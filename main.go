package main

import (
	"errors"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/serge-r/podreporter/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"net/url"
	"os"
	"strings"
	"time"
)

type options struct {
	LogType             string        `env:"LOG_TYPE" envDefault:"text"`
	LogLevel            string        `env:"LOG_LEVEL" envDefault:"info"`
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
	VaultEnvironment    []string      `env:"VAULT_ENVIRONMENT" envDefault:"production:development" envSeparator:":"`
	SlackBotToken       string        `env:"SLACK_BOT_TOKEN"`
	SlackAppToken       string        `env:"SLACK_APP_TOKEN"`
	MaxConcurrency      int           `env:"MAX_CONCURRENCY" envDefault:"2"`
}

func initLog(o *options) *log.Entry {
	switch strings.ToLower(o.LogType) {
	case "text":
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})

	default:
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	}

	switch strings.ToLower(o.LogLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	return log.WithField("context", "deploy")
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
	if options.SlackBotToken == "" {
		return nil, errors.New("slack BOT token not provided")
	}
	if options.SlackAppToken == "" {
		return nil, errors.New("slack APP token not provided")
	}
	if options.MaxConcurrency < 2 {
		return nil, errors.New("Please set max concurency >= 2")
	}
	return &options, nil
}

func main() {
	var datacenters []cmd.Datacenter

	// PromCreate options
	options, err := parseOptions()
	if err != nil {
		panic(err)
	}

	// PromCreate logs
	logger := initLog(options)

	logger.Infof("Start app")

	// PromCreate vault client
	vaultClient, err := cmd.VaultAuth(options.VaultURL,
		options.VaultTimeout,
		options.VaultSecretID,
		options.VaultRoleID,
	)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("Auth in vault successfully")

	// PromCreate prometheus client
	prom, err := cmd.PromCreate(options.PrometheusServerUrl.String(), options.PrometheusUsername, options.PrometheusPassword, options.PrometheusTimeout, logger)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Prometheus client ready")

	// Fill datacenters structure
	logger.Debug("I found next datacenters %v", options.Datacenters)
	for _, dc := range options.Datacenters {
		tempDc := cmd.Datacenter{}
		tempDc.Name = dc
		for _, environment := range options.VaultEnvironment {
			secretPath := fmt.Sprintf("%s/%s/%s", options.VaultSecretPath, environment, dc)
			logger.Debug(fmt.Sprintf("I will read secrets from %v", secretPath))
			config, err := cmd.VaultReturnSecret(vaultClient, secretPath, "config")
			if err != nil {
				logger.Fatal(err)
			}
			if config != nil {
				tempDc.KubeConfig = config
			}
		}
		if tempDc.KubeConfig == nil {
			logger.Warn("Cannot find config for %v+", tempDc.Name)
			continue
		}
		datacenters = append(datacenters, tempDc)
	}

	logger.Info("Creating Slack connection")
	// PromCreate slack
	slackClient := slack.New(
		options.SlackBotToken,
		slack.OptionDebug(false),
		slack.OptionAppLevelToken(options.SlackAppToken),
	)
	_, err = slackClient.AuthTest()
	if err != nil {
		logger.Fatal(err)
	}

	// Execute reporter
	logger.Info("Creating reporter")
	reporter := cmd.CreateReporter(datacenters, prom, slackClient, logger, options.MaxConcurrency)
	err = reporter.FillKubePods()
	if err != nil {
		logger.Errorf("Error filling pods: %v", err)
		os.Exit(1)
	}
	err = reporter.FillPrometheusInfo()
	if err != nil {
		logger.Errorf("Error get prometheus info: %v", err)
		os.Exit(1)
	}
	reporter.GetReport("lux")

}
