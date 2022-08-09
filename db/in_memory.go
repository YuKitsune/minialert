package db

import (
	"context"
	"fmt"
)

func SetupInMemoryDatabase() Repo {
	repo := &inMemoryRepo{
		registeredCommands: make([]CommandRegistration, 0),
		alertChannels:      make([]AlertsChannel, 0),
		adminUsers:         make([]AdminUser, 0),
		inhibitions:        make([]Inhibition, 0),
	}

	return repo
}

type inMemoryRepo struct {
	registeredCommands []CommandRegistration
	alertChannels      []AlertsChannel
	adminUsers         []AdminUser
	inhibitions        []Inhibition
}

func (r *inMemoryRepo) RegisterCommand(_ context.Context, guildId string, commandId string, commandName string) error {
	reg := CommandRegistration{
		GuildId:     guildId,
		CommandId:   commandId,
		CommandName: commandName,
	}
	r.registeredCommands = append(r.registeredCommands, reg)

	return nil
}

func (r *inMemoryRepo) GetRegisteredCommand(_ context.Context, guildId string) ([]CommandRegistration, error) {
	var commands []CommandRegistration
	for _, command := range r.registeredCommands {
		if command.GuildId == guildId {
			commands = append(commands, command)
		}
	}

	return commands, nil
}

func (r *inMemoryRepo) SetAlertsChannel(_ context.Context, guildId string, channelId string) error {
	r.alertChannels = append(r.alertChannels, AlertsChannel{GuildId: guildId, ChannelId: channelId})
	return nil
}

func (r *inMemoryRepo) GetAlertsChannel(_ context.Context, guildId string) (*AlertsChannel, error) {
	for _, channel := range r.alertChannels {
		if channel.GuildId == guildId {
			return &channel, nil
		}
	}

	return nil, fmt.Errorf("no alerts channel has been set")
}

func (r *inMemoryRepo) SetAdminUser(_ context.Context, guildId string, adminId string) error {
	r.adminUsers = append(r.adminUsers, AdminUser{GuildId: guildId, UserlId: adminId})
	return nil
}

func (r *inMemoryRepo) CreateInhibition(_ context.Context, guildId string, alertName string) error {
	r.inhibitions = append(r.inhibitions, Inhibition{GuildId: guildId, AlertName: alertName})
	return nil
}

func (r *inMemoryRepo) DeleteInhibition(_ context.Context, guildId string, alertName string) error {
	r.inhibitions = removeMatching(r.inhibitions, func(inhibition Inhibition) bool {
		return inhibition.GuildId == guildId && inhibition.AlertName == alertName
	})

	return nil
}

func (r *inMemoryRepo) ClearGuildInfo(_ context.Context, guildId string) error {
	r.registeredCommands = removeMatching(r.registeredCommands, func(command CommandRegistration) bool {
		return command.GuildId == guildId
	})

	r.alertChannels = removeMatching(r.alertChannels, func(alertChannel AlertsChannel) bool {
		return alertChannel.GuildId == guildId
	})

	r.adminUsers = removeMatching(r.adminUsers, func(adminUser AdminUser) bool {
		return adminUser.GuildId == guildId
	})

	r.inhibitions = removeMatching(r.inhibitions, func(inhibition Inhibition) bool {
		return inhibition.GuildId == guildId
	})

	return nil
}

func removeMatching[T any](s []T, fn func(v T) bool) []T {

	var indexes []int
	for i, item := range s {
		if fn(item) {
			indexes = append(indexes, i)
		}
	}

	for _, index := range indexes {
		s = remove(s, index)
	}

	return s
}

func remove[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
