package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/Clinet/discordgo-embed"
	"github.com/dustin/go-humanize"
)

type Duel struct {
	GuildID      string   `json:"guildID"`       //ID of guild hosting this duel, for accessing leaderboards
	Players      []string `json:"activePlayers"` //Players participating in this duel
	Dueler       string   `json:"dueler"`        //Player who created the original duel request
	RoundLimit   int      `json:"roundLimit"`    //How many rounds this duel should last for
	ScoreLimit   int      `json:"scoreLimit"`    //How many wins should be achieved per round
	Unranked     bool     `json:"unranked"`      //If unranked, this duel shouldn't apply leaderboard progressions

	Expires      time.Time             `json:"expires"`      //When this duel should expire
	CurrentRound int                   `json:"currentRound"` //0 indicates not started, and > RoundLimit indicates a finished duel
	DuelStats    map[string]*DuelStats `json:"duelStats"`    //Current player statistics for the duel, where key is Discord user ID
	Winners      []string              `json:"winners"`      //Empty until duel is won
	Losers       []string              `json:"losers"`       //Empty until duel is won
}

type DuelStats struct {
	Scores       []int `json:"score"`      //Round-ordered score list
	FinalScore   int   `json:"finalScore"` //Final score when duel ends
	Forfeit      bool  `json:"forfeit"`
	ForcedWinner bool `json:"ForcedWinner"`
}

func (d *Duel) IsHost(round int, player string) bool {
	currentHost := round
	if currentHost >= len(d.Players) {
		for currentHost >= len(d.Players) {
			currentHost -= len(d.Players)
		}
	}

	for i := 0; i < len(d.Players); i++ {
		if d.Players[i] == player {
			if currentHost == i {
				return true
			}
			return false
		}
	}
	return false
}
func (d *Duel) HasPlayer(player string) bool {
	for i := 0; i < len(d.Players); i++ {
		if d.Players[i] == player {
			return true
		}
	}
	return false
}

func (d *Duel) String(guildID string) string {
	msg := mention(d.Players[0], guildID, false) + " vs " + mention(d.Players[1], guildID, false)
	if len(d.Players) > 2 {
		for i := 2; i < len(d.Players); i++ {
			msg += " vs " + mention(d.Players[i], guildID, false)
		}
	}
	msg += fmt.Sprintf("\nDuel started by %s\n", mention(d.Dueler, guildID, false))

	timeLeft := d.Expires.Sub(time.Now())
	if timeLeft.Minutes() < 1 {
		msg += fmt.Sprintf("Duel expires %s", humanize.Time(d.Expires))
	} else {
		msg += fmt.Sprintf("Duel expires in over %s", humanize.Time(d.Expires))
	}
	return msg
}
func (d *Duel) Title(guildID string) string {
	title := fmt.Sprintf("%s vs %s", mention(d.Players[0], guildID, false), mention(d.Players[1], guildID, false))
	if len(d.Players) > 2 {
		for i := 2; i < len(d.Players); i++ {
			title += " vs " + mention(d.Players[i], guildID, false)
		}
	}
	return title
}
func (d *Duel) TitleWithMentions() string {
	title := fmt.Sprintf("<@%s> vs <@%s>", d.Players[0], d.Players[1])
	if len(d.Players) > 2 {
		for i := 2; i < len(d.Players); i++ {
			title += fmt.Sprintf(" vs <@%s>", d.Players[i])
		}
	}
	return title
}
func (d *Duel) Scores() string {
	scores := ""
	for i := 0; i < d.RoundLimit; i++ {
		if i == (d.CurrentRound-1) {
			scores += "> "
		}
		winner := ""
		winnerScore := -1
		for j := 0; j < len(d.Players); j++ {
			if d.IsHost(i, d.Players[j]) {
				scores += fmt.Sprintf("**%d**", d.DuelStats[d.Players[j]].Scores[i])
			} else {
				scores += fmt.Sprintf("%d", d.DuelStats[d.Players[j]].Scores[i])
			}
			if j != len(d.Players)-1 {
				scores += " - "
			}
			if d.DuelStats[d.Players[j]].Scores[i] > winnerScore {
				winner = d.Players[j]
				winnerScore = d.DuelStats[d.Players[j]].Scores[i]
			}
		}
		if i < d.CurrentRound-1 {
			scores += ": **" + mention(winner, d.GuildID, false) + "**"
		}
		if i != d.RoundLimit-1 {
			scores += "\n"
		}
	}
	if len(d.Winners) > 0 && len(d.Losers) > 0 {
		scores += fmt.Sprintf("\n%s wins: %d - %d", mention(d.Winners[0], d.GuildID, false), d.DuelStats[d.Winners[0]].FinalScore, d.DuelStats[d.Losers[0]].FinalScore)
	}
	return scores
}
func (d *Duel) Embed(guildID string) interface{} {
	duelEmbed := embed.NewEmbed().
		SetTitle(d.Title(guildID)).
		SetDescription("Duel started by " + mention(d.Dueler, guildID, false)).
		SetColor(config.ColorMain)

	if d.CurrentRound > 0 {
		duelEmbed.SetDescription(d.Scores())
		for player, duelStat := range d.DuelStats {
			if duelStat.ForcedWinner {
				duelEmbed.SetDescription("Duel was forcibly won by " + mention(player, guildID, false))
				break
			} else if duelStat.Forfeit {
				duelEmbed.SetDescription("Duel was forfeit by " + mention(player, guildID, false))
				break
			}
		}
		spectators := ""
		for spectator, _ := range leaderboards[d.GuildID].Spectators {
			if spectators != "" {
				spectators += ", "
			}
			spectators += mention(spectator, guildID, false)
		}
		duelEmbed.AddField("Spectators", spectators)
	} else {	
		timeLeft := d.Expires.Sub(time.Now())
		if timeLeft.Minutes() < 1 {
			duelEmbed.SetFooter("Expires " + humanize.Time(d.Expires))
		} else {
			duelEmbed.SetFooter("Expires in over " + humanize.Time(d.Expires))
		}
	}
	return duelEmbed
}
func (d *Duel) EmbedField(guildID string) *discordgo.MessageEmbedField {
	if d.CurrentRound > 0 {
		return &discordgo.MessageEmbedField{Name: d.Title(guildID), Value: d.Scores()}
	}
	timeLeft := d.Expires.Sub(time.Now())
	if timeLeft.Minutes() < 1 {
		return &discordgo.MessageEmbedField{Name: d.Title(guildID), Value: "Expires " + humanize.Time(d.Expires)}
	}
	return &discordgo.MessageEmbedField{Name: d.Title(guildID), Value: "Expires in over " + humanize.Time(d.Expires)}
}

