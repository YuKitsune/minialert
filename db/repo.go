package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type CollectionName string

const (
	CommandRegistrationsCollection CollectionName = "command_registrations"
	GuildConfigCollection          CollectionName = "guild_config"
)

func (c CollectionName) String() string {
	return string(c)
}

type GuildConfig struct {
	GuildId       string         `bson:"guild_id"`
	ScrapeConfigs []ScrapeConfig `bson:"scrape_configs"`
}

func NewGuildConfig(guildId string) *GuildConfig {
	return &GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: make([]ScrapeConfig, 0),
	}
}

type ScrapeConfig struct {
	Name                  string   `bson:"scrape_name"`
	Endpoint              string   `bson:"endpoint"`
	Username              string   `bson:"username"`
	Password              string   `bson:"password"`
	ScrapeIntervalMinutes int64    `bson:"scrape_interval_minutes"`
	AlertChannelId        string   `bson:"alert_channel_id"`
	InhibitedAlerts       []string `bson:"inhibited_alerts"`
}

type CommandRegistration struct {
	GuildId     string `bson:"guild_id"`
	CommandId   string `bson:"command_id"`
	CommandName string `bson:"command_name"`
}

type Callback func(ctx context.Context, db *mongo.Database) error
type Func func(ctx context.Context, cb Callback) error

type Repo interface {
	RegisterCommand(ctx context.Context, guildId string, commandId string, commandName string) error
	GetRegisteredCommands(ctx context.Context, guildId string) ([]CommandRegistration, error)
	GetGuildConfigs(ctx context.Context) ([]GuildConfig, error)
	GetGuildConfig(ctx context.Context, guildId string) (*GuildConfig, error)
	SetGuildConfig(ctx context.Context, config *GuildConfig) error
	DeleteGuildConfig(ctx context.Context, guildId string) error
	ClearGuildInfo(ctx context.Context, guildId string) error
}
