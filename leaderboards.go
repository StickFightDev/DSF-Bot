package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/Clinet/discordgo-embed"
)

var (
	regexRank = regexp.MustCompile(`((\(|\[|{)(X|S|A\+|A|B|C|D|E|F)(.*?)(\d*)(\)|\]|}))`)
	regexSpaces = regexp.MustCompile(`(\s+)`)
)

type Leaderboard struct {
	sync.Mutex //Enable mutex locking to prevent race conditions

	GuildID          string             `json:"guildID"`          //The guild ID of this leaderboard
	ChannelResults   string             `json:"channelResults"`   //The channel ID to post results to
	ChannelReminders string             `json:"channelReminders"` //The channel ID to post reminders to
	ActiveDuels      []*Duel            `json:"activeDuels"`      //Active duels in order of creation
	DuelHistory      []*Duel            `json:"duelHistory"`      //Duel graveyard for historical archives
	Ranks            []*Rank            `json:"ranks"`            //In-order list of ranks
	Players          map[string]*Player `json:"players"`          //Map of raw players and their stats, where key is Discord user ID
	Spectators       map[string]string  `json:"spectators"`       //Map of spectated duel players, where key is spectator's Discord user ID
}

func NewLeaderboard(guildID string) *Leaderboard {
	l := &Leaderboard{
		GuildID: guildID,
		ActiveDuels: make([]*Duel, 0),
		DuelHistory: make([]*Duel, 0),
		Ranks: config.Ranks,
		Players: make(map[string]*Player),
		Spectators: make(map[string]string),
	}
	return l
}

type Spec struct {
	DuelIndex int
	ChannelID string `json:"channelID"` //The channel of the spec message
	MessageID string `json:"messageID"` //The spec message to update with new scores
}

type Rank struct {
	Rank          string   `json:"rank"`          //The rank identifier, such as X or S
	Prefix        string   `json:"prefix"`        //The prefix for ranks in nicknames
	Suffix        string   `json:"suffix"`        //The suffix for ranks in nicknames
	PlayerLimit   int      `json:"playerLimit"`   //How many players may fill this rank
	IgnoreRankPos bool     `json:"ignoreRankPos"` //Whether to ignore rank positions when checking duel abilities (but still check rank difference)
	RoleID        string   `json:"roleID"`        //The role to assign to players of this rank
	Players       []string `json:"players"`       //In-order list of players for this rank
}
func (r *Rank) Style(position int) string {
	if position > 9 {
		return fmt.Sprintf("%s%d", r.Rank, position)
	}
	return fmt.Sprintf("%s %d", r.Rank, position)
}

func (l *Leaderboard) GetRawRank(rankTier string) (int, *Rank) {
	for rankIndex, rank := range l.Ranks {
		if rank.Rank == rankTier {
			return rankIndex, rank
		}
	}
	return -1, nil
}

//GetRank returns the rank index and the current player position starting from 1
//Access the *Rank with l.Ranks[rankIndex], determine success with >-1&&>0
func (l *Leaderboard) GetRank(duelee string) (int, int) {
	for rankIndex, rank := range l.Ranks {
		for playerIndex, player := range rank.Players {
			if player == duelee {
				return rankIndex, playerIndex+1
			}
		}
	}
	return -1, 0
}

