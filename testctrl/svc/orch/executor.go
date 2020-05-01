package orch

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/grpc/grpc/testctrl/svc/store"
	"github.com/grpc/grpc/testctrl/svc/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Executor executes sessions, returning an error if there is a problem with the infrastructure.
// Executors are expected to provision, monitor and clean up resources related to a session.
// An error is not indicative of a success or failure of the tests, but signals an orchestration
// issue.
type Executor interface {
	Execute(*types.Session) error
}

type kubeExecutor struct {
	name      string
	watcher   *Watcher
	eventChan <-chan *PodWatchEvent
	session   *types.Session
	pcd       podCreateDeleter
	store     store.Store
}

func newKubeExecutor(index int, pcd podCreateDeleter, watcher *Watcher, store store.Store) *kubeExecutor {
	return &kubeExecutor{
		name:    fmt.Sprintf("%d", index),
		watcher: watcher,
		pcd:     pcd,
		store:   store,
	}
}

func (k *kubeExecutor) Execute(session *types.Session) error {
	k.setSession(session)
	k.writeEvent(types.AcceptEvent, nil, "kubernetes executor %v assigned session %v", k.name, session.Name)

	k.writeEvent(types.ProvisionEvent, nil, "started provisioning components for session")
	err := k.provision()
	if err != nil {
		err = fmt.Errorf("failed to provision: %v", err)
		goto endSession
	}

	k.writeEvent(types.RunEvent, nil, "all containers appear healthy, monitoring during tests")
	err = k.monitor()
	if err != nil {
		err = fmt.Errorf("failed during test: %v", err)
	}

endSession:
	logs, logErr := k.getDriverLogs()
	if logErr != nil {
		logErr = fmt.Errorf("failed to fetch logs from driver: %v", logErr)
	}

	cleanErr := k.clean()
	if cleanErr != nil {
		cleanErr = fmt.Errorf("failed to tear-down resources: %v", cleanErr)
	}

	if logErr != nil {
		k.writeEvent(types.InternalErrorEvent, nil, logErr.Error())
	}

	if cleanErr != nil {
		k.writeEvent(types.InternalErrorEvent, logs, cleanErr.Error())
	}

	if err != nil {
		k.writeEvent(types.ErrorEvent, logs, err.Error())
		return err
	}

	k.writeEvent(types.DoneEvent, logs, "session completed, driver container had exit status of 0")
	return nil
}

func (k *kubeExecutor) provision() error {
	var components []*types.Component
	var workerIPs []string

	components = append(components, k.session.ServerWorkers()...)
	components = append(components, k.session.ClientWorkers()...)
	components = append(components, k.session.Driver)

	for _, component := range components {
		kind := strings.ToLower(component.Kind.String())

		if component.Kind == types.DriverComponent {
			component.Env["QPS_WORKERS"] = strings.Join(workerIPs, ",")
		}

		glog.Infof("kubeExecutor[%v]: creating %v component %v", k.name, kind, component.Name)

		pod := newSpecBuilder(k.session, component).Pod()
		if _, err := k.pcd.Create(pod); err != nil {
			return fmt.Errorf("could not create %v component %v: %v", component.Name, kind, err)
		}

		for {
			select {
			case event := <-k.eventChan:
				switch event.Health {
				case Ready:
					ip := event.PodIP
					if len(ip) > 0 {
						host := ip + ":10000"
						workerIPs = append(workerIPs, host)
						glog.V(2).Infof("kubeExecutor[%v]: component %v was assigned IP address %v",
							k.name, event.ComponentName, ip)
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
		glog.V(2).Infof("kubeExecutor[%v]: %v component %v is now ready", k.name, kind, component.Name)
	}

	return nil
}

func (k *kubeExecutor) monitor() error {
	glog.Infof("kubeExecutor[%v]: monitoring components while session %v runs", k.name, k.session.Name)

	for {
		select {
		case event := <-k.eventChan:
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

func (k *kubeExecutor) clean() error {
	glog.Infof("kubeExecutor[%v]: deleting components for session %v", k.name, k.session.Name)

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("session-name=%v", k.session.Name),
	}

	err := k.pcd.DeleteCollection(&metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return fmt.Errorf("unable to delete components: %v", err)
	}

	return nil
}

func (k *kubeExecutor) getDriverLogs() ([]byte, error) {
	return k.getLogs(k.session.Driver.Name)
}

func (k *kubeExecutor) getLogs(podName string) ([]byte, error) {
	req := k.pcd.GetLogs(podName, &corev1.PodLogOptions{})
	return req.DoRaw()
}

func (k *kubeExecutor) setSession(session *types.Session) {
	eventChan, _ := k.watcher.Subscribe(session.Name)
	k.eventChan = eventChan
	k.session = session
}

func (k *kubeExecutor) writeEvent(kind types.EventKind, driverLogs []byte, descriptionFmt string, args ...interface{}) {
	description := fmt.Sprintf(descriptionFmt, args...)

	if k.store != nil {
		k.store.StoreEvent(k.session.Name, &types.Event{
			SubjectName: k.session.Name,
			Kind:        kind,
			Time:        time.Now(),
			Description: description,
			DriverLogs:  driverLogs,
		})
	}

	if kind == types.InternalErrorEvent || kind == types.ErrorEvent {
		glog.Errorf("kubeExecutor[%v]: [%v]: %v", k.name, kind, description)
	} else {
		glog.Infof("kubeExecutor[%v]: [%v]: %v", k.name, kind, description)
	}
}
