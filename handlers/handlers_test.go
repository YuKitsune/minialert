package handlers

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/scraper"
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
	f.ActiveScrapers = util.RemoveMatches(f.ActiveScrapers, func(s Scraper) bool {
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
	if err != nil {
		t.Fail()
	}

	// Act
	alertNames, err := GetInhibitions(ctx, configName, guildId, repo)
	if err != nil {
		t.Fail()
	}

	// Assert
	if len(alertNames) != 1 {
		t.Fail()
	}

	if alertNames[0] != alertName {
		t.Fail()
	}
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
	if err != nil {
		t.Fail()
	}

	// Act
	err = RemoveScrapeConfig(ctx, repo, scrapeManager, guildId, configName)
	if err != nil {
		t.Fail()
	}

	// Assert
	guildConfig, err = repo.GetGuildConfig(ctx, guildId)
	if len(guildConfig.ScrapeConfigs) != 0 {
		t.Fail()
	}
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
	if err != nil {
		t.Fail()
	}

	scrapeManager.Start(guildId, &scrapeConfig)

	// Sanity check
	if len(scrapeManager.ActiveScrapers) != 1 {
		t.Fail()
	}

	// Act
	err = RemoveScrapeConfig(ctx, repo, scrapeManager, guildId, configName)
	if err != nil {
		t.Fail()
	}

	// Assert
	if len(scrapeManager.ActiveScrapers) != 0 {
		t.Fail()
	}
}