func (l *Leaderboard) FixRanks() {
	//Fix the rank positions first
	for rankIndex := 0; rankIndex < len(l.Ranks); rankIndex++ {
		if l.Ranks[rankIndex].PlayerLimit <= 0 {
			break //Stop processing once we reach the first rank with no player limit
		}

		if len(l.Ranks[rankIndex].Players) > l.Ranks[rankIndex].PlayerLimit {
			//Get a list of scratched players
			scratchedPlayers := l.Ranks[rankIndex].Players[l.Ranks[rankIndex].PlayerLimit:]
			log.Debug("rank has too many players: ", scratchedPlayers)
			//Remove the scratched players from the current rank
			l.Ranks[rankIndex].Players = l.Ranks[rankIndex].Players[:l.Ranks[rankIndex].PlayerLimit]
			//Remove the scratched players' roles
			if l.Ranks[rankIndex].RoleID != "" {
				for _, sP := range scratchedPlayers {
					Discord.GuildMemberRoleRemove(l.GuildID, sP, l.Ranks[rankIndex].RoleID)
				}
			}
			//Check to make sure there is a next rank
			if rankIndex+1 < len(l.Ranks) {
				//Append the list with the next rank's players
				scratchedPlayers = append(scratchedPlayers, l.Ranks[rankIndex+1].Players...)
				//Update the next rank's players with the new list
				l.Ranks[rankIndex+1].Players = scratchedPlayers
				log.Debug("migrated scratched players to new rank: ", scratchedPlayers)
			}
		} else if len(l.Ranks[rankIndex].Players) < l.Ranks[rankIndex].PlayerLimit {
			log.Debug("rank doesn't have enough players")
			if rankIndex+1 < len(l.Ranks) {
				for rI := rankIndex+1; rI < len(l.Ranks); rI++ {
					if len(l.Ranks[rankIndex].Players) == l.Ranks[rankIndex].PlayerLimit {
						break
					}

					//Find the next player to migrate to this rank
					if len(l.Ranks[rI].Players) > 0 {
						//Remove the player's old role
						if l.Ranks[rI].RoleID != "" {
							Discord.GuildMemberRoleRemove(l.GuildID, l.Ranks[rI].Players[0], l.Ranks[rI].RoleID)
						}
						//Append the player to this rank
						l.Ranks[rankIndex].Players = append(l.Ranks[rankIndex].Players, l.Ranks[rI].Players[0])
						log.Debug("new rank players: ", l.Ranks[rankIndex].Players)
						//Remove the player from their old rank
						if len(l.Ranks[rI].Players) > 1 {
							l.Ranks[rI].Players = l.Ranks[rI].Players[1:]
						} else {
							l.Ranks[rI].Players = make([]string, 0)
						}
						log.Debug("old rank players: ", l.Ranks[rI].Players)
						rI-- //Re-loop on the same rankIndex to look for the next player in it
					}
				}
			}
		}
	}

	//Fix the nicknames and roles once the positions are correct
	for rankIndex := 0; rankIndex < len(l.Ranks); rankIndex++ {
		for rankPos, player := range l.Ranks[rankIndex].Players {
			updated := false
			if l.Players[player].Rank != l.Ranks[rankIndex].Rank {
				l.Players[player].Rank = l.Ranks[rankIndex].Rank
				updated = true
			}
			if l.Players[player].RankPos != rankPos+1 && l.Ranks[rankIndex].PlayerLimit > 0 {
				l.Players[player].RankPos = rankPos+1
				updated = true
			}
			if updated {
				log.Trace("Updating player: ", player)
				nickname := mention(player, l.GuildID, true)
				oldNick := nickname
				rankPrefix := l.Ranks[rankIndex].Prefix + l.Ranks[rankIndex].Rank
				if l.Ranks[rankIndex].PlayerLimit > 0 {
					rankPrefix = l.Ranks[rankIndex].Prefix + l.Ranks[rankIndex].Style(rankPos+1)
				}
				rankPrefix += l.Ranks[rankIndex].Suffix
				oldRankPrefix := regexRank.FindAllString(nickname, 1)
				if len(oldRankPrefix) == 0 {
					nickname = rankPrefix + " " + nickname
				} else {
					nickname = strings.Replace(nickname, oldRankPrefix[0], rankPrefix, 1)
				}
				nickname = regexSpaces.ReplaceAllString(nickname, " ")
				if nickname != oldNick {
					log.Trace("Nickname change: " + oldNick + " > " + nickname)
					_, _ = Discord.GuildMemberEdit(l.GuildID, player, &discordgo.GuildMemberParams{
						Nick: nickname,
					})
					l.Players[player].Nickname = nickname
				}
				Discord.GuildMemberRoleAdd(l.GuildID, player, l.Ranks[rankIndex].RoleID)
			}
		}
	}
}

