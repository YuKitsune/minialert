package bot

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/db"
	"github.com/yukitsune/minialert/prometheus"
	"github.com/yukitsune/minialert/scraper"
	"github.com/yukitsune/minialert/util"
	"strconv"
)

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

func watchAlerts(done chan bool, s *discordgo.Session, repo db.Repo, scrapeManager *scraper.ScrapeManager, logger logrus.FieldLogger) {
	for {
		select {
		case results := <-scrapeManager.Chan():

			ctx := context.TODO()

			guildConfig, err := repo.GetGuildConfig(ctx, results.GuildId)
			if err != nil {
				logger.Errorf("Failed to get guild config: %s", err.Error())
				return
			}

			scrapeConfig, ok := util.FindMatching(guildConfig.ScrapeConfigs, func(cfg db.ScrapeConfig) bool {
				return cfg.Name == results.ScrapeConfigName
			})

			if !ok {
				logger.Warn("Guild config for %s doesn't contain a scrape config with the name %s", results.GuildId, results.ScrapeConfigName)
				return
			}

			filteredAlerts, err := prometheus.FilterAlerts(results.Alerts, scrapeConfig.Inhibitions)
			if err != nil {
				logger.Errorf("Failed to filter alerts: %s", err.Error())
				continue
			}

			sendAlertsToChannel(s, scrapeConfig.Name, scrapeConfig.AlertChannelId, filteredAlerts, logger)

		case <-done:
			logger.Debug("Stopping watchAlerts")
			return
		}
	}
}

func sendAlertsToChannel(s *discordgo.Session, configName string, channelId string, alerts prometheus.Alerts, logger logrus.FieldLogger) {
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
			CustomID: NewMessageInteractionId(InhibitAlertCommandName, configName, alertName).String(),
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
