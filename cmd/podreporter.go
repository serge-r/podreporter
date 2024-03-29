package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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

type task struct {
	pod *PodInfo
	dc  string
}

func CreateReporter(datacenters []Datacenter, prom *Prometheus, slackClient *slack.Client, logger *log.Entry, maxConcurrency int) *PodReporter {
	reporter := PodReporter{
		Datacenters:    datacenters,
		prom:           prom,
		slackClient:    slackClient,
		logger:         logger,
		maxConcurrency: maxConcurrency,
	}
	return &reporter
}

func (reporter *PodReporter) FillKubePods(namespaceSelector []string) error {
	var tempPods []PodInfo
	cluster := KubeCluster{}
	for i, dc := range reporter.Datacenters {
		err := cluster.AuthRemote(dc.KubeConfig)
		if err != nil {
			return err
		}
		tempPods, err = cluster.ReturnPods(namespaceSelector, reporter.logger)
		reporter.Datacenters[i].pods = tempPods
		if err != nil {
			return err
		}
	}
	return nil
}

func (reporter *PodReporter) worker(wg *sync.WaitGroup, T chan *task, R chan error, id int) {
	var cpu float64
	var ram float64

	for taskItem := range T {
		cpu = 0.0
		ram = 0.0

		dc := taskItem.dc

		reporter.logger.Debugf("ThreadID %d - Start query prom for DC %v and pod %v", id, dc, taskItem.pod.Name)
		contCPUQuery := fmt.Sprintf("max_over_time(sum(rate(container_cpu_usage_seconds_total{datacenter=\"%s\",namespace=\"%s\", pod=\"%s\", id=~\".*%s.*\"}))[7d:1m])",
			dc,
			taskItem.pod.Namespace,
			taskItem.pod.Name,
			taskItem.pod.Uid)
		contRAMQuery := fmt.Sprintf("max_over_time(sum(container_memory_rss{namespace=\"%s\",pod=\"%s\", id=~\".*%s.*\"})[7d:1m])",
			taskItem.pod.Namespace,
			taskItem.pod.Name,
			taskItem.pod.Uid)
		resultCPU, err := reporter.prom.InstanceQuery(contCPUQuery)
		if err != nil {
			reporter.logger.Debugf("Thread %d got an error %v", id, err)
			R <- err
			wg.Done()
			return
		}
		resultRAM, err := reporter.prom.InstanceQuery(contRAMQuery)
		if err != nil {
			reporter.logger.Debugf("Thread %d got an error %v", id, err)
			R <- err
			wg.Done()
			return
		}
		stringCPU := (*resultCPU).(string)
		stringRAM := (*resultRAM).(string)
		cpu, _ = strconv.ParseFloat(stringCPU, 64)
		ram, _ = strconv.ParseFloat(stringRAM, 64)
		//taskItem.pod.CPUMetric += cpu * 1000
		//taskItem.pod.RAMMetric += ram / 1024 / 1024
		taskItem.pod.UpdateMetrics(cpu, ram)
		taskItem.pod.SetRequestsRating()
		reporter.logger.Debugf("Thread id %d - PROM CPU is %f", id, cpu)
		reporter.logger.Debugf("Thread id %d - PROM RAM is %f", id, ram)
	}
	wg.Done()
	return
}

func (reporter *PodReporter) FillPrometheusInfo() error {

	tasksChannel := make(chan *task)
	errChannel := make(chan error)
	var err error = nil
	var wg sync.WaitGroup
	var counter = 0

	for i := 0; i < reporter.maxConcurrency; i++ {
		wg.Add(1)
		go reporter.worker(&wg, tasksChannel, errChannel, i)
	}

	reporter.logger.Debugf("Generated %d workers", reporter.maxConcurrency)

	start := time.Now()
	reporter.logger.Info("Starting processing pods...")
	go func() {
		for i, dc := range reporter.Datacenters {
			for j, pod := range dc.pods {
				newTask := task{&reporter.Datacenters[i].pods[j], dc.Name}
				select {
				case err = <-errChannel:
					close(tasksChannel)
					return
				case tasksChannel <- &newTask:
					reporter.logger.Debugf("Processing pod %v in dc %v", pod.Name, dc.Name)
				}
				counter++
			}
		}
		reporter.logger.Debugf("All tasks have been sending. Waiting until threads will be closing")
		close(tasksChannel)
	}()

	wg.Wait()
	elapsed := time.Since(start)
	reporter.logger.Infof("Processed %d pods for the %s", counter, elapsed)

	return err
}

