package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"strconv"
	"strings"
)

type InteractionName string

const (
	GetAlertsCommandName InteractionName = "get-alerts"

	ShowInhibitedAlertsCommandName InteractionName = "show-inhibited-alerts"
	InhibitAlertCommandName        InteractionName = "inhibit-alert"
	UninhibitAlertCommandName      InteractionName = "uninhibit-alert"

	SetAdminCommandName           InteractionName = "set-admin"
	CreateScrapeConfigCommandName InteractionName = "create-scrape-config"
	UpdateScrapeConfigCommandName InteractionName = "update-scrape-config"
	RemoveScrapeConfigCommandName InteractionName = "remove-scrape-config"
)

func (c InteractionName) String() string {
	return string(c)
}

type InteractionOption string

const (
	ChannelOption          InteractionOption = "channel"
	UserOption             InteractionOption = "user"
	AlertNameOption        InteractionOption = "alertname"
	ScrapeConfigNameOption InteractionOption = "scrape-config-name"
	EndpointOption         InteractionOption = "endpoint"
	UsernameOption         InteractionOption = "username"
	PasswordOption         InteractionOption = "password"
	IntervalOption         InteractionOption = "interval"
)

func (c InteractionOption) String() string {
	return string(c)
}

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

func getConfigCommand(create bool) *discordgo.ApplicationCommand {

	var name = UpdateScrapeConfigCommandName.String()
	if create {
		name = CreateScrapeConfigCommandName.String()
	}

	var description = "Updates an existing scrape config"
	if create {
		description = "Creates a new scrape config"
	}

	return &discordgo.ApplicationCommand{
		Name:        name,
		Description: description,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        ScrapeConfigNameOption.String(),
				Description: "The name of the scrape config",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        EndpointOption.String(),
				Description: "The endpoint to scrape",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    create,
			},
			{
				Name:        IntervalOption.String(),
				Description: "The interval (in minutes) at which to scrape the endpoint",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    create,
			},
			{
				Name:        ChannelOption.String(),
				Description: "The channel to send the alerts to",
				Type:        discordgo.ApplicationCommandOptionChannel,
				Required:    create,
			},
			{
				Name:        UsernameOption.String(),
				Description: "The username required to access the endpoint",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
			{
				Name:        PasswordOption.String(),
				Description: "The password required to access the endpoint",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
		},
	}
}

func getCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        GetAlertsCommandName.String(),
			Description: "List all currently firing alerts",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        ScrapeConfigNameOption.String(),
					Description: "The name of the scrape config",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        ShowInhibitedAlertsCommandName.String(),
			Description: "List all inhibited alerts",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        ScrapeConfigNameOption.String(),
					Description: "The name of the scrape config",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        InhibitAlertCommandName.String(),
			Description: "Inhibit an alert with the given name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        ScrapeConfigNameOption.String(),
					Description: "The name of the scrape config",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        AlertNameOption.String(),
					Description: "Alertname",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        UninhibitAlertCommandName.String(),
			Description: "Un-inhibit an alert with the given name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        ScrapeConfigNameOption.String(),
					Description: "The name of the scrape config",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        AlertNameOption.String(),
					Description: "Alertname",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        SetAdminCommandName.String(),
			Description: "Sets the administrator user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        UserOption.String(),
					Description: "User",
					Type:        discordgo.ApplicationCommandOptionUser,
					Required:    true,
				},
			},
		},
		getConfigCommand(false),
		getConfigCommand(true),
		{
			Name:        RemoveScrapeConfigCommandName.String(),
			Description: "Removes a scrape config",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        ScrapeConfigNameOption.String(),
					Description: "The name of the scrape config",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}
}

func getInteractionHandlers(repo db.Repo, scrapeManager *ScrapeManager) InteractionHandlers {
	return map[InteractionName]InteractionHandler{
		GetAlertsCommandName: getAlertsHandler(repo),

		ShowInhibitedAlertsCommandName: showInhibitedAlertsHandler(repo),
		InhibitAlertCommandName:        inhibitAlertHandler(repo),
		UninhibitAlertCommandName:      uninhibitAlertHandler(repo),

		SetAdminCommandName:           setAdminHandler(repo),
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

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})

	if err != nil {
		logger.Errorf("Failed to respond: %s", err.Error())
	}
}

func respondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger, message string) {
	respond(s, i, logger, fmt.Sprintf("✅ %s", message))
}

func respondWithWarning(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger, message string) {
	respond(s, i, logger, fmt.Sprintf("⚠️ %s", message))
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger, message string) {
	respond(s, i, logger, fmt.Sprintf("❌ %s", message))
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

		scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

		scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

		scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

		scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
			return cfg.Name == configName
		})
		if !ok {
			respondWithError(s, i, logger, fmt.Sprintf("Couldn't find scrape config with name \"%s\".", configName))
			return
		}

		scrapeConfig.Inhibitions = removeMatching(scrapeConfig.Inhibitions, func(inhib db.Inhibition) bool {
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

		scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
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

func setAdminHandler(repo db.Repo) InteractionHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		ctx := context.TODO()

		opts := getOptionMap(i.ApplicationCommandData().Options)

		adminOpt, ok := opts[UserOption]
		if !ok {
			respondWithWarning(s, i, logger, "User option is required.")
			return
		}

		user := adminOpt.UserValue(s)

		// Todo: Ensure user isn't a bot

		guildConfig, err := repo.GetGuildConfig(ctx, i.GuildID)
		if err != nil {
			logger.Errorf("Failed to get guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to set admin user.")
			return
		}

		guildConfig.AdminId = user.ID

		err = repo.SetGuildConfig(ctx, guildConfig)
		if err != nil {
			logger.Errorf("Failed to set guild config: %s", err.Error())
			respondWithError(s, i, logger, "Failed to set admin user.")
		}

		respondWithSuccess(s, i, logger, "Admin user set.")
	}
}

func createScrapeConfigCommandHandler(repo db.Repo, scrapeManager *ScrapeManager) InteractionHandler {
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

func updateScrapeConfigCommandHandler(repo db.Repo, scrapeManager *ScrapeManager) InteractionHandler {
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
		for _, cfg := range guildConfig.ScrapeConfigs {
			if cfg.Name == configName {
				scrapeConfig = &cfg
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
		scrapeManager.Restart(guildConfig.GuildId, scrapeConfig)

		respondWithSuccess(s, i, logger, "Start config updated.")
	}
}

func removeScrapeConfigCommandHandler(repo db.Repo, scrapeManager *ScrapeManager) InteractionHandler {
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

		scrapeManager.Stop(guildConfig.GuildId, configName)

		respondWithSuccess(s, i, logger, "Start config removed.")
	}
}

func onReadyHandler(cfg config.Bot, logger logrus.FieldLogger) func(s *discordgo.Session, r *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Infof("✅  Logged in as: %s#%s", s.State.User.Username, s.State.User.Discriminator)

		inviteLink := getInviteLink(cfg)
		logger.Infof("🔗 Invite link: %s", inviteLink)
	}
}

func onInteractionCreateHandler(interactionHandlers InteractionHandlers, messageInteractionHandlers MessageInteractionHandlers, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		entry := logger.WithField("guild_id", i.GuildID).
			WithField("interaction_id", i.Interaction.ID).
			WithField("interaction_type", i.Interaction.Type.String())

		if i.Type == discordgo.InteractionApplicationCommand {
			entry = entry.WithField("interaction_name", i.ApplicationCommandData().Name)
		} else if i.Type == discordgo.InteractionMessageComponent {
			entry = entry.WithField("interaction_custom_id", i.Interaction.MessageComponentData().CustomID)
		}

		entry.Debugln("interaction created")

		if i.Type == discordgo.InteractionApplicationCommand {
			if h, ok := interactionHandlers[InteractionName(i.ApplicationCommandData().Name)]; ok {
				h(s, i, entry)
			}
		} else if i.Type == discordgo.InteractionMessageComponent {
			customId := MessageInteractionId(i.Interaction.MessageComponentData().CustomID)
			commandName, ok := customId.Name()
			if !ok {
				entry.Errorf("unable to determine command name or value from custom_id: %s", customId)
			}

			if h, ok := messageInteractionHandlers[commandName]; ok {
				h(s, i, entry)
			}
		} else {
			entry.Warnf("unexpected interaction type: %s", i.Type.String())
		}
	}
}

