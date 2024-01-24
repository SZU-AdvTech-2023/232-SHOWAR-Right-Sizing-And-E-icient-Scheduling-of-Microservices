package main

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"scheduling/prometheus"
	"scheduling/util"
	"strconv"
)

func main() {
	// 遍历所有deploy
	clientset := util.GenerateClient()
	deployments, _ := clientset.AppsV1().Deployments("default").List(context.TODO(), metav1.ListOptions{})
	for _, deployment := range deployments.Items {
		// 获取过去5分钟内的平均响应时间
		serviceName, err := util.GetDeploymentServiceName(clientset, deployment.Name, deployment.Namespace)
		if err != nil {
			println(deployment.Name + "没有服务，跳过")
			continue
		}
		latency, err := prometheus.ObtainMetricNetworkLatency(serviceName, 14, 60*5)
		// 计算平均值
		avgLatency := 0.0
		for _, l := range latency {
			avgLatency += l
		}
		avgLatency = avgLatency / float64(len(latency))
		// 打上标签
		deployment.Annotations["avgLatency"] = strconv.FormatFloat(avgLatency, 'f', 3, 64)
		_, err = clientset.AppsV1().Deployments("default").Update(context.TODO(), &deployment, metav1.UpdateOptions{})
	}
}
