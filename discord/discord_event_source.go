package discord

import (
	"fmt"
	"net/url"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const discordTokenEnvVar = "NIA_DISCORD_BOT_TOKEN"
const botScope = "bot"
const permissions = discordgo.PermissionAllText | discordgo.PermissionAllChannel

//EventHandler is a struct which can handle all the events the discord listener generates.
type EventHandler interface {
	HandleMessage(*discordgo.MessageCreate)
}

//EventSource represents a connection to the Discord gateway
type EventSource struct {
	discordClient *discordgo.Session
	handler       EventHandler
}

//StartDiscordListener initializes an EventSource and starts listening for events from the discord gateway
func StartDiscordListener(handler EventHandler) (*EventSource, error) {
	//Get token from environment variable
	apiTok, exists := os.LookupEnv(discordTokenEnvVar)
	if !exists {
		logrus.Errorf("`%v` env variable was not set.", discordTokenEnvVar)
		return nil, fmt.Errorf("`%v` env variable was not set", discordTokenEnvVar)
	}

	//Create new client
	dc, err := discordgo.New("Bot " + apiTok)
	if err != nil {
		logrus.Warnf("Failed to create Discord gateway client due to %v", err)
		return nil, err
	}
	dispatch := EventSource{
		discordClient: dc,
		handler:       handler,
	}

	//Register event handlers
	dc.AddHandler(dispatch.dispatchMessageCreateEvent)

	//Register intents
	dc.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	//Open a websocket connection
	err = dc.Open()
	if err != nil {
		logrus.Errorf("Failed to connect to discord websockets gateway; encountered error %v", err)
		return nil, err
	}
	return &dispatch, nil
}

//BotAddURL generates a URL that can be used to add the bot to a server
func (d *EventSource) BotAddURL() (*url.URL, error) {
	user, err := d.discordClient.User("@me")
	if err != nil {
		return nil, err
	}
	clientID := user.ID

	url, err := url.Parse("https://discord.com/api/oauth2/authorize")
	if err != nil {
		return nil, err
	}
	q := url.Query()
	q.Set("client_id", clientID)
	q.Set("scope", botScope)
	q.Set("permissions", fmt.Sprintf("%d", permissions))
	url.RawQuery = q.Encode()

	return url, nil
}

//Close cleanly terminates the Discord connection
func (d *EventSource) Close() {
	logrus.Info("Terminating discord event listener...")
	_ = d.discordClient.Close()
}

//Session returns a handle to the underlying discordgo session
func (d *EventSource) Session() *discordgo.Session {
	return d.discordClient
}

func (d *EventSource) dispatchMessageCreateEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Ignore messages created by bot
	if m.Author.ID == s.State.User.ID {
		logrus.Debug("Got a message from self; Ignoring.")
		return
	}

	//Prevent panic from crashing the whole bot
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Bot handler thread panicked: %v", r)
		}
	}()

	//Dispatch to bot handlers
	d.handler.HandleMessage(m)

	//For debugging
	fmt.Printf("Got message `%v`\n", m.Content)
}
