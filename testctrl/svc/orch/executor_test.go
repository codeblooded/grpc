package orch

import (
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeFake "k8s.io/client-go/kubernetes/fake"
	corev1Fake "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/grpc/grpc/testctrl/svc/types"
)

func TestExecutorExecute(t *testing.T) {
	type action struct {
		targetKind types.ComponentKind
		status     corev1.ContainerStatus
	}

	cases := []struct {
		description string
		actions     []action
		errors      bool
	}{
		{
			description: "provision successful",
			actions: []action{
				{types.ServerComponent, ContainerStatus.running()},
				{types.ClientComponent, ContainerStatus.running()},
				{types.DriverComponent, ContainerStatus.running()},
				{types.DriverComponent, ContainerStatus.terminated(0)},
			},
			errors: false,
		},
	}

	for _, c := range cases {
		kubeSim := newKubeSimulator()

		watcher := NewWatcher(kubeSim)
		watcher.Start()
		defer watcher.Stop()

		driver := types.NewComponent(testContainerImage, types.DriverComponent)
		server := types.NewComponent(testContainerImage, types.ServerComponent)
		client := types.NewComponent(testContainerImage, types.ClientComponent)
		session := types.NewSession(driver, []*types.Component{server, client}, nil)

		go func() {
			time.Sleep(500 * time.Millisecond)

			for _, a := range c.actions {
				var target *types.Component

				switch a.targetKind {
				case types.DriverComponent:
					target = driver
				case types.ServerComponent:
					target = server
				case types.ClientComponent:
					target = client
				}

				if err := kubeSim.setStatus(target, a.status); err != nil {
					t.Fatalf("could not broadcast container status")
				}
			}
		}()

		e := newExecutor(0, kubeSim, watcher)
		err := e.Execute(session)
		if c.errors && err == nil {
			t.Errorf("case '%v' did not return an error", c.description)
		} else if !c.errors && err != nil {
			t.Errorf("case '%v' unexpectedly returned an error: %v", c.description, err)
		}
	}
}

func TestExecutorProvision(t *testing.T) {
	// test provision with successful pods
	fakePodInf := newFakePodInterface(t)

	driver := types.NewComponent(testContainerImage, types.DriverComponent)
	server := types.NewComponent(testContainerImage, types.ServerComponent)
	client := types.NewComponent(testContainerImage, types.ClientComponent)

	components := []*types.Component{server, client, driver}
	session := types.NewSession(driver, components[:2], nil)

	e := newExecutor(0, fakePodInf, nil)
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

	e = newExecutor(0, fakePodInf, nil)
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
		t.Errorf("unexpected error with failing pod")
	}
}

func TestExecutorMonitor(t *testing.T) {
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

	for _, c := range cases {
		fakePodInf := newFakePodInterface(t)

		driver := types.NewComponent(testContainerImage, types.DriverComponent)
		server := types.NewComponent(testContainerImage, types.ServerComponent)
		client := types.NewComponent(testContainerImage, types.ClientComponent)

		components := []*types.Component{server, client, driver}
		session := types.NewSession(driver, components[:2], nil)

		e := newExecutor(0, fakePodInf, nil)
		eventChan := make(chan *PodWatchEvent)
		e.eventChan = eventChan
		e.session = session

		go func() {
			eventChan <- &PodWatchEvent{
				SessionName:   session.Name,
				ComponentName: driver.Name,
				Pod:           c.event.Pod,
				PodIP:         c.event.PodIP,
				Health:        c.event.Health,
				Error:         c.event.Error,
			}
		}()

		err := e.monitor()
		if err == nil && c.errors {
			t.Errorf("case '%v' did not return error", c.description)
		} else if err != nil && !c.errors {
			t.Errorf("case '%v' unexpectedly returned error '%v'", c.description, err)
		}
	}
}

// TODO(@codeblooded): Refactor clean method, or choose to not test this method
//func TestExecutorClean(t *testing.T) {
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
//	e := newExecutor(0, fakePodInf, nil)
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

func newKubernetesFake(t *testing.T) *kubeFake.Clientset {
	t.Helper()
	return kubeFake.NewSimpleClientset()
}

func newFakePodInterface(t *testing.T) *corev1Fake.FakePods {
	t.Helper()
	return newKubernetesFake(t).CoreV1().Pods(corev1.NamespaceDefault).(*corev1Fake.FakePods)
}
