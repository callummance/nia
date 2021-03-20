package guildmodels

//MemberData represents the data stored on any given member
type MemberData struct {
	GuildID     string            `gorethink:"id[0]"`
	UserID      string            `gorethink:"id[1]"`
	Connections MemberConnections `gorethink:"connections"`
}

//MemberConnections contains a bit of data on a member
type MemberConnections struct {
	TwitchConnection *TwitchConnectionData `gorethink:"twitch_link,omitempty"`
}

//NumConnections returns the number of non-nil connections in a MemberConnections struct
func (cs *MemberConnections) NumConnections() int {
	res := 0
	if cs.TwitchConnection != nil {
		res++
	}
	return res
}

//TwitchConnectionData contains details on a link to a single twitch stream
type TwitchConnectionData struct {
	TwitchUID string `gorethink:"twitch_uid"`
}
