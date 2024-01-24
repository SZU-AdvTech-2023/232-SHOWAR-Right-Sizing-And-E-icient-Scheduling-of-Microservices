package main

import (
	"context"
	"fmt"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"math"
	"scheduling/autoscaling"
	"scheduling/prometheus"
	"scheduling/util"
	"strconv"
	"time"
)

const nameSpace string = util.Namespace

func main() {
	// 创建 Kubernetes 客户端
	clientset := util.GenerateClient()
	// 水平扩展每分钟跑一次，每次评估过去5分钟的资源
	for {
		// 获取所有deploy
		deploys, err := clientset.AppsV1().Deployments(nameSpace).List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			panic(err.Error())
		}
		// 遍历并打印 Pod 信息
		for _, deploy := range deploys.Items {
			// 跳过负载生成器
			if deploy.Name == "loadgenerator" {
				continue
			}
			// 使用 Label Selector 查询 Pod 列表
			pods, err := util.ObtainPodByDeploy(deploy, clientset)
			if err != nil {
				panic(err)
			}
			// 首先判断pod个数是否为1，如果是启动垂直伸缩
			if len(pods.Items) == 1 {
				pod := pods.Items[0]
				// 获取改pod的节点资源
				onePodUpdater(&pod, &deploy, clientset)
				//continue
			}
			//if true {
			//	continue
			//}
			// 获取服务
			serviceName, err := util.GetDeploymentServiceName(clientset, deploy.Name, deploy.Namespace)
			if err != nil {
				println(deploy.Name + "没有服务，跳过")
				continue
			}
			// 获取正常情况下的平均值
			//avgLatency, _ := strconv.ParseFloat(deploy.Annotations["avgLatency"], 64)
			// 延迟在1000ms内比较正常
			//avgLatency := 0.95
			latency, err := prometheus.ObtainMetricNetworkSuccessLv(serviceName, 1, 60)
			if err != nil {
				println(err.Error())
				continue
			}
			horizontals := autoscaling.MeanValue(latency)
			if math.IsNaN(horizontals) {
				continue
			}
			// 水平扩展的目标值
			if horizontals < 0.95 {
				// 水平扩展
				replicas := int32(*deploy.Spec.Replicas) + 1
				replicas = int32(math.Min(10.0, float64(replicas)))
				deploy.Spec.Replicas = &replicas
				printAction(&deploy, "horizatal", strconv.FormatInt(int64(replicas-1), 10), strconv.FormatInt(int64(replicas), 10))

			} else if horizontals > 0.99 {
				replicas := int32(*deploy.Spec.Replicas) - 1
				if replicas <= 0 {
					replicas = 1
				}
				deploy.Spec.Replicas = &replicas

				printAction(&deploy, "horizatal", strconv.FormatInt(int64(replicas+1), 10), strconv.FormatInt(int64(replicas), 10))

			}
			_, err = clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), &deploy, metav1.UpdateOptions{})

		}
		// 每分钟跑一次
		time.Sleep(60 * time.Second)
	}

}

