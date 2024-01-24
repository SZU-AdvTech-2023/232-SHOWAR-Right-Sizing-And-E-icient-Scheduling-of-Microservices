// main.go
package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"path/filepath"
	"scheduling/util"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/homedir"
)

func createClient() (*kubernetes.Clientset, error) {
	kubeconfigPath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
func createPodWatcher(clientset *kubernetes.Clientset, namespace string) cache.Controller {
	listWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		namespace,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		listWatcher,
		&v1.Pod{},
		0, // resyncPeriod
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, obj interface{}) {
				pod := obj.(*v1.Pod)

				if checkPodProbe(pod) == false {
					// 给pod的request和limits增加相关
					deployment, err := util.FindDeploymentForPod(clientset, pod)
					if err != nil {
						println(err.Error())
						return
					}
					// 获取deployment

					for i := range deployment.Spec.Template.Spec.Containers {
						millCpuValue := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu().MilliValue()
						millCpuValue = millCpuValue + 100
						millMemoryValue := deployment.Spec.Template.Spec.Containers[i].Resources.Requests.Memory().Value()
						millMemoryValue = millMemoryValue + 50*1024*1024
						deployment.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", millCpuValue))
						deployment.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", millCpuValue))
						deployment.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%d", millMemoryValue))
						deployment.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%d", millMemoryValue))
						deployment.Annotations["minCpu"] = fmt.Sprintf("%dm", millCpuValue)
						deployment.Annotations["minMemory"] = fmt.Sprintf("%d", millMemoryValue)

					}
					_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
					if err != nil {
						println(err.Error())
					}
				}
			},
		},
	)

	return controller
}
func checkPodProbe(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		readinessProbeStatus := containerStatus.Ready
		count := containerStatus.RestartCount
		if readinessProbeStatus == false && count >= 2 {
			return false
		}
	}

	return true
}
func main() {
	clientset, err := createClient()
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	namespace := util.Namespace
	stopCh := make(chan struct{})
	defer close(stopCh)

	controller := createPodWatcher(clientset, namespace)

	go controller.Run(stopCh)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
