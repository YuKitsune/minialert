package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"log"
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

type CommandHandler func(s *discordgo.Session, i *discordgo.InteractionCreate)
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

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})

	if err != nil {
		log.Printf("Failed to respond to interaction %s: %s", i.ID, err.Error())
	}
}

func respondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	respond(s, i, fmt.Sprintf("✅ %s", message))
}

func respondWithWarning(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	respond(s, i, fmt.Sprintf("⚠️ %s", message))
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	respond(s, i, fmt.Sprintf("❌ %s", message))
}

func getAlertsHandler(prometheusClient *prometheus.Client) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		alerts, err := prometheusClient.GetAlerts()
		if err != nil {
			log.Printf("Failed to get alerts: %s", err.Error())
			respondWithError(s, i, "Failed to get alerts.")
			return
		}

		sendAlertsToChannel(s, i.ChannelID, alerts)
	}
}

func setAlertsChannelHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if channelOption, ok := opts[ChannelOption]; ok {

			channel := channelOption.ChannelValue(s)

			if channel.Type != discordgo.ChannelTypeGuildText {
				respondWithWarning(s, i, "Alerts channel must be a text channel")
				return
			}

			err := repo.SetAlertsChannel(context.Background(), i.GuildID, channel.ID)
			if err != nil {
				log.Printf("Failed to set alerts channel for guild %s: %s", i.GuildID, err.Error())
				respondWithError(s, i, "Failed to set alerts channel.")
			}

			respondWithSuccess(s, i, "Alerts channel set.")
		} else {
			respondWithWarning(s, i, "Channel option is required.")
		}
	}
}

func setAdminHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if userOption, ok := opts[UserOption]; ok {

			user := userOption.UserValue(s)

			// Todo: Ensure user isn't a bot

			err := repo.SetAdminUser(context.Background(), i.GuildID, user.ID)

			if err != nil {
				log.Printf("Failed to set admin user for guild %s: %s", i.GuildID, err.Error())
				respondWithError(s, i, "Failed to set admin user.")
			}

			respondWithSuccess(s, i, "Admin user set.")
		} else {
			respondWithWarning(s, i, "User option is required.")
		}
	}
}

func showInhibitedAlertsHandler() CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		panic("not implemented!")
	}
}

func inhibitAlertHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if alertNameOption, ok := opts[AlertNameOption]; ok {

			alertName := alertNameOption.StringValue()

			err := repo.CreateInhibition(context.Background(), i.GuildID, alertName)
			if err != nil {
				log.Printf("Failed to add inhibition for guild %s: %s", i.GuildID, err.Error())
				respondWithError(s, i, "Failed to add inhibition.")
			}

			respondWithSuccess(s, i, "Inhibition added.")

		} else {
			respondWithWarning(s, i, "Could not find alert name option.")
		}
	}
}

func uninhibitAlertHandler(repo db.Repo) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		opts := getOptionMap(i.ApplicationCommandData().Options)

		if alertNameOption, ok := opts[AlertNameOption]; ok {

			alertName := alertNameOption.StringValue()

			err := repo.DeleteInhibition(context.Background(), i.GuildID, alertName)
			if err != nil {
				log.Printf("Failed to remove inhibition for guild %s: %s", i.GuildID, err.Error())
				respondWithError(s, i, "Failed to remove inhibition.")
			}

			respondWithSuccess(s, i, "Inhibition removed.")

		} else {
			respondWithWarning(s, i, "Could not find alert name option.")
		}
	}
}

func onReadyHandler(cfg config.Bot) func(s *discordgo.Session, r *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)

		inviteLink := getInviteLink(cfg)
		log.Printf("Invite link: %s", inviteLink)
	}
}

func onInteractionCreateHandler(handlers CommandHandlers) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		log.Printf("interaction created: %s", i.Interaction.Type.String())

		if h, ok := handlers[CommandName(i.ApplicationCommandData().Name)]; ok {
			h(s, i)
		}
	}
}