func (reporter *PodReporter) GetReport(slackChannel string) {

	var podsOutput = 5

	reporter.logger.Info("Generating report")
	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: "plain_text",
			Text: ":newspaper: Daily kubernetes resources news :newspaper:"}),
	}

	curTime := time.Now()
	curTimeLine := fmt.Sprintf("*%s* | Dops team", curTime.Format("01-02-2006"))

	blocks = append(blocks, slack.NewContextBlock("HeadLine", slack.MixedElement(slack.TextBlockObject{
		Type: "mrkdwn",
		Text: curTimeLine,
	})))
	blocks = append(blocks, slack.NewDividerBlock())

	for _, dc := range reporter.Datacenters {

		if podsOutput > len(dc.pods) {
			podsOutput = len(dc.pods)
		}

		dcString := fmt.Sprintf(":office: *Datacenter:* %s", strings.ToUpper(dc.Name))
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: dcString,
			}, nil, nil))

		// Sort by CPU
		sort.Sort(PodByMetricCPUDesc(dc.pods))
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Top %d pods by CPU*", podsOutput),
			}, nil, nil))

		podString := ""

		for i := 0; i < podsOutput; i++ {
			podString += fmt.Sprintf("*Ns:* %s\t*Pod:* %s\t*CPU:* %.1fm\t *Limits:* %.1fm\n",
				dc.pods[i].Namespace,
				dc.pods[i].Name,
				dc.pods[i].CPUMetric,
				dc.pods[i].CPULimits)
		}
		blocks = append(blocks, slack.NewContextBlock(dc.Name+"-CPU", slack.MixedElement(slack.TextBlockObject{
			Type: "mrkdwn",
			Text: podString,
		})))

		// Sort By RAM
		sort.Sort(PodByMetricRAMDesc(dc.pods))
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Top %d pods by RAM*", podsOutput),
			}, nil, nil))

		podString = ""

		for i := 0; i < podsOutput; i++ {
			podString += fmt.Sprintf("*Ns:* %s\t*Pod:* %s\t*RAM:* %.1fMi\t *Limits:* %.1fMi\n",
				dc.pods[i].Namespace,
				dc.pods[i].Name,
				dc.pods[i].RAMMetric,
				dc.pods[i].RAMLimits)
		}
		blocks = append(blocks, slack.NewContextBlock(dc.Name+"-RAM", slack.MixedElement(slack.TextBlockObject{
			Type: "mrkdwn",
			Text: podString,
		})))

		// Sort by CPU rating
		sort.Sort(PodByRatingCPU(dc.pods))
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Top %d pods with possible wrong CPU requests*", podsOutput),
			}, nil, nil))

		podString = ""

		for i := 0; i < podsOutput; i++ {
			if dc.pods[i].RatingCPU > 5 {
				continue
			}
			podString += fmt.Sprintf("*Ns:* %s\t*Pod:* %s\t*CPU:* %.1fm\t *Requests:* %.1fm\n",
				dc.pods[i].Namespace,
				dc.pods[i].Name,
				dc.pods[i].RAMMetric,
				dc.pods[i].CPURequsts)
		}
		blocks = append(blocks, slack.NewContextBlock(dc.Name+"R-RAM", slack.MixedElement(slack.TextBlockObject{
			Type: "mrkdwn",
			Text: podString,
		})))
		blocks = append(blocks, slack.NewDividerBlock())
	}

	_, _, _, err := reporter.slackClient.SendMessage(
		slackChannel,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
	)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
}
