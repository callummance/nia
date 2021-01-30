package bot

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const discordDevUIDEnvVar string = "NIA_DISCORD_DEV_UID"

const handleAddAdminRoleSyntax string = "`!addadminrole \"<role>\"` or `!addadminrole @<role>"

var regexHandleAddAdminMessage = regexp.MustCompile(`^\s*(<\@\&(\d*)\>)|(\"[^"]*\")|(\w*)\s*$`)

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
		matches := regexHandleAddAdminMessage.FindStringSubmatch(argString)
		switch {
		case matches[1] != "":
			//We have a role id directly
			rid := matches[2]
			result = b.addAdminRole(msg.GuildID, rid)
		case matches[3] != "":
			//We have a role name
			roleName := matches[4]
			result = b.addNamedAdminRole(msg.GuildID, roleName)
		case matches[5] != "":
			//We have a role name without quotation marks
			roleName := matches[4]
			result = b.addNamedAdminRole(msg.GuildID, roleName)
		default:
			//Nothing was provided
			result = SyntaxError{
				args:      argString,
				syntax:    handleAddAdminRoleSyntax,
				timeStamp: time.Now(),
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

func (b *NiaBot) addNamedAdminRole(gid string, roleName string) BotResult {
	guildRoles, err := b.DiscordSession().GuildRoles(gid)
	if err != nil {
		logrus.Warnf("Failed to get list of roles for guild %v due to error %v", gid, err)
		return InternalError{
			err:       err,
			timeStamp: time.Now(),
		}
	}
	for _, role := range guildRoles {
		if role.Name == roleName {
			return b.addAdminRole(gid, role.ID)
		}
	}
	return RoleNotFound{
		roleName:  roleName,
		timeStamp: time.Now(),
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

//syntax: !addmanagedrole "<role>" <type> [typeopts]
func (b *NiaBot) HandleAddManagedRoleMessage(msg *discordgo.MessageCreate) {

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
