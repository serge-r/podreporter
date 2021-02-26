package main

import (
	"errors"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/serge-r/podreporter/pkg/podreporter"
	prometheus "github.com/serge-r/podreporter/pkg/prometheus-client"
	vault "github.com/serge-r/podreporter/pkg/vault-client"
	"github.com/slack-go/slack"
	"log"
	"net/url"
	"os"
	"time"
)

type options struct {
	//Namespaces          []string      `env:"NAMESPACES" envSeparator:":"`
	Datacenters         []string      `env:"DATACENTERS" envSeparator:":"`
	PrometheusServerUrl url.URL       `env:"PROM_SERVER_URL"`
	PrometheusUsername  string        `env:"PROM_USERNAME"`
	PrometheusPassword  string        `env:"PROM_PASSWORD"`
	PrometheusTimeout   time.Duration `env:"PROM_TIMEOUT" envDefault:"5s"`
	VaultURL            url.URL       `env:"VAULT_URL"`
	VaultTimeout        time.Duration `env:"VAULT_TIMEOUT" envDefault:"5s"`
	VaultRoleID         string        `env:"VAULT_ROLE_ID"`
	VaultSecretID       string        `env:"VAULT_SECRET_ID"`
	VaultSecretPath     string        `env:"VAULT_SECRET_PATH"`
	VaultEnvironment    string      `env:"VAULT_ENVIRONMENT" envDefault:"production:development" envSeparator:":"`
	SlackToken          string        `env:"SLACK_TOKEN"`
	//SlackWebhook        string        `env:"SLACK_WEBHOOK"`
	//SlackChannel        string        `env:"SLACK_CHANNEL"`
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
	if options.PrometheusServerUrl.String() == "" {
		return nil, errors.New("prometheus server URL is not provided")
	}
	return &options, nil
}

func main() {
	var DCs []podreporter.Datacenter

	// Init options
	options, err := parseOptions()
	if err != nil {
		onError(err)
	}

	// Init vault client
	vaultClient, err := vault.VaultAuth(options.VaultURL,options.VaultTimeout, options.VaultSecretID, options.VaultRoleID)
	if err != nil {
		onError(err)
	}

	// Init prometheus client
	prom,err := prometheus.Init(options.PrometheusServerUrl.String(), options.PrometheusUsername, options.PrometheusPassword, options.PrometheusTimeout)
	if err != nil {
		onError(err)
	}

	// Fill datacenters structure
	for _,dc := range options.Datacenters {
		tempDc := podreporter.Datacenter{}
		tempDc.Name = dc
		for _,env := range options.VaultEnvironment {
			secretPath := fmt.Sprintf("%s/%s/%s", options.VaultSecretPath, env, dc)
			//fmt.Printf(secretPath)
			config, err := vault.VaultReturnSecret(vaultClient, secretPath, "config")
			if err != nil {
				//TODO: logging info
			}
			if config != nil {
				tempDc.KubeConfig = config
			}
		}
		if tempDc.KubeConfig == nil {
			//TODO: logginig warn
			continue
		}
		DCs = append(DCs, tempDc)
	}

	// Init slack
	slackClient := slack.New(options.SlackToken)
	_, err = slackClient.AuthTest()
	if err != nil {
		onError(err)
	}

	// Execute reporter
	reporter := podreporter.Init(DCs, prom, slackClient)
	reporter.Run()

}