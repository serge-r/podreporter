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

type PodInfo struct {
	Name        string
	Namespace   string
	Cluster     string
	Application string
	Uid         string
	CPUMetric   float64
	RAMMetric   float64
	CPULimits   float64
	RAMLimits   float64
	RatingCPU   int
	RatingRAM   int
}

type KubeCluster struct {
	Cluster string
	Config  *rest.Config
}

type PodByLimitCPU []PodInfo

type PodByLimitCPUDesc []PodInfo

type PodByLimitRAM []PodInfo

type PodByLimitRAMDesc []PodInfo

type PodByMetricCPU []PodInfo

type PodByMetricCPUDesc []PodInfo

type PodByMetricRAM []PodInfo

type PodByMetricRAMDesc []PodInfo

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
	var sumCPU float64
	var sumRAM float64
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
			sumCPU = 0
			sumRAM = 0
			logger.Debugf("Filling info for pod  %v", pod.Name)
			tempPod.Name = pod.Name
			tempPod.Uid = strings.Replace(string(pod.UID), "-", "_", -1)
			tempPod.Namespace = namespace.Name
			tempPod.Application = pod.Labels["app"]
			for _, cnt := range pod.Spec.Containers {
				cpu := cnt.Resources.Limits.Cpu().MilliValue()
				ram := cnt.Resources.Limits.Memory().MilliValue() / 1000 / 1024 / 1024
				sumCPU += float64(cpu)
				sumRAM += float64(ram)
			}
			tempPod.CPULimits = sumCPU
			tempPod.RAMLimits = sumRAM
			podsReport = append(podsReport, tempPod)
		}
	}
	logger.Debug("Pod structure complete")
	return podsReport, nil
}

// SetRating Rate pods
// If pod have no limits at least one container - his rate is 0
// If pod have a limits, but they are no more then 50% bigger then week metric - his rate is 1
// If pod have a limits and they are  bigger then 50% then week metrics - his rating is -1
func (pod *PodInfo) SetRating() {
	if pod.CPULimits == 0 {
		pod.RatingCPU = 0
	}
	if pod.RAMLimits == 0 {
		pod.RatingRAM = 0
	}
	if pod.CPULimits < (pod.CPULimits*50/100)+pod.CPUMetric {
		pod.RatingCPU = -1
	}
	if pod.RAMLimits < (pod.RAMLimits*50/100)+pod.RAMMetric {
		pod.RatingRAM = -1
	}
}

// Sorting pods, Limits CPU
func (pods PodByLimitCPU) Len() int { return len(pods) }

func (pods PodByLimitCPU) Less(i, j int) bool {
	return pods[i].CPULimits < pods[j].CPULimits
}

func (pods PodByLimitCPU) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM
func (pods PodByLimitRAM) Len() int { return len(pods) }

func (pods PodByLimitRAM) Less(i, j int) bool {
	return pods[i].RAMLimits < pods[j].RAMLimits
}

func (pods PodByLimitRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU
func (pods PodByMetricCPU) Len() int { return len(pods) }

func (pods PodByMetricCPU) Less(i, j int) bool {
	return pods[i].CPUMetric < pods[j].CPUMetric
}

func (pods PodByMetricCPU) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU
func (pods PodByMetricCPUDesc) Len() int { return len(pods) }

func (pods PodByMetricCPUDesc) Less(i, j int) bool {
	return pods[i].CPUMetric > pods[j].CPUMetric
}

func (pods PodByMetricCPUDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM
func (pods PodByMetricRAM) Len() int { return len(pods) }

func (pods PodByMetricRAM) Less(i, j int) bool {
	return pods[i].RAMMetric < pods[j].RAMMetric
}

func (pods PodByMetricRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM DESC
func (pods PodByMetricRAMDesc) Len() int { return len(pods) }

func (pods PodByMetricRAMDesc) Less(i, j int) bool {
	return pods[i].RAMMetric > pods[j].RAMMetric
}

func (pods PodByMetricRAMDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}