func (l *Leaderboard) ApplyRank(duelee, newRank string, position int, fixRanks bool, skipCancel bool) interface{} {
	l.Lock()
	defer l.Unlock()

	defer func(fR bool) {
		if fR {
			l.FixRanks() //Fix the rank order for player-limited ranks once done
		}
	}(fixRanks)

	//Remove the player's rank first
	rankIndex, rankPos := l.GetRank(duelee)
	if rankIndex > -1 && rankPos > 0 {
		playerIndex := rankPos-1
		l.Ranks[rankIndex].Players = append(l.Ranks[rankIndex].Players[:playerIndex], l.Ranks[rankIndex].Players[playerIndex+1:]...)
		if l.Ranks[rankIndex].RoleID != "" {
			Discord.GuildMemberRoleRemove(l.GuildID, duelee, l.Ranks[rankIndex].RoleID)
		}
	}

	//Cancel any active duels
	if !skipCancel {
		l.DuelCancel(duelee)
	}
	
	for rankIndex, rank := range l.Ranks {
		if rank.Rank == newRank {
			if position < 1 {
				position = len(rank.Players)+1
			}

			Discord.GuildMemberRoleAdd(l.GuildID, duelee, l.Ranks[rankIndex].RoleID)

			if position > 0 && position <= len(rank.Players) {
				position-- //The user may specify position 1, but we start from position 0
				if len(rank.Players) == position {
					l.Ranks[rankIndex].Players = append(rank.Players, duelee)
				} else {
					newPlayers := append(rank.Players[:position+1], rank.Players[position:]...)
					newPlayers[position] = duelee
					l.Ranks[rankIndex].Players = newPlayers
				}
				log.Info(fmt.Sprintf("Ranked %s at position %d of rank %s", mention(duelee, l.GuildID, false), position+1, newRank))
				return embed.NewGenericEmbed("Ranker", "Ranked %s at %s%d.", mention(duelee, l.GuildID, false), newRank, position+1)
			} else {
				l.Ranks[rankIndex].Players = append(rank.Players, duelee)
				log.Info("Ranked %s at bottom of rank %s", mention(duelee, l.GuildID, false), newRank)
				if rank.PlayerLimit > 0 {
					return embed.NewGenericEmbed("Ranker", "Ranked %s at the bottom of rank %s.", mention(duelee, l.GuildID, false), newRank)
				}
				return embed.NewGenericEmbed("Ranker", "Ranked %s at rank %s.", mention(duelee, l.GuildID, false), newRank)
			}
		}
	}

	nickname := regexRank.ReplaceAllString(mention(duelee, l.GuildID, true), "")
	l.Players[duelee].Nickname = nickname
	_, _ = Discord.GuildMemberEdit(l.GuildID, duelee, &discordgo.GuildMemberParams{
		Nick: nickname,
	})
	return embed.NewErrorEmbed("Ranker", "%s is now unranked.", mention(duelee, l.GuildID, false))
}

