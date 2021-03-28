package db

import (
	"fmt"

	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
)

const guildsTable string = "guilds"
const guildRolesTable string = "managed_roles"

//GetOrCreateGuild fetches a guild with a given ID from the database, creating a new one if it does not exist.
func (db *Connection) GetOrCreateGuild(id string) (*guildmodels.DiscordGuild, error) {
	var guildObj guildmodels.DiscordGuild
	res, err := rethink.Table(guildsTable).Get(id).Run(db.session)
	if err != nil {
		logrus.Errorf("Failed to query database for guild %v because: %v.", id, err)
		return nil, fmt.Errorf("failed to query database for guild %v because: %v", id, err)
	}
	defer res.Close()

	if res.IsNil() {
		//Create new guild object
		logrus.Infof("Inserting new guild id %v into database.", id)
		guildObj := guildmodels.DefaultGuild(id)
		resp, err := rethink.Table(guildsTable).Insert(guildObj).RunWrite(db.session)
		if err != nil {
			logrus.Errorf("Failed to insert new guild with id %v because: %v.", id, err)
			return nil, fmt.Errorf("failed to insert new guild with id %v because: %v", id, err)
		} else if resp.Inserted != 1 {
			logrus.Warnf("Expected to insert 1 new guild but recieved response %v.", resp)
		}
	} else {
		err = res.One(&guildObj)
		if err != nil {
			logrus.Errorf("Failed to read guild %v from database because: %v.", id, err)
			return nil, fmt.Errorf("failed to read guild %v from database because: %v", id, err)
		}
	}
	return &guildObj, nil
}

//AddAdminRole adds a roleID to the list of AdminRoles for the given guild. It returns the number of updated
//entries as well as any errors
func (db *Connection) AddAdminRole(gid string, roleID string) (int, error) {
	resp, err := rethink.Table(guildsTable).Get(gid).Update(map[string]interface{}{
		"admin_roles": rethink.Row.Field("admin_roles").SetInsert(roleID),
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Encountered error appending admin role to DB: %v", err)
		return 0, err
	} else if resp.Errors > 0 {
		err := fmt.Errorf("%v", resp.FirstError)
		logrus.Warnf("Encountered error appending admin role to DB: %v", err)
		return 0, err
	}
	return resp.Replaced, nil
}

//UpdateGuildNotificationChannels updates the notification channels assigned to a given guild stored in the database
func (db *Connection) UpdateGuildNotificationChannels(gid string, notifChans guildmodels.NotificationChannels) error {
	resp, err := rethink.Table(guildsTable).Get(gid).Update(map[string]interface{}{
		"notification_channels": notifChans,
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Encountered error updating guild notification channels: %v", err)
		return err
	} else if resp.Errors > 0 {
		err := fmt.Errorf("%v", resp.FirstError)
		logrus.Warnf("Encountered error updating guild notification channels: %v", err)
		return err
	}
	return nil
}
