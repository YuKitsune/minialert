package handlers

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/scraper"
	"github.com/yukitsune/minialert/slices"
	"testing"
)

type FakePrometheusClient struct {
	Alerts prometheus.Alerts
}

func (f *FakePrometheusClient) GetAlerts() (prometheus.Alerts, error) {
	return f.Alerts, nil
}

type Scraper struct {
	GuildId string
	Config  *db.ScrapeConfig
}

type FakeScrapeManager struct {
	ActiveScrapers []Scraper
}

func (f *FakeScrapeManager) Start(guildId string, config *db.ScrapeConfig) {
	f.ActiveScrapers = append(f.ActiveScrapers, Scraper{
		GuildId: guildId,
		Config:  config,
	})
}

func (f *FakeScrapeManager) Chan() chan scraper.ScrapeResult {
	return make(chan scraper.ScrapeResult)
}

func (f *FakeScrapeManager) Restart(_ string, _ *db.ScrapeConfig) error {
	return nil
}

func (f *FakeScrapeManager) Stop(guildId string, configName string) error {
	f.ActiveScrapers = slices.RemoveMatches(f.ActiveScrapers, func(s Scraper) bool {
		return s.GuildId == guildId && s.Config.Name == configName
	})

	return nil
}

func TestGetAlertsGetsAlertsFromPrometheus(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)

	// Create the guild config
	guildId := "foo"
	configName := "bar"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts:       []string{},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	value := "foo"
	alerts := prometheus.Alerts{
		prometheus.Alert{Value: value},
	}

	clientFactory := func(config *db.ScrapeConfig) prometheus.Client {
		return &FakePrometheusClient{Alerts: alerts}
	}

	// Act
	foundAlerts, err := GetAlerts(ctx, repo, clientFactory, guildId, configName)
	assert.NoError(t, err)

	// Assert
	assert.Len(t, foundAlerts, 1)
	assert.Equal(t, value, foundAlerts[0].Value)
}

func TestGetAlertsFiltersInhibitedAlerts(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)

	// Create the guild config
	guildId := "foo"
	configName := "bar"
	alertNameToFilter := "zig"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts: []string{
					alertNameToFilter,
				},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	value := "foo"
	valueToFilter := "bar"
	alertName := "zag"
	alerts := prometheus.Alerts{
		prometheus.Alert{
			Value: valueToFilter,
			Labels: map[string]string{
				"alertname": alertNameToFilter,
			},
		},
		prometheus.Alert{Value: value,
			Labels: map[string]string{
				"alertname": alertName,
			},
		},
	}

	clientFactory := func(config *db.ScrapeConfig) prometheus.Client {
		return &FakePrometheusClient{Alerts: alerts}
	}

	// Act
	foundAlerts, err := GetAlerts(ctx, repo, clientFactory, guildId, configName)
	assert.NoError(t, err)

	// Assert
	assert.Len(t, foundAlerts, 1)
	assert.Equal(t, foundAlerts[0].Value, value)
}

func TestInhibitAlertUpdatesConfig(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)

	guildId := "foo"
	configName := "bar"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts:       []string{},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	// Act
	alertName := "fizz"
	err = InhibitAlert(ctx, configName, guildId, alertName, repo)
	assert.NoError(t, err)

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	assert.NoError(t, err)

	scrapeConfig, ok := slices.FindMatching(guildConfig.ScrapeConfigs, func(c db.ScrapeConfig) bool {
		return c.Name == configName
	})

	assert.True(t, ok)
	assert.Contains(t, scrapeConfig.InhibitedAlerts, alertName)
}

func TestUninhibitAlertUpdatesConfig(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)

	guildId := "foo"
	configName := "bar"
	alertName := "zig"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts:       []string{alertName},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	// Act
	err = UninhibitAlert(ctx, configName, guildId, alertName, repo)
	assert.NoError(t, err)

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	assert.NoError(t, err)

	scrapeConfig, ok := slices.FindMatching(guildConfig.ScrapeConfigs, func(c db.ScrapeConfig) bool {
		return c.Name == configName
	})

	assert.True(t, ok)
	assert.Empty(t, scrapeConfig.InhibitedAlerts)
}

func TestGetInhibitionsGetsInhibitions(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)

	guildId := "foo"
	configName := "bar"
	alertName := "zig"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts:       []string{alertName},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	// Act
	alertNames, err := GetInhibitions(ctx, configName, guildId, repo)
	assert.NoError(t, err)

	// Assert
	assert.Len(t, alertNames, 1)
	assert.Equal(t, alertNames[0], alertName)
}

// Todo: CreateScrapeConfigCreatesScrapeConfig
// Todo: CreateScrapeConfigStartsScraper

func TestRemoveScrapeConfigRemovesScrapeConfig(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)
	scrapeManager := &FakeScrapeManager{}

	guildId := "foo"
	configName := "bar"
	alertName := "zig"
	guildConfig := &db.GuildConfig{
		GuildId: guildId,
		ScrapeConfigs: []db.ScrapeConfig{
			{
				Name:                  configName,
				Endpoint:              "",
				Username:              "",
				Password:              "",
				ScrapeIntervalMinutes: 0,
				AlertChannelId:        "",
				InhibitedAlerts:       []string{alertName},
			},
		},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	// Act
	err = RemoveScrapeConfig(ctx, repo, scrapeManager, guildId, configName)
	assert.NoError(t, err)

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	assert.NoError(t, err)
	assert.Empty(t, guildConfig.ScrapeConfigs)
}

func TestRemoveScrapeConfigStopsScraper(t *testing.T) {

	// Arrange
	ctx := context.Background()
	logger := logrus.New()
	repo := db.SetupInMemoryDatabase(logger)
	scrapeManager := &FakeScrapeManager{}

	guildId := "foo"
	configName := "bar"
	alertName := "zig"
	scrapeConfig := db.ScrapeConfig{
		Name:                  configName,
		Endpoint:              "",
		Username:              "",
		Password:              "",
		ScrapeIntervalMinutes: 0,
		AlertChannelId:        "",
		InhibitedAlerts:       []string{alertName},
	}
	guildConfig := &db.GuildConfig{
		GuildId:       guildId,
		ScrapeConfigs: []db.ScrapeConfig{scrapeConfig},
	}

	err := repo.SetGuildConfig(ctx, guildConfig)
	assert.NoError(t, err)

	scrapeManager.Start(guildId, &scrapeConfig)

	// Sanity check
	assert.Len(t, scrapeManager.ActiveScrapers, 1)

	// Act
	err = RemoveScrapeConfig(ctx, repo, scrapeManager, guildId, configName)
	assert.NoError(t, err)

	// Assert
	assert.Empty(t, scrapeManager.ActiveScrapers)
}
