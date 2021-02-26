package podreporter

import (
	"fmt"
	kube "github.com/serge-r/podreporter/pkg/kubernetes-client"
	prometheus "github.com/serge-r/podreporter/pkg/prometheus-client"
	"github.com/slack-go/slack"
)

type PodReporter struct {
	Datacenters []Datacenter
	Reports		[]PodReport
	prom 		*prometheus.Prometheus
	slackClient *slack.Client
}

type PodReport struct {
	AllPods				[]kube.PodInfo
	TopCPUPods 			[]kube.PodInfo
	TopRAMPods			[]kube.PodInfo
	PodWithWrongRAM 	[]kube.PodInfo
	PodWithWrongCPU		[]kube.PodInfo
}

type Datacenter struct {
	Name string
	KubeConfig []byte
	pods []kube.PodInfo
}

func Init(datacenters []Datacenter, prom *prometheus.Prometheus, slackClient *slack.Client) (*PodReporter) {
	reporter := PodReporter{
		Datacenters: datacenters,
		prom:	prom,
		slackClient: slackClient,
	}
	return &reporter
}

func (reporter *PodReporter) fillKubePods() error {
	cluster := kube.KubeCluster{}
	for _,dc := range reporter.Datacenters {
		err := cluster.AuthRemote(dc.KubeConfig)
		if err != nil {
			return err
		}
		dc.pods,err = cluster.ReturnPods("kube-system")
		if err != nil {
			return err
		}
	}
	return nil
}

func (reporter *PodReporter) Run() {
	for {
		reporter.getMessage()
		reporter.sendMessage()

	}
}

func (reporter *PodReporter) fillPrometheusInfo() error {
	for _,dc := range reporter.Datacenters {
		for _, pod := range dc.pods {
			for _,cont := range pod.Containers {
				contCPUQuery := fmt.Sprintf("max_over_time(rate(container_cpu_usage_seconds_total{namespace=\"%s\",pod=\"%s\",container=\"%s\",image!=\"\"}[1w]))",
					pod.Namespace,
					pod.Name,
					cont.Name)
				contRAMQuery := fmt.Sprintf("max_over_time(rate(container_memory_rss{namespace=\"%s\",pod=\"%s\",container=\"%s\",image!=\"\"}[1w]}))",
					pod.Namespace,
					pod.Name,
					cont.Name)
				resultCPU, err := reporter.prom.InstanceQuery(contCPUQuery)
				if err != nil {
					return err
				}
				resultRAM, err := reporter.prom.InstanceQuery(contRAMQuery)
				if err != nil {
					return err
				}
				cont.Resources.MetricCPU = resultCPU.Data.Result[0].Value[1].(int64)
				cont.Resources.MetricRAM = resultRAM.Data.Result[0].Value[1].(int64)
			}
		}
	}
	return nil
}

func (reporter *PodReporter) generateReport()  {


}

func (reporter *PodReporter) getMessage() {
	reporter.generateReport()

}

func (reporter *PodReporter) sendMessage() {


}