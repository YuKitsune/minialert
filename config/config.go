package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"strings"
	"time"
)

func Setup(logger logrus.FieldLogger) (Config, error) {

	v := viper.New()

	// Set defaults
	v.SetDefault("scrapeIntervalMinutes", 30)
	v.SetDefault("prometheus.username", "")
	v.SetDefault("prometheus.password", "")
	v.SetDefault("prometheus.timeoutSeconds", 5)
	v.SetDefault("database.scopes", []string{"bot", "application.commands"})
	v.SetDefault("log.level", "info")

	// Environment variables
	v.SetEnvPrefix("MINIALERT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	// Config file
	v.SetConfigName("minialert")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/minialert")
	v.AddConfigPath("../configs")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// Watch for changes
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		logger.Infof("Config file changed: ", e.Name)
	})

	// Load config from file
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	return NewConfigProvider(v), nil
}

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
