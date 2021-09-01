package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strconv"
)

type PodReporter struct {
	Datacenters    []Datacenter
	prom           *Prometheus
	slackClient    *slack.Client
	logger         *log.Entry
	maxConcurrency int
}

type Datacenter struct {
	Name       string
	KubeConfig []byte
	pods       []PodInfo
}

func CreateReporter(datacenters []Datacenter, prom *Prometheus, slackClient *slack.Client, logger *log.Entry) *PodReporter {
	reporter := PodReporter{
		Datacenters: datacenters,
		prom:        prom,
		slackClient: slackClient,
		logger:      logger,
	}
	return &reporter
}

func (reporter *PodReporter) FillKubePods() error {
	var tempPods []PodInfo
	cluster := KubeCluster{}
	for idx, dc := range reporter.Datacenters {
		err := cluster.AuthRemote(dc.KubeConfig)
		if err != nil {
			return err
		}
		tempPods, err = cluster.ReturnPods("jaeger", reporter.logger)
		reporter.Datacenters[idx].pods = tempPods
		if err != nil {
			return err
		}
	}
	return nil
}

func (reporter *PodReporter) FillPrometheusInfo() error {
	var cpu float64
	var ram float64
	for i, dc := range reporter.Datacenters {
		for j, pod := range dc.pods {
			reporter.logger.Debugf("Start query prom for DC %v and pod %v", dc.Name, pod.Name)
			contCPUQuery := fmt.Sprintf("max_over_time(sum(rate(container_cpu_usage_seconds_total{namespace=\"%s\", pod=\"%s\", id=~\".*%s.*\"}))[7d:1m])",
				pod.Namespace,
				pod.Name,
				pod.Uid)
			contRAMQuery := fmt.Sprintf("max_over_time(sum(container_memory_rss{namespace=\"%s\",pod=\"%s\", id=~\".*%s.*\"})[7d:1m])",
				pod.Namespace,
				pod.Name,
				pod.Uid)
			resultCPU, err := reporter.prom.InstanceQuery(contCPUQuery)
			if err != nil {
				return err
			}
			resultRAM, err := reporter.prom.InstanceQuery(contRAMQuery)
			if err != nil {
				return err
			}
			stringCPU := (*resultCPU).(string)
			stringRAM := (*resultRAM).(string)
			cpu, _ = strconv.ParseFloat(stringCPU, 32)
			ram, _ = strconv.ParseFloat(stringRAM, 32)
			reporter.Datacenters[i].pods[j].CPUMetric += cpu * 1000
			reporter.Datacenters[i].pods[j].RAMMetric += ram / 1024 / 1024
			reporter.logger.Debugf("PROM CPU is %f", cpu)
			reporter.logger.Debugf("PROM RAM is %f", ram)
		}
	}
	return nil
}

func (reporter *PodReporter) GetReport(dcname string) {

	reporter.logger.Info("Start generate report")
	for _, dc := range reporter.Datacenters {
		if dc.Name == dcname {
			for _, pod := range dc.pods {

				reporter.logger.Infof("Pod summary\n\n")

				reporter.logger.Infof("Pod: %s \t Namespace: %s \n", pod.Name, pod.Namespace)
				reporter.logger.Infof("Kube values\t CPU: %f MilliCores\t RAM: %f Mi\n", pod.CPULimits, pod.RAMLimits)
				reporter.logger.Infof("Prom values\t CPU: %f MilliCores\t RAM: %f Mi\n", pod.CPUMetric, pod.RAMMetric)

			}

			reporter.logger.Infof("\n\n\nTop 3 pods by max CPU")
			sort.Sort(PodByMetricCPUDesc(dc.pods))

			for i := 0; i < len(dc.pods); i++ {
				reporter.logger.Infof("Pod %s \t CPU: %f Millicores \n", dc.pods[i].Name, dc.pods[i].CPUMetric)
			}

			reporter.logger.Infof("\n\n\nTop 3 pods by max RAM")
			sort.Sort(PodByMetricRAMDesc(dc.pods))

			for i := 0; i < len(dc.pods); i++ {
				reporter.logger.Infof("Pod %s \t RAM: %f Mi \n", dc.pods[i].Name, dc.pods[i].RAMMetric)
			}
		}
	}
}
