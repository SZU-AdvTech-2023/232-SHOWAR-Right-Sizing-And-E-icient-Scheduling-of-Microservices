package util

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// UpdateAnnotation  更新pod annotation
func UpdateAnnotation(pod *v1.Pod, key string, value string, clientset *kubernetes.Clientset) (*v1.Pod, error) {
	pod.ObjectMeta.Annotations[key] = value
	return clientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
}