func onGuildCreated(commands []*discordgo.ApplicationCommand, repo db.Repo) func(s *discordgo.Session, i *discordgo.GuildCreate) {
	return func(s *discordgo.Session, i *discordgo.GuildCreate) {

		log.Printf("guild created: '%s' (%s)", i.Guild.Name, i.Guild.ID)

		for _, v := range commands {
			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, i.Guild.ID, v)
			if err != nil {
				log.Printf("Cannot create '%v' command: %v", v.Name, err)
			}

			log.Printf("Created '%v' command", v.Name)
			err = repo.RegisterCommand(context.Background(), i.Guild.ID, cmd.ID, cmd.Name)
			if err != nil {
				log.Printf("Failed to create commands for guild %s: %s", i.Guild.ID, err.Error())
			}
		}
	}
}

func onGuildDeleted(repo db.Repo) func(s *discordgo.Session, i *discordgo.GuildDelete) {
	return func(s *discordgo.Session, i *discordgo.GuildDelete) {

		log.Printf("guild deleted: %s (%s)", i.Guild.Name, i.Guild.ID)

		ctx := context.Background()

		commands, err := repo.GetRegisteredCommand(ctx, i.Guild.ID)
		if err != nil {
			log.Printf("Failed to get commands for guild %s: %s", i.Guild.ID, err.Error())
			return
		}

		// Delete commands from guild
		for _, v := range commands {
			err := s.ApplicationCommandDelete(s.State.User.ID, i.Guild.ID, v.CommandId)
			if err != nil {
				log.Panicf("Cannot delete '%v' command %v", v.CommandName, err)
			}
		}

		err = repo.ClearGuildInfo(ctx, i.Guild.ID)
		if err != nil {
			log.Printf("Failed to clear data for guild %s: %s", i.Guild.ID, err.Error())
			return
		}
	}
}

func getInviteLink(cfg config.Bot) string {
	scopesStr := strings.Join(cfg.Scopes(), "%20")
	link := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%s&scope=%s", cfg.ClientId(), cfg.Permissions(), scopesStr)
	return link
}

func RunBot(ctx context.Context, cfg config.Bot, repo db.Repo, prometheusClient *prometheus.Client, alertsChan chan prometheus.Alerts) error {

	commands := getCommands()
	commandHandlers := getCommandHandlers(prometheusClient, repo)

	// Create a new Discord session using the provided bot token.
	log.Println("Creating session...")
	s, err := discordgo.New("Bot " + cfg.Token())
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %s", err.Error())
	}

	// Configure event handlers
	s.AddHandler(onReadyHandler(cfg))
	s.AddHandler(onGuildCreated(commands, repo))
	s.AddHandler(onInteractionCreateHandler(commandHandlers))
	s.AddHandler(onGuildDeleted(repo))

	log.Println("Opening session...")
	err = s.Open()
	if err != nil {
		return fmt.Errorf("cannot open session: %s", err.Error())
	}

	defer s.Close()

	go sendAlerts(ctx, s, cfg.GuildId(), repo, alertsChan)

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

func sendAlerts(ctx context.Context, s *discordgo.Session, guildId string, repo db.Repo, alertsChan chan prometheus.Alerts) {
	for {
		select {
		case alerts := <-alertsChan:

			alertsChannel, err := repo.GetAlertsChannel(ctx, guildId)
			if err != nil {
				log.Printf("Failed to get alerts channel: %s", err.Error())
				continue
			}

			sendAlertsToChannel(s, alertsChannel.ChannelId, alerts)

		case <-ctx.Done():
			break
		}
	}
}

func sendAlertsToChannel(s *discordgo.Session, channelId string, alerts prometheus.Alerts) {
	for _, alert := range alerts {

		title := alert.Labels["alertname"]
		url := alert.Annotations["runbook_url"]
		description := alert.Annotations["description"]
		color, err := getColorFromSeverity(alert.Labels["severity"])
		if err != nil {
			log.Panicf("Failed to generate color for alert: %s\n", err.Error())
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
			log.Panicf("Failed to send message: %s\n", err.Error())
		}
	}
}
