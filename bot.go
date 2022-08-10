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

type CommandName string

const (
	GetAlertsCommandName           CommandName = "get-alerts"
	SetAlertsChannelCommandName    CommandName = "set-alerts-channel"
	SetAdminCommandName            CommandName = "set-admin"
	ShowInhibitedAlertsCommandName CommandName = "show-inhibited-alerts"
	InhibitAlertCommandName        CommandName = "inhibit-alert"
	UninhibitAlertCommandName      CommandName = "uninhibit-alert"
)

func (c CommandName) String() string {
	return string(c)
}

type CommandOption string

const (
	ChannelOption   CommandOption = "channel"
	UserOption      CommandOption = "user"
	AlertNameOption CommandOption = "alertname"
)

func (c CommandOption) String() string {
	return string(c)
}

type CommandHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger)
type CommandHandlers map[CommandName]CommandHandler

func getCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        GetAlertsCommandName.String(),
			Description: "List all currently firing alerts",
		},
		{
			Name:        SetAlertsChannelCommandName.String(),
			Description: "Sets the channel where alerts will be sent periodically",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        ChannelOption.String(),
					Description: "Channel",
					Required:    true,
				},
			},
		},
		{
			Name:        SetAdminCommandName.String(),
			Description: "Sets the administrator user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        UserOption.String(),
					Description: "User",
					Required:    true,
				},
			},
		},
		{
			Name:        ShowInhibitedAlertsCommandName.String(),
			Description: "List all inhibited alerts",
		},
		{
			Name:        InhibitAlertCommandName.String(),
			Description: "Inhibit an alert with the given name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        AlertNameOption.String(),
					Description: "Alertname",
					Required:    true,
				},
			},
		},
		{
			Name:        UninhibitAlertCommandName.String(),
			Description: "Un-inhibit an alert with the given name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        AlertNameOption.String(),
					Description: "Alertname",
					Required:    true,
				},
			},
		},
	}
}

func getCommandHandlers(prometheusClient *prometheus.Client, repo db.Repo) CommandHandlers {
	return map[CommandName]CommandHandler{
		GetAlertsCommandName:           getAlertsHandler(prometheusClient),
		SetAlertsChannelCommandName:    setAlertsChannelHandler(repo),
		SetAdminCommandName:            setAdminHandler(repo),
		ShowInhibitedAlertsCommandName: showInhibitedAlertsHandler(),
		InhibitAlertCommandName:        inhibitAlertHandler(repo),
		UninhibitAlertCommandName:      uninhibitAlertHandler(repo),
	}
}

func getOptionMap(options []*discordgo.ApplicationCommandInteractionDataOption) map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[CommandOption(opt.Name)] = opt
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

func getAlertsHandler(prometheusClient *prometheus.Client) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {
		alerts, err := prometheusClient.GetAlerts()
		if err != nil {
			logger.Errorf("Failed to get alerts: %s", err.Error())
			respondWithError(s, i, logger, "Failed to get alerts.")
			return
		}

		sendAlertsToChannel(s, i.ChannelID, alerts, logger)
	}
}

func setAlertsChannelHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if channelOption, ok := opts[ChannelOption]; ok {

			channel := channelOption.ChannelValue(s)

			if channel.Type != discordgo.ChannelTypeGuildText {
				respondWithWarning(s, i, logger, "Alerts channel must be a text channel")
				return
			}

			err := repo.SetAlertsChannel(context.Background(), i.GuildID, channel.ID)
			if err != nil {
				logger.Errorf("Failed to set alerts channel: %s", err.Error())
				respondWithError(s, i, logger, "Failed to set alerts channel.")
			}

			respondWithSuccess(s, i, logger, "Alerts channel set.")
		} else {
			respondWithWarning(s, i, logger, "Channel option is required.")
		}
	}
}

func setAdminHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if userOption, ok := opts[UserOption]; ok {

			user := userOption.UserValue(s)

			// Todo: Ensure user isn't a bot

			err := repo.SetAdminUser(context.Background(), i.GuildID, user.ID)

			if err != nil {
				logger.Errorf("Failed to set admin user: %s", err.Error())
				respondWithError(s, i, logger, "Failed to set admin user.")
			}

			respondWithSuccess(s, i, logger, "Admin user set.")
		} else {
			respondWithWarning(s, i, logger, "User option is required.")
		}
	}
}

func showInhibitedAlertsHandler() CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {
		logger.Errorln("Not implemented")
		respondWithError(s, i, logger, "Not implemented")
	}
}

func inhibitAlertHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if alertNameOption, ok := opts[AlertNameOption]; ok {

			alertName := alertNameOption.StringValue()

			err := repo.CreateInhibition(context.Background(), i.GuildID, alertName)
			if err != nil {
				logger.Errorf("Failed to add inhibition: %s", err.Error())
				respondWithError(s, i, logger, "Failed to add inhibition.")
			}

			respondWithSuccess(s, i, logger, "Inhibition added.")

		} else {
			respondWithWarning(s, i, logger, "Could not find alert name option.")
		}
	}
}

func uninhibitAlertHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, logger logrus.FieldLogger) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if alertNameOption, ok := opts[AlertNameOption]; ok {

			alertName := alertNameOption.StringValue()

			err := repo.DeleteInhibition(context.Background(), i.GuildID, alertName)
			if err != nil {
				logger.Errorf("Failed to remove inhibition: %s", err.Error())
				respondWithError(s, i, logger, "Failed to remove inhibition.")
			}

			respondWithSuccess(s, i, logger, "Inhibition removed.")

		} else {
			respondWithWarning(s, i, logger, "Could not find alert name option.")
		}
	}
}

func onReadyHandler(cfg config.Bot, logger logrus.FieldLogger) func(s *discordgo.Session, r *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Infof("✅  Logged in as: %s#%s", s.State.User.Username, s.State.User.Discriminator)

		inviteLink := getInviteLink(cfg)
		logger.Infof("🔗 Invite link: %s", inviteLink)
	}
}

func onInteractionCreateHandler(handlers CommandHandlers, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		entry := logger.WithField("guild_id", i.GuildID).
			WithField("interaction_id", i.Interaction.ID).
			WithField("interaction_type", i.Interaction.Type.String()).
			WithField("interaction_name", i.ApplicationCommandData().Name)

		entry.Debugln("interaction created")

		if h, ok := handlers[CommandName(i.ApplicationCommandData().Name)]; ok {
			h(s, i, entry)
		}
	}
}

func onGuildCreated(commands []*discordgo.ApplicationCommand, repo db.Repo, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.GuildCreate) {
	return func(s *discordgo.Session, i *discordgo.GuildCreate) {

		ctxLogger := logger.WithField("guild_id", i.Guild.ID)
		ctxLogger.Debugln("Guild created")

		for _, v := range commands {
			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, i.Guild.ID, v)
			if err != nil {
				ctxLogger.Errorf("Cannot create '%v' command: %v", v.Name, err)
			}

			ctxLogger.Debugf("Created '%v' command", v.Name)
			err = repo.RegisterCommand(context.Background(), i.Guild.ID, cmd.ID, cmd.Name)
			if err != nil {
				ctxLogger.Errorf("Failed to register command: %s", err.Error())
			}
		}
	}
}

func onGuildDeleted(repo db.Repo, logger logrus.FieldLogger) func(s *discordgo.Session, i *discordgo.GuildDelete) {
	return func(s *discordgo.Session, i *discordgo.GuildDelete) {

		ctxLogger := logger.WithField("guild_id", i.Guild.ID)
		ctxLogger.Debugln("Guild deleted")

		ctx := context.Background()

		commands, err := repo.GetRegisteredCommand(ctx, i.Guild.ID)
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

func RunBot(ctx context.Context, cfg config.Bot, logger logrus.FieldLogger, repo db.Repo, prometheusClient *prometheus.Client, alertsChan chan prometheus.Alerts) error {

	commands := getCommands()
	commandHandlers := getCommandHandlers(prometheusClient, repo)

	// Create a new Discord session using the provided bot token.
	logger.Infoln("📡 Creating session...")
	s, err := discordgo.New("Bot " + cfg.Token())
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %s", err.Error())
	}

	// Configure event handlers
	s.AddHandler(onReadyHandler(cfg, logger))
	s.AddHandler(onGuildCreated(commands, repo, logger))
	s.AddHandler(onInteractionCreateHandler(commandHandlers, logger))
	s.AddHandler(onGuildDeleted(repo, logger))

	logger.Infoln("📡 Opening session...")
	err = s.Open()
	if err != nil {
		return fmt.Errorf("cannot open session: %s", err.Error())
	}

	defer s.Close()

	go sendAlerts(ctx, s, cfg.GuildId(), repo, logger, alertsChan)

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

func sendAlerts(ctx context.Context, s *discordgo.Session, guildId string, repo db.Repo, logger logrus.FieldLogger, alertsChan chan prometheus.Alerts) {
	for {
		select {
		case alerts := <-alertsChan:
			alertsChannel, err := repo.GetAlertsChannel(ctx, guildId)
			if err != nil {
				logger.Debugf("Failed to get alerts channel: %s", err.Error())
				continue
			}

			sendAlertsToChannel(s, alertsChannel.ChannelId, alerts, logger)

		case <-ctx.Done():
			break
		}
	}
}

func sendAlertsToChannel(s *discordgo.Session, channelId string, alerts prometheus.Alerts, logger logrus.FieldLogger) {
	for _, alert := range alerts {

		title := alert.Labels["alertname"]
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
			CustomID: "foobar",
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

		_, err = s.ChannelMessageSendComplex(channelId, message)
		if err != nil {
			logger.Errorf("Failed to send message to channel %s: %s\n", channelId, err.Error())
		}
	}
}
