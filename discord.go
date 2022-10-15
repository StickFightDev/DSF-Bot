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
	log.Info("... Saving states")
	if err := saveJSON(leaderboards, "states/leaderboards.json"); err != nil {
		log.Error(err)
	}
	log.Info("Good-bye!")
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
	for Discord == nil {
		if Discord != nil {
			break //Wait for Discord to finish connecting, just in case we're called early
		}
	}
	Discord.User = event.User
	log.Info("Logged into Discord as ", Discord.User, "!")

	log.Info("Loading active duels...")
	for _, leaderboard := range leaderboards {
		for duelIndex := range leaderboard.ActiveDuels {
			leaderboard.TrackDuel(duelIndex)
		}
	}

	Discord.RegisterApplicationCmds()
}

func discordGuildMemberRemove(session *discordgo.Session, member *discordgo.GuildMemberRemove) {
	if _, exists := leaderboards[member.GuildID]; exists {
		leaderboards[member.GuildID].DeletePlayer(member.User.ID)
	}
}

func discordInteractionCreate(session *discordgo.Session, event *discordgo.InteractionCreate) {
	log.Trace("INTERACTION: ", event.ID, event.AppID, event.Type, event.Data, event.GuildID, event.ChannelID, event.Message, event.AppPermissions, event.Member, event.User, event.Locale, event.GuildLocale, event.Token, event.Version)
	
	//Create a new leaderboard for this guild if it doesn't exist yet
	if _, exists := leaderboards[event.GuildID]; !exists {
		leaderboards[event.GuildID] = NewLeaderboard(event.GuildID)
	}
	if leaderboards[event.GuildID].Spectators == nil {
		leaderboards[event.GuildID].Spectators = make(map[string]string)
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
		if _, exists := leaderboards[event.GuildID].Spectators[event.Member.User.ID]; !exists {
			return
		}

		specPlayer := leaderboards[event.GuildID].Spectators[event.Member.User.ID]
		duelIndex, duel := leaderboards[event.GuildID].GetActiveDuel(specPlayer)
		if duelIndex < 0 || duel == nil {
			return
		}

		switch msgData.CustomID {
		case "sWP1":
			duel.Win(duel.Players[0])
		case "sWP2":
			duel.Win(duel.Players[1])
		case "sDP1":
			duel.Discount(duel.Players[0])
		case "sDP2":
			duel.Discount(duel.Players[1])
		case "sFP1":
			duel.Forfeit(duel.Players[0])
		case "sFP2":
			duel.Forfeit(duel.Players[1])
		}

		specEmbed := embed.NewEmbed().
			SetTitle(duel.Title(event.GuildID)).
			SetDescription(duel.Scores()).
			SetColor(config.ColorMain)
		components := specComponents

		duelIndex, _ = leaderboards[event.GuildID].GetActiveDuel(specPlayer)
		if duelIndex < 0 || duelIndex >= len(leaderboards[event.GuildID].ActiveDuels) {
			components = nil
			specEmbed = duel.Embed(event.GuildID).(*embed.Embed)
		} else {
			leaderboards[event.GuildID].ActiveDuels[duelIndex] = duel
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

		err := session.InteractionRespond(event.Interaction, resp)
		if err != nil {
			log.Error(err)
		} else {
			log.Debug("Responded to button ", msgData.CustomID, " with response: ", fmt.Sprintf("%v", resp))
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