package autoscaling

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeValue struct {
	Limit int64
}

func ObtainNode(clientset *kubernetes.Clientset) map[string]NodeValue {
	// 获取所有节点列表
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	reMap := make(map[string]NodeValue)
	// 遍历节点并获取 CPU 上限值
	for _, node := range nodes.Items {
		nodeName := node.ObjectMeta.Name
		allocatable := node.Status.Allocatable
		cpuLimit := allocatable.Cpu().MilliValue() // 获取 CPU 上限值，以毫核（mCPU）为单位
		var nodeValue NodeValue
		fmt.Printf("Node: %s, CPU Limit: %d mCPU\n", nodeName, cpuLimit)
		nodeValue.Limit = cpuLimit
		reMap[nodeName] = nodeValue
	}
	return reMap
}
