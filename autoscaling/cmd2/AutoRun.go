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
			if deploy.Name == "loadgenerator" || deploy.Name == "pythonclient" || deploy.Name == "my-job3" || deploy.Name == "monitoring-mem" {
				continue
			}
			// 首先判断pod个数是否为1，如果是启动垂直伸缩
			// 获取改pod的节点资源
			keep := onePodUpdater(&deploy, clientset)
			// 获取服务
			serviceName, err := util.GetDeploymentServiceName(clientset, deploy.Name, deploy.Namespace)
			if err != nil {
				println(deploy.Name + "没有服务，跳过")
				continue
			}
			// 获取服务延迟
			//avgLatency, _ := strconv.ParseFloat(deploy.Annotations["avgLatency"], 64)
			// 延迟在1000ms内比较正常
			//avgLatency := 0.95
			latency, err := prometheus.ObtainMetricNetworkLatency(serviceName, 1, 60)
			// 获取请求成功率
			successLv, err := prometheus.ObtainMetricNetworkSuccessLv(serviceName, 1, 60)

			if err != nil {
				println(err.Error())
				continue
			}
			// 获取水平伸缩值
			horizontals := autoscaling.MeanValue(latency)
			// 计算平均请求成功率
			meanSuccessLv := autoscaling.MeanValue(successLv)

			horizontalsNum := horizontals / 200
			horizontalsNum = math.Max(horizontalsNum, 2)
			// 成功率小于1也要扩容
			if meanSuccessLv < 0.99 {
				horizontalsNum = math.Max(horizontalsNum, float64(*deploy.Spec.Replicas)+1)
			}
			if math.IsNaN(horizontals) {
				continue
			}
			// 水平扩展的目标值
			replicas := int32(horizontalsNum)
			replicas = int32(math.Min(10.0, float64(replicas)))
			if *deploy.Spec.Replicas != replicas {
				// 获取旧的副本个数
				oldReplicas := *deploy.Spec.Replicas
				// 判断是否降落，如果是降落的操作，需要缓慢下降，同时判断当前cpu使用率是否超过1，如果超过1，应该考虑保持当前状态，不让副本减少
				if replicas < oldReplicas {
					// 判断窗口
					outerWindow := false
					pastTime := deploy.Annotations["lastExpandReplicasTimes"]
					if pastTime != "" {
						pastTimeT, err := util.String2Time(pastTime)
						if err == nil {
							currentTime := time.Now()
							currentTime.Add(-5 * time.Minute)
							if currentTime.After(pastTimeT) {
								outerWindow = true
							}
						}
					} else {
						outerWindow = true
					}
					if outerWindow {
						if !keep {
							replicas = int32(math.Max(float64(oldReplicas-1), 2))
						} else {
							replicas = int32(math.Max(float64(oldReplicas), 2))
						}
					}

				} else if replicas > oldReplicas {
					// 添加annotation
					deploy.Annotations["lastReplicas"] = strconv.FormatInt(int64(replicas), 10)
					// 添加时间
					deploy.Annotations["lastExpandReplicasTimes"] = util.Time2String(time.Now())
				}
				deploy.Spec.Replicas = &replicas
				printAction(&deploy, "horizatal", strconv.FormatInt(int64(oldReplicas), 10), strconv.FormatInt(int64(replicas), 10))
			}
			// 更新pod
			updateDeploy := &deploy
			for true {
				// 获取最新的deploy
				newDeploy, _ := clientset.AppsV1().Deployments(nameSpace).Get(context.TODO(), deploy.Name, metav1.GetOptions{})
				// 将老的deploy的replicas以及mem和cpu还有annotation等赋值给新的
				newDeploy.Spec.Replicas = deploy.Spec.Replicas
				if deploy.Annotations["lastReplicas"] != "" {
					newDeploy.Annotations["lastReplicas"] = deploy.Annotations["lastReplicas"]
				}
				if deploy.Annotations["lastExpandReplicasTimes"] != "" {
					newDeploy.Annotations["lastExpandReplicasTimes"] = deploy.Annotations["lastExpandReplicasTimes"]
				}
				for i := range newDeploy.Spec.Template.Spec.Containers {
					newDeploy.Spec.Template.Spec.Containers[i].Resources.Requests = deploy.Spec.Template.Spec.Containers[i].Resources.Requests.DeepCopy()
					newDeploy.Spec.Template.Spec.Containers[i].Resources.Limits = deploy.Spec.Template.Spec.Containers[i].Resources.Limits.DeepCopy()
					newDeploy.Spec.Template.ObjectMeta.Annotations = map[string]string{}
					if deploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+newDeploy.Spec.Template.Spec.Containers[i].Name] != "" {
						newDeploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+newDeploy.Spec.Template.Spec.Containers[i].Name] =
							deploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+newDeploy.Spec.Template.Spec.Containers[i].Name]
					}
					if deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+newDeploy.Spec.Template.Spec.Containers[i].Name] != "" {
						newDeploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+newDeploy.Spec.Template.Spec.Containers[i].Name] =
							deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+newDeploy.Spec.Template.Spec.Containers[i].Name]
					}
				}
				updateDeploy = newDeploy
				_, err = clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), updateDeploy, metav1.UpdateOptions{})
				if err != nil {
					println(err.Error())

				} else {
					break
				}
			}

		}
		// 每分钟跑一次
		time.Sleep(60 * time.Second)
	}

}