func onGuildCreated(commands []*discordgo.ApplicationCommand, repo db.Repo, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.GuildCreate) {
	return func(s *discordgo.Session, i *discordgo.GuildCreate) {

		ctx := context.TODO()

		ctxLogger := logger.WithField("guild_id", i.Guild.ID)
		ctxLogger.Debugln("Guild created")

		ctxLogger.Debugln("Creating config...")

		cfg := db.NewGuildConfig(i.Guild.ID)
		err := repo.SetGuildConfig(ctx, cfg)
		if err != nil {
			ctxLogger.Errorf("Failed to set guild config: %v", err.Error())
		}

		ctxLogger.Debugln("Config created")

		ctxLogger.Debugln("Creating commands...")

		for _, v := range commands {
			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, i.Guild.ID, v)
			if err != nil {
				ctxLogger.Errorf("Cannot create '%v' command: %v", v.Name, err)
				continue
			}

			ctxLogger.Debugf("Created '%v' command", v.Name)
			err = repo.RegisterCommand(context.Background(), i.Guild.ID, cmd.ID, cmd.Name)
			if err != nil {
				ctxLogger.Errorf("Failed to register command: %s", err.Error())
			}
		}

		ctxLogger.Debugln("Commands created")
	}
}

func onGuildDeleted(repo db.Repo, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.GuildDelete) {
	return func(s *discordgo.Session, i *discordgo.GuildDelete) {

		ctxLogger := logger.WithField("guild_id", i.Guild.ID)
		ctxLogger.Debugln("Guild deleted")

		ctx := context.Background()

		commands, err := repo.GetRegisteredCommands(ctx, i.Guild.ID)
		if err != nil {
			ctxLogger.Errorf("Failed to get commands: %s", err.Error())
			return
		}

		// Delete commands from guild
		for _, v := range commands {
			err = s.ApplicationCommandDelete(s.State.User.ID, i.Guild.ID, v.CommandId)
			if err != nil {
				ctxLogger.Errorf("Cannot delete '%v' command: %v", v.CommandName, err.Error())
			}
		}

		err = repo.ClearGuildInfo(ctx, i.Guild.ID)
		if err != nil {
			ctxLogger.Errorf("Failed to clear guild data: %s", err.Error())
			return
		}
	}
}

func getInviteLink(cfg config.Bot) string {
	scopesStr := strings.Join(cfg.Scopes(), "%20")
	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%s&scope=%s", cfg.ClientId(), cfg.Permissions(), scopesStr)
	return link
}

func RunBot(ctx context.Context, cfg config.Bot, logger logrus.FieldLogger, repo db.Repo) error {

	scrapeResultsChan := make(chan ScrapeResult)
	scrapeManager := NewScrapeManager(scrapeResultsChan, logger)

	commands := getCommands()
	interactionHandlers := getInteractionHandlers(repo, scrapeManager)
	componentInteractionHandlers := getMessageInteractionHandlers(repo)

	// Create a new Discord session using the provided bot token.
	logger.Infoln("📡 Creating session...")
	s, err := discordgo.New("Bot " + cfg.Token())
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %s", err.Error())
	}

	// Configure event handlers
	s.AddHandler(onReadyHandler(cfg, logger))
	s.AddHandler(onGuildCreated(commands, repo, logger))
	s.AddHandler(onInteractionCreateHandler(interactionHandlers, componentInteractionHandlers, logger))
	s.AddHandler(onGuildDeleted(repo, logger))

	logger.Infoln("📡 Opening session...")
	err = s.Open()
	if err != nil {
		return fmt.Errorf("cannot open session: %s", err.Error())
	}

	defer s.Close()

	guildConfigs, err := repo.GetGuildConfigs(ctx)
	if err != nil {
		return err
	}

	for _, guildConfig := range guildConfigs {
		for _, scrapeConfig := range guildConfig.ScrapeConfigs {
			scrapeManager.Start(guildConfig.GuildId, &scrapeConfig)
		}
	}

	go watchAlerts(ctx, s, repo, logger, scrapeResultsChan)

	<-ctx.Done()

	return nil
}

