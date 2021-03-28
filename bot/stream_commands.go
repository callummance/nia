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
	oldStream, newStream, err := b.DBConnection.SetTwitchConnectionData(msg.GuildID, msg.Author.ID, broadcaster.ID)
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
	//If there is an oldStream, we need to do some more cleaning up
	if oldStream != nil && oldStream.TwitchUID != newStream.TwitchUID {
		//Check if there are any others in the guild linked to the same stream
		linkedMembers, err := b.DBConnection.GetMemberByConnection(guildmodels.MemberConnections{TwitchConnection: oldStream}, &msg.GuildID, nil)
		if err != nil {
			logrus.Errorf("Failed to look up remaining members linked to twitch stream ID %v in guild %v due to error %v", oldStream.TwitchUID, msg.GuildID, err)
		} else {
			if linkedMembers != nil && len(linkedMembers) >= 0 {
				//There are other members in the guild with the same stream linked, so no need to remove anything else
				logrus.Debugf("No need to remove any posts as there still exists at least one linked member in the same guild")
			} else {
				postsToRemove := make([]guildmodels.MessageRef, 0)
				for _, post := range oldStream.DiscordStatusPosts {
					if post.GuildID == msg.GuildID {
						postsToRemove = append(postsToRemove, post)
					}
				}
				//Remove alert posts as that user was the only one in the guild with that channel linked
				b.removeAlertPosts(postsToRemove)
			}
		}
		//Remove now streaming roles from user if their new stream is not also streaming
		if !newStream.IsLive {
			b.unassignLiveRoles(msg.Author.ID, msg.GuildID)
		}
		//If there are no other members with the same stream linked, we should remove it from the DB and unsubscribe from twitch alerts
		globalLinkedMembers, err := b.DBConnection.GetMemberByConnection(guildmodels.MemberConnections{TwitchConnection: oldStream}, nil, nil)
		if err != nil {
			logrus.Errorf("Failed to look up remaining members linked to twitch stream ID %v in globally due to error %v", oldStream.TwitchUID, err)
		} else {
			if globalLinkedMembers == nil || len(globalLinkedMembers) == 0 {
				//Unsubscribe from eventsub notifications
				err := t.UnsubscribeFromStream(oldStream.TwitchUID)
				if err != nil {
					logrus.Errorf("Failed to unsubscribe from twitch alerts for stream uid %v due to error %v", oldStream.TwitchUID, err)
				}
				//Delete twitch stream from DB
				err = b.DBConnection.DeleteTwitchStream(oldStream.TwitchUID)
				if err != nil {
					logrus.Errorf("Failed to remove twitch uid %v from DB due to error %v", oldStream.TwitchUID, err)
				}
			}
		}
	}
	err = t.SubscribeToStream(newStream.TwitchUID)
	if err != nil {
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: "Encountered error whilst subscribing to twitch updates. Please try again later or contact a developer.",
			data:        map[string]string{"Error": err.Error()},
			timestamp:   time.Now(),
		}
	}
	if !newStream.IsLive {
		//update newly connected stream
		err := t.ForceStreamUpdate(newStream.TwitchUID)
		if err != nil {
			return NiaResponsePartialSuccess{
				command:     commandName,
				commandMsg:  msg.Content,
				description: "Failed to fetch current state of the provided stream. Alerts and roles should still be applied the next time you start streaming.",
				data:        map[string]string{"Error": err.Error()},
				timestamp:   time.Now(),
			}
		}
	} else {
		//assign roles and make post as needed
		err := b.SetUserStreaming(newStream.TwitchUID, msg.Author.ID, msg.GuildID)
		if err != nil {
			return NiaResponsePartialSuccess{
				command:     commandName,
				commandMsg:  msg.Content,
				description: "Failed to set your role and send alert. Alerts and roles should still be applied the next time you start streaming.",
				data:        map[string]string{"Error": err.Error()},
				timestamp:   time.Now(),
			}
		}
	}
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
