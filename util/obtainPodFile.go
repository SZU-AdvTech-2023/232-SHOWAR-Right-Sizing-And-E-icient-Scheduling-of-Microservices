package util

import (
	"bytes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func ObtainPodFile() string {

	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		kubeconfig = "/Users/username/.kube/config" // 替换为你的 kubeconfig 路径
	}
	// 创建 Kubernetes 客户端配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	clientset := GenerateClient()
	podName := "cpustress-deployment2-78568fc6d6-w7jvj"
	namespace := "autoscale"
	containerName := "cpustress"
	filePath := "/proc/schedstat"

	req := clientset.CoreV1().RESTClient().
		Get().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   []string{"cat", filePath},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Container: containerName,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		panic(err.Error())
	}
	var stdoutBytes, _ bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdoutBytes,
		Stderr: &stdoutBytes,
		Tty:    false,
	})
	if err != nil {
		panic(err.Error())
	}
	// 将读取的内容从字节切片中转换为字符串
	stdoutStr := stdoutBytes.String()

	//// 打印读取的内容
	//fmt.Println("File content:")
	//fmt.Println(stdoutStr)

	//stderrStr := stderrBytes.String()

	//fmt.Println(stderrStr)
	return stdoutStr
}
