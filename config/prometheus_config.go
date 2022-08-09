package config

import (
	"github.com/spf13/viper"
	"time"
)

type Prometheus interface {
	Endpoint() string
	BasicAuth() (hasCreds bool, username string, password string)
	Timeout() time.Duration
}

type viperPrometheusConfig struct {
	v *viper.Viper
}

func (c *viperPrometheusConfig) Endpoint() string {
	if !c.v.IsSet("prometheus.endpoint") {
		panic("no prometheus endpoint was provided")
	}

	endpoint := c.v.GetString("prometheus.endpoint")
	return endpoint
}

func (c *viperPrometheusConfig) BasicAuth() (hasCreds bool, username string, password string) {
	un := c.v.GetString("prometheus.username")
	pw := c.v.GetString("prometheus.password")

	if len(un) > 0 || len(pw) > 0 {
		return true, un, pw
	}

	return false, "", ""
}

func (c *viperPrometheusConfig) Timeout() time.Duration {
	timeout := c.v.GetInt("prometheus.timeoutSeconds")
	return time.Duration(timeout) * time.Second
}
