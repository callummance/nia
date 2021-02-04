package bot

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
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
