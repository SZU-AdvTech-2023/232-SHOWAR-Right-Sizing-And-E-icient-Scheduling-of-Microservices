// main.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"path/filepath"
	"scheduling/util"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/homedir"
)

type RecordData struct {
	Time      int64 `json:"time"`
	FailTimes int64 `json:"fail_times"`
}

var records []RecordData
var times = 0

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
				// 检查Pod是否处于失败状态，根据您的需求定义失败状态的条件
				if isPodInFailedState(pod) {
					times += 1
					//
					records = append(records, RecordData{
						FailTimes: int64(times),
						Time:      time.Now().Unix(),
					})
					//
					outputFile(records)
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
func outputFile(data []RecordData) {
	// 将数组序列化为JSON格式
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("JSON序列化错误:", err)
		return
	}

	// 将JSON数据写入文件
	err = ioutil.WriteFile("data2.json", jsonData, 0644)
	if err != nil {
		fmt.Println("写入文件错误:", err)
		return
	}

	fmt.Println("数据已成功写入文件.")
}
