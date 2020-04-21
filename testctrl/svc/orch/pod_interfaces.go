package orch

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type PodCreator interface {
	Create(*corev1.Pod) (*corev1.Pod, error)
}

type PodDeleter interface {
	DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
}

type PodLogGetter interface {
	GetLogs(podName string, opts *corev1.PodLogOptions) *rest.Request
}

type PodCreateDeleter interface {
	PodCreator
	PodDeleter
	PodLogGetter
}

type PodWatcher interface {
	Watch(metav1.ListOptions) (watch.Interface, error)
}

type PodInterface interface {
	PodCreateDeleter
	PodWatcher
}
