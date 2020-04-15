package orch

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/grpc/grpc/testctrl/svc/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Executor struct {
	name      string
	watcher   *Watcher
	eventChan <-chan *PodWatchEvent
	session   *types.Session
	pcd       PodCreateDeleter
}

func NewExecutor(index int, pcd PodCreateDeleter, watcher *Watcher) *Executor {
	return &Executor{
		name:    fmt.Sprintf("%d", index),
		watcher: watcher,
		pcd:     pcd,
	}
}

func (e *Executor) Execute(session *types.Session) error {
	eventChan, _ := e.watcher.Subscribe(session.Name)
	e.eventChan = eventChan

	var err error
	re := &RuntimeError{
		Executor: e,
		Session:  session,
	}

	e.session = session

	if err = e.provision(e.pcd); err != nil {
		re.Add(fmt.Errorf("failed to provision: %v", err))
		goto endSession
	}

	if err = e.monitor(); err != nil {
		re.Add(fmt.Errorf("failed during test: %v", err))
		goto endSession
	}

endSession:
	if logs, err := e.getDriverLogs(e.pcd); err == nil {
		glog.Infof("executor[%v]: found logs for component (driver) %v: %s",
			e.name, e.session.Driver.Name, logs)
	}

	if err = e.clean(e.pcd); err != nil {
		re.Add(fmt.Errorf("failed to teardown resources: %v", err))
	}

	return re.Wrap()
}

func (e *Executor) provision(pc PodCreator) error {
	var components []*types.Component
	var workerIPs []string

	components = append(components, e.session.ServerWorkers()...)
	components = append(components, e.session.ClientWorkers()...)
	components = append(components, e.session.Driver)

	for _, component := range components {
		kind := strings.ToLower(component.Kind.String())

		if component.Kind == types.DriverComponent {
			component.Env["QPS_WORKERS"] = strings.Join(workerIPs, ",")
		}

		glog.Infof("executor[%v]: creating %v component %v", e.name, kind, component.Name)

		pod := NewSpecBuilder(e.session, component).Pod()
		if _, err := pc.Create(pod); err != nil {
			return fmt.Errorf("could not create %v component %v: %v", component.Name, kind, err)
		}

		for {
			select {
			case event := <-e.eventChan:
				switch event.Health {
				case Ready:
					ip := event.PodIP
					if len(ip) > 0 {
						host := ip + ":10000"
						workerIPs = append(workerIPs, host)
						glog.V(2).Infof("executor[%v]: component %v was assigned IP address %v",
							e.name, event.ComponentName, ip)
						goto componentProvisioned

					}
				case Failed:
					return fmt.Errorf("provision failed due to component %v: %v", event.ComponentName, event.Error)
				default:
					continue
				}
			// TODO(#54): Add timeout/deadline for provisioning resources
			default:
				time.Sleep(1 * time.Second)
			}
		}

	componentProvisioned:
		glog.V(2).Infof("executor[%v]: %v component %v is now ready", e.name, kind, component.Name)
	}

	return nil
}

func (e *Executor) monitor() error {
	glog.Infof("executor[%v]: monitoring components while session %v runs", e.name, e.session.Name)

	for {
		select {
		case event := <-e.eventChan:
			switch event.Health {
			case Succeeded:
				return nil // no news is good news :)
			case Failed:
				return fmt.Errorf("component %v has failed: %v", event.ComponentName, event.Error)
			}

			// TODO(#54): Add timeout/deadline for test execution (see concerns on GitHub)
		}
	}
}

func (e *Executor) clean(pd PodDeleter) error {
	glog.Infof("executor[%v]: deleting components for session %v", e.name, e.session.Name)

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("session-name=%v", e.session.Name),
	}

	err := pd.DeleteCollection(&metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return fmt.Errorf("unable to delete components: %v", err)
	}

	return nil
}

func (e *Executor) getDriverLogs(plg PodLogGetter) ([]byte, error) {
	return e.getLogs(plg, e.session.Driver.Name)
}

func (e *Executor) getLogs(plg PodLogGetter, podName string) ([]byte, error) {
	req := plg.GetLogs(podName, &corev1.PodLogOptions{})
	return req.DoRaw()
}

type RuntimeError struct {
	Executor     *Executor
	Session      *types.Session
	Component    *types.Component
	Message      string
	NestedErrors []error
}

func (re *RuntimeError) Wrap() error {
	if len(re.NestedErrors) < 1 {
		return nil
	}

	return re
}

func (re *RuntimeError) Add(err error) {
	re.NestedErrors = append(re.NestedErrors, err)
}

func (re *RuntimeError) Error() string {
	b := &strings.Builder{}

	if re.Executor != nil {
		b.WriteString(fmt.Sprintf("executor[%v]: ", re.Executor.name))
	}

	if re.Session != nil {
		b.WriteString(fmt.Sprintf("session %v ", re.Session.Name))
	}

	if re.Component != nil {
		b.WriteString(fmt.Sprintf("%v component %v ",
			strings.ToLower(re.Component.Kind.String()), re.Component.Name))
	}

	count := len(re.NestedErrors)
	b.WriteString(fmt.Sprintf("had %d errors: ", count))

	for i, e := range re.NestedErrors {
		b.WriteString(fmt.Sprintf("(%d) %v", i+1, e.Error()))
		if (i + 1) != count {
			b.WriteString(", ")
		}
	}

	return b.String()
}
