package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/Clinet/discordgo-embed"
)

type DiscordClient struct {
	*discordgo.Session
	User *discordgo.User
}
func (dc *DiscordClient) Shutdown() {
	dc.Sync()
	if err := dc.Offline(); err != nil {
		log.Error(err)
	}
	if err := dc.Close(); err != nil {
		log.Error(err)
	}
}
func (dc *DiscordClient) Sync() {
	log.Info("... Saving states")
	if err := saveJSON(leaderboards, "states/leaderboards.json"); err != nil {
		log.Error(err)
	} else {
		log.Info("Saved states successfully")
	}
}

func (dc *DiscordClient) Online() error {
	idleSince := int(0)
	return dc.UpdateStatusComplex(discordgo.UpdateStatusData{
		IdleSince: &idleSince,
		AFK: false,
		Status: "",
	})
}
func (dc *DiscordClient) Offline() error {
	idleSince := int(1)
	return dc.UpdateStatusComplex(discordgo.UpdateStatusData{
		IdleSince: &idleSince,
		AFK: true,
		Status: "",
	})
}

func (dc *DiscordClient) RegisterApplicationCmds() {
	log.Info("... Registering application commands")
	_, err := Discord.ApplicationCommandBulkOverwrite(Discord.State.User.ID, "", cmds)
	if err != nil {
		log.Fatal("Unable to register commands: ", err)
	}
	appCmds, err := Discord.ApplicationCommands(Discord.State.User.ID, "")
	if err != nil {
		log.Fatal("Unable to get list of commands: ", err)
	}
	for _, appCmd := range appCmds {
		exists := false
		for _, cmd := range cmds {
			if appCmd.Name == cmd.Name {
				exists = true
				break
			}
		}
		if !exists {
			err = Discord.ApplicationCommandDelete(Discord.State.User.ID, "", appCmd.ID)
			if err != nil {
				log.Fatal("Unable to delete old command: ", err)
			}
		}
	}
	log.Info("Application commands registered!")
}

func discordReady(session *discordgo.Session, event *discordgo.Ready) {
	defer Discord.Sync()

	for Discord == nil {
		if Discord != nil {
			break //Wait for Discord to finish connecting, just in case we're called early
		}
	}
	Discord.User = event.User
	log.Info("Logged into Discord as ", Discord.User, "!")

	Discord.RegisterApplicationCmds()
	if err := Discord.Online(); err != nil {
		log.Error(err)
	}

	log.Info("Loading active duels...")
	for _, leaderboard := range leaderboards {
		leaderboard.TrackDuels()
	}
}

func discordGuildMemberRemove(session *discordgo.Session, member *discordgo.GuildMemberRemove) {
	defer Discord.Sync()

	if _, exists := leaderboards[member.GuildID]; exists {
		leaderboards[member.GuildID].DeletePlayer(member.User.ID)
	}
}

