package main

import (
	"github.com/bwmarrin/discordgo"
)

type Spec struct {
	PlayerID        string                 `json:"playerID"`        //The player being spectated
	LastInteraction *discordgo.Interaction `json:"lastInteraction"` //The last interaction from the spec
}

func NewSpec(playerID string, interaction *discordgo.Interaction) *Spec {
	return &Spec{PlayerID: playerID, LastInteraction: interaction}
}

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