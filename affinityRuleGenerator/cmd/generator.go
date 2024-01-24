package main

import (
	"context"
	"errors"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"scheduling/autoscaling"
	"scheduling/prometheus"
	"scheduling/util"
	"time"
)

const nameSpace = util.Namespace

func main() {
	for {
		// 创建 Kubernetes 客户端
		clientset := util.GenerateClient()
		// 水平扩展每分钟跑一次，每次评估过去5分钟的资源
		for {
			// 获取所有deploy
			deploys, err := clientset.AppsV1().Deployments(nameSpace).List(context.TODO(), metav1.ListOptions{})

			if err != nil {
				panic(err.Error())
			}
			// 亲和性规则
			generateAntifity(deploys, clientset)
		}
		time.Sleep(5 * time.Minute)
	}
}
func generateAntifity(deploys *v12.DeploymentList, clientset *kubernetes.Clientset) {

	// 生成亲和性规则
	for i := 0; i < len(deploys.Items)-1; i++ {
		for j := i + 1; j < len(deploys.Items); j++ {
			podsI, _ := clientset.CoreV1().Pods(nameSpace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(
					&metav1.LabelSelector{MatchLabels: map[string]string{"app": deploys.Items[i].Spec.Selector.MatchLabels["app"]}},
				),
			})
			podsCpud1, _ := computeCpus(*podsI)
			podsJ, _ := clientset.CoreV1().Pods(nameSpace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(
					&metav1.LabelSelector{MatchLabels: map[string]string{"app": deploys.Items[j].Spec.Selector.MatchLabels["app"]}},
				),
			})
			podsCpud2, _ := computeCpus(*podsJ)
			if autoscaling.AffinityGenerator(podsCpud1, podsCpud2) {
				// 生成亲和性规则
				// Create PodAffinity rule
				podAffinityTerm := v1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploys.Items[j].Spec.Template.ObjectMeta.Labels["app"],
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				}
				deploys.Items[i].Spec.Template.Spec.Affinity.PodAffinity = &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{podAffinityTerm},
				}
				clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), &deploys.Items[i], metav1.UpdateOptions{})

				podAffinityTerm = v1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploys.Items[i].Spec.Template.ObjectMeta.Labels["app"],
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				}
				deploys.Items[j].Spec.Template.Spec.Affinity.PodAffinity = &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{podAffinityTerm},
				}
				clientset.AppsV1().Deployments(nameSpace).Update(context.TODO(), &deploys.Items[j], metav1.UpdateOptions{})

			}

		}
	}
}

func computeCpus(pods v1.PodList) ([]float64, error) {
	var cpus []float64
	for _, pod := range pods.Items {
		cpu, err := prometheus.ObtainMetricCpu(pod.ObjectMeta.Name, 1, 60*5)
		if err != nil {
			return nil, errors.New("计算错误")
		}
		if cpus == nil {
			cpus = cpu
		} else {
			for i := 0; i < len(cpu); i++ {
				cpus[i] += cpu[i]
			}
		}
	}
	return cpus, nil
}
