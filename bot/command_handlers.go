package bot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/scraper"
	"github.com/yukitsune/minialert/util"
	"strings"
)

type InteractionHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger)
type InteractionHandlers map[InteractionName]InteractionHandler

const MessageInteractionIdSeparator = ":"

type MessageInteractionId string

func (id MessageInteractionId) Name() (InteractionName, bool) {
	parts := strings.Split(id.String(), MessageInteractionIdSeparator)
	if len(parts) != 2 {
		return "", false
	}

	return InteractionName(parts[0]), true
}

func (id MessageInteractionId) Values() ([]string, bool) {
	parts := strings.Split(id.String(), MessageInteractionIdSeparator)
	if len(parts) < 2 {
		return nil, false
	}

	return parts[1:], true
}

func (id MessageInteractionId) String() string {
	return string(id)
}

func NewMessageInteractionId(name InteractionName, values ...string) MessageInteractionId {
	var str strings.Builder
	str.WriteString(name.String())

	for _, v := range values {
		str.WriteString(MessageInteractionIdSeparator)
		str.WriteString(v)
	}

	return MessageInteractionId(str.String())
}

type MessageInteractionHandlers map[InteractionName]InteractionHandler

func getInteractionHandlers(repo db.Repo, scrapeManager *scraper.ScrapeManager) InteractionHandlers {
	return map[InteractionName]InteractionHandler{
		GetAlertsCommandName: getAlertsHandler(repo),

		ShowInhibitedAlertsCommandName: showInhibitedAlertsHandler(repo),
		InhibitAlertCommandName:        inhibitAlertHandler(repo),
		UninhibitAlertCommandName:      uninhibitAlertHandler(repo),

		CreateScrapeConfigCommandName: createScrapeConfigCommandHandler(repo, scrapeManager),
		UpdateScrapeConfigCommandName: updateScrapeConfigCommandHandler(repo, scrapeManager),
		RemoveScrapeConfigCommandName: removeScrapeConfigCommandHandler(repo, scrapeManager),
	}
}

func getMessageInteractionHandlers(repo db.Repo) MessageInteractionHandlers {
	return map[InteractionName]InteractionHandler{
		InhibitAlertCommandName: inhibitAlertFromMessageHandler(repo),
	}
}

func getOptionMap(options []*discordgo.ApplicationCommandInteractionDataOption) map[InteractionOption]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[InteractionOption]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[InteractionOption(opt.Name)] = opt
	}

	return optionMap
}

func getAlertsHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get inhibited alerts.")
			return
		}

		scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		client := prometheus.NewPrometheusClientFromScrapeConfig(scrapeConfig)
		alerts, err := client.GetAlerts()

		filteredAlerts, err := filterAlerts(alerts, scrapeConfig.Inhibitions)
		if err != nil {
			logger.Errorf("Failed to filter alerts: %s", err.Error())
			respondWithError(s, i, logger, "Failed to filter alerts.")
			return
		}

		sendAlertsToChannel(s, scrapeConfig, filteredAlerts, logger)
	}
}

func showInhibitedAlertsHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get inhibited alerts.")
			return
		}

		scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		if len(scrapeConfig.Inhibitions) == 0 {
			respond(s, i, logger, fmt.Sprintf("No inhibitions set for %s.", configName))
			return
		}

		var content string
		for i2, inhibition := range scrapeConfig.Inhibitions {
			content += inhibition.AlertName
			if i2 != len(scrapeConfig.Inhibitions)-1 {
				content += ", "
			}
		}

		respond(s, i, logger, content)
	}
}

func inhibitAlertHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		alertNameOpt, ok := opts[AlertNameOption]
		if !ok {
			respondWithError(s, i, logger, "Could not find alert name option.")
			return
		}

		alertName := alertNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get inhibited alerts.")
			return
		}

		scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		scrapeConfig.Inhibitions = append(scrapeConfig.Inhibitions, db.Inhibition{AlertName: alertName})

		err = repo.SetGuildConfig(ctx, guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to add inhibition.")
			return
		}

		respondWithSuccess(s, i, logger, "Inhibition added.")
	}
}

func uninhibitAlertHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		alertNameOpt, ok := opts[AlertNameOption]
		if !ok {
			respondWithError(s, i, logger, "Could not find alert name option.")
			return
		}

		alertName := alertNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get inhibited alerts.")
			return
		}

		scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		scrapeConfig.Inhibitions = util.RemoveMatching(scrapeConfig.Inhibitions, func(inhib db.Inhibition) bool {
			return inhib.AlertName == alertName
		})

		err = repo.SetGuildConfig(ctx, guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to remove inhibition.")
			return
		}

		respondWithSuccess(s, i, logger, "Inhibition removed.")
	}
}

func inhibitAlertFromMessageHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		customId := MessageInteractionId(i.Interaction.MessageComponentData().CustomID)
		values, ok := customId.Values()
		if !ok || len(values) != 2 {
			respondWithWarning(s, i, logger, fmt.Sprintf("Received unknown custom_id: %s", customId))
			return
		}

		configName := values[0]
		alertName := values[1]

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get inhibited alerts.")
			return
		}

		scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		scrapeConfig.Inhibitions = append(scrapeConfig.Inhibitions, db.Inhibition{AlertName: alertName})

		err = repo.SetGuildConfig(ctx, guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to add inhibition.")
			return
		}

		respondWithSuccess(s, i, logger, "Inhibition added.")
	}
}

