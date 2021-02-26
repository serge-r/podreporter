package kubernetes_client

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PodInfo struct {
	Name       		string
	Namespace  		string
	Cluster    		string
	Application 	string
	Containers 		[]ContainerInfo
}

type ContainerInfo struct {
	Name      string
	Image     string
	Resources *ResourcesInfo
}

type ResourcesInfo struct {
	LimitsCPU int64
	LimitsRAM int64
	MetricCPU int64
	MetricRAM int64
}

type KubeCluster struct {
	Cluster string
	Config     *rest.Config
}

func (kub *KubeCluster) AuthRemote(configFile []byte) error {
		config, err := clientcmd.RESTConfigFromKubeConfig(configFile)
		if err != nil {
			return err
		}
		kub.Config = config
		return nil
}

func (kub *KubeCluster) AuthLocal() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	kub.Config = config
	return nil
}

func (kub *KubeCluster) ReturnPods(namespacesList string) ([]PodInfo,error) {

	var podsReport []PodInfo
	var tempPod PodInfo
	var tempCont ContainerInfo
	var namespacesSelector string

	if len(namespacesList) > 0 {
		namespacesSelector = fmt.Sprintf("metadata.Name!=%s", namespacesList)
	}

	clientset, err := kubernetes.NewForConfig(kub.Config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		TypeMeta:             metav1.TypeMeta{},
		LabelSelector:        "",
		FieldSelector:        namespacesSelector,
		Watch:                false,
		AllowWatchBookmarks:  false,
		ResourceVersion:      "",
		ResourceVersionMatch: "",
		TimeoutSeconds:       nil,
		Limit:                0,
		Continue:             "",
	})

	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaces.Items {
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil,err
		}

		for _, pod := range pods.Items {
			for _, cnt := range pod.Spec.Containers {
				tempPod.Name = pod.Name
				tempPod.Namespace = namespace.Name
				tempPod.Application = pod.Labels["app"]
				tempCont.Name = cnt.Name
				tempCont.Image = cnt.Image
				tempCont.Resources.LimitsCPU = cnt.Resources.Limits.Cpu().MilliValue()
				tempCont.Resources.LimitsRAM = cnt.Resources.Limits.Memory().MilliValue() / 1000 / 1024 / 1024
				tempPod.Containers = append(tempPod.Containers, tempCont)
				podsReport = append(podsReport, tempPod)
			}
		}
	}
	return podsReport,nil
}
