package cmd

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

type KubeCluster struct {
	Cluster string
	Config  *rest.Config
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

func (kub *KubeCluster) ReturnPods(namespacesList string, logger *log.Entry) ([]PodInfo, error) {

	var podsReport []PodInfo
	var tempPod PodInfo
	var namespacesSelector string

	if len(namespacesList) > 0 {
		namespacesSelector = fmt.Sprintf("metadata.name=%s", namespacesList)
	}

	clientset, err := kubernetes.NewForConfig(kub.Config)
	if err != nil {
		return nil, err
	}
	logger.Infof("Trying to get namespaces from kubernetes")
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
		logger.Debugf("Trying to get pods from from namespace %v", namespace.Name)
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		logger.Debugf("I found %d pods in namespace %v", len(pods.Items), namespace.Name)
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			tempPod.CPULimits = 0
			tempPod.RAMLimits = 0
			tempPod.CPURequsts = 0
			tempPod.RAMRequests = 0

			logger.Debugf("Filling info for pod  %v", pod.Name)
			tempPod.Name = pod.Name
			tempPod.Uid = strings.Replace(string(pod.UID), "-", "_", -1)
			tempPod.Namespace = namespace.Name
			tempPod.Application = pod.Labels["app"]
			for _, cnt := range pod.Spec.Containers {
				tempPod.CPULimits += float64(cnt.Resources.Limits.Cpu().MilliValue())
				tempPod.RAMLimits += float64(cnt.Resources.Limits.Memory().MilliValue() / 1000 / 1024 / 1024)
				tempPod.CPURequsts += float64(cnt.Resources.Requests.Cpu().MilliValue())
				tempPod.RAMRequests += float64(cnt.Resources.Requests.Memory().MilliValue() / 1000 / 1024 / 1024)
			}
			podsReport = append(podsReport, tempPod)
		}
	}
	logger.Debug("Pod structure complete")
	return podsReport, nil
}
