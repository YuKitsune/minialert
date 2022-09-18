package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Todo:
// 	1. The goroutines and channels are kinda wacky... Let's fix them...
// 	2. Implement cobra
// 	3. Dockerize and mongo-ize(?)

func main() {

	shutdownChan := getShutdownSignalChan()

	// Build the logger now so we can log stuff
	// We'll configure it once we've loaded the config
	logger := logrus.New()

	cfg, _ := config.Setup(logger)
	configureLogging(logger, cfg.Log())

	cfg.Debug(logger.WriterLevel(logrus.DebugLevel))

	client := http.Client{
		Timeout: cfg.Prometheus().Timeout(),
	}

	var creds *prometheus.BasicAuthDetails
	if hasCreds, username, password := cfg.Prometheus().BasicAuth(); hasCreds {
		creds = &prometheus.BasicAuthDetails{
			Username: username,
			Password: password,
		}
	}

	promClient := prometheus.NewClientWithBasicAuth(client, cfg.Prometheus().Endpoint(), creds)

	alertsChan := make(chan prometheus.Alerts)
	errorsChan := make(chan error)

	// Todo: Only start scraping when a bot joins a guild
	// Todo: Start one goroutine per guild
	// Todo: Stop the goroutine when the bot leaves a guild, or the application exits
	go scrape(cfg.ScrapeInterval(), promClient, alertsChan, errorsChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbFunc := db.SetupInMemoryDatabase()

	go func() {
		err := RunBot(ctx, cfg.Bot(), logger, dbFunc, promClient, alertsChan)

		if err != nil {
			errorsChan <- err
		}
	}()

	waitForSignal(logger, shutdownChan, errorsChan)
}

func scrape(period time.Duration, prometheusClient *prometheus.Client, alertsChan chan prometheus.Alerts, errChan chan error) {
	for range time.Tick(period) {

		alerts, err := prometheusClient.GetAlerts()
		if err != nil {
			errChan <- err
			continue
		}

		alertsChan <- alerts
	}
}

func getShutdownSignalChan() chan os.Signal {
	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGKILL,
		os.Kill,
	)

	return shutdownSignalChan
}

func waitForSignal(logger *logrus.Logger, shutdownSignalChan chan os.Signal, errorChan chan error) {
	select {
	case sig := <-shutdownSignalChan:
		logger.Infof("Signal caught: %s\n", sig.String())
		break
	case err := <-errorChan:
		logger.Errorf("Error: %s\n", err.Error())
		break
	}
}

func configureLogging(logger *logrus.Logger, cfg config.Log) {

	lvl := cfg.Level()
	logger.SetLevel(lvl)
}
