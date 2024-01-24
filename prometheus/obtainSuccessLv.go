package prometheus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const prometheusURL = "http://localhost:9091"

func ObtainSuccessLv(serviceName string) NetworkResponse {
	//sum by(destination_service_name)(istio_requests_total{destination_service_name="frontend", response_code="200"})/sum by(destination_service_name)( istio_requests_total{destination_service_name="frontend"})
	realTime := time.Now().Unix()
	realTimeF := float64(realTime)
	url := "http://localhost:9091/api/v1/query?query=sum%20by(destination_service_name)(istio_requests_total%7Bdestination_service_name%3D%22" +
		serviceName + "%22%2C%20response_code%3D~%22200%7C302%22%7D)%2Fsum%20by(destination_service_name)(%20istio_requests_total%7Bdestination_service_name%3D%22" + serviceName + "%22%7D)&time=" + fmt.Sprintf("%.3f", realTimeF) + "&_=1699014084606" // 将YOUR_API_ENDPOINT替换为您的API地址
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

func ObtainMetricNetworkSuccessLv(serviceName string, step int64, seconds int64) ([]float64, error) {
	//return obtainData2(step, seconds, "histogram_quantile(0.90%2C%20sum(rate(istio_request_duration_milliseconds_bucket%7Bdestination_service_name%3D%22"+serviceName+"%22%7D%5B5m%5D))%20by%20(le))")
	return obtainData2(step, seconds,
		"sum (irate(istio_requests_total{destination_service_name=~\""+serviceName+".*\", response_code=~\"2..\"}[1m]))/sum (irate(istio_requests_total{destination_service_name=~\""+serviceName+".*\"}[1m]))")

}
