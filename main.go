/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"log"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		panic(err)
	}
	// 创建clientset对象
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	//创建stopCh对象，用于在程序进程退出之前通知Informer退出，因为Informer是一个持久运行的goroutine
	stopCh := make(chan struct{})
	defer close(stopCh)

	//实例化ShareInformer对象，一个参数是clientset,另一个是time.Minute用于设置多久进行一次resync(重新同步)
	//resync会周期性的执行List操作，将所有的资源存放在Informer Store中，如果参数为0,则禁止resync操作
	sharedInformers := informers.NewSharedInformerFactory(clientSet, time.Minute)
	//得到具体Pod资源的informer对象
	informer := sharedInformers.Core().V1().Pods().Informer()
	// 为Pod资源添加资源事件回调方法，支持AddFunc、UpdateFunc及DeleteFunc
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			//在正常情况下，kubernetes其他组件在使用Informer机制时触发资源事件回调方法，将资源对象推送到WorkQueue或其他队列中，
			//这里是直接输出触发的资源事件
			myObj := obj.(metav1.Object)
			log.Printf("New Pod Added to Store: %s", myObj.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oObj := oldObj.(metav1.Object)
			nObj := newObj.(metav1.Object)
			log.Printf("%s Pod Updated to %s", oObj.GetName(), nObj.GetName())

		},
		DeleteFunc: func(obj interface{}) {
			myObj := obj.(metav1.Object)
			log.Printf("Pod Deleted from Store: %s", myObj.GetName())
		},
	})
	//通过Run函数运行当前Informer,内部为Pod资源类型创建Informer
	informer.Run(stopCh)

}