func createScrapeConfigCommandHandler(repo db.Repo, scrapeManager *scraper.ScrapeManager) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		endpointOpt, ok := opts[EndpointOption]
		if !ok {
			respondWithError(s, i, logger, "Endpoint is required.")
			return
		}

		intervalMinsOpt, ok := opts[IntervalOption]
		if !ok {
			respondWithError(s, i, logger, "Start interval is required.")
			return
		}

		channelOpt, ok := opts[ChannelOption]
		if !ok {
			respondWithError(s, i, logger, "Alerts channel is required.")
			return
		}

		channel := channelOpt.ChannelValue(s)
		if channel.Type != discordgo.ChannelTypeGuildText {
			respondWithError(s, i, logger, "Alerts channel must be a text channel.")
			return
		}

		scrapeConfig := &db.ScrapeConfig{
			Name:                  configNameOpt.StringValue(),
			Endpoint:              endpointOpt.StringValue(),
			ScrapeIntervalMinutes: intervalMinsOpt.IntValue(),
			AlertChannelId:        channel.ID,
			Inhibitions:           make([]db.Inhibition, 0),
		}

		usernameOpt, _ := opts[UsernameOption]
		passwordOpt, _ := opts[PasswordOption]
		if usernameOpt != nil && passwordOpt != nil {
			scrapeConfig.Username = usernameOpt.StringValue()
			scrapeConfig.Password = passwordOpt.StringValue()
		}

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to create scrape config.")
			return
		}

		for _, cfg := range guildConfig.ScrapeConfigs {
			if cfg.Name == scrapeConfig.Name {
				respondWithError(s, i, logger, fmt.Sprintf("There is already a scrape config with the name \"%s\".", scrapeConfig.Name))
				return
			}
		}

		guildConfig.ScrapeConfigs = append(guildConfig.ScrapeConfigs, *scrapeConfig)

		err = repo.SetGuildConfig(context.Background(), guildConfig)
		if err != nil {
			logger.Errorf("Failed to update guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to create scrape config.")
			return
		}

		scrapeManager.Start(guildConfig.GuildId, scrapeConfig)

		respondWithSuccess(s, i, logger, "Start config created.")
	}
}

func updateScrapeConfigCommandHandler(repo db.Repo, scrapeManager *scraper.ScrapeManager) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to guild scrape config.")
		}

		var scrapeConfig *db.ScrapeConfig
		for i, cfg := range guildConfig.ScrapeConfigs {
			if cfg.Name == configName {
				scrapeConfig = &guildConfig.ScrapeConfigs[i]
				break
			}
		}

		if scrapeConfig == nil {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		endpointOpt, ok := opts[EndpointOption]
		if ok {
			scrapeConfig.Endpoint = endpointOpt.StringValue()
		}

		usernameOpt, ok := opts[UsernameOption]
		if ok {
			scrapeConfig.Username = usernameOpt.StringValue()
		}

		passwordOpt, ok := opts[PasswordOption]
		if ok {
			scrapeConfig.Password = passwordOpt.StringValue()
		}

		intervalMinsOpt, ok := opts[IntervalOption]
		if ok {
			scrapeConfig.ScrapeIntervalMinutes = intervalMinsOpt.IntValue()
		}

		channelOpt, ok := opts[ChannelOption]
		if ok {
			channel := channelOpt.ChannelValue(s)
			if channel.Type != discordgo.ChannelTypeGuildText {
				respondWithError(s, i, logger, "Alerts channel must be a text channel.")
				return
			}

			scrapeConfig.AlertChannelId = channel.ID
		}

		err = repo.SetGuildConfig(context.Background(), guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to update scrape config.")
			return
		}

		// Restart the scrape after updating the config
		err = scrapeManager.Restart(guildConfig.GuildId, scrapeConfig)
		if err != nil {
			respondWithError(s, i, logger, "Failed to restart scraper.")
			logger.Fatalf("Failed to restart scraper: %s", err.Error())
		}

		respondWithSuccess(s, i, logger, "Start config updated.")
	}
}

func removeScrapeConfigCommandHandler(repo db.Repo, scrapeManager *scraper.ScrapeManager) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		configNameOpt, ok := opts[ScrapeConfigNameOption]
		if !ok {
			respondWithError(s, i, logger, "Name is required.")
			return
		}

		configName := configNameOpt.StringValue()

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to guild scrape config.")
		}

		removed := false
		for i, cfg := range guildConfig.ScrapeConfigs {
			if cfg.Name == configName {
				guildConfig.ScrapeConfigs = append(guildConfig.ScrapeConfigs[:i], guildConfig.ScrapeConfigs[i+1:]...)
				removed = true
				break
			}
		}

		if !removed {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		err = repo.SetGuildConfig(context.Background(), guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to remove scrape config.")
			return
		}

		err = scrapeManager.Stop(guildConfig.GuildId, configName)
		if err != nil {
			respondWithError(s, i, logger, "Failed to stop scraper.")
			logger.Fatalf("Failed to stop scraper: %s", err.Error())
		}

		respondWithSuccess(s, i, logger, "Start config removed.")
	}
}
