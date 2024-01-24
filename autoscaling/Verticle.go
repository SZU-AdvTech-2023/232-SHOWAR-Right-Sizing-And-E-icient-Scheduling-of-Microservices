package autoscaling

import (
	"math"
)

func Verticle(cpu []float64) float64 {

	// 计算平均值
	sum := 0.0
	for _, value := range cpu {
		sum += value
	}
	average := sum / float64(len(cpu))

	// 计算方差
	variance := 0.0
	for _, value := range cpu {
		diff := value - average
		variance += diff * diff
	}
	variance = variance / float64(len(cpu))
	// 标准差是方差的平方根
	stdDev := math.Sqrt(variance)
	return average + 3*stdDev
}