func discordInteractionCreate(session *discordgo.Session, event *discordgo.InteractionCreate) {
	log.Trace("INTERACTION: ", event.ID, event.AppID, event.Type, event.Data, event.GuildID, event.ChannelID, event.Message, event.AppPermissions, event.Member, event.User, event.Locale, event.GuildLocale, event.Token, event.Version)
	defer Discord.Sync()

	//Create a new leaderboard for this guild if it doesn't exist yet
	if _, exists := leaderboards[event.GuildID]; !exists {
		leaderboards[event.GuildID] = NewLeaderboard(event.GuildID)
	}
	leaderboards[event.GuildID].InitPlayer(event.Member.User.ID)

	switch event.Type {
	case discordgo.InteractionApplicationCommand:
		cmdName := event.ApplicationCommandData().Name
		resp := cmd(session, event, cmdName, event.ApplicationCommandData().Options)
		if resp != nil {
			switch resp.(type) {
			case *discordgo.InteractionResponse:
				respInter := resp.(*discordgo.InteractionResponse)
				err := session.InteractionRespond(event.Interaction, respInter)
				if err != nil {
					log.Error(err)
				} else {
					log.Debug("Responded to command ", cmdName, " with response: ", respInter.Data.Content)
				}
			case *embed.Embed, *discordgo.MessageEmbed:
				respEmbed := cmdRespEmbed(resp)
				err := session.InteractionRespond(event.Interaction, respEmbed)
				if err != nil {
					log.Error(err)
				} else {
					log.Debug("Responded to command ", cmdName, " with response: ", fmt.Sprintf("%v", respEmbed))
				}
			default:
				log.Error("unknown response type for resp from command ", cmdName)
			}
		} else {
			log.Info("No response to interaction for command ", cmdName)
		}
	case discordgo.InteractionMessageComponent:
		msgData := event.MessageComponentData()
		if msgData.ComponentType != discordgo.ButtonComponent {
			return
		}

		duelIndex, duel := leaderboards[event.GuildID].GetSpectatorDuel(event.Member.User.ID)
		if duelIndex < 0 || duel == nil {
			return
		}

		leaderboards[event.GuildID].ActiveDuels[duelIndex].Spectators[event.Member.User.ID].LastInteraction = event.Interaction

		switch msgData.CustomID {
		case "sWP1":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Win(duel.Players[0])
		case "sWP2":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Win(duel.Players[1])
		case "sDP1":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Discount(duel.Players[0])
		case "sDP2":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Discount(duel.Players[1])
		case "sFP1":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Forfeit(duel.Players[0])
		case "sFP2":
			leaderboards[event.GuildID].ActiveDuels[duelIndex].Forfeit(duel.Players[1])
		}

		specEmbed := leaderboards[event.GuildID].ActiveDuels[duelIndex].Embed(event.GuildID).(*embed.Embed)
		components := specComponents

		duelIndex, _ = leaderboards[event.GuildID].GetSpectatorDuel(event.Member.User.ID)
		if duelIndex < 0 || duelIndex >= len(leaderboards[event.GuildID].ActiveDuels) {
			components = make([]discordgo.MessageComponent, 0)
		} else {
			if len(leaderboards[event.GuildID].ActiveDuels[duelIndex].Winners) > 0 {
				specEmbed.SetFooter("Please use /quote to provide the winner's quote! The duel won't archive and the result won't post until then.")
			}
		}

		resp := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				CustomID: "spec",
				Embeds: []*discordgo.MessageEmbed{specEmbed.MessageEmbed},
				Components: components,
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}

		for spectator, _ := range leaderboards[event.GuildID].ActiveDuels[duelIndex].Spectators {
			if err := session.InteractionRespond(leaderboards[event.GuildID].ActiveDuels[duelIndex].Spectators[spectator].LastInteraction, resp); err != nil {
				log.Error(err)
				delete(leaderboards[event.GuildID].ActiveDuels[duelIndex].Spectators, spectator)
			} else {
				log.Debug("Responded to button ", msgData.CustomID, " with response: ", fmt.Sprintf("%v", resp))
			}
		}
	}
}

func mention(userID, guildID string, skipCache bool) string {
	leaderboards[guildID].InitPlayer(userID)
	if !skipCache{
		if player, exists := leaderboards[guildID].Players[userID]; exists {
			if player.Nickname != "" {
				return player.Nickname
			}
		}
	}

	member, err := Discord.GuildMember(guildID, userID)
	if err == nil {
		if member.Nick != "" {
			leaderboards[guildID].Players[userID].Nickname = member.Nick
			return member.Nick
		}
		if member.User != nil {
			leaderboards[guildID].Players[userID].Nickname = member.User.Username
			return member.User.Username
		}
	}

	return ""
}
func guildName(guildID string) string {
	guild, err := Discord.Guild(guildID)
	if err == nil {
		return guild.Name
	}

	return ""
}
