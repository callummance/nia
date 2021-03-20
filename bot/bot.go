package bot

import (
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/callummance/nia/db"
	"github.com/callummance/nia/discord"
	"github.com/callummance/nia/twitch"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
)

//NiaBot represents an instance of the discord bot, containing handles to the various external connections.
type NiaBot struct {
	DiscordConnection *discord.EventSource
	DBConnection      *db.DBConnection
	TwitchConnection  *twitch.EventSource
}

//Init creates a new NiaBot instance
func Init() (*NiaBot, error) {
	var res NiaBot
	//Start database connection
	db, err := db.Init()
	if err != nil {
		logrus.Errorf("Cannot start bot due to error initializing database connection: %v", err)
		return nil, err
	}

	//Start discord connection
	disc, err := discord.StartDiscordListener(&res)
	if err != nil {
		logrus.Errorf("Cannot start bot due to error initializing discord connection: %v", err)
		return nil, err
	}

	//Try to start twitch connection
	uids, err := db.GetAllTwitchUIDs()
	if err != nil {
		logrus.Errorf("Failed to initialize twitch listener due to error %v. Continuing without twitch functionality.", err)
	} else {
		twitch, err := twitch.StartTwitchListener(&res, uids)
		if err != nil {
			logrus.Errorf("Failed to initialize twitch listener due to error %v. Continuing without twitch functionality.", err)
		} else {
			res.TwitchConnection = twitch
		}
	}

	res.DiscordConnection = disc
	res.DBConnection = db

	return &res, nil
}

//BotAddURL generates a URL that can be used to add the bot to a server
func (b *NiaBot) BotAddURL() (*url.URL, error) {
	return b.DiscordConnection.BotAddURL()
}

//DiscordSession returns a handle to the underlying discord session
func (b *NiaBot) DiscordSession() *discordgo.Session {
	return b.DiscordConnection.Session()
}

//Close cleanly terminates the bot instance
func (b *NiaBot) Close() {
	log.Info("Terminating bot...")
	b.DiscordConnection.Close()
	b.DBConnection.Close()
}
