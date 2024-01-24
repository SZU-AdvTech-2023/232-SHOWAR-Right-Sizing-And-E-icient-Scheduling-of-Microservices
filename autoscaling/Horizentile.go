package autoscaling

import (
	"math"
)

func Horizontal(cpus []float64, target float64, nodeValue NodeValue) float64 {
	// 计算离散函数的积分
	integral := 0.0
	var e []float64
	for _, value := range cpus {
		e = append(e, value-target)
	}
	for _, value := range e {
		integral += value
	}
	// 计算平均 导数
	var xSum = 0.0
	for i := 0; i < len(e)-1; i++ {
		xSum += math.Abs(e[i+1] - e[i])
	}
	xSum = xSum / float64((len(e) - 1))

	var computed = xSum + integral + e[len(e)-1]/3
	return computed
}
func MeanValue(cpus []float64) float64 {
	integral := 0.0

	for _, value := range cpus {
		integral += value
	}

	var computed = integral / float64(len(cpus))
	return computed
}
