package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Database interface {
	Uri() (string, error)
	Database() (string, error)
	UseInMemoryDatabase() bool
}

type viperDatabaseConfig struct {
	v *viper.Viper
}

func (c *viperDatabaseConfig) Uri() (string, error) {
	if !c.v.IsSet("database.uri") {
		return "", fmt.Errorf("no mongodb uri was provided")
	}

	uri := c.v.GetString("database.uri")
	return uri, nil
}

func (c *viperDatabaseConfig) Database() (string, error) {
	if !c.v.IsSet("database.name") {
		return "", fmt.Errorf("no database name was provided")
	}

	name := c.v.GetString("database.name")
	return name, nil
}

func (c *viperDatabaseConfig) UseInMemoryDatabase() bool {
	if !c.v.IsSet("database.inMemory") {
		return false
	}

	return c.v.GetBool("database.inMemory")
}
