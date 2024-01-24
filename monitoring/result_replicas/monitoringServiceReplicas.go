package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/signal"
	"scheduling/util"
	"syscall"
	"time"
)

type RecordData struct {
	Time     int64 `json:"time"`
	Replicas int32 `json:"replicas"`
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

		replicas := int32(0)
		for _, deployment := range deployments.Items {
			replicas += *deployment.Spec.Replicas
			//println("deploy:"+deployment.Name+";limit :"+strconv.FormatInt(limitMem, 10), ";real:"+strconv.FormatFloat(memory[0], 'f', 3, 64))
		}
		records = append(records, RecordData{
			Time:     time.Now().Unix(),
			Replicas: replicas,
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
	err = ioutil.WriteFile("replicas.json", jsonData, 0644)
	if err != nil {
		fmt.Println("写入文件错误:", err)
		return
	}

	fmt.Println("数据已成功写入文件.")
}
