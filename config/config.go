package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

func Setup(configFile string, v *viper.Viper, logger logrus.FieldLogger) Config {

	// Set defaults
	v.SetDefault("prometheus.timeoutSeconds", 5)
	v.SetDefault("bot.scopes", []string{"bot", "application.commands"})
	v.SetDefault("log.level", "info")

	// Environment variables
	v.SetEnvPrefix("MINIALERT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	// Config file
	if len(configFile) > 0 {
		logger.Infof("🔧 Loading config from %s", configFile)
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("minialert")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/minialert")
		v.AddConfigPath("../configs")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Watch for changes
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		logger.Infof("♻️ Config file changed: ", e.Name)
	})

	// Load config from file
	_ = v.ReadInConfig()
	return NewConfigProvider(v)
}

type Config interface {
	Database() Database
	Bot() Bot
	Log() Log
	Debug() string
}

type viperConfig struct {
	v   *viper.Viper
	db  *viperDatabaseConfig
	bot *viperBotConfig
	log *viperLogConfig
}

func NewConfigProvider(v *viper.Viper) Config {
	return &viperConfig{
		v:   v,
		db:  &viperDatabaseConfig{v},
		bot: &viperBotConfig{v},
		log: &viperLogConfig{v},
	}
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

func (c *viperConfig) Debug() string {
	return fmt.Sprintf("%#v", c.v.AllSettings())
}
