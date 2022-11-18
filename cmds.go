package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/Clinet/discordgo-embed"
	//"github.com/JoshuaDoes/json"
	//"github.com/JoshuaDoes/logger"
)

var (
	cmds []*discordgo.ApplicationCommand
	minPosition float64 = float64(1)
)

func init() {
	cmdDuel := &discordgo.ApplicationCommand{
		Name: "duel",
		Description: "Initializes a duel with the specified player",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdForfeit := &discordgo.ApplicationCommand{
		Name: "forfeit",
		Description: "Forfeits an active duel",
	}
	cmdForceDuel := &discordgo.ApplicationCommand{
		Name: "forceduel",
		Description: "Forcefully creates an active duel with the specified players",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player1",
				Description: "The first player to duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "player2",
				Description: "The second player to duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdTournamentDuel := &discordgo.ApplicationCommand{
		Name: "tournamentduel",
		Description: "Forcefully creates a tournament duel with leaderboard progression disabled",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player1",
				Description: "The first player to duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "player2",
				Description: "The second player to duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdAddWin := &discordgo.ApplicationCommand{
		Name: "addwin",
		Description: "Adds the specified +/- wins to the player",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player who will receive these wins",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "wins",
				Description: "How many wins to give/take",
				Type: discordgo.ApplicationCommandOptionInteger,
				Required: true,
			},
		},
	}
	cmdAddLoss := &discordgo.ApplicationCommand{
		Name: "addloss",
		Description: "Adds the specified +/- losses to the player",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player who will receive these losses",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "losses",
				Description: "How many losses to give/take",
				Type: discordgo.ApplicationCommandOptionInteger,
				Required: true,
			},
		},
	}
	cmdForceWin := &discordgo.ApplicationCommand{
		Name: "win",
		Description: "Forces the specified duelee to win their active duel",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player who will win their duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdForceCancel := &discordgo.ApplicationCommand{
		Name: "cancel",
		Description: "Forces the specified duelee to cancel their duel",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player who will cancel their duel",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdExtend := &discordgo.ApplicationCommand{
		Name: "extend",
		Description: "Extends the specified duelee's duel by duration",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to extend the duel duration of",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "duration",
				Description: "The duration (ex: 2d5h for 2 days and 5 hours) to extend by",
				Type: discordgo.ApplicationCommandOptionString,
				Required: true,
			},
		},
	}
	cmdShorten := &discordgo.ApplicationCommand{
		Name: "shorten",
		Description: "Shortens the specified duelee's duel by duration",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to shorten the duel duration of",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "duration",
				Description: "The duration (ex: 2d5h for 2 days and 5 hours) to shorten by",
				Type: discordgo.ApplicationCommandOptionString,
				Required: true,
			},
		},
	}
	cmdExpire := &discordgo.ApplicationCommand{
		Name: "expire",
		Description: "Applies rank droppings to and tries to expire the duel of the specified duelee",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to drop rankings of, and try to expire the duel of",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdMyDuel := &discordgo.ApplicationCommand{
		Name: "a",
		Description: "Displays statistics about your pending or active duel",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to view",
				Type: discordgo.ApplicationCommandOptionUser,
			},
		},
	}
	cmdAllDuels := &discordgo.ApplicationCommand{
		Name: "aa",
		Description: "Displays statistics about all pending and active duels",
	}
	cmdLeaderboard := &discordgo.ApplicationCommand{
		Name: "aaa",
		Description: "Shows the current leaderboards for this server",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "all",
				Description: "Tries to show all rankings, overriding specific rankings",
				Type: discordgo.ApplicationCommandOptionBoolean,
			},
			&discordgo.ApplicationCommandOption{
				Name: "rank",
				Description: "Shows the specified ranking only",
				Type: discordgo.ApplicationCommandOptionString,
			},
		},
	}
	cmdChannel := &discordgo.ApplicationCommand{
		Name: "channel",
		Description: "Sets the channels the bot can automatically post to",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "results",
				Description: "The bot will post duel results here",
				Type: discordgo.ApplicationCommandOptionChannel,
			},
			&discordgo.ApplicationCommandOption{
				Name: "reminders",
				Description: "When DMs fail, the bot will remind players about their duels here",
				Type: discordgo.ApplicationCommandOptionChannel,
			},
		},
	}
	cmdRank := &discordgo.ApplicationCommand{
		Name: "rank",
		Description: "Forces a given rank onto the specified player",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to forcibly rank",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "rank",
				Description: "The rank to give",
				Type: discordgo.ApplicationCommandOptionString,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "position",
				Description: "The position within the rank to place this player",
				Type: discordgo.ApplicationCommandOptionInteger,
				MinValue: &minPosition,
			},
		},
	}
	cmdUnrank := &discordgo.ApplicationCommand{
		Name: "unrank",
		Description: "Unranks the specified player",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "The player to unrank",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		},
	}
	cmdRankRole := &discordgo.ApplicationCommand{
		Name: "rankrole",
		Description: "Sets the role to use for a given rank (and batch removes the old role if previously set)",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "rank",
				Description: "The rank to assign a role to",
				Type: discordgo.ApplicationCommandOptionString,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "role",
				Description: "The role to assign to this rank",
				Type: discordgo.ApplicationCommandOptionRole,
				Required: true,
			},
		},
	}
	cmdSpectate := &discordgo.ApplicationCommand{
		Name: "spec",
		Description: "STARTS THE DUEL and generates a spectating interface for it",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "player",
				Description: "Spectates this player's duel, STARTING THE DUEL",
				Type: discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
			&discordgo.ApplicationCommandOption{
				Name: "setfirsthost",
				Description: "USELESS IF DUEL ALREADY STARTED, sets this player as the host for round one",
				Type: discordgo.ApplicationCommandOptionUser,
			},
		},
	}
	cmdQuote := &discordgo.ApplicationCommand{
		Name: "quote",
		Description: "Sets the winner's quote for a duel before posting the result",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "quote",
				Description: "The quote to be posted for the winner",
				Type: discordgo.ApplicationCommandOptionString,
				Required: true,
			},
		},
	}
	cmdSteamID := &discordgo.ApplicationCommand{
		Name: "steamid",
		Description: "Set your Steam ID for Steam features, such as using your live Steam nick",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Name: "steamid",
				Description: "Your Steam user ID",
				Type: discordgo.ApplicationCommandOptionInteger,
				Required: true,
			},
		},
	}
	
	cmds = []*discordgo.ApplicationCommand{
		cmdDuel, cmdForfeit,
		cmdForceDuel, cmdTournamentDuel,
		cmdAddWin, cmdAddLoss,
		cmdForceWin, cmdForceCancel,
		cmdExtend, cmdShorten, cmdExpire,
		cmdMyDuel, cmdAllDuels,
		cmdLeaderboard, cmdChannel,
		cmdRank, cmdUnrank, cmdRankRole,
		cmdSpectate, cmdQuote,
		cmdSteamID,
	}
}

