package handlers

import (
	"context"
	"fmt"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/scraper"
	"github.com/yukitsune/minialert/slices"
)

func GetAlerts(ctx context.Context, repo db.Repo, clientFactory prometheus.ClientFactory, guildId string, configName string) (prometheus.Alerts, error) {

	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild config: %s", err.Error())
	}

	scrapeConfig, ok := slices.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
		return cfg.Name == configName
	})
	if !ok {
		return nil, fmt.Errorf("couldn't find scrape config with name \"%s\"", configName)
	}

	client := clientFactory(scrapeConfig)
	alerts, err := client.GetAlerts()

	filteredAlerts, err := prometheus.FilterAlerts(alerts, scrapeConfig.InhibitedAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to filter alerts: %s", err.Error())
	}

	return filteredAlerts, nil
}

func GetInhibitions(ctx context.Context, configName string, guildId string, repo db.Repo) ([]string, error) {

	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild config: %s", err.Error())
	}

	scrapeConfig, ok := slices.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
		return cfg.Name == configName
	})
	if !ok {
		return nil, fmt.Errorf("couldn't find scrape config with name \"%s\"", configName)
	}

	return scrapeConfig.InhibitedAlerts, nil
}

func InhibitAlert(ctx context.Context, configName string, guildId string, alertName string, repo db.Repo) error {
	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return err
	}

	var scrapeConfig *db.ScrapeConfig
	for i, config := range guildConfig.ScrapeConfigs {
		if config.Name == configName {
			// Note: Here, `config` is a copy, need to use the index instead
			// https://stackoverflow.com/a/65292086
			scrapeConfig = &guildConfig.ScrapeConfigs[i]
		}
	}

	if scrapeConfig == nil {
		return fmt.Errorf("couldn't find scrape config with name \"%s\"", configName)
	}

	scrapeConfig.InhibitedAlerts = append(scrapeConfig.InhibitedAlerts, alertName)

	// Bug: Not working. Seems that the guild config isn't getting updated
	err = repo.SetGuildConfig(ctx, guildConfig)

	return err
}

func UninhibitAlert(ctx context.Context, configName string, guildId string, alertName string, repo db.Repo) error {
	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return fmt.Errorf("failed to get guild config: %s", err.Error())
	}

	scrapeConfig, ok := slices.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
		return cfg.Name == configName
	})
	if !ok {
		return fmt.Errorf("couldn't find scrape config with name \"%s\"", configName)
	}

	scrapeConfig.InhibitedAlerts = slices.RemoveMatches(scrapeConfig.InhibitedAlerts, func(inhibitedAlert string) bool {
		return inhibitedAlert == alertName
	})

	err = repo.SetGuildConfig(ctx, guildConfig)
	return err
}

func GetScrapeConfigs(ctx context.Context, repo db.Repo, guildId string) ([]db.ScrapeConfig, error) {
	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild config: %s", err)
	}

	return guildConfig.ScrapeConfigs, nil
}

func RemoveScrapeConfig(ctx context.Context, repo db.Repo, scrapeManager scraper.ScrapeManager, guildId string, configName string) error {

	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return fmt.Errorf("failed to get guild config: %s", err)
	}

	removed := false
	for i, cfg := range guildConfig.ScrapeConfigs {
		if cfg.Name == configName {
			guildConfig.ScrapeConfigs = append(guildConfig.ScrapeConfigs[:i], guildConfig.ScrapeConfigs[i+1:]...)
			removed = true
			break
		}
	}

	if !removed {
		return fmt.Errorf("couldn't find scrape config with name: \"%s\"", configName)
	}

	err = repo.SetGuildConfig(ctx, guildConfig)
	if err != nil {
		return fmt.Errorf("failed to set guild config: %s", err.Error())
	}

	err = scrapeManager.Stop(guildConfig.GuildId, configName)
	if err != nil {
		return fmt.Errorf("failed to stop scraper: %s", err.Error())
	}

	return nil
}
