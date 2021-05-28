package db

import (
	"fmt"

	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/gorethink/gorethink.v3/encoding"
)

const membersTable string = "members"
const twitchTable string = "twitch"

//GetTwitchConnectionData returns exactly one twitch connection object for the given member if it exists.
func (db *Connection) GetTwitchConnectionData(guildID, userID string) (*guildmodels.TwitchStream, error) {
	id := []string{guildID, userID}
	query := rethink.Table(membersTable).Get(id).Merge(func(p rethink.Term) interface{} {
		return map[string]interface{}{
			"connections": map[string]interface{}{
				"twitch_link": rethink.Table(twitchTable).Get(p.Field("connections").Field("twitch_link")),
			},
		}
	}).Field("connections").Field("twitch_link")
	res, err := query.Run(db.session)
	defer res.Close()
	if err != nil {
		logrus.Warnf("Failed to get twitch connection data for member %v:%v due to error %v", guildID, userID, err)
		return nil, err
	}
	var data guildmodels.TwitchStream
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
func (db *Connection) GetAllTwitchUIDs() ([]string, error) {
	query := rethink.Table(twitchTable).Field("tid")
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
func (db *Connection) GetMemberByConnection(connection guildmodels.MemberConnections, guildID, userID *string) ([]guildmodels.MemberData, error) {
	rethink.SetVerbose(true)
	//Check that exactly 1 connection is set
	if connection.NumConnections() != 1 {
		return nil, fmt.Errorf("member lookup requires exactly 1 connection, %d were provided", connection.NumConnections())
	}
	//Filter by connections struct
	query := rethink.Table(membersTable).Filter(map[string]interface{}{
		"connections": connection,
	})
	//Filter by userID
	if userID != nil {
		query = query.Filter(func(member rethink.Term) rethink.Term {
			return member.Field("id").Nth(0).Eq(*userID)
		})
	}
	//Filter by guildID
	if guildID != nil {
		query = query.Filter(func(member rethink.Term) rethink.Term {
			return member.Field("id").Nth(1).Eq(*guildID)
		})
	}
	//Join member data with twich data
	query = query.Merge(func(p rethink.Term) interface{} {
		return map[string]interface{}{
			"connections": map[string]interface{}{
				"twitch_link": rethink.Table(twitchTable).Get(p.Field("connections").Field("twitch_link")),
			},
		}
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

//SetTwitchConnectionData updates the stored twitch connection for a given member, returning the new TwitchStream value as well as
//the previous value if it was set.
func (db *Connection) SetTwitchConnectionData(guildID, userID, twitchUID string) (*guildmodels.TwitchStream, *guildmodels.TwitchStream, error) {
	//Get Twitch stream struct
	twitch, err := db.GetTwitchStream(twitchUID)
	if err != nil {
		return nil, nil, err
	}
	//Document to be inserted (or updated)
	doc := guildmodels.MemberData{
		GuildID: guildID,
		UserID:  userID,
		Connections: guildmodels.MemberConnections{
			TwitchConnection: twitch,
		},
	}
	logrus.Trace("Inserting memberdata struct %#v", doc)
	query := rethink.Table(membersTable).Insert(doc, rethink.InsertOpts{
		ReturnChanges: "always",
		Conflict:      "update",
	})
	res, err := query.RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to get twitch connection data for member %v:%v due to error %v", guildID, userID, err)
		return nil, nil, err
	}
	logrus.Tracef("Got result %#v from twitch connection data update", res)
	changes := res.Changes
	//If we got changes back from the DB query
	if len(changes) >= 1 {
		oldVal := changes[0].OldValue
		//If there was an old value (ie. if there was an existing link for that user)
		if oldVal != nil {
			var oldValStruct struct {
				GuildID     string            `gorethink:"id[0]"`
				UserID      string            `gorethink:"id[1]"`
				Connections map[string]string `gorethink:"connections"`
			}
			encoding.Decode(&oldValStruct, oldVal)
			oldStreamUID := oldValStruct.Connections["twitch_link"]
			logrus.Debugf("Retrieved Twitch UID %v as old (replaced) stream.", oldStreamUID)
			if oldStreamUID != "" {
				oldStreamStruct, err := db.GetTwitchStream(oldStreamUID)
				if err != nil {
					logrus.Warnf("Failed to fetch TwitchStream struct for user %v's old stream with uid %v due to error %v", userID, oldStreamUID, err)
					return nil, twitch, nil
				}
				return oldStreamStruct, twitch, nil
			}
		}
		//If there was no old value, just return the new one
		return nil, twitch, err
	}
	return nil, nil, nil
}

//GetTwitchStream returns a TwitchStream struct for the stream with the provided uid. If it does not exist, a new one will be created and returned.
func (db *Connection) GetTwitchStream(uid string) (*guildmodels.TwitchStream, error) {
	//Document to be inserted (or updated)
	doc := guildmodels.TwitchStream{
		TwitchUID: uid,
	}
	logrus.Trace("Inserting twitch struct %#v", doc)
	query := rethink.Table(twitchTable).Insert(doc, rethink.InsertOpts{
		ReturnChanges: "always",
		Conflict: func(id, oldDoc, newDoc rethink.Term) interface{} {
			return oldDoc
		},
	})
	res, err := query.RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to get stream struct for twitch uid %v due to error %v", uid, err)
		return nil, err
	}
	logrus.Tracef("Got result %#v from twitch connection data update", res)
	changes := res.Changes
	if len(changes) >= 1 {
		stream := changes[0].NewValue
		if stream != nil {
			var streamData guildmodels.TwitchStream
			encoding.Decode(&streamData, stream)
			return &streamData, nil
		}
		return nil, fmt.Errorf("got nil value when looking up twitch stream in database")
	}
	return nil, fmt.Errorf("twitch stream insertion did not return any changes")
}

//DeleteTwitchStream removes a twitch stream from the database
func (db *Connection) DeleteTwitchStream(uid string) error {
	//Remove any links to this stream
	_, err := rethink.Table(membersTable).Filter(map[string]interface{}{
		"connections": map[string]interface{}{
			"twitch_link": uid,
		},
	}).Update(map[string]interface{}{
		"connections": map[string]interface{}{
			"twitch_link": rethink.Literal(nil),
		},
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to delete any remaining member links before removing twitch stream with UID %v due to error %v", uid, err)
		return err
	}
	//Delete stream
	_, err = rethink.Table(twitchTable).Get(uid).Delete().RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to delete twitch stream with UID %v due to error %v", uid, err)
		return err
	}
	return nil
}

//AddDiscordStatusPost inserts a message reference into the status posts array for the provided twitch UID in the database
func (db *Connection) AddDiscordStatusPost(tid string, post *guildmodels.MessageRef) error {
	_, err := rethink.Table(twitchTable).Get(tid).Update(func(t rethink.Term) interface{} {
		return t.Merge(map[string]interface{}{
			"posts": t.Field("posts").Default([]interface{}{}).SetInsert(post),
		})
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to insert discord status post %v into DB for twitch uid %v due to error %v", post, tid, err)
		return err
	}
	return nil
}

//RemoveDiscordStatusPost removes the given message reference from the status posts array for the provided twitch UID in the database
func (db *Connection) RemoveDiscordStatusPost(uid string, post *guildmodels.MessageRef) error {
	_, err := rethink.Table(twitchTable).Get(uid).Update(func(t rethink.Term) interface{} {
		return t.Merge(map[string]interface{}{
			"posts": t.Field("posts").SetDifference(post),
		})
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warn("Failed to remove discord status post %v from DB for twitch uid %v due to error %v", post, uid, err)
		return err
	}
	return nil
}

//ClearDiscordStatusPosts removes all message references from the status posts array for the provided twitch UID in the database
func (db *Connection) ClearDiscordStatusPosts(uid string) error {
	_, err := rethink.Table(twitchTable).Get(uid).Update(map[string]interface{}{
		"posts": []guildmodels.MessageRef{},
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warn("Failed to remove discord status posts from DB for twitch uid %v due to error %v", uid, err)
		return err
	}
	return nil
}

//SetTwitchStreamLive updates the database to reflect whether the provided twitch stream is live or not.
func (db *Connection) SetTwitchStreamLive(uid string, isLive bool) error {
	stream := guildmodels.TwitchStream{
		TwitchUID: uid,
		IsLive:    isLive,
	}

	return db.updateTwitchStream(&stream)
}

func (db *Connection) updateTwitchStream(stream *guildmodels.TwitchStream) error {
	_, err := rethink.Table(twitchTable).Get(stream.TwitchUID).Update(stream).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to update twitch stream %v in database due to error %v", stream, err)
		return err
	}
	return nil
}
