package util

import (
	"fmt"
	"time"
)

// Time2String 时间转字符串
func Time2String(t time.Time) string {
	// 将时间转为字符串
	timeStr := t.Format("2006-01-02 15:04:05")
	return timeStr
}

// String2Time 字符串转时间
func String2Time(str string) (time.Time, error) {
	// 将字符串转为时间
	parsedTime, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		fmt.Println("解析时间出错:", err)
		return time.Time{}, err
	}
	return parsedTime, nil
}