func cmdRespContent(content string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	}
}
func cmdRespEmbed(msgEmbed interface{}) *discordgo.InteractionResponse {
	respEmbed := &discordgo.MessageEmbed{}
	switch msgEmbed.(type) {
	case *discordgo.MessageEmbed:
		respEmbed = msgEmbed.(*discordgo.MessageEmbed)
	case *embed.Embed:
		respEmbed = msgEmbed.(*embed.Embed).MessageEmbed
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{respEmbed},
		},
	}
}

func cmd(session *discordgo.Session, interaction *discordgo.InteractionCreate, cmdName string, opts []*discordgo.ApplicationCommandInteractionDataOption) interface{} {
	staffRole := "1013192283283275937"
	guildMember, err := session.GuildMember(interaction.GuildID, interaction.Member.User.ID)
	if err != nil {
		log.Trace("Unable to get guild member: ", err)
		return nil
	}

	isStaff := false
	for _, roleID := range guildMember.Roles {
		if roleID == staffRole {
			isStaff = true
			break
		}
	}

	switch cmdName {
	case "duel":
		players := make([]string, 0)
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				players = append(players, opts[i].UserValue(session).ID)
			}
		}
		return leaderboards[interaction.GuildID].NewDuel(interaction.Member.User.ID, players, 2, 25, false, false)
	case "forfeit":
		return leaderboards[interaction.GuildID].DuelForfeit(interaction.Member.User.ID, false)
	case "forceduel":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		players := make([]string, 0)
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player1", "player2", "player3", "player4":
				players = append(players, opts[i].UserValue(session).ID)
			}
		}
		return leaderboards[interaction.GuildID].NewDuel(players[0], players[1:], 2, 25, true, false)
	case "tournamentduel":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		players := make([]string, 0)
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player1", "player2", "player3", "player4":
				players = append(players, opts[i].UserValue(session).ID)
			}
		}
		return leaderboards[interaction.GuildID].NewDuel(players[0], players[1:], 2, 25, true, true)
	case "addwin":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		wins := int64(0)
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "wins":
				wins = opts[i].IntValue()
			}
		}

		leaderboards[interaction.GuildID].Players[player].MatchesWon += wins
		return leaderboards[interaction.GuildID].EmbedStats(interaction.GuildID, interaction.Member.User.ID, player)
	case "addloss":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		losses := int64(0)
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "losses":
				losses = opts[i].IntValue()
			}
		}

		leaderboards[interaction.GuildID].Players[player].MatchesLost += losses
		return leaderboards[interaction.GuildID].EmbedStats(interaction.GuildID, interaction.Member.User.ID, player)
	case "win":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := opts[0].UserValue(session).ID
		return leaderboards[interaction.GuildID].DuelWin(player)
	case "extend":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		duration := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "duration":
				duration = opts[i].StringValue()
			}
		}
		return leaderboards[interaction.GuildID].DuelExtend(player, duration)
	case "shorten":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		duration := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "duration":
				duration = opts[i].StringValue()
			}
		}
		return leaderboards[interaction.GuildID].DuelShorten(player, duration)
	case "expire":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := opts[0].UserValue(session).ID
		return leaderboards[interaction.GuildID].DuelForfeit(player, true)
	case "a": //MyDuel
		player := interaction.Member.User.ID
		if len(opts) > 0 {
			player = opts[0].UserValue(session).ID
		}
		return leaderboards[interaction.GuildID].EmbedStats(interaction.GuildID, interaction.Member.User.ID, player)
	case "aa": //AllDuels
		return leaderboards[interaction.GuildID].EmbedDuels(interaction.GuildID)
	case "aaa":
		all := false
		displayRank := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name{
			case "all":
				all = opts[i].BoolValue()
			case "rank":
				displayRank = opts[i].StringValue()
			}
		}
		if all {
			displayRank = "all"
		}
		return leaderboards[interaction.GuildID].EmbedRanks(interaction.GuildID, displayRank)
	case "channel":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		results := ""
		reminders := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "results":
				results = opts[i].ChannelValue(session).ID
			case "reminders":
				reminders = opts[i].ChannelValue(session).ID
			}
		}
		if results != "" {
			leaderboards[interaction.GuildID].ChannelResults = results
		}
		if reminders != "" {
			leaderboards[interaction.GuildID].ChannelReminders = reminders
		}
		return embed.NewGenericEmbed("Channels", "Results: <#%s>\nReminders: <#%s>", leaderboards[interaction.GuildID].ChannelResults, leaderboards[interaction.GuildID].ChannelReminders)
	case "rank":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		rank := ""
		position := -1
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "rank":
				rank = opts[i].StringValue()
			case "position":
				position = int(opts[i].IntValue())
			}
		}

		Discord.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Syncing ranks...",
			},
		})
		rankEmbed := leaderboards[interaction.GuildID].ApplyRank(player, rank, position, true, true).(*discordgo.MessageEmbed)
		_, err := Discord.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{rankEmbed},
		})
		log.Trace(err)

		return nil
	case "unrank":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			}
		}
		return leaderboards[interaction.GuildID].ApplyRank(player, "unranked", 0, false, false)
	case "rankrole":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		tier := ""
		role := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "rank":
				tier = opts[i].StringValue()
			case "role":
				role = opts[i].RoleValue(session, interaction.GuildID).ID
			}
		}

		rankIndex, rank := leaderboards[interaction.GuildID].GetRawRank(tier)
		if rankIndex < 0 || rank == nil {
			return embed.NewErrorEmbed("Rank Role", "Unknown rank %s.", rank.Rank)
		}

		leaderboards[interaction.GuildID].Ranks[rankIndex].RoleID = role

		if len(rank.Players) > 0 {
			Discord.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			for _, player := range rank.Players {
				if rank.RoleID != "" {
					Discord.GuildMemberRoleRemove(interaction.GuildID, player, rank.RoleID)
				}
				Discord.GuildMemberRoleAdd(interaction.GuildID, player, role)
			}
		}

		_, err := Discord.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed.NewGenericEmbed("Rank Role", "Changed rank %s to role <@&%s>.", rank.Rank, role)},
		})
		log.Trace(err)
		return nil
	case "spec":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		host := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			case "setfirsthost":
				host = opts[i].UserValue(session).ID
			}
		}

		duelIndex, _ := leaderboards[interaction.GuildID].GetActiveDuel(player)
		if duelIndex < 0 || leaderboards[interaction.GuildID].ActiveDuels[duelIndex] == nil {
			return embed.NewErrorEmbed("Spec", "No active duel for %s.", mention(player, interaction.GuildID, true))
		}
		if host != "" && leaderboards[interaction.GuildID].ActiveDuels[duelIndex].CurrentRound <= 0 {
			if !leaderboards[interaction.GuildID].ActiveDuels[duelIndex].HasPlayer(host) {
				return embed.NewErrorEmbed("Spec", "Unable to set host %s for this duel, they aren't participating!", mention(host, interaction.GuildID, true))
			}
			players := []string{host}
			for _, nonHost := range leaderboards[interaction.GuildID].ActiveDuels[duelIndex].Players {
				if nonHost != host {
					players = append(players, nonHost)
				}
			}
			leaderboards[interaction.GuildID].ActiveDuels[duelIndex].Players = players
		}

		leaderboards[interaction.GuildID].ActiveDuels[duelIndex].Spectators[interaction.Member.User.ID] = NewSpec(player, interaction.Interaction)
		leaderboards[interaction.GuildID].ActiveDuels[duelIndex].Start()

		specEmbed := leaderboards[interaction.GuildID].ActiveDuels[duelIndex].Embed(interaction.GuildID).(*embed.Embed)

		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				CustomID: "spec",
				Embeds: []*discordgo.MessageEmbed{specEmbed.MessageEmbed},
				Components: specComponents,
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}
	case "quote":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		quote := opts[0].StringValue()
		leaderboards[interaction.GuildID].DuelQuote(interaction.Member.User.ID, quote)
	case "cancel":
		if !isStaff {
			return embed.NewErrorEmbed("Missing Permissions", "I'm sorry, Dave. I'm afraid I can't do that.")
		}

		player := ""
		for i := 0; i < len(opts); i++ {
			switch opts[i].Name {
			case "player":
				player = opts[i].UserValue(session).ID
			}
		}
		return leaderboards[interaction.GuildID].DuelCancel(player)
	}
	return embed.NewErrorEmbed("Invalid Command", "How'd you even come up with that one?")
}