func onePodUpdater(pod *v1.Pod, deploy *v12.Deployment, clientset *kubernetes.Clientset) bool {
	// 扩展资源
	// 更新每个容器
	// 返回值判断是否需要进行 水平伸缩

	for i, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			// istio-proxy 为代理，跳过
			continue
		}
		cpu, err := prometheus.ObtainContainerMetricCpu(pod.ObjectMeta.Name, container.Name, 1, 60*5)
		memory, err := prometheus.ObtainContainerMetricMemory(pod.ObjectMeta.Name, container.Name, 1, 60*5)

		if err != nil {
			print(err.Error())
			continue
		}
		// 计算垂直计算的值
		verticleCpuCompute := autoscaling.Verticle(cpu)
		verticleMemCompute := autoscaling.Verticle(memory)

		lastCpuVerticle, err := strconv.ParseFloat(pod.ObjectMeta.Annotations["verticle-cpu-"+container.Name], 3)
		lastMemoryVerticle, err := strconv.ParseFloat(pod.ObjectMeta.Annotations["verticle-mem-"+container.Name], 3)

		if err != nil || lastCpuVerticle == 0.0 || lastMemoryVerticle == 0.0 {
			// 说明节点现在没有值，或者节点值的设置是错误的
			if err != nil {
				println(err.Error())

			}
			// 更新数据
			pod.ObjectMeta.Annotations["verticle-cpu-"+container.Name] = strconv.FormatFloat(verticleCpuCompute, 'f', 3, 64)
			pod.ObjectMeta.Annotations["verticle-mem-"+container.Name] = strconv.FormatFloat(verticleMemCompute, 'f', 3, 64)

			_, err = clientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})

			if err != nil {
				panic(err.Error())
			}
			continue
		}

		computedCpu := verticleCpuCompute
		// 确保最小资源
		computedCpu = math.Max(computedCpu, 0.128)

		computedMem := verticleMemCompute
		computedMem = computedMem / (1024 * 1024)
		// 确保最小资源
		computedMem = math.Max(computedMem, 128)

		// 获取limit
		limitCpu := float64(container.Resources.Limits.Cpu().MilliValue())
		limitCpu = limitCpu / 1000.0
		// mem的值需要确定一下
		_ = float64(container.Resources.Limits.Memory().Value())
		// cpu更新
		if verticleCpuCompute > lastCpuVerticle {
			// 判断是否超过15%
			if verticleCpuCompute > lastCpuVerticle*1.15 {
				if computedCpu > 1 {
					replicas := int32(*deploy.Spec.Replicas) + 1
					deploy.Spec.Replicas = &replicas
					deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(1.0, 'f', 3, 64))
					deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(1.0, 'f', 3, 64))
					_, err = clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
					printAction(deploy, "horizatal", strconv.FormatInt(int64(replicas-1), 10), strconv.FormatInt(int64(replicas), 10))
					return true
				}

				deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 3, 64))
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 3, 64))
				deploy.Spec.Template.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
				if deploy.Spec.Template.ObjectMeta.Annotations == nil {
					deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				deploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+container.Name] = strconv.FormatFloat(verticleCpuCompute, 'f', 3, 64)
				printAction(deploy, "verticle-cpu", strconv.FormatFloat(lastCpuVerticle, 'f', 3, 64), strconv.FormatFloat(computedCpu, 'f', 3, 64))
			}
		} else if verticleCpuCompute < lastCpuVerticle {
			if verticleCpuCompute < lastCpuVerticle*0.55 {
				// 收缩资源
				// 更新每个容器
				// 保证最低的垂直伸缩
				settingCpu := compare(deploy.Annotations["minCpu"], resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 3, 64)))

				deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = settingCpu
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = settingCpu
				deploy.Spec.Template.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
				if deploy.Spec.Template.ObjectMeta.Annotations == nil {
					deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				deploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+container.Name] = strconv.FormatFloat(verticleCpuCompute, 'f', 3, 64)
				printAction(deploy, "verticle-cpu", strconv.FormatFloat(lastCpuVerticle, 'f', 3, 64), strconv.FormatFloat(computedCpu, 'f', 3, 64))
			}
		}
		// 内存更新
		if verticleMemCompute > lastMemoryVerticle {
			// 判断是否超过15%
			if verticleMemCompute > lastMemoryVerticle*1.15 {

				deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", int(computedMem)))
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", int(computedMem)))
				deploy.Spec.Template.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
				if deploy.Spec.Template.ObjectMeta.Annotations == nil {
					deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+container.Name] = strconv.FormatFloat(verticleMemCompute, 'f', 3, 64)
				printAction(deploy, "verticle-mem", strconv.FormatFloat(lastMemoryVerticle, 'f', 3, 64), strconv.FormatFloat(verticleMemCompute, 'f', 3, 64))
			}
		} else if verticleMemCompute < lastMemoryVerticle {
			if verticleMemCompute < lastMemoryVerticle*0.55 {
				// 收缩资源
				// 更新每个容器
				settingMemory := compare(deploy.Annotations["minMemory"], resource.MustParse(fmt.Sprintf("%fMi", computedMem)))
				deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceMemory] = settingMemory
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] = settingMemory
				deploy.Spec.Template.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
				if deploy.Spec.Template.ObjectMeta.Annotations == nil {
					deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+container.Name] = strconv.FormatFloat(verticleMemCompute, 'f', 3, 64)
				printAction(deploy, "verticle-mem", strconv.FormatFloat(lastMemoryVerticle, 'f', 3, 64), strconv.FormatFloat(verticleMemCompute, 'f', 3, 64))
			}
		}
	}

	_, err := clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
	if err != nil {
		print(err.Error())
	}
	return false
}

func compare(minCpuStr string, settingCpu resource.Quantity) resource.Quantity {
	if minCpuStr == "" {
		return settingCpu
	}
	minCpu := resource.MustParse(minCpuStr)

	if minCpu.Cmp(settingCpu) > 1 {
		settingCpu = minCpu
	}
	return settingCpu
}

func printAction(deploy *v12.Deployment, action string, from string, to string) {
	fmt.Println("Deploy\tAction\tFrom\tTo")
	fmt.Println(deploy.Name + "\t" + action + "\t" + from + "\t" + to)
}
