package autoscaling

import (
	"math"
)

func mean(arr []float64) float64 {
	sum := 0.0
	for _, value := range arr {
		sum += value
	}
	return sum / float64(len(arr))
}

func stdDev(arr []float64, mean float64) float64 {
	sumSquaredDiffs := 0.0
	for _, value := range arr {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}
	return math.Sqrt(sumSquaredDiffs / float64(len(arr)))
}

func covariance(arr1, arr2 []float64, mean1, mean2 float64) float64 {
	sumProduct := 0.0
	for i := 0; i < len(arr1); i++ {
		sumProduct += (arr1[i] - mean1) * (arr2[i] - mean2)
	}
	return sumProduct / float64(len(arr1))
}

func pearsonCorrelation(arr1, arr2 []float64) float64 {
	mean1 := mean(arr1)
	mean2 := mean(arr2)

	stdDev1 := stdDev(arr1, mean1)
	stdDev2 := stdDev(arr2, mean2)

	covar := covariance(arr1, arr2, mean1, mean2)

	return covar / (stdDev1 * stdDev2)
}
func AffinityGenerator(resource1 []float64, resource2 []float64) bool {
	correlation := pearsonCorrelation(resource1, resource2)

	if correlation < -0.8 {
		return true
	}
	return false
}
func AntiAffinityGenerator(resource1 []float64, resource2 []float64) bool {
	correlation := pearsonCorrelation(resource1, resource2)

	if correlation < -1.0 {
		return true
	}
	return false
}