func (l *Leaderboard) TrackDuel(duelIndex int) {
	if duelIndex < 0 || duelIndex >= len(l.ActiveDuels) {
		return
	}

	duel := l.ActiveDuels[duelIndex]
	waitDuration := duel.Expires.Sub(time.Now())
	time.AfterFunc(waitDuration, func() {
		//Make sure the duel still exists by time it expires
		_, active := l.GetActiveDuel(duel.Dueler)
		if active == nil {
			return
		}

		for _, player := range duel.Players {
			if player == duel.Dueler {
				continue
			}

			_ = l.DuelForfeit(player, false)
			break
		}

		duelEmbed := duel.Embed(l.GuildID).(*embed.Embed)
		duelEmbed.SetTitle("Duel Expired")
		duelEmbed.SetDescription(duel.Title(l.GuildID))
		Discord.ChannelMessageSendComplex(l.ChannelReminders, &discordgo.MessageSend{
			Embed: duelEmbed.MessageEmbed,
		})
	})

	//Prepare notices for additional reminders
	if waitDuration.Hours() > 72 {
		remindDay3 := duel.Expires.Add(time.Hour * -72)
		remindDuration := remindDay3.Sub(time.Now())
		time.AfterFunc(remindDuration, func() {
			duelEmbed := duel.Embed(l.GuildID).(*embed.Embed)
			duelEmbed.SetTitle("3 Day Duel Warning")
			duelEmbed.SetDescription(duel.Title(l.GuildID))
			//send DM to duelers
			for _, player := range duel.Players {
				privChannel, err := Discord.UserChannelCreate(player)
				if err != nil {
					log.Trace(err)
				} else {
					_, err := Discord.ChannelMessageSendComplex(privChannel.ID, &discordgo.MessageSend{
						Embed: duelEmbed.MessageEmbed,
					})
					if err != nil {
						log.Error(err)
						_, err = Discord.ChannelMessageSendComplex(l.ChannelReminders, &discordgo.MessageSend{
							Content: duel.TitleWithMentions(),
							Embed: duelEmbed.MessageEmbed,
						})
						if err != nil {
							log.Error(err)
						}
						return
					}
				}
			}
		})
	}
	if waitDuration.Hours() > 24 {
		remindDay1 := duel.Expires.Add(time.Hour * -24)
		remindDuration := remindDay1.Sub(time.Now())
		time.AfterFunc(remindDuration, func() {
			duelEmbed := duel.Embed(l.GuildID).(*embed.Embed)
			duelEmbed.SetTitle("1 Day Duel Warning")
			duelEmbed.SetDescription(duel.Title(l.GuildID))
			//send DM to duelers
			for _, player := range duel.Players {
				privChannel, err := Discord.UserChannelCreate(player)
				if err != nil {
					log.Trace(err)
				} else {
					_, err := Discord.ChannelMessageSendComplex(privChannel.ID, &discordgo.MessageSend{
						Embed: duelEmbed.MessageEmbed,
					})
					if err != nil {
						log.Error(err)
						_, err = Discord.ChannelMessageSendComplex(l.ChannelReminders, &discordgo.MessageSend{
							Content: duel.TitleWithMentions(),
							Embed: duelEmbed.MessageEmbed,
						})
						if err != nil {
							log.Error(err)
						}
						return
					}
				}
			}
		})
	}
}

