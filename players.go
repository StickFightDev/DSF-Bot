package main

import (
	"time"
)

type Player struct {
	Nickname       string    `json:"nickname"`
	SteamID        CSteamID  `json:"steamID"`
	MatchesWon     int64     `json:"matchesWon"`
	MatchesLost    int64     `json:"matchesLost"`
	MatchesForfeit int       `json:"matchesForfeit"` //How many matches have been forfeit/expired in a row, reset on win/loss
	Rank           string    `json:"rank"`
	RankPos        int       `json:"rankPos"`
	ImmuneUntil    time.Time `json:"immuneUntil"`
}