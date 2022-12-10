package handlers

import (
	"context"
	"fmt"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/util"
)

func GetAlerts(ctx context.Context, repo db.Repo, clientFactory prometheus.ClientFactory, guildId string, configName string) (prometheus.Alerts, error) {

	guildConfig, err := repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild config: %s", err.Error())
	}

	scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

	scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

	//for _, config := range guildConfig.ScrapeConfigs {
	//	if config.Name == configName {
	//		scrapeConfig = &config
	//	}
	//}

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

	scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
		return cfg.Name == configName
	})
	if !ok {
		fmt.Errorf("couldn't find scrape config with name \"%s\"", configName)
	}

	scrapeConfig.InhibitedAlerts = util.RemoveMatching(scrapeConfig.InhibitedAlerts, func(inhibitedAlert string) bool {
		return inhibitedAlert == alertName
	})

	err = repo.SetGuildConfig(ctx, guildConfig)
	return err
}
