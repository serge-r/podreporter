package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Prometheus struct {
	server   *url.URL
	username string
	password string
	timeout  time.Duration
	isAuth   bool
	logger   *log.Entry
}

type Response struct {
	Status    string   `json:"status"`
	Data      *Data    `json:"data"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric interface{} `json:"metric,omitempty"`
	Value  *Value      `json:"value"`
}

type Value [2]interface{}

func PromCreate(server string, username string, password string, timeout time.Duration, logger *log.Entry) (*Prometheus, error) {
	var prom = Prometheus{}
	serverUrlString := fmt.Sprintf("http://%s/api/v1/query", server)
	serverUrl, err := url.Parse(serverUrlString)
	if err != nil {
		return nil, err
	}
	if len(username) > 0 {
		prom.username = username
		prom.password = password
		prom.isAuth = true
	}
	prom.server = serverUrl
	prom.timeout = timeout
	prom.logger = logger
	return &prom, nil
}

func (prom *Prometheus) InstanceQuery(query string) (*interface{}, error) {
	client := &http.Client{Timeout: prom.timeout}
	var data = url.Values{}
	var result = Response{}
	var zeroResult interface{} = "0"

	data.Set("query", query)
	prom.logger.Debugf("Qyery is %v", query)
	req, err := http.NewRequest("POST", prom.server.String(), strings.NewReader(data.Encode()))

	if err != nil {
		return nil, err
	}
	if prom.isAuth {
		req.SetBasicAuth(prom.username, prom.password)
		req.BasicAuth()
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, err := client.Do(req)
	prom.logger.Debugf("Response status: %d", resp.StatusCode)
	prom.logger.Debugf("Response headers: %v", resp.Header)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > 399 {
		err = fmt.Errorf("server returns %s", resp.Status)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	prom.logger.Debugf("Response body is: %v", string(body))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Data.Result) > 0 {
		//prom.logger.Debugf("Result is %v", result.Data.Result)
		return &result.Data.Result[0].Value[1], err
	}
	return &zeroResult, nil
}
