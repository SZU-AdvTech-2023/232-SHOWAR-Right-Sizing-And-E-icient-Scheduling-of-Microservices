package main

import (
	"context"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"scheduling/util"
)

func main() {
	clientset := util.GenerateClient()
	eviction := policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "adservice-7b89c8f7c8-2jgrh",
		},
	}
	err := clientset.CoreV1().Pods("default").EvictV1(context.TODO(), &eviction)
	if err != nil {
		println(err.Error())
	}
}
