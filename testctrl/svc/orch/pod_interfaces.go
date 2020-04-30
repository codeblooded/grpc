package orch

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type podCreator interface {
	Create(*corev1.Pod) (*corev1.Pod, error)
}

type podDeleter interface {
	DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
}

type podLogGetter interface {
	GetLogs(podName string, opts *corev1.PodLogOptions) *rest.Request
}

type podCreateDeleter interface {
	podCreator
	podDeleter
	podLogGetter
}

type podWatcher interface {
	Watch(metav1.ListOptions) (watch.Interface, error)
}
