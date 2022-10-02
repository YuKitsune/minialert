package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

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
