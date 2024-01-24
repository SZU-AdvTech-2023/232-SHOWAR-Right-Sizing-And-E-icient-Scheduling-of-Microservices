package main

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"scheduling/autoscaling"
	"scheduling/prometheus"
	"scheduling/util"
	"strconv"
	"strings"
	"time"
)

func test() {
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
	for {
		// 获取所有 Pod 列表
		pods, err := clientset.CoreV1().Pods("autoscale").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		// 获取所有节点的cpu的上限值
		nodeValueMap := autoscaling.ObtainNode(clientset)
		// 遍历并打印 Pod 信息
		for _, pod := range pods.Items {
			// 获取改pod的节点资源
			nodeValue := nodeValueMap[pod.Spec.NodeName]
			println(nodeValue.Limit)
			//autoscaling.Verticle()
			// 获取pod 过去10s内的cpu的值
			cpu, _ := prometheus.ObtainMetricCpu(pod.ObjectMeta.Name, 1, 10)
			// 计算垂直计算的值
			verticleCompute := autoscaling.Verticle(cpu)

			lastVerticle, err := strconv.ParseFloat(pod.ObjectMeta.Annotations["verticle"], 3)
			if err != nil { // 说明节点现在没有值，或者节点值的设置是错误的
				print(err.Error())
				// 更新annotation数据
				_, err := util.UpdateAnnotation(&pod, "verticle", strconv.FormatFloat(verticleCompute, 'f', 3, 64), clientset)
				if err != nil {
					panic(err.Error())
				}
				continue
			}
			print(lastVerticle)
			if verticleCompute > lastVerticle {
				// 判断是否超过15%
				if verticleCompute > lastVerticle*1.15 {
					// 扩展资源
					// 更新每个容器
					for i, container := range pod.Spec.Containers {
						cpuVal, _ := container.Resources.Requests.Cpu().AsInt64()
						computedCpu := float64(cpuVal) * 0.9 * (1.0 + 0.15)
						if computedCpu == 0 {
							computedCpu = 1.0
						}
						if computedCpu > 1000 {
							autoscaling.Horizontal(cpu, 0, nodeValue)
							continue
						}
						if pod.Spec.Containers[i].Resources.Requests == nil {
							pod.Spec.Containers[i].Resources.Requests = make(v1.ResourceList)
						}
						if pod.Spec.Containers[i].Resources.Limits == nil {
							pod.Spec.Containers[i].Resources.Limits = make(v1.ResourceList)
						}
						pod.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 1, 64))
						pod.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 1, 64))

						//newPodScale, err = c.podScalesClientset.SystemautoscalerV1beta1().PodScales(podScale.Namespace).Update(context.TODO(), podScale, *opts)
					}

				}
			} else if verticleCompute < lastVerticle {
				if verticleCompute < lastVerticle*0.85 {
					// 收缩资源
					// 更新每个容器
					for i, container := range pod.Spec.Containers {
						quantity := container.Resources.Requests[v1.ResourceCPU]
						cpuVal := quantity.String()
						computedCpu := parseResourceToFloat(cpuVal) * 0.9 * (1.0 - 0.15)
						if computedCpu == 0 {
							computedCpu = 1.0
						}
						strconv.FormatFloat(computedCpu, 'f', 3, 64)
						pod.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse("5")

					}
				}
			}

			// 设置节点标签
			pod.ObjectMeta.Annotations["verticle"] = strconv.FormatFloat(verticleCompute, 'f', 3, 64)

			// 提交更新后的 Pod 对象
			_, err = clientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), &pod, metav1.UpdateOptions{
				DryRun: []string{metav1.DryRunAll},
			})
			if err != nil {
				panic(err.Error())
			}
		}
		// 不断循环
		time.Sleep(10 * time.Second)
	}

}
func parseResourceToFloat(resource string) float64 {
	// 解析资源字符串并将其转换为浮点数
	// 例如，将 "100m" 转换为 0.1
	if strings.HasSuffix(resource, "m") {
		value, err := strconv.ParseFloat(strings.TrimSuffix(resource, "m"), 64)
		if err != nil {
			panic(err)
		}
		return value / 1000.0 // 转换为核
	}

	// 如果没有单位，将其视为核
	value, err := strconv.ParseFloat(resource, 64)
	if err != nil {
		panic(err)
	}
	return value
}
