package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
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
