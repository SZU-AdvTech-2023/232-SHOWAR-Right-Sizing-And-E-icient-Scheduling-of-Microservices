package util

import (
	"context"
	"fmt"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func GetDeploymentServiceName(clientset *kubernetes.Clientset, deploymentName, namespace string) (string, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// 获取服务名称标签
	serviceNameLabel, found := deployment.Spec.Template.Labels["app"]
	if !found {
		return "", fmt.Errorf("Service name label not found in Deployment")
	}

	return serviceNameLabel, nil
}
func FindDeploymentForPod(clientset *kubernetes.Clientset, pod *v1.Pod) (*appv1.Deployment, error) {
	podLabels := labels.Set(pod.Labels)
	deploymentList, err := clientset.AppsV1().Deployments(pod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, deployment := range deploymentList.Items {
		// 检查 Deployment 的 Selector 是否与 Pod 的标签匹配
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			return nil, err
		}
		if selector.Matches(podLabels) {
			return &deployment, nil
		}
	}

	return nil, fmt.Errorf("No matching Deployment found for the Pod")
}