// 垂直伸缩器
func onePodUpdater(deploy *v12.Deployment, clientset *kubernetes.Clientset) bool {
	// 扩展资源
	// 更新每个容器
	// 返回值判断是否需要进行 水平伸缩
	for i, container := range deploy.Spec.Template.Spec.Containers {
		if container.Name == "istio-proxy" {
			// istio-proxy 为代理，跳过
			continue
		}
		// 通过deploy获取对应的pod
		cpu, err := prometheus.ObtainDeployContainerMetricCpu(deploy, clientset, container.Name, 1, 60*5)
		memory, err := prometheus.ObtainDeployContainerMetricMemory(deploy, clientset, container.Name, 1, 60*5)

		if err != nil {
			print(err.Error())
			continue
		}
		// 计算垂直计算的值
		verticleCpuCompute := autoscaling.Verticle(cpu)
		verticleMemCompute := autoscaling.Verticle(memory)
		// 通过模板得Annotation取值
		lastCpuVerticle, err := strconv.ParseFloat(deploy.Spec.Template.Annotations["verticle-cpu-"+container.Name], 3)
		lastMemoryVerticle, err := strconv.ParseFloat(deploy.Spec.Template.Annotations["verticle-mem-"+container.Name], 3)

		if err != nil || lastCpuVerticle == 0.0 || lastMemoryVerticle == 0.0 {
			// 说明节点现在没有值，或者节点值的设置是错误的
			if err != nil {
				println(err.Error())

			}
			if verticleMemCompute == 0 || verticleCpuCompute == 0 {
				return false
			}
			if deploy.Spec.Template.ObjectMeta.Annotations == nil {
				deploy.Spec.Template.ObjectMeta.Annotations = map[string]string{}
			}
			// 更新数据
			deploy.Spec.Template.ObjectMeta.Annotations["verticle-cpu-"+container.Name] = strconv.FormatFloat(verticleCpuCompute, 'f', 3, 64)
			deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+container.Name] = strconv.FormatFloat(verticleMemCompute, 'f', 3, 64)
			return false
		}

		computedCpu := verticleCpuCompute
		// 确保最小资源
		computedCpu = math.Max(computedCpu, 0.128)

		computedMem := verticleMemCompute
		computedMem = computedMem / (1024 * 1024)
		// 确保最小资源
		computedMem = math.Max(computedMem, 128)
		// 判断是否有2个以上得replicas
		if verticleCpuCompute > lastCpuVerticle*1.15 {
			if *deploy.Spec.Replicas < 2 {
				replicNum := *deploy.Spec.Replicas
				replicNum = replicNum + 1
				deploy.Spec.Replicas = &replicNum
				// 先更新至少2个replica才更新
				return false
			}
		}
		// cpu更新
		if verticleCpuCompute > lastCpuVerticle {
			// 判断是否超过15%
			if verticleCpuCompute > lastCpuVerticle*1.15 {
				if computedCpu > 1 {
					replicas := int32(*deploy.Spec.Replicas) + 1
					deploy.Spec.Replicas = &replicas
					deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(1.0, 'f', 3, 64))
					deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(2.0, 'f', 3, 64))
					printAction(deploy, "horizatal", strconv.FormatInt(int64(replicas-1), 10), strconv.FormatInt(int64(replicas), 10))
					return true
				}

				deploy.Spec.Template.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu, 'f', 3, 64))
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = resource.MustParse(strconv.FormatFloat(computedCpu*2, 'f', 3, 64))
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
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] = compare(deploy.Annotations["minCpu"],
					resource.MustParse(strconv.FormatFloat(computedCpu*2, 'f', 3, 64)))
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
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", int(computedMem*2)))
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
				deploy.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] = compare(deploy.Annotations["minMemory"],
					resource.MustParse(fmt.Sprintf("%dMi", int(computedMem*2))))
				deploy.Spec.Template.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
				if deploy.Spec.Template.ObjectMeta.Annotations == nil {
					deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
				deploy.Spec.Template.ObjectMeta.Annotations["verticle-mem-"+container.Name] = strconv.FormatFloat(verticleMemCompute, 'f', 3, 64)
				printAction(deploy, "verticle-mem", strconv.FormatFloat(lastMemoryVerticle, 'f', 3, 64), strconv.FormatFloat(verticleMemCompute, 'f', 3, 64))
			}
		}
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
