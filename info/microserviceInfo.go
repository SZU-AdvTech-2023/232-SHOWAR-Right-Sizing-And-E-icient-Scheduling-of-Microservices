package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"scheduling/prometheus"
	"scheduling/util"
)

const namespace = util.Namespace

func main() {
	client := util.GenerateClient()
	deployments, _ := client.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	// 表格标题

	for _, deployment := range deployments.Items {
		println(deployment.Name)
		// 获取每个pod的资源使用率
		// 表格数据
		fmt.Printf("%-35s %-10s %-5s\n", "Name", "CPU(m)", "Memory(Mi)")
		fmt.Println("---------------------------------------------------------------")

		// 通过的deploy获取pod
		podList, err := util.ObtainPodByDeploy(deployment, client)
		if err != nil {
			panic(err)
		}
		for _, pod := range podList.Items {
			podName := pod.Name
			cpu := prometheus.RealTimeMetric("sum by(pod) (rate(container_cpu_usage_seconds_total{pod=~\"" + podName + "\", container!=\"\"}[1m])) ")
			mem := prometheus.RealTimeMetric("sum by(pod) (container_memory_working_set_bytes{pod=\"" + podName + "\", container!=\"\"})")
			fmt.Printf("%-35s %-10f %-5f\n", podName, cpu*1000, mem/(1024*1024))
		}
		fmt.Println("---------------------------------------------------------------")

		serviceName, err := util.GetDeploymentServiceName(client, deployment.Name, deployment.Namespace)
		if err != nil {
			println(deployment.Name + "没有服务，跳过")
			continue
		}
		networkMetric := prometheus.ObtainRealTimeNetworkMetric(serviceName)
		if len(networkMetric.Data.Result) == 0 {
			continue
		}
		fmt.Printf("%-15s %-20s %-10s\n", "NetworkMetric", serviceName, networkMetric.Data.Result[0].Value[1])
		fmt.Println("---------------------------------------------------------------")

	}
}
