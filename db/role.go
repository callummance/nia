package db

import (
	"fmt"

	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
)

//AddManagedRoleRule inserts a new managed role rule struct into the database
func (db *DBConnection) AddManagedRoleRule(rule guildmodels.ManagedRoleRule) error {
	resp, err := rethink.Table(guildRolesTable).Insert(rule).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Encountered error inserting managed role  rule %v into database: %v.", rule, err)
	} else if resp.Errors > 0 {
		err := fmt.Errorf("%v", resp.FirstError)
		logrus.Warnf("Encountered error appending admin role to DB: %v", err)
		return err
	}
	return nil
}

//LookupRolesByEmote takes a message ID as well as its channel and guild, along with an emoji ID.
//It then returns any managed role rules that include that reaction.
func (db *DBConnection) LookupRolesByEmote(msgID string, chanID string, guildID string, emojiID string) ([]guildmodels.ManagedRoleRule, error) {
	filter := map[string]interface{}{
		"guild_id": guildID,
		"role_assignment": map[string]interface{}{
			"type": "reaction",
			"reaction_opts": map[string]interface{}{
				"message_id": msgID,
				"channel_id": chanID,
				"emoji_id":   emojiID,
			},
		},
	}
	logrus.Debugf("Looking up role by emote with filter %#v", filter)
	query := rethink.Table(guildRolesTable).Filter(filter)
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Encountered error looking up role rule for emote %v on mesage %v:%v into database: %v.", emojiID, chanID, msgID, err)
		return nil, err
	}
	var matchingRoleRules []guildmodels.ManagedRoleRule
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&matchingRoleRules)
	if err != nil {
		logrus.Warnf("Encountered error looking up role rule for emote %v on mesage %v:%v into database: %v.", emojiID, chanID, msgID, err)
		return nil, err
	}
	return matchingRoleRules, nil
}
