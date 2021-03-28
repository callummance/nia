package bot

import (
	"fmt"
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
	commandName := "!addadminrole"
	var result NiaResponse
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		errorTxt := fmt.Sprintf("Failed to check if message came from admin due to error %v", err)
		result = NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if !isFromAdmin {
		errorTxt := "The !addadminrole command can only be run by admins (including the server owner)."
		result = NiaResponseNotAllowed{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else {
		//Interpret and run the command
		argString := strings.TrimPrefix(msg.Content, "!addadminrole")
		argString = strings.TrimLeft(argString, " ")
		matchingRole, err := b.interpretRoleString(argString, msg.GuildID)
		if err != nil {
			result = NiaResponseInternalError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: fmt.Sprintf("Something unexpected went wrong whilst trying to read %v as a role", argString),
				data:        map[string]string{"Error": err.Error()},
				timestamp:   time.Now(),
			}
		} else if matchingRole == nil {
			result = NiaResponseSyntaxError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: fmt.Sprintf("%v does not seem to be a valid role", argString),
				syntax:      handleAddAdminRoleSyntax,
				timestamp:   time.Now(),
			}
		} else {
			result = b.addAdminRole(msg.GuildID, matchingRole.ID, msg.Content)
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err = b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

func (b *NiaBot) addAdminRole(gid string, roleID string, msgContent string) NiaResponse {
	commandName := "!addadminrole"
	//Make sure guild exists
	_, err := b.DBConnection.GetOrCreateGuild(gid)
	if err != nil {
		errorTxt := fmt.Sprintf("Encountered error %v when trying to add role %v to admins on server %v", err, roleID, gid)
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msgContent,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	}
	//Add role to list
	noUpdated, err := b.DBConnection.AddAdminRole(gid, roleID)
	if err != nil {
		errorTxt := fmt.Sprintf("Encountered database error %v when trying to add role %v to admins on server %v", err, roleID, gid)
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msgContent,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if noUpdated == 0 {
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msgContent,
			description: fmt.Sprintf("Role %v is already set as an admin", roleID),
			syntax:      handleAddAdminRoleSyntax,
			timestamp:   time.Now(),
		}
	}
	return NiaResponseSuccess{
		command:    commandName,
		commandMsg: msgContent,
		timestamp:  time.Now(),
	}
}

const handleAddManagedRoleSyntax string = "```" +
	`!addmanagedrole "<role>" <method> [options]
Options depend on the role assignment method selected as follows:

	!addmanagedrole "<role>" reaction <post> <emoji> [flags] 
		Allows assignment of roles based on users reacting with any chosen reaction to the provided post.

		<role> may be the role name enclosed in double quotation marks or an @mention.
		<post> may be a message link (recommended) or ID of the post (Right click -> copy ID if in developer mode) and channel in the format <channel_id>:<post_id>.
		<emoji> should be an emoji.
		[flags] can be any number of optional flags from the following: 
			"clearafter": Remove reaction after assigningthe role
			"initialreact": Bot should create an initial reaction
			"noremove": Bot should not remove role if reaction is removed
			
	!addmanagedrole "<role>" nowstreaming
		Assigns a role to users for as long as their linked twitch account is live ` +
	"```"

var regexHandleAddManagedRoleMessage = regexp.MustCompile(`^\s*((?:"?<\@\&\d*\>"?)|(?:\"[^"]*\")|(?:\w*))\s*(reaction|nowstreaming)\s*(.*)$`)

//HandleAddManagedRoleMessage handles a message starting with the !addmanagedrole command
//syntax: !addmanagedrole "<role>" <type> [typeopts]
func (b *NiaBot) HandleAddManagedRoleMessage(msg *discordgo.MessageCreate) {
	commandName := "!addmanagedrole"
	var result NiaResponse
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		errorTxt := fmt.Sprintf("Failed to check if message came from admin due to error %v", err)
		result = NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if !isFromAdmin {
		errorTxt := "The !addmanagedrole command can only be run by admins."
		result = NiaResponseNotAllowed{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else {
		//Interpret and run the command
		argString := strings.TrimPrefix(msg.Content, "!addmanagedrole")
		argString = strings.TrimLeft(argString, " ")
		matches := regexHandleAddManagedRoleMessage.FindStringSubmatch(argString)
		if matches == nil {
			result = NiaResponseSyntaxError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: fmt.Sprintf("*%v* doesn't seem to be the correct syntax for an !addmanagedrole command", argString),
				syntax:      handleAddManagedRoleSyntax,
				timestamp:   time.Now(),
			}
		} else {
			role, err := b.interpretRoleString(matches[1], msg.GuildID)
			if err != nil {
				result = NiaResponseInternalError{
					command:     commandName,
					commandMsg:  msg.Content,
					description: fmt.Sprintf("Something unexpected went wrong whilst trying to read %v as a role", matches[1]),
					data:        map[string]string{"Error": err.Error()},
					timestamp:   time.Now(),
				}
			} else if role == nil {
				result = NiaResponseSyntaxError{
					command:     commandName,
					commandMsg:  msg.Content,
					description: fmt.Sprintf("%v does not seem to be a valid role", argString),
					syntax:      handleAddAdminRoleSyntax,
					timestamp:   time.Now(),
				}
			}
			opts := matches[3]
			switch matches[2] {
			case "reaction":
				result = b.handleAddReactionManagedRoleMessage(role.ID, opts, msg)
			case "nowstreaming":
				result = b.handleAddNowStreamingManagedRoleMessage(role.ID, msg.Message)
			}
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err = b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

var addReactionManagedRoleOptsRegex = regexp.MustCompile(`^\s*((?:https://discord\.com/channels/\d+/\d{18}/(?:\d{18}))|(?:\d{18}):(?:\d{18}))\s*((?:<a?:(?:[^:]+):(?:\d+)>)|(?:\S{1,4}))\s*((?:(?:clearafter|initialreact|noremove)\s*)*)\s*$`)

//syntax: !addmamangedrole "<role>" reaction <post> <emoji> [flags]
func (b *NiaBot) handleAddReactionManagedRoleMessage(roleID string, opts string, msg *discordgo.MessageCreate) NiaResponse {
	commandName := "!addmanagedrole"
	matches := addReactionManagedRoleOptsRegex.FindStringSubmatch(opts)
	if matches == nil {
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: fmt.Sprintf("%v doesn't seem to be the correct syntax for adding a reaction-based managed role", msg.Content),
			syntax:      handleAddManagedRoleSyntax,
			timestamp:   time.Now(),
		}
	}
	message := matches[1]
	emote := matches[2]
	flags := strings.Split(matches[3], " ")

	chanID, msgID := b.interpretMessageRef(message)
	if chanID == nil || msgID == nil {
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: fmt.Sprintf("I couldn't work out which message you were referring to with %v", message),
			syntax:      handleAddManagedRoleSyntax,
			timestamp:   time.Now(),
		}
	}
	emoteID := b.interpretEmoji(emote)
	if emoteID == nil {
		return NiaResponseSyntaxError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: fmt.Sprintf("%v doesn't seem to be a valid emote...", emote),
			syntax:      handleAddManagedRoleSyntax,
			timestamp:   time.Now(),
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
		errorTxt := fmt.Sprintf("Encountered error %v when trying to add role %v to managed roles on server %v", err, roleID, msg.GuildID)
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	}
	return NiaResponseSuccess{
		command:    commandName,
		commandMsg: msg.Content,
		timestamp:  time.Now(),
	}
}

func (b *NiaBot) handleAddNowStreamingManagedRoleMessage(roleID string, msg *discordgo.Message) NiaResponse {
	commandName := "!addmanagedrole"
	roleAssStruct := guildmodels.RoleAssignment{
		AssignmentType: "nowlive",
	}
	rule := guildmodels.ManagedRoleRule{
		RoleID:         roleID,
		GuildID:        msg.GuildID,
		RoleAssignment: roleAssStruct,
	}
	err := b.DBConnection.AddManagedRoleRule(rule)
	if err != nil {
		errorTxt := fmt.Sprintf("Encountered error %v when trying to add role %v to managed roles on server %v", err, roleID, msg.GuildID)
		return NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	}
	return NiaResponseSuccess{
		command:    commandName,
		commandMsg: msg.Content,
		timestamp:  time.Now(),
	}
}

//HandleInitReactionsMessage handles a message containing an add initial reactions command
//command format: !initreactions
func (b *NiaBot) HandleInitReactionsMessage(msg *discordgo.MessageCreate) {
	commandName := "!initreactions"
	var result NiaResponse
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		errorTxt := fmt.Sprintf("Failed to check if message came from admin due to error %v", err)
		result = NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if !isFromAdmin {
		errorTxt := "The !initreactions command can only be run by admins."
		result = NiaResponseNotAllowed{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else {
		//Run the command
		relevantRoles, err := b.DBConnection.GetGuildRolesWithInitialReact(msg.GuildID)
		if err != nil {
			result = NiaResponseInternalError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: "Failed to fetch relevant roles from database",
				data:        map[string]string{"Error": err.Error()},
				timestamp:   time.Now(),
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
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err = b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

const handlePurgeRoleSyntax string = "```" +
	`!purgerole "<role>"` +
	"```"

//HandlePurgeRoleMessage handles a message containing a purge role command
//command format: !purgerole <role>
func (b *NiaBot) HandlePurgeRoleMessage(msg *discordgo.MessageCreate) {
	commandName := "!purgerole"
	var result NiaResponse
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		errorTxt := fmt.Sprintf("Failed to check if message came from admin due to error %v", err)
		result = NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if !isFromAdmin {
		errorTxt := "The !purgerole command can only be run by admins."
		result = NiaResponseNotAllowed{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else {
		argString := strings.TrimPrefix(msg.Content, "!purgerole")
		argString = strings.TrimLeft(argString, " ")
		matchingRole, err := b.interpretRoleString(argString, msg.GuildID)
		if err != nil {
			result = NiaResponseInternalError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: "Couldn't read provided role",
				data:        map[string]string{"Error": err.Error()},
				timestamp:   time.Now(),
			}
		} else if matchingRole == nil {
			result = NiaResponseSyntaxError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: fmt.Sprintf("I couldn't find a role for %v", argString),
				syntax:      handlePurgeRoleSyntax,
				timestamp:   time.Now(),
			}
		} else {
			isManaged, err := b.DBConnection.IsManagedRole(msg.GuildID, matchingRole.ID)
			if err != nil {
				result = NiaResponseInternalError{
					command:     commandName,
					commandMsg:  msg.Content,
					description: "Failed to look up that role in the database",
					data:        map[string]string{"Error": err.Error()},
					timestamp:   time.Now(),
				}
			} else if !isManaged {
				result = NiaResponseSyntaxError{
					command:     commandName,
					commandMsg:  msg.Content,
					description: fmt.Sprintf("Role %v is not managed by this bot", argString),
					syntax:      handlePurgeRoleSyntax,
					timestamp:   time.Now(),
				}
			} else {
				problemMembers, problemRules, err := b.doRolePurge(msg.Message, matchingRole)
				if err != nil {
					result = NiaResponseInternalError{
						command:     commandName,
						commandMsg:  msg.Content,
						description: "Failed to purge role",
						data:        map[string]string{"Error": err.Error()},
						timestamp:   time.Now(),
					}
				} else if problemMembers == nil && problemRules == nil {
					result = NiaResponseSuccess{
						command:    commandName,
						commandMsg: msg.Content,
						timestamp:  time.Now(),
					}
				} else {
					data := make(map[string]string, 2)
					if problemMembers != nil || len(problemMembers) == 0 {
						causesMap := make(map[string][]string)
						for _, issue := range problemMembers {
							causesMap[issue.err.Error()] = append(causesMap[issue.err.Error()], issue.member.Nick)
						}
						failedMembersString := ""
						for issue, members := range causesMap {
							failedMembersString += fmt.Sprintf("Failed to remove role from members %v due to error %v", strings.Join(members, ", "), issue)
						}
						data["Failed to remove role from members"] = failedMembersString
					}
					if problemRules != nil || len(problemRules) == 0 {
						failedRulesString := ""
						for _, issue := range problemRules {
							failedRulesString += fmt.Sprintf("Failed to undo role assignment rule %#v due to error %v", issue.rule, issue.err)
						}
						data["Failed to undo rules"] = failedRulesString
					}
					result = NiaResponsePartialSuccess{
						command:     commandName,
						commandMsg:  msg.Content,
						description: "Purge role command completed, but with some errors",
						data:        data,
						timestamp:   time.Now(),
					}
				}
			}
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err = b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
	if err != nil {
		logrus.Errorf("Failed to send response to command due to error %v", err)
	}
}

type failedRoleRemoval struct {
	member *discordgo.Member
	err    error
}

type failedRoleRuleReset struct {
	rule *guildmodels.ManagedRoleRule
	err  error
}

//Returns a list of members whose role could not be removed
func (b *NiaBot) doRolePurge(msg *discordgo.Message, role *discordgo.Role) ([]failedRoleRemoval, []failedRoleRuleReset, error) {
	//Get list of members with that role
	var relevantMembers []*discordgo.Member
	for member := range b.DiscordConnection.GuildMembersIter(msg.GuildID) {
		if member.Error != nil {
			return nil, nil, member.Error
		} else if member.Member != nil {
			userRoles := member.Member.Roles
			for _, roleID := range userRoles {
				//If user has the role
				if roleID == role.ID {
					relevantMembers = append(relevantMembers, member.Member)
					break
				}
			}
		}
	}
	//Remove role from each member
	var errs []failedRoleRemoval
	for _, member := range relevantMembers {
		err := b.DiscordSession().GuildMemberRoleRemove(msg.GuildID, member.User.ID, role.ID)
		if err != nil {
			errs = append(errs, failedRoleRemoval{
				member: member,
				err:    err,
			})
			logrus.Infof("Failed to remove role %v from user %v becuase %v", role, member, err)
		}
	}
	//Get list of associated role assignments
	rules, err := b.DBConnection.GetRoleRules(msg.GuildID, role.ID)
	if err != nil {
		logrus.Warnf("Failed to lookup rules to be undone for role %v due to error %v.", role, err)
		return errs, nil, err
	}
	//Undo each of those role assignments
	var failedRuleResets []failedRoleRuleReset
	for _, rule := range rules {
		logrus.Debugf("Undoing rule %v", rule)
		err := b.undoRoleRule(&rule.RoleAssignment)
		if err != nil {
			failedRuleResets = append(failedRuleResets, failedRoleRuleReset{
				rule: &rule,
				err:  err,
			})
		}
	}
	return errs, failedRuleResets, nil
}

const handleSetNotificationChannelSyntax = "```" +
	`!setnotificationchannel <notification_type> <channel>
	currently the only supported <notification_type> is 'twitch'
	<channel> can either be the name of a channel or a link to the channel (eg. #channel)` +
	"```"

var setnotificationchannelRegex = regexp.MustCompile(`(twitch)\s*("?(?:<#(?:\d+)>)|#?(?:[\w_-]+)"?\s*)`)

//HandleSetNotificationChannel handles a message from an admin setting a certain channel as the target for
//alert messages
func (b *NiaBot) HandleSetNotificationChannel(msg *discordgo.MessageCreate) {
	commandName := "!setnotificationchannel"
	var result NiaResponse
	//Check sender is admin
	isFromAdmin, err := b.isFromAdmin(msg.Member, msg.Author, msg.GuildID)
	if err != nil {
		errorTxt := fmt.Sprintf("Failed to check if message came from admin due to error %v", err)
		result = NiaResponseInternalError{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else if !isFromAdmin {
		errorTxt := "The !setnotificationchannel command can only be run by admins."
		result = NiaResponseNotAllowed{
			command:     commandName,
			commandMsg:  msg.Content,
			description: errorTxt,
			timestamp:   time.Now(),
		}
	} else {
		argString := strings.TrimPrefix(msg.Content, "!setnotificationchannel")
		argString = strings.TrimLeft(argString, " ")
		matches := regexHandleAddManagedRoleMessage.FindStringSubmatch(argString)
		if matches == nil {
			result = NiaResponseSyntaxError{
				command:     commandName,
				commandMsg:  msg.Content,
				description: fmt.Sprintf("*%v* doesn't seem to be the correct syntax for an !setnotificationchannel command", argString),
				syntax:      handleAddManagedRoleSyntax,
				timestamp:   time.Now(),
			}
		} else {
			ch, err := b.interpretChannelString(matches[2], msg.GuildID)
			if err != nil {
				result = NiaResponseInternalError{
					command:     commandName,
					commandMsg:  msg.Content,
					description: fmt.Sprintf("Something unexpected went wrong whilst trying to read %v as a channel", matches[2]),
					data:        map[string]string{"Error": err.Error()},
					timestamp:   time.Now(),
				}
			} else {
				switch matches[2] {
				case "twitch":
					channels := guildmodels.NotificationChannels{
						StreamNotificationsChannel: &ch.ID,
					}
					err := b.DBConnection.UpdateGuildNotificationChannels(msg.GuildID, channels)
					if err != nil {
						result = NiaResponseInternalError{
							command:     commandName,
							commandMsg:  msg.Content,
							description: "Something unexpected went wrong whilst trying to write update to database",
							data:        map[string]string{"Error": err.Error()},
							timestamp:   time.Now(),
						}
					}
				}
			}
		}
	}
	//Respond
	result.WriteToLog()
	resp := result.DiscordResponse()
	msgRef := discordgo.MessageReference{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   msg.GuildID,
	}
	resp.Reference = &msgRef
	_, err = b.DiscordSession().ChannelMessageSendComplex(msg.ChannelID, resp)
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
