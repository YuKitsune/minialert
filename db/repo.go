package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type CollectionName string

const (
	CommandRegistrationsCollection CollectionName = "command_registrations"
	AlertChannelsCollection        CollectionName = "alert_channels"
	AdminUsersCollection           CollectionName = "admin_users"
	InhibitionsCollection          CollectionName = "inhibitions"
)

func (c CollectionName) String() string {
	return string(c)
}

type CommandRegistration struct {
	CommandId   string `bson:"command_id"`
	GuildId     string `bson:"guild_id"`
	CommandName string `bson:"command_name"`
}

type AlertsChannel struct {
	GuildId   string `bson:"guild_id"`
	ChannelId string `bson:"channel_id"`
}

type AdminUser struct {
	GuildId string `bson:"guild_id"`
	UserlId string `bson:"user_id"`
}

type Inhibition struct {
	GuildId   string `bson:"guild_id"`
	AlertName string `bson:"alert_name"`
}

type Callback func(ctx context.Context, db *mongo.Database) error
type Func func(ctx context.Context, cb Callback) error

type Repo interface {
	RegisterCommand(ctx context.Context, guildId string, commandId string, commandName string) error
	GetRegisteredCommand(ctx context.Context, guildId string) ([]CommandRegistration, error)
	SetAlertsChannel(ctx context.Context, guildId string, channelId string) error
	GetAlertsChannel(ctx context.Context, guildId string) (*AlertsChannel, error)
	SetAdminUser(ctx context.Context, guildId string, channelId string) error
	CreateInhibition(ctx context.Context, guildId string, alertName string) error
	GetInhibitions(ctx context.Context, guildId string) ([]Inhibition, error)
	DeleteInhibition(ctx context.Context, guildId string, alertName string) error
	ClearGuildInfo(ctx context.Context, guildId string) error
}
