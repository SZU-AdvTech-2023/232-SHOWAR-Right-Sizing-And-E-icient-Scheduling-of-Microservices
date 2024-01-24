package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"scheduling/autoscaling"
	"testing"
)

func TestObtainNode(t *testing.T) {
	// 使用命令行标志来指定 kubeconfig 文件路径
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		kubeconfig = "/Users/username/.kube/config" // 替换为你的 kubeconfig 路径
	}

	// 创建 Kubernetes 客户端配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	autoscaling.ObtainNode(clientset)
}
