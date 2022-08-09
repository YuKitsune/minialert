package config

import "github.com/spf13/viper"

type Database interface {
	Uri() string
	Database() string
}

type viperDatabaseConfig struct {
	v *viper.Viper
}

func (c *viperDatabaseConfig) Uri() string {
	if !c.v.IsSet("database.uri") {
		panic("no mongodb uri was provided")
	}

	uri := c.v.GetString("database.uri")
	return uri
}

func (c *viperDatabaseConfig) Database() string {
	if !c.v.IsSet("database.name") {
		panic("no database name was provided")
	}

	name := c.v.GetString("database.name")
	return name
}
