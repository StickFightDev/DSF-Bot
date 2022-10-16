package main

import (
	"regexp"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"github.com/Philipp15b/go-steamapi"
	"github.com/microcosm-cc/bluemonday"
)

var (
	steamUsernames = make(map[uint64]string)
	stripTags *bluemonday.Policy
)

func init() {
	stripTags = bluemonday.StrictPolicy()
}

//CSteamID holds a Steam client ID and its username
type CSteamID struct {
	ID           uint64
	Username     string
	NormUsername string
}

//NewCSteamID returns a new Steam client ID
func NewCSteamID(steamID uint64) CSteamID {
	clientID := CSteamID{
		ID: steamID,
	}

	return clientID
}

//GetUsername returns the username of the CSteamID and caches it in memory
func (cSteamID CSteamID) GetUsername() string {
	if cSteamID.Username != "" {
		return cSteamID.Username
	}

	if steamUsername, ok := steamUsernames[cSteamID.ID]; ok {
		cSteamID.Username = steamUsername
		return steamUsername
	}

	summaries, err := steamapi.GetPlayerSummaries([]uint64{cSteamID.ID}, "8FAF40F156C0D7DAF869385A3FF4EE1C")
	if err != nil {
		return ""
	}

	if len(summaries) == 0 {
		return ""
	}

	steamUsernames[cSteamID.ID] = summaries[0].PersonaName
	cSteamID.Username = steamUsernames[cSteamID.ID]
	return cSteamID.Username
}

//GetNormalizedUsername returns a normalized version of the username of the CSteamID and caches it in memory
func (cSteamID CSteamID) GetNormalizedUsername() string {
	if cSteamID.NormUsername != "" {
		return cSteamID.NormUsername
	}

	username := cSteamID.GetUsername()
	username = regexp.MustCompile(`<.*?>`).ReplaceAllString(username, "")
	username = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(username, "")
	username = stripTags.Sanitize(username)

	bytes := make([]byte, len(username))
	normalize := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	_, _, err := normalize.Transform(bytes, []byte(username), true)
	if err != nil {
		return username
	}

	username = string(bytes)
	cSteamID.NormUsername = username
	return cSteamID.NormUsername
}

//CompareCSteamID evaluates if a CSteamID is the same as another
func (cSteamID CSteamID) CompareCSteamID(compareSteamID CSteamID) bool {
	return cSteamID.ID == compareSteamID.ID
}

//CompareSteamID evaluates if a CSteamID matches a SteamID
func (cSteamID CSteamID) CompareSteamID(steamID uint64) bool {
	return cSteamID.ID == steamID
}
