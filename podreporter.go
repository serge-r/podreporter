package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/caarlos0/env/v6"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/url"
	"os"
	"time"

	prometheus "github.com/serge-r/podreporter/pkg/prometheus-client"
	vault "github.com/serge-r/podreporter/pkg/vault-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type podInfo struct {
	Name       string
	Namespace  string
	Datacenter string
	Containers []containerInfo
}

type containerInfo struct {
	Name      string
	Image     string
	Resources *resourcesInfo
}

type resourcesInfo struct {
	LimitsCPU int64
	LimitsRAM int64
	MetricCPU int64
	MetricRAM int64
}

type kubeCluster struct {
	Datacenter string
	Config     *rest.Config
}

type options struct {
	Namespaces          []string      `env:"NAMESPACES" envSeparator:":"`
	Datacenters         []string      `env:"DATACENTERS" envSeparator:":"`
	AuthType            string        `env:"AUTH_TYPE" envDefault:"vault"`
	PrometheusServerUrl url.URL       `env:"PROM_SERVER_URL"`
	PrometheusUsername  string        `env:"PROM_USERNAME"`
	PrometheusPassword  string        `env:"PROM_PASSWORD"`
	PrometheusTimeout   time.Duration `env:"PROM_TIMEOUT" envDefault:"5s"`
	VaultURL            url.URL       `env:"VAULT_URL"`
	VaultTimeout        time.Duration `env:"VAULT_TIMEOUT" envDefault:"5s"`
	VaultRoleID         string        `env:"VAULT_ROLE_ID"`
	VaultSecretID       string        `env:"VAULT_SECRET_ID"`
	VaultSecretPath     string        `env:"VAULT_SECRET_PATH"`
	VaultEnvironment    []string      `env:"VAULT_ENVIRONMENT" envDefault:"development" envSeparator:":"`
	SlackToken          string        `env:"SLACK_TOKEN"`
	SlackWebhook        string        `env:"SLACK_WEBHOOK"`
	SlackChannel        string        `env:"SLACK_CHANNEL"`
}

func onError(err error) {
	log.Println(err)
	os.Exit(1)
}

func parseOptions() (*options, error) {
	options := options{}
	if err := env.Parse(&options); err != nil {
		return nil, err
	}
	switch options.AuthType {
	case "vault":
		if len(options.Datacenters) == 0 {
			return nil, errors.New("datacenters list is not provided")
		}
		if options.VaultURL.String() == "" {
			return nil, errors.New("vault server URL is not provided")
		}
		if options.VaultRoleID == "" {
			return nil, errors.New("vault role ID is not provided")
		}
		if options.VaultSecretID == "" {
			return nil, errors.New("vault secret ID is not provided")
		}
		if options.VaultSecretPath == "" {
			return nil, errors.New("vault secret path is not provided")
		}
	}
	if options.PrometheusServerUrl.String() == "" {
		return nil, errors.New("prometheus server URL is not provided")
	}
	return &options, nil
}

func (kub *kubeCluster) Auth(authType string, configFile []byte) error {
	switch authType {
	case "incluster":
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		kub.Config = config
		return nil
	default:
		config, err := clientcmd.RESTConfigFromKubeConfig(configFile)
		if err != nil {
			return err
		}
		kub.Config = config
		return nil
	}
}

func (kub *kubeCluster) ReturnPods(authtype []string, namespacesList string) *[]podInfo {

	var podsReport []podInfo
	var tempPod podInfo
	var tempCont containerInfo
	var namespacesSelector string

	if len(namespacesList) > 0 {
		namespacesSelector = fmt.Sprintf("metadata.Name!=%s", namespacesList)
	}

	clientset, err := kubernetes.NewForConfig(kub.Config)
	if err != nil {
		onError(err)
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
		onError(err)
	}

	//fmt.Printf("I found a %d namespaces!",len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			onError(err)
		}
		//fmt.Printf("There are %d pods in the namespace %s\n", len(pods.Items), namespace.Name)

		for _, pod := range pods.Items {
			//log.Printf("In pod %s I found %d containers\n", pod.Name, len(pod.Spec.Containers))
			for _, cnt := range pod.Spec.Containers {
				tempPod.Name = pod.Name
				tempPod.Namespace = namespace.Name
				tempCont.Name = cnt.Name
				tempCont.Image = cnt.Image
				tempCont.Resources.LimitsCPU = cnt.Resources.Limits.Cpu().MilliValue()
				tempCont.Resources.LimitsRAM = cnt.Resources.Limits.Memory().MilliValue() / 1000 / 1024 / 1024
				tempPod.Containers = append(tempPod.Containers, tempCont)
				podsReport = append(podsReport, tempPod)
			}
		}
	}
	return &podsReport
}

func (podinfo *podInfo) getPrometheusInfo(prom *prometheus.Prometheus, timeout int) error {

	for _, cont := range podinfo.Containers {
		contCPUQuery := fmt.Sprintf("max_over_time(rate(container_cpu_usage_seconds_total{namespace=\"%s\",pod=\"%s\",container=\"%s\",image!=\"\"}[1w]))",
			podinfo.Namespace,
			podinfo.Name,
			cont.Name)
		contRAMQuery := fmt.Sprintf("max_over_time(rate(container_memory_rss{namespace=\"%s\",pod=\"%s\",container=\"%s\",image!=\"\"}[1w]}))",
			podinfo.Namespace,
			podinfo.Name,
			cont.Name)
		resultCPU, err := prom.InstanceQuery(contCPUQuery, timeout)
		if err != nil {
			return err
		}
		resultRAM, err := prom.InstanceQuery(contRAMQuery, timeout)
		if err != nil {
			return err
		}
		cont.Resources.MetricCPU = resultCPU.Data.Result[0].Value[1].(int64)
		cont.Resources.MetricRAM = resultRAM.Data.Result[0].Value[1].(int64)
	}
	return nil
}

func main() {
	options, err := parseOptions()
	if err != nil {
		onError(err)
	}
	fmt.Println("Just a test")

	vaultClient, err := vault.VaultAuth(options.VaultURL,options.VaultTimeout, options.VaultSecretID, options.VaultRoleID)
	if err != nil {
		onError(err)
	}
	secretPath := fmt.Sprintf("%s%s/%s",options.VaultSecretPath,options.VaultEnvironment[0],options.Datacenters[0])
	fmt.Printf(secretPath)
	config,err := vault.VaultReturnSecret(vaultClient,secretPath,"config")
	if err != nil {
		onError(err)
	}
	fmt.Printf("%s", config)


}
