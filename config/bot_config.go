package config

import "github.com/spf13/viper"

type Bot interface {
	Token() string
	ClientId() string
	Permissions() string
	Scopes() []string
}

type viperBotConfig struct {
	v *viper.Viper
}

func (c *viperBotConfig) Token() string {
	if !c.v.IsSet("bot.token") {
		panic("no discord bot token was provided")
	}

	token := c.v.GetString("bot.token")
	return token
}

func (c *viperBotConfig) ClientId() string {
	if !c.v.IsSet("bot.clientId") {
		panic("no discord bot client id was provided")
	}

	clientID := c.v.GetString("bot.clientId")
	return clientID
}

func (c *viperBotConfig) Permissions() string {
	if !c.v.IsSet("bot.permissions") {
		panic("no discord bot permissions were provided")
	}

	perms := c.v.GetString("bot.permissions")
	return perms
}

func (c *viperBotConfig) Scopes() []string {
	if !c.v.IsSet("bot.scopes") {
		panic("no discord bot scopes were provided")
	}

	scopes := c.v.GetStringSlice("bot.scopes")
	return scopes
}
