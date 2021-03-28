package bot

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
)

//Allows @mentions, double quotation marked roles or roles only made up from letters
var roleRegex = regexp.MustCompile(`^\s*("?<\@\&(\d*)\>"?)|(\"[^"]*\")|(\w*)\s*$`)

func (b *NiaBot) interpretRoleString(roleStr string, guildID string) (*discordgo.Role, error) {
	guildRoles, err := b.DiscordSession().GuildRoles(guildID)
	if err != nil {
		logrus.Warnf("Failed to fetch guild roles for guild id %v", guildID)
		return nil, err
	}
	matches := roleRegex.FindStringSubmatch(roleStr)

	switch {
	case matches == nil:
		return nil, fmt.Errorf("empty role identifier was provided")
	case matches[1] != "":
		//We have a role id directly
		rid := matches[2]
		for _, guildRole := range guildRoles {
			if guildRole.ID == rid {
				return guildRole, nil
			}
		}
		return nil, nil
	case matches[3] != "":
		//We have a role name
		roleName := matches[4]
		for _, guildRole := range guildRoles {
			if guildRole.Name == roleName {
				return guildRole, nil
			}
		}
		return nil, nil
	case matches[5] != "":
		//We have a role name without quotation marks
		roleName := matches[4]
		for _, guildRole := range guildRoles {
			if guildRole.Name == roleName {
				return guildRole, nil
			}
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("%v was not a valid role string format", roleStr)
	}
}

var chanRegex = regexp.MustCompile(`^\s*"?(?:<#(?P<ch_id>\d+)>)|#?(?P<ch_name>[\w_-]+)"?\s*$`)

func (b *NiaBot) interpretChannelString(chanStr string, guildID string) (*discordgo.Channel, error) {
	matches := chanRegex.FindStringSubmatch(chanStr)
	switch {
	case matches == nil:
		return nil, fmt.Errorf("unrecognized channel identifier was provided")
	case matches[chanRegex.SubexpIndex("ch_id")] != "":
		//We have a channel link
		chID := matches[chanRegex.SubexpIndex("ch_id")]
		ch, err := b.DiscordSession().Channel(chID)
		if err != nil {
			logrus.Warnf("Failed to fetch data for channel %v whilst interpreting channel specifier %v due to error %v", chanStr, chanStr, err)
			return nil, err
		}
		return ch, nil
	case matches[chanRegex.SubexpIndex("ch_name")] != "":
		//We have a channel name
		chName := matches[chanRegex.SubexpIndex("ch_name")]
		guildChannels, err := b.DiscordSession().GuildChannels(guildID)
		if err != nil {
			logrus.Warnf("Failed to fetch channel list for guild %v whilst interpreting channel specifier %v due to error %v", guildID, chanStr, err)
			return nil, err
		}
		for _, ch := range guildChannels {
			if ch.Name == chName {
				//Found it \o/
				return ch, nil
			}
		}
		return nil, fmt.Errorf("couldn't find any channel called %v; it may be worth using a channel link", chName)
	default:
		return nil, fmt.Errorf("%v was not a valid channel string specifier", chanStr)
	}
}

//This is kind of a mess and waay too greedy but the symbol other category doesn't seem to work with RE2 so eh ¯\_(ツ)_/¯
//TODO: replace this with something better
const unicodeEmojiRegex = `(\S{1,4})`

var emojiRegex = regexp.MustCompile(`(<(a?):([^:]+):(\d+)>)|` + unicodeEmojiRegex)

func (b *NiaBot) interpretEmoji(emojiStr string) *string {
	matches := emojiRegex.FindStringSubmatch(emojiStr)
	switch {
	case matches == nil:
		return nil
	case matches[1] != "":
		//Discord guild emoji
		_ /*animatedFlag*/ = matches[2]
		name := matches[3]
		id := matches[4]
		apiName := fmt.Sprintf("%v:%v", name, id)
		return &apiName
	case matches[5] != "":
		//Unicode emoji
		return &matches[5]
	default:
		return nil
	}
}

//Allows message links or IDs
var messageRegex = regexp.MustCompile(`(?:https://discord\.com/channels/\d+/(\d{18})/(\d{18}))|(?:(\d{18}):(\d{18}))`)

func (b *NiaBot) interpretMessageRef(messageStr string) (*string, *string) {
	matches := messageRegex.FindStringSubmatch(messageStr)
	switch {
	case matches == nil:
		return nil, nil
	case matches[1] != "":
		//Message link
		return &matches[1], &matches[2]
	case matches[3] != "":
		//Message ID
		return &matches[3], &matches[4]
	default:
		return nil, nil
	}
}

func (b *NiaBot) undoRoleRule(rule *guildmodels.RoleAssignment) error {
	switch rule.AssignmentType {
	case "reaction":
		return b.resetAssignmentReactions(rule.ReactionRoleData)
	default:
		//noop
		return nil
	}
}
