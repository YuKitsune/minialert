package bot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/scraper"
	"strings"
)

type Bot struct {
	cfg                          config.Bot
	session                      *discordgo.Session
	repo                         db.Repo
	scrapeManager                *scraper.ScrapeManager
	doneChan                     chan bool
	commands                     []*discordgo.ApplicationCommand
	interactionHandlers          InteractionHandlers
	componentInteractionHandlers MessageInteractionHandlers
	logger                       logrus.FieldLogger
}

func New(cfg config.Bot, repo db.Repo, scrapeManager *scraper.ScrapeManager, logger logrus.FieldLogger) *Bot {
	commands := getCommands()
	interactionHandlers := getInteractionHandlers(repo, scrapeManager)
	componentInteractionHandlers := getMessageInteractionHandlers(repo)

	return &Bot{
		cfg:                          cfg,
		repo:                         repo,
		scrapeManager:                scrapeManager,
		commands:                     commands,
		interactionHandlers:          interactionHandlers,
		componentInteractionHandlers: componentInteractionHandlers,
		logger:                       logger,
		doneChan:                     make(chan bool, 1),
	}
}

func (b *Bot) Start(ctx context.Context) error {

	// Create a new Discord session using the provided bot token.
	b.logger.Infoln("📡 Starting session...")
	s, err := discordgo.New("Bot " + b.cfg.Token())
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %s", err.Error())
	}

	// Configure event handlers
	s.AddHandler(onReadyHandler(b.cfg, b.logger))
	s.AddHandler(onGuildCreated(b.commands, b.repo, b.logger))
	s.AddHandler(onInteractionCreateHandler(b.interactionHandlers, b.componentInteractionHandlers, b.logger))
	s.AddHandler(onGuildDeleted(b.repo, b.logger))

	// Open the session (connect)
	b.logger.Infoln("📡 Opening session...")
	err = s.Open()
	if err != nil {
		return fmt.Errorf("cannot open session: %s", err.Error())
	}

	// Start scraping for each config
	guildConfigs, err := b.repo.GetGuildConfigs(ctx)
	if err != nil {
		return err
	}

	for _, guildConfig := range guildConfigs {
		for _, scrapeConfig := range guildConfig.ScrapeConfigs {
			b.scrapeManager.Start(guildConfig.GuildId, &scrapeConfig)
		}
	}

	go watchAlerts(b.doneChan, s, b.repo, b.scrapeManager, b.logger)
	b.session = s

	return nil
}

func (b *Bot) Close() error {
	b.doneChan <- true
	b.logger.Infoln("👋 Closing session...")
	return b.session.Close()
}

func getInviteLink(cfg config.Bot) string {
	scopesStr := strings.Join(cfg.Scopes(), "%20")
	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%s&scope=%s", cfg.ClientId(), cfg.Permissions(), scopesStr)
	return link
}

func hasMatching[T any](ts []T, fn func(v T) bool) bool {
	for _, t := range ts {
		if fn(t) {
			return true
		}
	}

	return false
}

func removeMatching[T any](s []T, match func(t T) bool) []T {
	for i, t := range s {
		if match(t) {
			s[i] = s[len(s)-1]
			return s[:len(s)-1]
		}
	}

	return s
}

func findMatching[T any](s []T, match func(t T) bool) (*T, bool) {
	for _, t := range s {
		if match(t) {
			return &t, true
		}
	}

	return nil, false
}
