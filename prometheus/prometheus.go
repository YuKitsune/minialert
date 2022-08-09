package prometheus

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	return NewClientWithBasicAuth(client, endpoint, nil)
}

func NewClientWithBasicAuth(client http.Client, endpoint string, creds *BasicAuthDetails) *Client {
	return &Client{
		client:           client,
		endpoint:         endpoint,
		basicAuthDetails: creds,
	}
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

	fmt.Printf("Status: %s\n", res.Status)

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