func (l *Leaderboard) NewDuel(dueler string, duelees []string, roundLimit, scoreLimit int, force, unranked bool) interface{} {
	l.Lock()
	defer l.Unlock()

	l.InitPlayer(dueler)

	duelerRankIndex, duelerRankPos := l.GetRank(dueler)
	if duelerRankIndex == -1 || duelerRankPos == 0 {
		if !force {
			return embed.NewErrorEmbed("Duel", "Dueler %s is not ranked yet!", mention(dueler, l.GuildID, false))
		}
	}

	//Prevent players from participating in multiple duels
	if _, active := l.GetActiveDuel(dueler); active != nil {
		return embed.NewErrorEmbed("Duel", "Dueler is already in an active duel with other players.")
	}

	for _, duelee := range duelees {
		if duelee == dueler {
			return embed.NewErrorEmbed("Duel", "Dueler may not self-duel!")
		}
		l.InitPlayer(duelee)
		if _, active := l.GetActiveDuel(duelee); active != nil {
			return embed.NewErrorEmbed("Duel", "%s is already participating in another duel.", mention(duelee, l.GuildID, false))
		}
		if !l.Players[duelee].ImmuneUntil.IsZero() {
			if time.Now().Before(l.Players[duelee].ImmuneUntil) {
				if !force {
					return embed.NewErrorEmbed("Duel", "Duelee %s is currently in a 24 hour grace period and cannot be dueled yet.", mention(duelee, l.GuildID, false))
				}
			} else {
				l.Players[duelee].ImmuneUntil = time.Time{}
			}
		}

		dueleeRankIndex, dueleeRankPos := l.GetRank(duelee)
		if dueleeRankIndex == -1 || dueleeRankPos == 0 {
			if !force {
				return embed.NewErrorEmbed("Duel", "%s is not ranked yet!", mention(duelee, l.GuildID, false))
			}
		}

		if !force {
			if l.Ranks[dueleeRankIndex].PlayerLimit > 0 || l.Ranks[duelerRankIndex].PlayerLimit > 0 { //ensure a player is rank-restricted before checking
				if dueleeRankIndex < duelerRankIndex { //ex: X:0 is less than S:1
					if (duelerRankIndex-dueleeRankIndex) > 1 { //ex: A+:2 - X:0 = 2
						return embed.NewErrorEmbed("Duel", "%s is rank %s%d, %d ranks above %s at rank %s%d.", mention(duelee, l.GuildID, false), l.Ranks[dueleeRankIndex].Rank, dueleeRankPos, duelerRankIndex-dueleeRankIndex, mention(dueler, l.GuildID, false), l.Ranks[duelerRankIndex].Rank, duelerRankPos)
					}
					if !l.Ranks[dueleeRankIndex].IgnoreRankPos {
						//ex: S:1-X:0=1 or X:0-X:0=0, code continues
						if duelerRankPos > 1 { //ex: S2
							return embed.NewErrorEmbed("Duel", "%s is rank %s%d but must be rank %s1 in order to duel %s at rank %s%d.", mention(dueler, l.GuildID, false), l.Ranks[duelerRankIndex].Rank, duelerRankPos, l.Ranks[duelerRankIndex].Rank, mention(duelee, l.GuildID, false), l.Ranks[dueleeRankIndex].Rank, dueleeRankPos)
						}
						if dueleeRankPos != len(l.Ranks[dueleeRankIndex].Players) { //if X10 exists and duelee is X9
							return embed.NewErrorEmbed("Duel", "%s is rank %s%d, more than one position into the rank above %s at rank %s%d.", mention(duelee, l.GuildID, false), l.Ranks[dueleeRankIndex].Rank, dueleeRankPos, mention(dueler, l.GuildID, false), l.Ranks[duelerRankIndex].Rank, duelerRankPos)
						}
					}
				}
				if dueleeRankIndex == duelerRankIndex { //ex: X == X
					if dueleeRankPos < duelerRankPos { //ex: X1 < X2
						if (duelerRankPos-dueleeRankPos) > (dueleeRankIndex+1) { //ex: (X3 - X1 = 2) > (X:0 + 1 = 1)
							return embed.NewErrorEmbed("Duel", "%s is rank %s%d, %d position(s) above %s at rank %s%d.", mention(duelee, l.GuildID, false), l.Ranks[dueleeRankIndex].Rank, dueleeRankPos, mention(dueler, l.GuildID, false), l.Ranks[duelerRankIndex].Rank, duelerRankPos)
						}
						//ex: (X3 - X2 = 1) =< (X:0 + 1 = 1) && //ex: (S3 - S1 = 2) =< (S:1 + 1 = 2), code continues
					}
				}
			}
		}
	}

	duelees = append([]string{dueler}, duelees...)
	duel := NewDuel(l.GuildID, dueler, duelees, roundLimit, scoreLimit, unranked)
	l.ActiveDuels = append(l.ActiveDuels, duel) //Add the new duel to the list of active duels
	l.TrackDuel(len(l.ActiveDuels)-1) //Track the new duel

	duelEmbed := duel.Embed(l.GuildID).(*embed.Embed)

	//send DM to duelers
	for _, player := range duel.Players {
		privChannel, err := Discord.UserChannelCreate(player)
		if err != nil {
			log.Trace(err)
		} else {
			_, err := Discord.ChannelMessageSendComplex(privChannel.ID, &discordgo.MessageSend{
				Embed: duelEmbed.MessageEmbed,
			})
			if err != nil {
				log.Error(err)
				_, err = Discord.ChannelMessageSendComplex(l.ChannelReminders, &discordgo.MessageSend{
					Content: duel.TitleWithMentions(),
					Embed: duelEmbed.MessageEmbed,
				})
				if err != nil {
					log.Error(err)
				}
				break
			}
		}
	}

	return duelEmbed
}