func getColorFromSeverity(severity string) (int64, error) {
	switch severity {
	case "warning":
		return strconv.ParseInt("ffaa00", 16, 64)
	case "critical":
		return strconv.ParseInt("ff0000", 16, 64)
	default:
		return strconv.ParseInt("ffffff", 16, 64)
	}
}

func getFieldsFromLabels(alert prometheus.Alert) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField
	for k, v := range alert.Labels {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   k,
			Value:  v,
			Inline: false,
		})
	}

	return fields
}

func watchAlerts(ctx context.Context, s *discordgo.Session, repo db.Repo, logger logrus.FieldLogger, resultsChan chan ScrapeResult) {
	for {
		select {
		case results := <-resultsChan:

			guildConfig, err := repo.GetGuildConfig(ctx, results.GuildId)
			if err != nil {
				logger.Errorf("Failed to get guild config: %s", err.Error())
				return
			}

			scrapeConfig, ok := findMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
				return cfg.Name == results.ScrapeConfigName
			})

			if !ok {
				logger.Warn("Guild config for %s doesn't contain a scrape config with the name %s", results.GuildId, results.ScrapeConfigName)
				return
			}

			filteredAlerts, err := filterAlerts(results.Alerts, scrapeConfig.Inhibitions)
			if err != nil {
				logger.Errorf("Failed to filter alerts: %s", err.Error())
				continue
			}

			sendAlertsToChannel(s, scrapeConfig, filteredAlerts, logger)

		case <-ctx.Done():
			break
		}
	}
}

func filterAlerts(alerts prometheus.Alerts, inhibitions []db.Inhibition) (prometheus.Alerts, error) {

	var newAlerts prometheus.Alerts
	for _, alert := range alerts {
		if !hasMatching(inhibitions, func(inhibition db.Inhibition) bool {
			alertName := alert.Labels["alertname"]
			return inhibition.AlertName == alertName
		}) {
			newAlerts = append(newAlerts, alert)
		}
	}

	return newAlerts, nil
}

func hasMatching[T any](ts []T, fn func(v T) bool) bool {
	for _, t := range ts {
		if fn(t) {
			return true
		}
	}

	return false
}

func sendAlertsToChannel(s *discordgo.Session, scrapeConfig *db.ScrapeConfig, alerts prometheus.Alerts, logger logrus.FieldLogger) {
	for _, alert := range alerts {

		alertName := alert.Labels["alertname"]

		title := alertName
		url := alert.Annotations["runbook_url"]
		description := alert.Annotations["description"]
		color, err := getColorFromSeverity(alert.Labels["severity"])
		if err != nil {
			logger.Errorf("Failed to generate color for alert: %s", err.Error())
		}

		fields := getFieldsFromLabels(alert)

		embed := &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeRich,
			Title:       title,
			URL:         url,
			Description: description,
			Timestamp:   alert.ActiveAt.Format("2006-01-02T15:04:05-0700"),
			Color:       int(color),
			Fields:      fields,
		}

		inhibitButtonComponent := discordgo.Button{
			Label:    "Inhibit",
			Style:    discordgo.DangerButton,
			CustomID: NewMessageInteractionId(InhibitAlertCommandName, scrapeConfig.Name, alertName).String(),
		}

		message := &discordgo.MessageSend{
			Embed: embed,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						inhibitButtonComponent,
					},
				},
			},
		}

		_, err = s.ChannelMessageSendComplex(scrapeConfig.AlertChannelId, message)
		if err != nil {
			logger.Errorf("Failed to send message to channel %s: %s\n", scrapeConfig.AlertChannelId, err.Error())
		}
	}
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
