package handlers

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/util"
	"golang.org/x/exp/slices"
	"testing"
)

type FakePrometheusClient struct {
	Alerts prometheus.Alerts
}

func (f *FakePrometheusClient) GetAlerts() (prometheus.Alerts, error) {
	return f.Alerts, nil
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
	if err != nil {
		t.Fail()
	}

	value := "foo"
	alerts := prometheus.Alerts{
		prometheus.Alert{Value: value},
	}

	clientFactory := func(config *db.ScrapeConfig) prometheus.Client {
		return &FakePrometheusClient{Alerts: alerts}
	}

	// Act
	foundAlerts, err := GetAlerts(ctx, repo, clientFactory, guildId, configName)
	if err != nil {
		t.Fail()
	}

	// Assert
	if foundAlerts[0].Value != value {
		t.Fail()
	}
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
	if err != nil {
		t.Fail()
	}

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
	if err != nil {
		t.Fail()
	}

	// Assert
	if len(foundAlerts) != 1 {
		t.Fail()
	}

	if foundAlerts[0].Value != value {
		t.Fail()
	}
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
	if err != nil {
		t.Fail()
	}

	// Act
	alertName := "fizz"
	err = InhibitAlert(ctx, configName, guildId, alertName, repo)
	if err != nil {
		t.Fail()
	}

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		t.Fail()
	}

	scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(c db.ScrapeConfig) bool {
		return c.Name == configName
	})

	if !ok {
		t.Fail()
	}

	if !slices.Contains(scrapeConfig.InhibitedAlerts, alertName) {
		t.Fail()
	}
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
	if err != nil {
		t.Fail()
	}

	// Act
	err = UninhibitAlert(ctx, configName, guildId, alertName, repo)
	if err != nil {
		t.Fail()
	}

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	if err != nil {
		t.Fail()
	}

	scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(c db.ScrapeConfig) bool {
		return c.Name == configName
	})

	if !ok {
		t.Fail()
	}

	if len(scrapeConfig.InhibitedAlerts) != 0 {
		t.Fail()
	}
}

// Todo: GetInhibitions
// Todo: RemoveScrapeConfig
