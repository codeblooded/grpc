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

type executor struct {
	name      string
	watcher   *Watcher
	eventChan <-chan *PodWatchEvent
	session   *types.Session
	pcd       podCreateDeleter
}

func newExecutor(index int, pcd podCreateDeleter, watcher *Watcher) *executor {
	return &executor{
		name:    fmt.Sprintf("%d", index),
		watcher: watcher,
		pcd:     pcd,
	}
}

func (e *executor) Execute(session *types.Session) error {
	var err error

	e.setSession(session)

	if err = e.provision(); err != nil {
		err = fmt.Errorf("failed to provision: %v", err)
		goto endSession
	}

	if err = e.monitor(); err != nil {
		err = fmt.Errorf("failed during test: %v", err)
		goto endSession
	}

endSession:
	if logs, err := e.getDriverLogs(); err == nil {
		glog.Infof("executor[%v]: found logs for component (driver) %v: %s",
			e.name, e.session.Driver.Name, logs)
	}

	if err = e.clean(); err != nil {
		glog.Errorf("executor[%v]: failed to teardown resources for session %v: %v",
			e.name, session.Name, err)
	}

	return err
}

func (e *executor) provision() error {
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

		pod := newSpecBuilder(e.session, component).Pod()
		if _, err := e.pcd.Create(pod); err != nil {
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

func (e *executor) monitor() error {
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

func (e *executor) clean() error {
	glog.Infof("executor[%v]: deleting components for session %v", e.name, e.session.Name)

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("session-name=%v", e.session.Name),
	}

	err := e.pcd.DeleteCollection(&metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return fmt.Errorf("unable to delete components: %v", err)
	}

	return nil
}

func (e *executor) getDriverLogs() ([]byte, error) {
	return e.getLogs(e.session.Driver.Name)
}

func (e *executor) getLogs(podName string) ([]byte, error) {
	req := e.pcd.GetLogs(podName, &corev1.PodLogOptions{})
	return req.DoRaw()
}

func (e *executor) setSession(session *types.Session) {
	eventChan, _ := e.watcher.Subscribe(session.Name)
	e.eventChan = eventChan
	e.session = session
}