func (l *Leaderboard) DuelForfeit(duelee string, forceDropRank bool) interface{} {
	rI, rP := l.GetRank(duelee)
	if rI < 0 || rP <= 0 {
		return embed.NewErrorEmbed("Duel", "Duelee %s has no rank and cannot possibly forfeit a duel.")
	}

	l.Players[duelee].MatchesForfeit++

	activeIndex, _ := l.GetActiveDuel(duelee)

	//Drop rankings for letting the duel forfeit
	if activeIndex >= 0 || forceDropRank {
		dropCount := l.Players[duelee].MatchesForfeit
		if l.Ranks[rI].PlayerLimit > 0 {
			if (rP+dropCount) > l.Ranks[rI].PlayerLimit {
				dropCount -= (l.Ranks[rI].PlayerLimit - rP)
				rI++
				rP = dropCount
			} else {
				rP += dropCount
			}
		} else {
			rI++
			rP = 1
		}
		l.ApplyRank(duelee, l.Ranks[rI].Rank, rP, true, true)
	}

	if activeIndex >= 0 {
		l.ActiveDuels[activeIndex].Forfeit(duelee)

		return embed.NewGenericEmbed("Duel", "Forfeit duel.")
	}
	return embed.NewErrorEmbed("Duel", "No duels available for duelee to forfeit.")
}

func (l *Leaderboard) DuelWin(duelee string) interface{} {
	if activeIndex, _ := l.GetActiveDuel(duelee); activeIndex >= 0 {
		l.ActiveDuels[activeIndex].ForceWin(duelee)
		return embed.NewGenericEmbed("Duel", "Forced duelee to win their duel.")
	}
	return embed.NewErrorEmbed("Duel", "No duels available for duelee to forcibly win.")
}

func (l *Leaderboard) DuelCancel(duelee string) interface{} {
	if activeIndex, _ := l.GetActiveDuel(duelee); activeIndex >= 0 {
		l.ActiveDuels = append(l.ActiveDuels[:activeIndex], l.ActiveDuels[activeIndex+1:]...)
		return embed.NewGenericEmbed("Duel", "Cancelled duel.")
	}
	return embed.NewErrorEmbed("Duel", "No duels available for duelee to cancel.")
}

func (l *Leaderboard) DuelExtend(duelee, duration string) interface{} {
	dur, err := ParseDuration(duration, false)
	if err != nil {
		return embed.NewErrorEmbed("Duel", "Unknown duration %s, unable to extend duel timer.", duration)
	}
	if duelIndex, _ := l.GetActiveDuel(duelee); duelIndex >= 0 {
		l.ActiveDuels[duelIndex].Expires = l.ActiveDuels[duelIndex].Expires.Add(dur)
		return l.ActiveDuels[duelIndex].Embed(l.GuildID)
	}
	return embed.NewErrorEmbed("Duel", "Unable to find an active duel to extend for %s.", mention(duelee, l.GuildID, false))
}
func (l *Leaderboard) DuelShorten(duelee, duration string) interface{} {
	dur, err := ParseDuration(duration, true)
	if err != nil {
		return embed.NewErrorEmbed("Duel", "Unknown duration %s, unable to shorten duel timer.", duration)
	}
	if duelIndex, _ := l.GetActiveDuel(duelee); duelIndex >= 0 {
		l.ActiveDuels[duelIndex].Expires = l.ActiveDuels[duelIndex].Expires.Add(dur)
		return l.ActiveDuels[duelIndex].Embed(l.GuildID)
	}
	return embed.NewErrorEmbed("Duel", "Unable to find an active duel to shorten for %s.", mention(duelee, l.GuildID, false))
}

