package main

import (
	"github.com/bwmarrin/discordgo"
)

var (
	specComponents = []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Yellow",
					Style: discordgo.SuccessButton,
					CustomID: "sWP1",
				},
				discordgo.Button{
					Label: "Blue",
					Style: discordgo.SuccessButton,
					CustomID: "sWP2",
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "DC",
					Style: discordgo.DangerButton,
					CustomID: "sDP1",
				},
				discordgo.Button{
					Label: "DC",
					Style: discordgo.DangerButton,
					CustomID: "sDP2",
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Forfeit",
					Style: discordgo.PrimaryButton,
					CustomID: "sFP1",
				},
				discordgo.Button{
					Label: "Forfeit",
					Style: discordgo.PrimaryButton,
					CustomID: "sFP2",
				},
			},
		},
	}
)