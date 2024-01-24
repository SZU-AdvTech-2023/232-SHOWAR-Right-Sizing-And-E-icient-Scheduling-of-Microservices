package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type NetworkResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]interface{} `json:"metric"`
			Value  []interface{}          `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func ObtainRealTimeNetworkMetric(serviceName string) NetworkResponse {
	realTime := time.Now().Unix()
	realTimeF := float64(realTime)
	url := "http://localhost:9091/api/v1/query?query=histogram_quantile(0.90%2C%20sum(rate(istio_request_duration_milliseconds_bucket%7Bdestination_service_name%3D%22" + serviceName + "%22%7D%5B5m%5D))%20by%20(le))&time=" + fmt.Sprintf("%.3f", realTimeF) + "&_=1699014084606" // 将YOUR_API_ENDPOINT替换为您的API地址
	// 发起 GET 请求
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return NetworkResponse{}
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return NetworkResponse{}
	}

	// 解析JSON响应
	var response NetworkResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return NetworkResponse{}
	}

	return response
}
func ObtainMetricNetworkLatency(serviceName string, step int64, seconds int64) ([]float64, error) {
	//return obtainData2(step, seconds, "histogram_quantile(0.90%2C%20sum(rate(istio_request_duration_milliseconds_bucket%7Bdestination_service_name%3D%22"+serviceName+"%22%7D%5B5m%5D))%20by%20(le))")
	return obtainData2(step, seconds, "histogram_quantile(0.90, sum(rate(istio_request_duration_milliseconds_bucket{destination_service_name=\""+serviceName+"\"}[5m])) by (le))")

}
func obtainData2(step int64, seconds int64, queryData string) ([]float64, error) {
	body, err := requestPrometheus2(step, seconds, queryData)
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

func requestPrometheus2(step int64, seconds int64, queryData string) ([]byte, error) {
	// Prometheus 服务器的地址和端口
	prometheusURL := "http://localhost:9091/api/v1/query_range" // 替换为你的 Prometheus 服务器的 URL
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
