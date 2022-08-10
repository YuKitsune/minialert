package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Log interface {
	Level() logrus.Level
}

type viperLogConfig struct {
	v *viper.Viper
}

func (c *viperLogConfig) Level() logrus.Level {

	level := c.v.GetString("log.level")
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		panic(fmt.Sprintf("could not parse log.level: %s", err.Error()))
	}

	return lvl
}
