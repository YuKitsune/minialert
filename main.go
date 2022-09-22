package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"os"
	"os/signal"
	"syscall"
)

// Todo:
//  1. Config not updating
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := db.SetupInMemoryDatabase()

	errorsChan := make(chan error)

	go func() {
		err := RunBot(ctx, cfg.Bot(), logger, repo)

		if err != nil {
			errorsChan <- err
		}
	}()

	waitForSignal(logger, shutdownChan, errorsChan)
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
