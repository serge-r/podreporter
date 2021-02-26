package prometheus_client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Prometheus struct {
	server *url.URL
	username string
	password string
	timeout time.Duration
	isAuth bool
}

type Response struct {
	Status    string	`json:"status"`
	Data      *Data		`json:"data"`
	ErrorType string	`json:"errorType,omitempty"`
	Error     string	`json:"error,omitempty"`
	Warnings  []string	`json:"warnings,omitempty"`
}

type Data struct {
	ResultType string	`json:"resultType"`
	Result []Result		`json:"result"`
}

type Result struct {
	Metric interface{}	`json:"metric,omitempty"`
	Value *Value		`json:"value"`
}

type Value [2]interface{}

func Init(server string, username string, password string, timeout time.Duration) (*Prometheus,error) {
	var prom = Prometheus{}
	serverUrlString := fmt.Sprintf("http://%s/api/v1/query",server)
	serverUrl, err := url.Parse(serverUrlString)
	if err != nil {
		return nil,err
	}
	if len(username) > 0 {
		prom.username = username
		prom.password = password
		prom.isAuth = true
	}
	prom.server = serverUrl
	prom.timeout = timeout
	return &prom,nil
}

func (prom *Prometheus) InstanceQuery(query string) (*Response,error)  {
	client := &http.Client{Timeout: prom.timeout}
	var data = url.Values{}
	var result = Response{}

	data.Set("query", query)
	req, err := http.NewRequest("POST", prom.server.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil,err
	}
	if prom.isAuth {
		req.SetBasicAuth(prom.username, prom.password)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp,err := client.Do(req)
	if err != nil {
		return nil,err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil,err
	}
	err = json.Unmarshal(body,&result)
	return &result,err
}