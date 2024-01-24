package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"scheduling/util"
	"strconv"
	"time"
)

func QueryMetric() {
	// Prometheus 服务器的地址和端口
	prometheusURL := "http://localhost:9090/metrics" // 替换为你的 Prometheus 服务器的 URL

	// 发起 HTTP GET 请求获取 Prometheus 指标数据
	response, err := http.Get(prometheusURL)
	if err != nil {
		fmt.Printf("Failed to retrieve Prometheus metrics: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Failed to retrieve Prometheus metrics. Status code: %v\n", response.Status)
		return
	}

	// 读取响应体中的指标数据
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}

	// 现在 body 变量中包含了 Prometheus 的指标数据
	fmt.Printf("Prometheus Metrics:\n%s\n", body)
}

type Metric struct {
	Instance string `json:"instance"`
}

// Result 结构体用于表示 JSON 中的 "result" 部分
type Result struct {
	Metric Metric          `json:"metric"`
	Values [][]interface{} `json:"values"`
}

// Data 结构体用于表示 JSON 中的 "data" 部分
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

// Response 结构体用于表示整个 JSON 数据
type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

func obtainData(step int64, seconds int64, queryData string) ([]float64, error) {
	body, err := requestPrometheus(step, seconds, queryData)
	if err != nil {
		return nil, err
	}

	// 创建 Response 结构体
	var response2 Response

	// 解码 JSON 数据
	if err := json.Unmarshal([]byte(body), &response2); err != nil {
		fmt.Println("解析JSON失败:", err)
		return nil, err
	}
	if len(response2.Data.Result) <= 0 {
		return nil, errors.New("读取数据出错，可能pod容器出错")
	}
	var res []float64
	for _, item := range response2.Data.Result[0].Values {

		floatvalue, _ := item[1].(string)
		floatvalue2, _ := strconv.ParseFloat(floatvalue, 3)
		res = append(res, floatvalue2)

	}
	return res, nil
}
func obtainDataWithTime(step int64, seconds int64, queryData string) ([][]interface{}, error) {
	body, err := requestPrometheus(step, seconds, queryData)
	if err != nil {
		return nil, err
	}

	// 创建 Response 结构体
	var response2 Response

	// 解码 JSON 数据
	if err := json.Unmarshal([]byte(body), &response2); err != nil {
		fmt.Println("解析JSON失败:", err)
		return nil, err
	}
	if len(response2.Data.Result) <= 0 {
		return nil, errors.New("读取数据出错，可能pod容器出错")
	}
	return response2.Data.Result[0].Values, nil
}

func requestPrometheus(step int64, seconds int64, queryData string) ([]byte, error) {
	// Prometheus 服务器的地址和端口
	prometheusURL := "http://localhost:9090/api/v1/query_range" // 替换为你的 Prometheus 服务器的 URL
	request, err := http.NewRequest("GET", prometheusURL, nil)
	query := request.URL.Query()
	query.Add("query", queryData)
	end := float64(time.Now().Unix())
	start := end - float64(seconds)
	// 拼接参数
	query.Add("start", strconv.FormatFloat(start, 'f', 3, 64))
	query.Add("end", strconv.FormatFloat(end, 'f', 3, 64))
	query.Add("step", strconv.FormatInt(step, 10))
	// 请求数据
	request.URL.RawQuery = query.Encode()
	response, err := http.DefaultClient.Do(request)
	//response, err := http.Get(prometheusURL)
	if err != nil {
		fmt.Printf("Failed to retrieve Prometheus metrics: %v\n", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Failed to retrieve Prometheus metrics. Status code: %v\n", response.Status)
		return nil, err
	}

	// 读取响应体中的指标数据
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return nil, err
	}
	return body, nil
}

func ObtainMetricCpu(podName string, step int64, seconds int64) ([]float64, error) {
	return obtainData(step, seconds, "sum by(pod) (rate(container_cpu_usage_seconds_total{pod=~\""+podName+"\", container!=\"\"}[1m])) ")
}

