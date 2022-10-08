package bot

import "github.com/bwmarrin/discordgo"

type InteractionName string

const (
	GetAlertsCommandName InteractionName = "get-alerts"

	ShowInhibitedAlertsCommandName InteractionName = "show-inhibited-alerts"
	InhibitAlertCommandName        InteractionName = "inhibit-alert"
	UninhibitAlertCommandName      InteractionName = "uninhibit-alert"

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
