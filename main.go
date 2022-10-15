package main

import (
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/JoshuaDoes/json"
	"github.com/JoshuaDoes/logger"
)

type Config struct {
	Verbosity int `json:"verbosity"` //How verbose logs should be
	Token string `json:"token"` //Bot token to login with
	OwnerIDs []string `json:"ownerIDs"` //Discord user IDs for accessing debug commands
	Ranks []*Rank `json:"ranks"` //The ranks to track

	ChannelResults string `json:"channelResults"` //TODO: Move to leaderboards with /channel to specify //The channelID to post duel results to
	ColorMain int `json:"colorMain"` //Main color of embeds
}

var (
	log *logger.Logger
	Discord *DiscordClient
	config *Config
)

var (
	leaderboards map[string]*Leaderboard
)

func init() {
	if err := loadJSON(&config, "config.json"); err != nil {
		log.Error(err)
		panic(err)
	}

	log = logger.NewLogger("SF", config.Verbosity)

	if err := loadJSON(&leaderboards, "states/leaderboards.json"); err != nil {
		log.Error(err)
		log.Warn("If the leaderboards state exists, back it up before exiting!")
	}
	if leaderboards == nil {
		leaderboards = make(map[string]*Leaderboard)
	}
}

func main() {
	log.Debug("... Initializing Discord session")
	discordSession, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Fatal(err)
	}
	Discord = &DiscordClient{discordSession, nil}

	log.Info("... Registering Discord event handlers")
	Discord.AddHandler(discordReady)
	Discord.AddHandler(discordGuildMemberRemove)
	Discord.AddHandler(discordInteractionCreate)

	log.Info("... Connecting to Discord")
	err = Discord.Open()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Connected to Discord!")
	defer Discord.Shutdown()

	//Make a channel to listen to OS signals on
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT) //Request notifications for SIGINT signals
	signal.Notify(sc, syscall.SIGKILL) //Request notifications for SIGKILL signals
	//watchdogTicker := time.Tick(watchdogDelay) //TODO: Spawn separate process and watch over it

	//Loop endlessly so we don't exit until required to
	for {
		select {
		//Check for one of our registered signals from the OS
		case sig, ok := <-sc:
			if ok {
				log.Trace("Received signal: ", sig)
				return
			}
		}
	}
}

func saveJSON(data interface{}, path string) error {
	dataJSON, err := json.Marshal(data, true)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, dataJSON, 0644)
	return err
}
func loadJSON(data interface{}, path string) error {
	dataJSON, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dataJSON, data)
	return err
}

//intDiff returns the difference between two ints, always >= 0
func intDiff(int1, int2 int) int {

	intDiff := int1-int2
	if intDiff < 0 {
		intDiff *= -1
	}
	return intDiff
}
