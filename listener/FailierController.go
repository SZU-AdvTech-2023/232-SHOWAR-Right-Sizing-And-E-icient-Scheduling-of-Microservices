// main.go
package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
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
				oldPod := oldObj.(*v1.Pod)
				pod := obj.(*v1.Pod)
				// 检查Pod是否处于失败状态，根据您的需求定义失败状态的条件
				if isPodInFailedState(pod) {
					println("update one")
					println("old pod:" + oldPod.Name)
					println("new pod:" + pod.Name)
					// 处理Pod失败的逻辑
					fmt.Printf("Pod %s in namespace %s has failed\n", pod.Name, pod.Namespace)
					// 直接对失败的pod进行一次扩容, 超过10个则不扩容
					deployment, err := util.FindDeploymentForPod(clientset, pod)
					if err != nil {
						println(err)
						return
					}
					if *deployment.Spec.Replicas > 10 {
						return
					}
					replicas := *deployment.Spec.Replicas
					replicas = replicas + 1
					deployment.Spec.Replicas = &replicas
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
func isPodInFailedState(pod *v1.Pod) bool {
	// 检查Pod的Phase是否为Failed
	if pod.Status.Phase == v1.PodFailed {
		return true
	}

	// 检查每个容器的状态
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == "CrashLoopBackOff" {
			return true
		}
	}

	// 如果没有任何条件符合，可根据自定义的其他标志或指标来判断
	// 这里可以添加额外的逻辑

	return false
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
