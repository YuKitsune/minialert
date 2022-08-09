package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"strings"
)

func Setup() (Config, error) {

	v := viper.New()

	// Set defaults
	v.SetDefault("scrapeIntervalMinutes", 30)
	v.SetDefault("prometheus.username", "")
	v.SetDefault("prometheus.password", "")
	v.SetDefault("prometheus.timeoutSeconds", 5)
	v.SetDefault("database.scopes", []string{"bot", "application.commands"})

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
		// Todo: Log
		fmt.Println("Config file changed:", e.Name)
	})

	// Load config from file
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error
			// Todo: trace logs
			fmt.Println("No config file found")
		} else {
			return nil, err
		}
	}

	// Todo: Hide behind debug flag
	v.Debug()

	return NewConfigProvider(v), nil
}
