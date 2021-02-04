package bot

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
)

const discordDevUIDEnvVar string = "NIA_DISCORD_DEV_UID"

const handleAddAdminRoleSyntax string = "`!addadminrole \"<role>\"` or `!addadminrole @<role>"

//HandleAddAdminMessage handles a message containing an add admin role command
//command format: !addadminrole <role>
func (b *NiaBot) HandleAddAdminMessage(msg *discordgo.MessageCreate) {
	var result BotResult
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		logrus.Warnf("Failed to check if message came from admin due to error %v", err)
		result = InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	} else if !isFromAdmin {
		result = CommandNeedsAdmin{
			command:   "!addadminrole",
			timeStamp: time.Now(),
		}
	} else {
		//Interpret and run the command
		argString := strings.TrimPrefix(msg.Content, "!addadminrole")
		argString = strings.TrimLeft(argString, " ")
		matchingRole, err := b.interpretRoleString(argString, msg.GuildID)
		if err != nil {
			result = InternalError{
				err:       err,
				timeStamp: time.Now(),
			}
		} else if matchingRole == nil {
			result = RoleNotFound{
				roleName:  argString,
				timeStamp: time.Now(),
			}
		} else {
			result = b.addAdminRole(msg.GuildID, matchingRole.ID)
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordMessage()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	_, err = b.DiscordSession().ChannelMessageSendReply(msg.ChannelID, resp, &msgRef)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

func (b *NiaBot) addAdminRole(gid string, roleID string) BotResult {
	//Make sure guild exists
	_, err := b.DBConnection.GetOrCreateGuild(gid)
	if err != nil {
		logrus.Warnf("Encountered error %v when trying to add role %v to admins on server %v", err, roleID, gid)
		return InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	}
	//Add role to list
	noUpdated, err := b.DBConnection.AddAdminRole(gid, roleID)
	if err != nil {
		logrus.Warnf("Encountered error %v when trying to add role %v to admins on server %v", err, roleID, gid)
		return InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	} else if noUpdated == 0 {
		return AdminRoleAlreadyExists{
			roleID:    roleID,
			timeStamp: time.Now(),
		}
	}
	return AdminRoleAdded{
		timeStamp: time.Now(),
	}
}

const handleAddManagedRoleSyntax string = "```" +
	`!addmanagedrole "<role>" <method> [options]
Options depend on the role assignment method selected as follows:

	!addmanagedrole "<role>" reaction <post> <emoji> [flags] 

		<role> may be the role name enclosed in double quotation marks or an @mention.
		<post> may be a message link (recommended) or ID of the post (Right click -> copy ID if in developer mode) and channel in the format <channel_id>:<post_id>.
		<emoji> should be an emoji.
		[flags] can be any number of optional flags from the following: 
			"clearafter": Remove reaction after assigningthe role
			"initialreact": Bot should create an initial reaction
			"noremove": Bot should not remove role if reaction is removed` +
	"```"

var regexHandleAddManagedRoleMessage = regexp.MustCompile(`^\s*((?:"?<\@\&\d*\>"?)|(?:\"[^"]*\")|(?:\w*))\s*(reaction)\s*(.*)$`)

//HandleAddManagedRoleMessage handles a message starting with the !addmanagedrole command
//syntax: !addmanagedrole "<role>" <type> [typeopts]
func (b *NiaBot) HandleAddManagedRoleMessage(msg *discordgo.MessageCreate) {
	var result BotResult
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		logrus.Warnf("Failed to check if message came from admin due to error %v", err)
		result = InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	} else if !isFromAdmin {
		result = CommandNeedsAdmin{
			command:   "!addmanagedrole",
			timeStamp: time.Now(),
		}
	} else {
		//Interpret and run the command
		argString := strings.TrimPrefix(msg.Content, "!addmanagedrole")
		argString = strings.TrimLeft(argString, " ")
		matches := regexHandleAddManagedRoleMessage.FindStringSubmatch(argString)
		if matches == nil {
			result = SyntaxError{
				args:      argString,
				syntax:    handleAddManagedRoleSyntax,
				timeStamp: time.Now(),
			}
		} else {
			role, err := b.interpretRoleString(matches[1], msg.GuildID)
			if err != nil {
				result = InternalError{
					err:       err,
					timeStamp: time.Now(),
				}
			}
			opts := matches[3]
			switch matches[2] {
			case "reaction":
				result = b.handleAddReactionManagedRoleMessage(role.ID, opts, msg)
			}
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordMessage()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	_, err = b.DiscordSession().ChannelMessageSendReply(msg.ChannelID, resp, &msgRef)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}

}

var addReactionManagedRoleOptsRegex = regexp.MustCompile(`^\s*((?:https://discord\.com/channels/\d+/\d{18}/(?:\d{18}))|(?:\d{18}):(?:\d{18}))\s*((?:<a?:(?:[^:]+):(?:\d+)>)|(?:\S{1,4}))\s*((?:clearafter|initialreact|noremove)\s*)\s*$`)

//syntax: !addmamangedrole "<role>" reaction <post> <emoji> [flags]
func (b *NiaBot) handleAddReactionManagedRoleMessage(roleID string, opts string, msg *discordgo.MessageCreate) BotResult {
	matches := addReactionManagedRoleOptsRegex.FindStringSubmatch(opts)
	if matches == nil {
		return SyntaxError{
			args:      msg.Content,
			syntax:    handleAddManagedRoleSyntax,
			timeStamp: time.Now(),
		}
	}
	message := matches[1]
	emote := matches[2]
	flags := strings.Split(matches[3], " ")

	chanID, msgID := b.interpretMessageRef(message)
	if chanID == nil || msgID == nil {
		return InvalidMessageRef{
			ref:       message,
			timeStamp: time.Now(),
		}
	}
	emoteID := b.interpretEmoji(emote)
	if emoteID == nil {
		return InvalidEmote{
			emote:     emote,
			timeStamp: time.Now(),
		}
	}

	var shouldClear bool
	var initialReact bool
	var noRemove bool
	for _, flag := range flags {
		switch flag {
		case "clearafter":
			shouldClear = true
		case "initialreact":
			initialReact = true
			//Add reaction
			err := b.DiscordSession().MessageReactionAdd(*chanID, *msgID, *emoteID)
			if err != nil {
				logrus.Error("Failed to add initial emote %v to message %v due to error %v", emoteID, msgID, err)
			}
		case "noremove":
			noRemove = true
		case "":
			continue
		default:
			logrus.Warnf("Got unexpected flag for reaction role assignment add: %v", flag)
		}
	}

	reactRoleAssignStruct := guildmodels.ReactionRoleAssign{
		MsgID:                *msgID,
		ChanID:               *chanID,
		EmojiID:              *emoteID,
		ShouldClear:          shouldClear,
		BotShouldReact:       initialReact,
		DisallowRoleRemoveal: noRemove,
	}

	roleAssignmentStruct := guildmodels.RoleAssignment{
		AssignmentType:   "reaction",
		ReactionRoleData: &reactRoleAssignStruct,
	}

	rule := guildmodels.ManagedRoleRule{
		RoleID:         roleID,
		GuildID:        msg.GuildID,
		RoleAssignment: roleAssignmentStruct,
	}

	err := b.DBConnection.AddManagedRoleRule(rule)
	if err != nil {
		logrus.Warnf("Encountered error %v when trying to add role %v to managed roles on server %v", err, roleID, msg.GuildID)
		return InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	}
	return ManagedRoleAdded{
		timeStamp: time.Now(),
	}
}

//HandleInitReactionsMessage handles a message containing an add initial reactions command
//command format: !initreactions
func (b *NiaBot) HandleInitReactionsMessage(msg *discordgo.MessageCreate) {
	var result BotResult
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		logrus.Warnf("Failed to check if message came from admin due to error %v", err)
		result = InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	} else if !isFromAdmin {
		result = CommandNeedsAdmin{
			command:   "!addadminrole",
			timeStamp: time.Now(),
		}
	} else {
		//Run the command
		relevantRoles, err := b.DBConnection.GetGuildRolesWithInitialReact(msg.GuildID)
		if err != nil {
			result = InternalError{
				err:       err,
				timeStamp: time.Now(),
			}
		} else {
			for _, role := range relevantRoles {
				chanID := role.RoleAssignment.ReactionRoleData.ChanID
				msgID := role.RoleAssignment.ReactionRoleData.MsgID
				emoteID := role.RoleAssignment.ReactionRoleData.EmojiID
				err := b.DiscordSession().MessageReactionAdd(chanID, msgID, emoteID)
				if err != nil {
					logrus.Error("Failed to add initial emote %v to message %v due to error %v", emoteID, msgID, err)
				}
			}
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordMessage()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	_, err = b.DiscordSession().ChannelMessageSendReply(msg.ChannelID, resp, &msgRef)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

/**************************
/     Utility Functions
/**************************/

func (b *NiaBot) isFromAdmin(member *discordgo.Member, user *discordgo.User, guildID string) (bool, error) {
	//Works if from dev
	if isDev(user.ID) {
		return true, nil
	}
	//Works if from server owner
	guild, err := b.DiscordSession().Guild(guildID)
	if err != nil {
		logrus.Warnf("Failed to fetch guild object from Discord API when checking if user %v is admin for server %v", user.ID, guildID)
		return false, err
	} else if guild.OwnerID == user.ID {
		return true, nil
	}
	//Works if user has an admin role
	localGuild, err := b.DBConnection.GetOrCreateGuild(guildID)
	if err != nil {
		logrus.Warnf("Failed to fetch guild object from Database when checking if user %v is admin for server %v", user.ID, guildID)
		return false, err
	}
	for _, adminRole := range localGuild.AdminRoles {
		for _, senderRole := range member.Roles {
			if adminRole == senderRole {
				return true, nil
			}
		}
	}
	return false, nil
}

func isDev(userID string) bool {
	devUID, exists := os.LookupEnv(discordDevUIDEnvVar)
	if !exists {
		return false
	}
	return userID == devUID
}
