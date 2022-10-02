package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yukitsune/minialert/bot"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/grace"
	"github.com/yukitsune/minialert/scraper"
	"github.com/yukitsune/minialert/version"
	"log"
)

// Todo:
// 	1. Dockerize and mongo-ize(?)

var rootCmd = &cobra.Command{
	Use:   "minialert <command> [flags]",
	Short: "Minialert is a lightweight alert management Discord bot for prometheus",
	Long:  `a lightweight alert management Discord bot for prometheus built with love by YuKitsune in Go.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Version)
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the discord bot",
	RunE:  run,
}

var configFile string

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "use a specific config file, uses /etc/minialert or the current directory by default")
	rootCmd.PersistentFlags().Bool("debug", false, "use verbose logging")
	_ = viper.BindPFlag("log.debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		grace.FExitFromError(log.Writer(), err)
	}
}

func run(_ *cobra.Command, _ []string) error {

	ctx, cancel := context.WithCancel(context.Background())

	// Build the logger now, so we can log stuff
	// We'll configure it once we've loaded the config
	logger := logrus.New()

	cfg, _ := config.Setup(configFile, viper.GetViper(), logger)
	configureLogging(logger, cfg.Log())

	logger.Debugf("Config %s", cfg.Debug())

	repo := db.SetupInMemoryDatabase(logger)
	scrapeManager := scraper.NewScrapeManager(logger)

	b := bot.New(cfg.Bot(), repo, scrapeManager, logger)

	errorsChan := make(chan error)
	go func() {
		err := b.Start(ctx)

		if err != nil {
			errorsChan <- err
		}
	}()

	grace.WaitForShutdownSignalOrError(logger, errorsChan, func() error {
		cancel()
		return b.Close()
	})

	return nil
}

func configureLogging(logger *logrus.Logger, cfg config.Log) {

	lvl := cfg.Level()
	logger.SetLevel(lvl)
}
