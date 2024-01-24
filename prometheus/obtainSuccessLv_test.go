package prometheus

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"scheduling/util"
	"testing"
)

func TestObtainSuccessLv(t *testing.T) {
	clientset := util.GenerateClient()
	deploys, _ := clientset.AppsV1().Deployments(util.Namespace).List(context.TODO(), metav1.ListOptions{})
	for _, deploy := range deploys.Items {
		serviceName, err := util.GetDeploymentServiceName(clientset, deploy.Name, deploy.Namespace)
		if err != nil {
			println(deploy.Name + "没有服务，跳过")
			continue
		}
		println(ObtainSuccessLv(serviceName).Data.Result)
	}
}
func TestObtainMetricNetworkSuccessLv(t *testing.T) {
	clientset := util.GenerateClient()
	deploys, _ := clientset.AppsV1().Deployments(util.Namespace).List(context.TODO(), metav1.ListOptions{})
	for _, deploy := range deploys.Items {
		serviceName, err := util.GetDeploymentServiceName(clientset, deploy.Name, deploy.Namespace)
		if err != nil {
			println(deploy.Name + "没有服务，跳过")
			continue
		}
		println(ObtainMetricNetworkSuccessLv(serviceName, 1, 20))
	}
}
