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

//LookupNowLiveRoles returns a list of all roles in the given server which should be assigned when a member is online on a streaming
//platform.
func (db *DBConnection) LookupNowLiveRoles(guildID string) ([]guildmodels.ManagedRoleRule, error) {
	filter := map[string]interface{}{
		"guild_id": guildID,
		"role_assignment": map[string]interface{}{
			"type": "nowlive",
		},
	}
	logrus.Debugf("Looking up nowlive roles with filter %#v", filter)
	query := rethink.Table(guildRolesTable).Filter(filter)
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Encountered error looking up streaming assigned role for guild %v in database: %v.", guildID, err)
		return nil, err
	}
	var matchingRoleRules []guildmodels.ManagedRoleRule
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&matchingRoleRules)
	if err != nil {
		logrus.Warnf("Encountered error looking up streaming assigned role for guild %v in database: %v.", guildID, err)
		return nil, err
	}
	return matchingRoleRules, nil
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

//GetGuildRolesWithInitialReact takes a guild ID and returns a slice of all role assignment rules for that server
//that both use reactions for role assignment and for which the bost should make an initial reaction.
func (db *DBConnection) GetGuildRolesWithInitialReact(guildID string) ([]guildmodels.ManagedRoleRule, error) {
	filter := map[string]interface{}{
		"guild_id": guildID,
		"role_assignment": map[string]interface{}{
			"type": "reaction",
			"reaction_opts": map[string]interface{}{
				"bot_should_react": true,
			},
		},
	}
	logrus.Debugf("Looking up roles with initial react with filter %#v", filter)
	query := rethink.Table(guildRolesTable).Filter(filter)
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Encountered error looking up roles with initial reaction for guild %v: %v.", guildID, err)
		return nil, err
	}
	var matchingRoleRules []guildmodels.ManagedRoleRule
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&matchingRoleRules)
	if err != nil {
		logrus.Warnf("Encountered error looking up roles with initial reaction for guild %v: %v.", guildID, err)
		return nil, err
	}
	return matchingRoleRules, nil
}

//GetRoleRules returns all role assignment rules for a given role in a given server
func (db *DBConnection) GetRoleRules(guildID string, roleID string) ([]guildmodels.ManagedRoleRule, error) {
	filter := map[string]interface{}{
		"guild_id": guildID,
		"role_id":  roleID,
	}
	logrus.Debugf("Looking up rules by role with filter %#v", filter)
	query := rethink.Table(guildRolesTable).Filter(filter)
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Encountered error looking up rules for role %v in guild %v: %v.", roleID, guildID, err)
		return nil, err
	}
	var matchingRoleRules []guildmodels.ManagedRoleRule
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&matchingRoleRules)
	if err != nil {
		logrus.Warnf("Encountered error looking up rules for role %v in guild %v: %v.", roleID, guildID, err)
		return nil, err
	}
	return matchingRoleRules, nil
}

//IsManagedRole returns true iff we have any rules stored for the given roleID in the given guildID
func (db *DBConnection) IsManagedRole(guildID string, roleID string) (bool, error) {
	filter := map[string]interface{}{
		"guild_id": guildID,
		"role_id":  roleID,
	}
	logrus.Debugf("Looking up rules by role with filter %#v", filter)
	query := rethink.Table(guildRolesTable).Filter(filter)
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Encountered error looking up rules for role %v in guild %v: %v.", roleID, guildID, err)
		return false, err
	}
	if res.IsNil() {
		return true, nil
	}
	return false, nil
}
