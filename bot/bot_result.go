package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const (
	successMessageColour int = 0x28bd00
	warnMessageColour    int = 0xbdb900
	errorMessageColour   int = 0xbd1b00
)

//NiaResponse represents the result of a command which can be both communicated over discord and written to the log.
type NiaResponse interface {
	DiscordResponse() *discordgo.MessageSend
	WriteToLog()
}

//NiaResponseSuccess will be returned when a command has been successfully completed
type NiaResponseSuccess struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//The time the success was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponseSuccess) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("Completed %v command successfully!", r.command)
	embed := discordgo.MessageEmbed{
		Title:       "Success! \\o/",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       successMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponseSuccess) WriteToLog() {
	logrus.Infof("%v Completed command %v successfully.", logLineLabel(r.timestamp), r.commandMsg)
}

//NiaResponsePartialSuccess will be returned when a command has executed but with issues
type NiaResponsePartialSuccess struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//A human-readable description of the issue
	description string
	//A map containing fields which should be included in the embed
	data map[string]string
	//The time the success was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponsePartialSuccess) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("Completed %v command but with errors: \n%v", r.command, r.description)
	embed := discordgo.MessageEmbed{
		Title:       "Partial success...",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       warnMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
		Fields: stringMapToFields(r.data),
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponsePartialSuccess) WriteToLog() {
	logrus.Infof("%v Completed command %v but with errors: %v.", logLineLabel(r.timestamp), r.commandMsg, r.data)
}

//NiaResponseSyntaxError will be returned when there was an issue with the user's input
type NiaResponseSyntaxError struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//A human-readable description of the issue
	description string
	//A description of the correct syntax
	syntax string
	//The time the error was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponseSyntaxError) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("Sorry, but there was a problem with the data you supplied for the %v command: \n%v", r.command, r.description)
	fields := map[string]string{
		"Your command":   r.commandMsg,
		"Correct syntax": r.syntax,
	}
	embed := discordgo.MessageEmbed{
		Title:       "Uh-oh, there was something wrong with that command",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       errorMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
		Fields: stringMapToFields(fields),
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponseSyntaxError) WriteToLog() {
	logrus.Infof("%v Syntax error in command %v: %v", logLineLabel(r.timestamp), r.commandMsg, r.description)
}

//NiaResponseInternalError will be returned when there was some kind of error within the bot or when communicating with
//APIs
type NiaResponseInternalError struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//A human-readable description of the issue
	description string
	//A map containing fields which should be included in the embed
	data map[string]string
	//The time the error was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponseInternalError) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("Oops! I encountered an unexpected error whilst running your %v command. Please try again later or file a bug report.", r.command)
	dataWithDescription := r.data
	dataWithDescription["Error"] = r.description
	embed := discordgo.MessageEmbed{
		Title:       "Oops, something went wrong ;w;",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       errorMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
		Fields: stringMapToFields(dataWithDescription),
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponseInternalError) WriteToLog() {
	logrus.Infof("%v Internal error in whilst executing command %v: %v | data: %v", logLineLabel(r.timestamp), r.commandMsg, r.description, r.data)
}

//NiaResponseNotAllowed will be returned when a user tried to run a command that they do not have the correct role for
type NiaResponseNotAllowed struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//A human-readable description of the issue
	description string
	//The time the error was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponseNotAllowed) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("I'm sorry Dave, I can't let you do that...")
	fields := map[string]string{
		"Reason":  r.description,
		"Command": r.commandMsg,
	}
	embed := discordgo.MessageEmbed{
		Title:       "That's illegal m8",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       errorMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
		Fields: stringMapToFields(fields),
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponseNotAllowed) WriteToLog() {
	logrus.Infof("%v Rejected command `%v` as the sender did not have the correct priveliges | description: %v", logLineLabel(r.timestamp), r.commandMsg, r.description)
}

//NiaResponseFeatureNotEnabled will be returned when a user tried to run a command which requires an unloaded module
type NiaResponseFeatureNotEnabled struct {
	//The base command name
	command string
	//The entire text contents of the message
	commandMsg string
	//The name of the feature which was disabled
	disabledFeature string
	//The time the error was logged at
	timestamp time.Time
}

//DiscordResponse builds a MessageSend object which can be sent back to whoever sent a command message.
func (r NiaResponseFeatureNotEnabled) DiscordResponse() *discordgo.MessageSend {
	description := fmt.Sprintf("Sorry, buy the '%v' command requires a feature which is not currently running.", r.command)
	fields := map[string]string{
		"Required Feature(s)": r.disabledFeature,
	}
	embed := discordgo.MessageEmbed{
		Title:       "Required feature is not activated",
		Type:        discordgo.EmbedTypeRich,
		Description: description,
		Timestamp:   r.timestamp.Format(time.RFC3339),
		Color:       errorMessageColour,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Log ID: %d", r.timestamp.UnixNano()),
		},
		Fields: stringMapToFields(fields),
	}
	msg := discordgo.MessageSend{
		Embed: &embed,
		TTS:   false,
		Files: []*discordgo.File{},
	}
	return &msg
}

//WriteToLog dumps data on a discord command response to the log
func (r NiaResponseFeatureNotEnabled) WriteToLog() {
	logrus.Infof("%v Rejected command `%v` as required feature %v is not loaded", logLineLabel(r.timestamp), r.commandMsg, r.disabledFeature)
}

/////////////////////
//Utility Functions//
/////////////////////
func writeLogRef(t time.Time) string {
	return fmt.Sprintf("More details can be found on log line %v", t.UnixNano())
}

func logLineLabel(t time.Time) string {
	return fmt.Sprintf("#%v# | ", t.UnixNano())
}

func stringMapToFields(fields map[string]string) []*discordgo.MessageEmbedField {
	var res []*discordgo.MessageEmbedField
	for fieldName, content := range fields {
		field := discordgo.MessageEmbedField{
			Name:   fieldName,
			Value:  content,
			Inline: false,
		}
		res = append(res, &field)
	}
	return res
}
