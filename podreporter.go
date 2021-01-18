package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type containerInfo struct {
	containerName string
	cpuCores int64
	memBytes int64
}

type podInfo struct {
	podname    string
	containers []containerInfo
}

type namespaceInfo struct {
	namespace string
	pods []podInfo
}

type report struct {
	dc string
	namespaces [] namespaceInfo
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	var namespace *string
	namespace = flag.String("namespace", "default", "namespace for check pods")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the namespace %s\n", len(pods.Items), *namespace)
	for i := 0; i < len(pods.Items); i++ {
		pod := pods.Items[i]
		fmt.Printf("In pod %s I found %d containers\n", pod.Name, len(pod.Spec.Containers))
		for j := 0 ; j < len(pod.Spec.Containers); j ++ {
			cnt := pod.Spec.Containers[j]
			fmt.Printf("Container %s with limits: CPU %d (mils) and RAM %d MB\n", cnt.Name, cnt.Resources.Limits.Cpu().MilliValue(), cnt.Resources.Limits.Memory().MilliValue()/ 1000 /1024 /1024)
		}


	}

}
