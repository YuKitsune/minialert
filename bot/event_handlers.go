package bot

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/minialert/config"
	"github.com/yukitsune/minialert/db"
)

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

		ctx := context.TODO()

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
