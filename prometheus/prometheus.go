package prometheus

import (
	"bytes"
	"encoding/json"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/util"
	"io/ioutil"
	"net/http"
	"time"
)

type Alerts []Alert

type Response struct {
	Data struct {
		Alerts []Alert `json:"alerts"`
	} `json:"data"`
	Status string `json:"status"`
}

type Alert struct {
	ActiveAt    time.Time
	Annotations map[string]string
	Labels      map[string]string
	State       string
	Value       string
}

type BasicAuthDetails struct {
	Username string
	Password string
}

type Client struct {
	client           http.Client
	endpoint         string
	basicAuthDetails *BasicAuthDetails
}

func NewPrometheusClient(client http.Client, endpoint string) *Client {
	return NewPrometheusClientWithBasicAuth(client, endpoint, nil)
}

func NewPrometheusClientWithBasicAuth(client http.Client, endpoint string, creds *BasicAuthDetails) *Client {
	return &Client{
		client:           client,
		endpoint:         endpoint,
		basicAuthDetails: creds,
	}
}

func NewPrometheusClientFromScrapeConfig(config *db.ScrapeConfig) *Client {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	if len(config.Username) > 0 && len(config.Password) > 0 {
		return NewPrometheusClientWithBasicAuth(client, config.Endpoint, &BasicAuthDetails{Username: config.Username, Password: config.Password})
	}

	return NewPrometheusClient(client, config.Endpoint)
}

func (c *Client) GetAlerts() (Alerts, error) {
	req, err := http.NewRequest("GET", c.endpoint, bytes.NewReader([]byte{}))
	if err != nil {
		return nil, err
	}

	if c.basicAuthDetails != nil {
		req.SetBasicAuth(c.basicAuthDetails.Username, c.basicAuthDetails.Password)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resData Response
	err = json.Unmarshal(resBytes, &resData)
	if err != nil {
		return nil, err
	}

	return resData.Data.Alerts, nil
}

func FilterAlerts(alerts Alerts, inhibitedAlerts []string) (Alerts, error) {

	var newAlerts Alerts
	for _, alert := range alerts {
		if !util.HasMatching(inhibitedAlerts, func(inhibitedAlert string) bool {
			alertName := alert.Labels["alertname"]
			return inhibitedAlert == alertName
		}) {
			newAlerts = append(newAlerts, alert)
		}
	}

	return newAlerts, nil
}
