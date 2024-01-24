package util

import (
	"context"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ObtainPodByDeploy(deploy v12.Deployment, clientset *kubernetes.Clientset) (*v1.PodList, error) {
	selector := metav1.LabelSelector{MatchLabels: map[string]string{"app": deploy.Spec.Selector.MatchLabels["app"]}}
	options := metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(&selector)}
	pods, err := clientset.CoreV1().Pods(deploy.Namespace).List(context.TODO(), options)
	return pods, err
}
