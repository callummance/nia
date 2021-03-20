package db

import (
	"fmt"

	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/gorethink/gorethink.v3/encoding"
)

const membersTable string = "members"

//GetTwitchConnectionData returns exactly one twitch connection object for the given member if it exists.
func (db *DBConnection) GetTwitchConnectionData(guildID, userID string) (*guildmodels.TwitchConnectionData, error) {
	id := []string{guildID, userID}
	query := rethink.Table(membersTable).Get(id).Field("connections").Field("twitch_link")
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Failed to get twitch connection data for member %v:%v due to error %v", guildID, userID, err)
		return nil, err
	}
	var data guildmodels.TwitchConnectionData
	if res.IsNil() {
		return nil, nil
	}
	err = res.One(&data)
	if err != nil {
		logrus.Warnf("Failed to retrieve document for twitch connection on member %v:%v due to error %v", guildID, userID, err)
		return nil, nil
	}
	return &data, nil
}

//GetAllTwitchUIDs returns a list of all twitch braodcaster UIDs that have been registered by members
func (db *DBConnection) GetAllTwitchUIDs() ([]string, error) {
	query := rethink.Table(membersTable).HasFields(map[string]interface{}{
		"connections": map[string]interface{}{
			"twitch_link": map[string]interface{}{
				"twitch_uid": true,
			},
		},
	}).Field("connections").Field("twitch_link").Field("twitch_uid")
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Failed to enumerate twitch channels due to error %v", err)
		return nil, err
	}
	var data []string
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&data)
	if err != nil {
		logrus.Warnf("Failed to enumerate twitch channels due to error %v", err)
		return nil, err
	}
	return data, nil
}

//GetMemberByConnection looks up members by connection. The MemberConnections struct should have exactly one non-nil connection.
func (db *DBConnection) GetMemberByConnection(connection guildmodels.MemberConnections) ([]guildmodels.MemberData, error) {
	//Check that exactly 1 connection is set
	if connection.NumConnections() != 1 {
		return nil, fmt.Errorf("member lookup requires exactly 1 connection, %d were provided", connection.NumConnections())
	}
	query := rethink.Table(membersTable).Filter(map[string]interface{}{
		"connections": connection,
	})
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Failed to lookup members for connection %v due to error %v", connection, err)
		return nil, err
	}
	var data []guildmodels.MemberData
	if res.IsNil() {
		return nil, nil
	}
	err = res.All(&data)
	if err != nil {
		logrus.Warnf("Failed to retrieve member documents for connection %v due to error %v", connection, err)
		return nil, nil
	}
	return data, nil
}

//SetTwitchConnectionData updates the stored twitch connection for a given member, returning the previous value if it was set.
func (db *DBConnection) SetTwitchConnectionData(guildID, userID string, data guildmodels.TwitchConnectionData) (*guildmodels.TwitchConnectionData, error) {
	//Document to be inserted (or updated)
	doc := guildmodels.MemberData{
		GuildID: guildID,
		UserID:  userID,
		Connections: guildmodels.MemberConnections{
			TwitchConnection: &data,
		},
	}
	logrus.Trace("Inserting memberdata struct %#v", doc)
	query := rethink.Table(membersTable).Insert(doc, rethink.InsertOpts{
		ReturnChanges: true,
		Conflict:      "update",
	})
	res, err := query.RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to get twitch connection data for member %v:%v due to error %v", guildID, userID, err)
		return nil, err
	}
	logrus.Debug("Got result %#v from twitch connection data update", res)
	changes := res.Changes
	if len(changes) >= 1 {
		oldVal := changes[0].OldValue
		if oldVal != nil {
			var oldMemberData guildmodels.MemberData
			encoding.Decode(&oldMemberData, oldVal)
			return oldMemberData.Connections.TwitchConnection, nil
		}
		return nil, nil
	}
	return nil, nil
}
