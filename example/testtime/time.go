package main

import (
	"fmt"
	"time"
)

func main() {
	currentTime := time.Now()

	// 将时间转为字符串
	timeStr := currentTime.Format("2006-01-02 15:04:05")
	fmt.Println("时间转为字符串:", timeStr)

	// 将字符串转为时间
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		fmt.Println("解析时间出错:", err)
		return
	}
	fmt.Println("字符串转为时间:", parsedTime)
}
