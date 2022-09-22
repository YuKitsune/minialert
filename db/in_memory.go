package db

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

func SetupInMemoryDatabase(logger logrus.FieldLogger) Repo {
	repo := &inMemoryRepo{
		registeredCommands: make([]CommandRegistration, 0),
		guildConfigs:       make([]GuildConfig, 0),
		logger:             logger,
	}

	return repo
}

type inMemoryRepo struct {
	registeredCommands []CommandRegistration
	guildConfigs       []GuildConfig
	logger             logrus.FieldLogger
}

func (r *inMemoryRepo) RegisterCommand(_ context.Context, guildId string, commandId string, commandName string) error {
	reg := CommandRegistration{
		GuildId:     guildId,
		CommandId:   commandId,
		CommandName: commandName,
	}
	r.registeredCommands = append(r.registeredCommands, reg)

	r.logger.Debugf("Registering command: %+v", reg)

	return nil
}

func (r *inMemoryRepo) GetRegisteredCommands(_ context.Context, guildId string) ([]CommandRegistration, error) {
	var commands []CommandRegistration
	for _, command := range r.registeredCommands {
		if command.GuildId == guildId {
			commands = append(commands, command)
		}
	}

	return commands, nil
}

func (r *inMemoryRepo) GetGuildConfigs(_ context.Context) ([]GuildConfig, error) {
	return r.guildConfigs, nil
}

func (r *inMemoryRepo) GetGuildConfig(_ context.Context, guildId string) (*GuildConfig, error) {
	for _, config := range r.guildConfigs {
		if config.GuildId == guildId {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("no config found for guild %s", guildId)
}

func (r *inMemoryRepo) SetGuildConfig(_ context.Context, config *GuildConfig) error {

	r.logger.Debugf("Setting guild config: %+v", config)

	for i, cfg := range r.guildConfigs {
		if cfg.GuildId == config.GuildId {
			r.guildConfigs[i] = *config
			return nil
		}
	}

	r.guildConfigs = append(r.guildConfigs, *config)
	return nil
}

func (r *inMemoryRepo) DeleteGuildConfig(_ context.Context, guildId string) error {
	for i, cfg := range r.guildConfigs {
		if cfg.GuildId == guildId {
			r.guildConfigs = append(r.guildConfigs[:i], r.guildConfigs[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("no config found for guild %s", guildId)
}

func (r *inMemoryRepo) ClearGuildInfo(_ context.Context, guildId string) error {
	r.registeredCommands = removeMatching(r.registeredCommands, func(command CommandRegistration) bool {
		return command.GuildId == guildId
	})

	r.guildConfigs = removeMatching(r.guildConfigs, func(config GuildConfig) bool {
		return config.GuildId == guildId
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
