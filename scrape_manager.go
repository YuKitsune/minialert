package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"time"
)

// Todo: Composite scrape ID made up of guild ID and scrape ID

type ScrapeResult struct {
	GuildId          string
	ScrapeConfigName string
	Alerts           prometheus.Alerts
}

type ScrapeManager struct {
	logger      logrus.FieldLogger
	resultsChan chan ScrapeResult
	quitters    map[string]func()
}

func NewScrapeManager(resultsChan chan ScrapeResult, logger logrus.FieldLogger) *ScrapeManager {
	return &ScrapeManager{
		logger:      logger,
		resultsChan: resultsChan,
		quitters:    make(map[string]func()),
	}
}

func (m *ScrapeManager) Start(guildId string, config *db.ScrapeConfig) {
	quit := make(chan bool)
	m.quitters[config.Name] = func() {
		quit <- true
	}

	scrapeLogger := m.logger.WithField("scrape_config_name", config.Name)

	go m.scrape(guildId, config, scrapeLogger, quit, m.resultsChan)
}

func (m *ScrapeManager) Restart(guildId string, config *db.ScrapeConfig) {
	m.Stop(guildId, config.Name)
	m.Start(guildId, config)
}

func (m *ScrapeManager) Stop(guildId string, name string) error {
	quit, ok := m.quitters[name]
	if !ok {
		return fmt.Errorf("no scrapers running for %s", name)
	}

	quit()
	delete(m.quitters, name)
	return nil
}

func (m *ScrapeManager) scrape(guildId string, config *db.ScrapeConfig, logger logrus.FieldLogger, quitChan chan bool, resultsChan chan ScrapeResult) {
	dur := time.Duration(config.ScrapeIntervalMinutes) * time.Minute
	client := prometheus.NewPrometheusClientFromScrapeConfig(config)

	logger.Info("Scraper started")

	for {
		select {
		case <-time.Tick(dur):

			logger.Debug("Beginning scrape")
			alerts, err := client.GetAlerts()
			if err != nil {
				logger.WithField("scrape_config_name", config.Name).Errorf("Error occurred while scraping: %s", err.Error())
				continue
			}

			res := ScrapeResult{
				GuildId:          guildId,
				ScrapeConfigName: config.Name,
				Alerts:           alerts,
			}

			resultsChan <- res

		case <-quitChan:
			logger.Infof("Scraper stopped")
			break
		}
	}
}