func NewDuel(guildID, dueler string, players []string, roundLimit, scoreLimit int, unranked bool) *Duel {
	duelStats := make(map[string]*DuelStats)
	for _, player := range players {
		duelStats[player] = &DuelStats{
			Scores: make([]int, roundLimit),
		}
	}
	return &Duel{
		GuildID: guildID,
		Dueler: dueler,
		Players: players,
		RoundLimit: roundLimit,
		ScoreLimit: scoreLimit,
		Unranked: unranked,
		Expires: time.Now().AddDate(0, 0, 7),
		DuelStats: duelStats,
	}
}

func (d *Duel) Init() {
	for _, player := range d.Players {
		if duelStat, exists := d.DuelStats[player]; !exists || duelStat == nil {
			log.Trace("Duel init for duel stats on player " + player)
			d.DuelStats[player] = &DuelStats{}
		}
		if d.DuelStats[player].Scores == nil || len(d.DuelStats[player].Scores) == 0 {
			log.Trace("Duel init for player scores on player " + player)
			d.DuelStats[player].Scores = make([]int, d.RoundLimit)
		}
	}
}

func (d *Duel) Start() {
	if d.CurrentRound > 0 && d.CurrentRound <= d.RoundLimit {
		return
	}
	if d.CurrentRound > d.RoundLimit {
		return
	}
	d.CurrentRound = 1

	return
}

func (d *Duel) CheckWinner() {
	winners := make([]string, 0)
	losers := make([]string, 0)

	for _, player := range d.Players {
		log.Trace(player, " ", d.CurrentRound, " ", d.DuelStats[player])
		if d.DuelStats[player].Forfeit {
			d.CurrentRound = d.RoundLimit + 1
			for _, tmp := range d.Players {
				if tmp != player {
					winners = append(winners, tmp)
				} else {
					losers = append(losers, tmp)
				}
			}

			d.Winners = winners
			d.Losers = losers
			leaderboards[d.GuildID].DuelEnd(d)
			return
		} else if d.DuelStats[player].ForcedWinner {
			d.CurrentRound = d.RoundLimit + 1
			for _, tmp := range d.Players {
				if tmp != player {
					losers = append(losers, tmp)
				} else {
					winners = append(winners, tmp)
				}
			}

			d.Winners = winners
			d.Losers = losers
			leaderboards[d.GuildID].DuelEnd(d)
			return
		}
	}

	if d.CurrentRound > 0 {
		for _, player := range d.Players {
			playerScore := d.DuelStats[player].Scores[d.CurrentRound-1]
			if playerScore >= d.ScoreLimit {
				d.CurrentRound++
				if d.CurrentRound > d.RoundLimit {
					finalScores := make([]int, len(d.Players))
					for tmpIndex, tmp := range d.Players {
						score := 0
						for _, roundScore := range d.DuelStats[tmp].Scores {
							score += roundScore
						}
						finalScores[tmpIndex] = score
						d.DuelStats[tmp].FinalScore = score
					}
					winnerIndex := -1
					winnerScore := 0
					for tmpIndex, finalScore := range finalScores {
						if finalScore > winnerScore {
							winnerIndex = tmpIndex
							winnerScore = finalScore
						}
					}
					for tmpIndex, tmp := range d.Players {
						if tmpIndex == winnerIndex {
							winners = append(winners, tmp)
						} else {
							losers = append(losers, tmp)
						}
					}
					d.Winners = winners
					d.Losers = losers
					leaderboards[d.GuildID].DuelEnd(d)
				}

				return
			}
		}
	}
}

func (d *Duel) Win(duelee string) {
	duelStat := d.DuelStats[duelee]
	duelStat.Scores[d.CurrentRound-1]++
	d.DuelStats[duelee] = duelStat
	d.CheckWinner()
}
func (d *Duel) ForceWin(duelee string) {
	duelStat := d.DuelStats[duelee]
	for i := 0; i < len(duelStat.Scores); i++ {
		duelStat.Scores[i] = d.ScoreLimit
	}
	duelStat.ForcedWinner = true
	d.DuelStats[duelee] = duelStat
	d.CurrentRound = d.RoundLimit
	d.CheckWinner()
}
func (d *Duel) Discount(duelee string) {
	duelStat := d.DuelStats[duelee]
	if duelStat.Scores[d.CurrentRound-1] > 0 {
		duelStat.Scores[d.CurrentRound-1]--
	}
	d.DuelStats[duelee] = duelStat
	d.CheckWinner()
}
func (d *Duel) Forfeit(duelee string) {
	log.Trace(duelee)
	log.Trace(d.DuelStats)
	log.Trace(d.DuelStats[duelee])
	duelStat := d.DuelStats[duelee]
	log.Trace(duelStat)
	duelStat.Forfeit = true
	d.DuelStats[duelee] = duelStat
	d.CheckWinner()
}
