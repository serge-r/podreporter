package main

import (
	"context"
	"flag"
	"fmt"
	"k8s.io/client-go/rest"
	"os"
	//prometheus "github.com/ryotarai/prometheus-query/client"
	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//"github.com/ymotongpoo/datemaki"
)

type container struct {
	name []string
	cpu int64
	ram int64
}

type options struct {
	namespace string
	format string
	server string
	query  string
	start  string
	end    string
	step   string
}

func parseFlags() options {
	namespace := flag.String("namespace", "default","provide namespace to exclude from")
	format := flag.String("format", "json", "Format (available formats are json, tsv and csv)")
	server := flag.String("server", os.Getenv("PROMETHEUS_SERVER"), "Prometheus server URL like 'https://prometheus.example.com' (can be set by PROMETHEUS_SERVER environment variable)")
	query := flag.String("query", "", "Query")
	start := flag.String("start", "1 hour ago", "Start time")
	end := flag.String("end", "now", "End time")
	step := flag.String("step", "15s", "Step")

	flag.Parse()

	return options{
		namespace: *namespace,
		format: *format,
		server: *server,
		query:  *query,
		start:  *start,
		end:    *end,
		step:   *step,
	}
}

func kube_init(kubeconfig *string)  {

}

func main() {

	options := parseFlags()

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespaces,err := clientset.CoreV1().Namespaces().List(context.TODO(),metav1.ListOptions{
		TypeMeta:             metav1.TypeMeta{},
		LabelSelector:        "",
		FieldSelector:        fmt.Sprintf("metadata.name!=%s",options.namespace),
		Watch:                false,
		AllowWatchBookmarks:  false,
		ResourceVersion:      "",
		ResourceVersionMatch: "",
		TimeoutSeconds:       nil,
		Limit:                0,
		Continue:             "",
	})
	if err != nil {
		panic(err.Error())ß
	}
	fmt.Printf("I found a %s namespaces!",len(namespaces.Items))
	for _,namespace := range namespaces.Items {
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the namespace %s\n", len(pods.Items), namespace.Name)

		for i := 0; i < len(pods.Items); i++ {
			pod := pods.Items[i]
			fmt.Printf("In pod %s I found %d containers\n", pod.Name, len(pod.Spec.Containers))
			for j := 0 ; j < len(pod.Spec.Containers); j ++ {
				cnt := pod.Spec.Containers[j]
				fmt.Printf("Container %s with limits: CPU %d (mils) and RAM %d MB\n", cnt.Name, cnt.Resources.Limits.Cpu().MilliValue(), cnt.Resources.Limits.Memory().MilliValue() / 1000 /1024 /1024)
			}

		}
	}
}
