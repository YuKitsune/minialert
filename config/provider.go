package config

import (
	"fmt"
	"github.com/spf13/viper"
	"io"
	"time"
)

type Config interface {
	ScrapeInterval() time.Duration
	Prometheus() Prometheus
	Database() Database
	Bot() Bot
	Log() Log
	Debug(w io.Writer)
}

type viperConfig struct {
	v    *viper.Viper
	prom *viperPrometheusConfig
	db   *viperDatabaseConfig
	bot  *viperBotConfig
	log  *viperLogConfig
}

func NewConfigProvider(v *viper.Viper) Config {
	return &viperConfig{
		v:    v,
		prom: &viperPrometheusConfig{v},
		db:   &viperDatabaseConfig{v},
		bot:  &viperBotConfig{v},
		log:  &viperLogConfig{v},
	}
}

func (c *viperConfig) ScrapeInterval() time.Duration {
	intervalMinutes := c.v.GetInt("scrapeIntervalMinutes")
	return time.Duration(intervalMinutes) * time.Minute
}

func (c *viperConfig) Prometheus() Prometheus {
	return c.prom
}

func (c *viperConfig) Database() Database {
	return c.db
}

func (c *viperConfig) Bot() Bot {
	return c.bot
}

func (c *viperConfig) Log() Log {
	return c.log
}

func (c *viperConfig) Debug(w io.Writer) {
	fmt.Fprintf(w, "%#v", c.v.AllSettings())
}