func ObtainContainerMetricCpu(podName string, containerName string, step int64, seconds int64) ([]float64, error) {
	return obtainData(step, seconds, "sum by(pod) (rate(container_cpu_usage_seconds_total{pod=~\""+podName+"\", container=\""+containerName+"\"}[1m])) ")
}
func ObtainContainerMetricCpuWithTime(podName string, containerName string, step int64, seconds int64) ([][]interface{}, error) {
	return obtainDataWithTime(step, seconds, "sum by(pod) (rate(container_cpu_usage_seconds_total{pod=~\""+podName+"\", container=\""+containerName+"\"}[1m])) ")
}
func ObtainDeployContainerMetricCpu(deployment *v12.Deployment, clientset *kubernetes.Clientset, containerName string, step int64, seconds int64) ([]float64, error) {
	pods, err := util.ObtainPodByDeploy(*deployment, clientset)
	if err != nil {
		return nil, err
	}

	var resultMap = make(map[float64][]float64)
	for _, pod := range pods.Items {
		cpu, err := ObtainContainerMetricCpuWithTime(pod.Name, containerName, step, seconds)
		if err != nil {
			continue
		}
		for _, value := range cpu {

			floatvalue, _ := value[1].(string)
			floatvalue2, _ := strconv.ParseFloat(floatvalue, 3)
			time := value[0].(float64)
			resultMap[time] = append(resultMap[time], floatvalue2)
		}
	}
	var res []float64
	for _, value := range resultMap {
		res = append(res, mean(value))
	}
	return res, nil
}
func mean(arr []float64) float64 {
	sum := 0.0
	for _, value := range arr {
		sum += value
	}
	return sum / float64(len(arr))
}
func ObtainMetricMemory(podName string, step int64, seconds int64) ([]float64, error) {
	return obtainData(step, seconds, "sum by(pod) (container_memory_working_set_bytes{pod=\""+podName+"\", container!=\"\"})")
}
func ObtainContainerMetricMemory(podName string, containerName string, step int64, seconds int64) ([]float64, error) {
	return obtainData(step, seconds, "sum by(pod)(container_memory_working_set_bytes{pod=\""+podName+"\", container=\""+containerName+"\"}) ")
}
func ObtainContainerMetricMemoryWithTime(podName string, containerName string, step int64, seconds int64) ([][]interface{}, error) {
	return obtainDataWithTime(step, seconds, "sum by(pod)(container_memory_working_set_bytes{pod=\""+podName+"\", container=\""+containerName+"\"}) ")
}
func ObtainDeployContainerMetricMemory(deployment *v12.Deployment, clientset *kubernetes.Clientset, containerName string, step int64, seconds int64) ([]float64, error) {
	pods, err := util.ObtainPodByDeploy(*deployment, clientset)
	if err != nil {
		return nil, err
	}

	var resultMap = make(map[float64][]float64)

	for _, pod := range pods.Items {
		cpu, err := ObtainContainerMetricMemoryWithTime(pod.Name, containerName, step, seconds)
		if err != nil {
			continue
		}
		for _, value := range cpu {

			floatvalue, _ := value[1].(string)
			floatvalue2, _ := strconv.ParseFloat(floatvalue, 3)
			time := value[0].(float64)
			resultMap[time] = append(resultMap[time], floatvalue2)
		}
	}
	var res []float64
	for _, value := range resultMap {
		res = append(res, mean(value))
	}
	return res, nil
}
func RealTimeMetric(promeQL string) float64 {
	body, err := requestPrometheus(1, 10, promeQL)

	if err != nil {
		panic(err)
	}

	// 创建 Response 结构体
	var response2 Response

	// 解码 JSON 数据
	if err := json.Unmarshal([]byte(body), &response2); err != nil {
		panic(err)
	}
	var res []float64
	for _, item := range response2.Data.Result[0].Values {
		floatvalue, _ := item[1].(string)
		floatvalue2, _ := strconv.ParseFloat(floatvalue, 3)
		res = append(res, floatvalue2)

	}
	return res[len(res)-1]
}
func computeMatrix(data [][]float64) (result []float64) {
	// 获取数组的行数和列数
	rows := len(data)
	cols := len(data[0])

	// 创建一个数组用于存放每列的累加和
	sums := make([]float64, cols)

	// 遍历每行，将每列的元素相加
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			sums[j] += data[i][j]
		}
	}

	// 计算平均值
	averages := make([]float64, cols)
	for j := 0; j < cols; j++ {
		averages[j] = float64(sums[j]) / float64(rows)
	}
	result = sums
	return
}
