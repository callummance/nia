package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

//HandleMessage is called upon every recieved message. It checks if the message is a command, and executes it.
func (b *NiaBot) HandleMessage(msg *discordgo.MessageCreate) {
	if msg.Content[0] == '!' {
		//We have a command
		words := strings.SplitN(msg.Content, " ", 2)
		command := strings.TrimLeft(words[0], "!")
		switch command {
		case "addadminrole":
			b.HandleAddAdminMessage(msg)
		case "addmanagedrole":
			b.HandleAddManagedRoleMessage(msg)
		}

	}
}
