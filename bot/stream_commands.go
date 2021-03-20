package bot

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/callummance/nia/guildmodels"
	"github.com/callummance/nia/twitch"
	"github.com/sirupsen/logrus"
)

const handleRegisterTwitchSyntax string = "```" +
	`!registertwitch "<twitch>"
	<twitch> can be a twitch username or channel URL` +
	"```"

var broadcasterURLRegex = regexp.MustCompile(`!registertwitch\s+"?(?:(?:https?://)?(?:(?:www|go|m)\.)?twitch\.tv/)?(?P<username>[a-zA-Z0-9_]{4,25})"?`)

//HandleRegisterTwitchCommandMessage takes a message from any server member and registers a twitch channel for them
func (b *NiaBot) HandleRegisterTwitchCommandMessage(msg *discordgo.MessageCreate) {
	result := b.registerTwitch(msg.Message)
	result.WriteToLog()
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err := b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

func (b *NiaBot) registerTwitch(msg *discordgo.Message) NiaResponse {
	commandName := "!registertwitch"
	t, errResp := b.getTwitchClient(commandName, msg.Content)
	if errResp != nil {
		return *errResp
	}
	matches := broadcasterURLRegex.FindStringSubmatch(msg.Content)
	unameIdx := broadcasterURLRegex.SubexpIndex("username")
	if matches == nil || len(matches) <= unameIdx {
		//no match
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: "I couldn't understand that",
			syntax:      handleRegisterTwitchSyntax,
			timestamp:   time.Now(),
		}
	}
	username := matches[unameIdx]
	//Check username is valid
	broadcaster, err := t.GetBroadcasterDeets(username)
	if err != nil {
		//Could not find anyone with that username
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: fmt.Sprintf("I couldn't find any user with the username %v", username),
			syntax:      handleRegisterTwitchSyntax,
			timestamp:   time.Now(),
		}
	}
	//We have a valid broadcaster, so save it to the database and register a subscription
	connection := guildmodels.TwitchConnectionData{
		TwitchUID: broadcaster.ID,
	}
	_, err = b.DBConnection.SetTwitchConnectionData(msg.GuildID, msg.Author.ID, connection)
	if err != nil {
		//DB error of some kind
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: "Encountered internal database error whilst saving twitch connection details",
			data:        map[string]string{"Error": err.Error()},
			timestamp:   time.Now(),
		}
	}
	t.SubscribeToUID(broadcaster.ID)
	return NiaResponseSuccess{
		command:    commandName,
		commandMsg: msg.Content,
		timestamp:  time.Now(),
	}
}

func (b *NiaBot) getTwitchClient(command, msgContent string) (*twitch.EventSource, *NiaResponseFeatureNotEnabled) {
	if b.TwitchConnection == nil {
		return nil, &NiaResponseFeatureNotEnabled{
			command:         command,
			commandMsg:      msgContent,
			disabledFeature: "twitch_integration",
			timestamp:       time.Now(),
		}
	}
	return b.TwitchConnection, nil
}