func (l *Leaderboard) DuelEnd(duel *Duel) {
	if activeIndex, _ := l.GetActiveDuel(duel.Dueler); activeIndex >= 0 {
		for spectator, spec := range l.Spectators {
			for _, player := range duel.Players {
				if spec == player {
					delete(l.Spectators, spectator)
					break
				}
			}
		}

		winners := ""
		for _, winner := range duel.Winners {
			if winners != "" {
				winners += ", "
			}
			winners += fmt.Sprintf("**%d** - %s", duel.DuelStats[winner].FinalScore, mention(winner, l.GuildID, false))
			l.Players[winner].MatchesWon++
			l.Players[winner].ImmuneUntil = time.Now().AddDate(0, 0, 1)
		}

		losers := ""
		for _, loser := range duel.Losers {
			if losers != "" {
				losers += ", "
			}
			losers += fmt.Sprintf("**%d** - %s", duel.DuelStats[loser].FinalScore, mention(loser, l.GuildID, false))
			l.Players[loser].MatchesLost++
		}

		duelEmbed := duel.Embed(l.GuildID).(*embed.Embed)
		if !duel.Unranked {
			_, err := Discord.ChannelMessageSendComplex(l.ChannelResults, &discordgo.MessageSend{
				Embed: duelEmbed.MessageEmbed,
			})
			log.Trace(err)
		}

		//Apply leaderboard progression for the winner
		if !duel.Unranked {
			if len(duel.Winners) == 1 {
				wRI, wRP := l.GetRank(duel.Winners[0])
				for _, loser := range duel.Losers {
					lRI, lRP := l.GetRank(loser)
					if lRI > wRI {
						continue
					}
					if lRI == wRI && lRP > wRP {
						continue
					}
					wRI = lRI
					wRP = lRP
				}
				l.ApplyRank(duel.Winners[0], l.Ranks[wRI].Rank, wRP, true, true)
			}
		}

		l.ArchiveDuel(activeIndex)
	}
}

func (l *Leaderboard) ArchiveDuel(duelIndex int) {
	if duelIndex < 0 || duelIndex >= len(l.ActiveDuels) {
		return
	}
	l.DuelHistory = append(l.DuelHistory, l.ActiveDuels[duelIndex])
	l.ActiveDuels = append(l.ActiveDuels[:duelIndex], l.ActiveDuels[duelIndex+1:]...)
}
func (l *Leaderboard) GetActiveDuel(duelee string) (int, *Duel) {
	for activeIndex, active := range l.ActiveDuels {
		for _, activePlayer := range active.Players {
			if activePlayer == duelee {
				l.ActiveDuels[activeIndex].Init()
				return activeIndex, active
			}
		}
	}
	return -1, nil
}

func (l *Leaderboard) InitPlayer(duelee string) {
	if l.Players == nil {
		l.Players = make(map[string]*Player) //Initialize an empty player map if there is none yet
	}
	if _, ok := l.Players[duelee]; !ok {
		l.Players[duelee] = &Player{}
	}
}
func (l *Leaderboard) DeletePlayer(duelee string) {
	if memberNick := mention(duelee, l.GuildID, false); memberNick != "" { //Make sure this player has been used for the bot before
		//Forfeit any existing duel
		_, duel := leaderboards[l.GuildID].GetActiveDuel(duelee)
		if duel != nil {
			duel.Forfeit(duelee)
		}

		//Derank them to sync leaderboard updates
		_ = leaderboards[l.GuildID].ApplyRank(duelee, "unranked", 0, false, true)

		//Remove their player stats
		delete(leaderboards[l.GuildID].Players, duelee)

		//Tell the world
		goodbyeEmbed := embed.NewGenericEmbed("%s has left the server, losing their rank and player stats. Leaderboards have been synced.", memberNick)
		Discord.ChannelMessageSendComplex(l.ChannelReminders, &discordgo.MessageSend{
			Embed: goodbyeEmbed,
		})
	}
}

