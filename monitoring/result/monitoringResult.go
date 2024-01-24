package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/signal"
	"scheduling/prometheus"
	"scheduling/util"
	"strconv"
	"syscall"
	"time"
)

type RecordData struct {
	Time     int64   `json:"time"`
	Limit    float64 `json:"limit"`
	RealData float64 `json:"real_data"`
	Span     float64 `json:"span"`
}

const namespace = util.Namespace

func main() {
	// 创建一个信号通道
	stopChan := make(chan os.Signal, 1)
	// 将 interrupt 信号（Ctrl+C）和 syscall.SIGTERM 信号添加到通道
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Press Ctrl+C to stop")
	var records []RecordData
	// 阻塞等待信号或一段时间
	for {
		// 模拟程序执行的一些操作
		//获取所有服务
		client := util.GenerateClient()
		deployments, _ := client.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		limitTotal := 0.0
		realTotal := 0.0

		for _, deployment := range deployments.Items {
			// 获取每个服务的limit以及真实使用率
			limitMem := deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().Value()

			// 获取真实使用率
			// 获取对应的pod
			pods, err := util.ObtainPodByDeploy(deployment, client)
			if err != nil {
				println(err.Error())
				continue
			}
			memory := 0.0

			for _, pod := range pods.Items {

				podMemory, err := prometheus.ObtainContainerMetricMemory(pod.Name, deployment.Spec.Template.Spec.Containers[0].Name, 1, 1)
				if err != nil {
					//println(pod.Items[0].Name)
					//println(deployment.Spec.Template.Spec.Containers[0].Name)
					println(err.Error())
					continue
				}
				memory += podMemory[0]
			}

			limitTotal += float64(limitMem)
			realTotal += memory / float64(len(pods.Items))
			//println("deploy:"+deployment.Name+";limit :"+strconv.FormatInt(limitMem, 10), ";real:"+strconv.FormatFloat(memory[0], 'f', 3, 64))
		}
		println("limit:" + strconv.FormatFloat(limitTotal, 'f', 3, 64) + ";real:" + strconv.FormatFloat(realTotal, 'f', 3, 64))
		records = append(records, RecordData{
			Time:     time.Now().Unix(),
			RealData: realTotal,
			Limit:    limitTotal,
			Span:     limitTotal - realTotal,
		})
		time.Sleep(1 * time.Second)
		// 阻塞等待信号或一段时间
		select {
		case <-stopChan:
			// 收到停止信号时退出循环
			// 执行相关的任务收集
			outputFile(records)
			os.Exit(0)
		case <-time.After(time.Second):
			// 在这里执行你的主循环中的操作
		}
	}

}

func outputFile(data []RecordData) {
	// 将数组序列化为JSON格式
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("JSON序列化错误:", err)
		return
	}

	// 将JSON数据写入文件
	err = ioutil.WriteFile("data.json", jsonData, 0644)
	if err != nil {
		fmt.Println("写入文件错误:", err)
		return
	}

	fmt.Println("数据已成功写入文件.")
}
