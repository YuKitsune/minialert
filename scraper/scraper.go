package scraper

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"time"
)

type ScrapeManager interface {
	Start(guildId string, config *db.ScrapeConfig)
	Chan() chan ScrapeResult
	Restart(guildId string, config *db.ScrapeConfig) error
	Stop(guildId string, configName string) error
}

type key string

func newQuitterKey(guildId string, configName string) key {
	return key(fmt.Sprintf("%s:%s", guildId, configName))
}

type ScrapeResult struct {
	GuildId          string
	ScrapeConfigName string
	Alerts           prometheus.Alerts
}

type scrapeManager struct {
	clientFactory prometheus.ClientFactory
	logger        logrus.FieldLogger
	resultsChan   chan ScrapeResult
	quitters      map[key]func()
}

func NewScrapeManager(clientFactory prometheus.ClientFactory, logger logrus.FieldLogger) ScrapeManager {
	return &scrapeManager{
		clientFactory: clientFactory,
		logger:        logger,
		resultsChan:   make(chan ScrapeResult),
		quitters:      make(map[key]func()),
	}
}

func (m *scrapeManager) Start(guildId string, config *db.ScrapeConfig) {
	quit := make(chan bool)
	key := newQuitterKey(guildId, config.Name)
	m.quitters[key] = func() {
		quit <- true
	}

	scrapeLogger := m.logger.WithField("scrape_config_name", config.Name)

	go m.scrape(guildId, config, scrapeLogger, quit)
}

func (m *scrapeManager) Chan() chan ScrapeResult {
	return m.resultsChan
}

func (m *scrapeManager) Restart(guildId string, config *db.ScrapeConfig) error {
	if err := m.Stop(guildId, config.Name); err != nil {
		return err
	}

	m.Start(guildId, config)
	return nil
}

func (m *scrapeManager) Stop(guildId string, name string) error {
	key := newQuitterKey(guildId, name)
	quit, ok := m.quitters[key]
	if !ok {
		return fmt.Errorf("no scrapers running for %s in guild %s", name, guildId)
	}

	quit()
	delete(m.quitters, key)
	return nil
}

func (m *scrapeManager) scrape(guildId string, config *db.ScrapeConfig, logger logrus.FieldLogger, quitChan chan bool) {
	dur := time.Duration(config.ScrapeIntervalMinutes) * time.Minute
	client := m.clientFactory(config)

	ctxLogger := logger.
		WithField("guild_id", guildId).
		WithField("scrape_config_name", config.Name)

	ctxLogger.Debug("Scraper started")

	for {
		select {
		case <-time.Tick(dur):

			ctxLogger.Debug("Beginning scrape")
			alerts, err := client.GetAlerts()
			if err != nil {
				ctxLogger.Errorf("Error occurred while scraping: %s", err.Error())
				continue
			}

			res := ScrapeResult{
				GuildId:          guildId,
				ScrapeConfigName: config.Name,
				Alerts:           alerts,
			}

			m.Chan() <- res

		case <-quitChan:
			ctxLogger.Debug("Scraper stopped")
			return
		}
	}
}
