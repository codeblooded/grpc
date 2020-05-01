package orch

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1Fake "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/grpc/grpc/testctrl/svc/types"
)

func TestKubeExecutorProvision(t *testing.T) {
	// test provision with successful pods
	fakePodInf := newFakePodInterface(t)

	driver := types.NewComponent(testContainerImage, types.DriverComponent)
	server := types.NewComponent(testContainerImage, types.ServerComponent)
	client := types.NewComponent(testContainerImage, types.ClientComponent)

	components := []*types.Component{server, client, driver}
	session := types.NewSession(driver, components[:2], nil)

	e := newKubeExecutor(0, fakePodInf, nil, nil)
	eventChan := make(chan *PodWatchEvent)
	e.eventChan = eventChan
	e.session = session

	go func() {
		for _, c := range components {
			eventChan <- &PodWatchEvent{
				SessionName:   session.Name,
				ComponentName: c.Name,
				Pod:           nil,
				PodIP:         "127.0.0.1",
				Health:        Ready,
				Error:         nil,
			}
		}
	}()

	if err := e.provision(); err != nil {
		t.Fatalf("unexpected error in provision: %v", err)
	}

	pods, err := listPods(t, fakePodInf)
	if err != nil {
		t.Fatalf("could not list pods from provision: %v", err)
	}

	expectedNames := []string{driver.Name, server.Name, client.Name}
	for _, en := range expectedNames {
		found := false

		for _, pod := range pods {
			if strings.Compare(pod.ObjectMeta.Name, en) == 0 {
				found = true
			}
		}

		if !found {
			t.Errorf("provision did not create pod for component %v", en)
		}
	}

	// test provision with an unsuccessful pod
	fakePodInf = newFakePodInterface(t)

	driver = types.NewComponent(testContainerImage, types.DriverComponent)
	server = types.NewComponent(testContainerImage, types.ServerComponent)
	client = types.NewComponent(testContainerImage, types.ClientComponent)

	components = []*types.Component{server, client, driver}
	session = types.NewSession(driver, components[:2], nil)

	e = newKubeExecutor(0, fakePodInf, nil, nil)
	eventChan = make(chan *PodWatchEvent)
	e.eventChan = eventChan
	e.session = session

	go func() {
		for _, c := range components[:2] {
			eventChan <- &PodWatchEvent{
				SessionName:   session.Name,
				ComponentName: c.Name,
				Pod:           nil,
				PodIP:         "127.0.0.1",
				Health:        Ready,
				Error:         nil,
			}
		}

		eventChan <- &PodWatchEvent{
			SessionName:   session.Name,
			ComponentName: components[2].Name,
			Pod:           nil,
			PodIP:         "",
			Health:        Failed,
			Error:         nil,
		}
	}()

	if err := e.provision(); err == nil {
		t.Errorf("expected error for failing pod, but returned nil")
	}
}

func TestKubeExecutorMonitor(t *testing.T) {
	cases := []struct {
		description string
		event       *PodWatchEvent
		errors      bool
	}{
		{
			description: "success event received",
			event:       &PodWatchEvent{Health: Succeeded},
			errors:      false,
		},
		{
			description: "failure event received",
			event:       &PodWatchEvent{Health: Failed},
			errors:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			fakePodInf := newFakePodInterface(t)

			driver := types.NewComponent(testContainerImage, types.DriverComponent)
			server := types.NewComponent(testContainerImage, types.ServerComponent)
			client := types.NewComponent(testContainerImage, types.ClientComponent)

			components := []*types.Component{server, client, driver}
			session := types.NewSession(driver, components[:2], nil)

			e := newKubeExecutor(0, fakePodInf, nil, nil)
			eventChan := make(chan *PodWatchEvent)
			e.eventChan = eventChan
			e.session = session

			go func() {
				eventChan <- &PodWatchEvent{
					SessionName:   session.Name,
					ComponentName: driver.Name,
					Pod:           tc.event.Pod,
					PodIP:         tc.event.PodIP,
					Health:        tc.event.Health,
					Error:         tc.event.Error,
				}
			}()

			err := e.monitor()
			if err == nil && tc.errors {
				t.Errorf("case '%v' did not return error", tc.description)
			} else if err != nil && !tc.errors {
				t.Errorf("case '%v' unexpectedly returned error '%v'", tc.description, err)
			}
		})
	}
}

// TODO(@codeblooded): Refactor clean method, or choose to not test this method
//func TestKubeExecutorClean(t *testing.T) {
//	fakePodInf := newFakePodInterface(t)
//
//	driver := types.NewComponent(testContainerImage, types.DriverComponent)
//	server := types.NewComponent(testContainerImage, types.ServerComponent)
//	client := types.NewComponent(testContainerImage, types.ClientComponent)
//
//	session := types.NewSession(driver, []*types.Component{server, client}, nil)
//	driverPod := newSpecBuilder(session, driver).Pod()
//	serverPod := newSpecBuilder(session, server).Pod()
//	clientPod := newSpecBuilder(session, client).Pod()
//
//	fakePodInf.Create(driverPod)
//	fakePodInf.Create(serverPod)
//	fakePodInf.Create(clientPod)
//
//	pods, err := listPods(t, fakePodInf)
//	if err != nil {
//		t.Fatalf("could not list pods: %v", err)
//	}
//	podCountBefore := len(pods)
//
//	e := newKubeExecutor(0, fakePodInf, nil, nil)
//	e.session = session
//	if err = e.clean(fakePodInf); err != nil {
//		t.Fatalf("returned an error unexpectedly: %v", err)
//	}
//
//	pods, err = listPods(t, fakePodInf)
//	if err != nil {
//		t.Fatalf("could not list pods: %v", err)
//	}
//	podCountAfter := len(pods)
//
//	if podCountAfter-podCountBefore != 3 {
//		t.Fatalf("deletion did not remove all pods, %v remain", podCountAfter)
//	}
//}

func listPods(t *testing.T, fakePodInf *corev1Fake.FakePods) ([]corev1.Pod, error) {
	t.Helper()

	podList, err := fakePodInf.List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("setup failed, could not fetch pod list from kubernetes fake")
	}
	return podList.Items, nil
}