func (l *Leaderboard) EmbedStats(guildID, requesterID, playerID string) interface{} {
	l.InitPlayer(playerID)
	player := l.Players[playerID]

	duelEmbed := embed.NewEmbed().
		SetTitle("My Duel").
		SetDescription("You are not participating in a duel right now.").
		SetColor(config.ColorMain)

	if requesterID != playerID {
		duelEmbed.SetTitle("Duel Stats")
		duelEmbed.SetDescription(mention(playerID, l.GuildID, true) + " is not participating in a duel right now.")
	}

	if _, active := l.GetActiveDuel(playerID); active != nil {
		duelEmbed = active.Embed(l.GuildID).(*embed.Embed)
	}

	rankIndex, rankPos := l.GetRank(playerID)
	if rankIndex > -1 {
		rank := l.Ranks[rankIndex]
		rankID := rank.Rank
		if rank.PlayerLimit > 0 {
			rankID = fmt.Sprintf("%s%d", rankID, rankPos)
		}
		duelEmbed.SetFooter(fmt.Sprintf("%s | %d wins | %d losses", rankID, player.MatchesWon, player.MatchesLost))
	} else {
		duelEmbed.SetFooter("Unranked")
	}

	return duelEmbed
}
func (l *Leaderboard) EmbedDuels(guildID string) interface{} {
	duelsEmbed := embed.NewEmbed().
		SetTitle("Active Duels for " + guildName(guildID)).
		SetColor(config.ColorMain)
	for _, active := range l.ActiveDuels {
		duelsEmbed.Fields = append(duelsEmbed.Fields, active.EmbedField(guildID))
	}
	if len(duelsEmbed.Fields) == 0 {
		duelsEmbed.SetDescription("No duels are active right now!")
	}
	return duelsEmbed
}
func (l *Leaderboard) EmbedRanks(guildID, displayRank string) interface{} {
	ranksEmbed := embed.NewEmbed().
		SetTitle("Leaderboards for " + guildName(guildID)).
		SetFooter("If you see invalid users, scroll to the bottom of the user list and/or change channels, then come back here.").
		SetColor(config.ColorMain)

	rankExists := false //Determines if "Rank doesn't exist!" should instead be "No rankings for X available yet!"
	for _, rank := range l.Ranks {
		if rank.Rank == displayRank {
			rankExists = true
		}
		if displayRank == "" {
			if rank.PlayerLimit <= 0 || len(rank.Players) == 0 {
				continue
			}
		} else {
			if displayRank != "all" && displayRank != rank.Rank {
				continue
			}
		}
		rankData := ""
		for playerIndex, player := range rank.Players {
			if rankData != "" {
				rankData += "\n"
			}
			if rank.PlayerLimit > 0 {
				rankData += fmt.Sprintf("**%s**: <@%s>", rank.Style(playerIndex+1), player)
			} else {
				rankData += fmt.Sprintf("<@%s>", player)
			}
		}
		if rankData != "" {
			ranksEmbed.AddField(rank.Prefix + rank.Rank + rank.Suffix, rankData)
		}
	}
	if len(ranksEmbed.Fields) == 0 {
		if rankExists {
			ranksEmbed.SetDescription("No rankings for " + displayRank + " available yet!")
		} else if displayRank == "all" || displayRank == "" {
			ranksEmbed.SetDescription("No rankings available yet!")
		} else {
			ranksEmbed.SetDescription("Rank doesn't exist!")
		}
	}
	log.Trace(ranksEmbed)
	return ranksEmbed.InlineAllFields()
}
