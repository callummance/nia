package bot

//import (
//	"strings"
//
//	"github.com/bwmarrin/discordgo"
//	"github.com/sirupsen/logrus"
//)
//
//func (b *NiaBot) HandleRegisterTwitchCommand(msg *discordgo.Message) BotResult {
//	var result BotResult
//	argString := strings.TrimPrefix(msg.Content, "!registertwitch")
//	twitchRef := strings.TrimLeft(argString, " ")
//
//	//Respond
//	result.WriteToLog()
//	resp := result.DiscordMessage()
//	msgRef := discordgo.MessageReference{
//		MessageID: msg.ID,
//		ChannelID: msg.ChannelID,
//		GuildID:   msg.GuildID,
//	}
//	_, err := b.DiscordSession().ChannelMessageSendReply(msg.ChannelID, resp, &msgRef)
//	if err != nil {
//		logrus.Errorf("Failed to send response to command due to error %v", err)
//	}
//}
//
