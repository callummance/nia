package guildmodels

//MemberData represents the data stored on any given member
type MemberData struct {
	GuildID     string            `gorethink:"id[0]"`
	UserID      string            `gorethink:"id[1]"`
	Connections MemberConnections `gorethink:"connections"`
}

//MemberConnections contains a bit of data on a member
type MemberConnections struct {
	TwitchConnection *TwitchStream `gorethink:"twitch_link,omitempty,reference" gorethink_ref:"tid"`
}

//NumConnections returns the number of non-nil connections in a MemberConnections struct
func (cs *MemberConnections) NumConnections() int {
	res := 0
	if cs.TwitchConnection != nil {
		res++
	}
	return res
}

//TwitchStream contains details on a link to a single twitch stream as well as data on its current state
type TwitchStream struct {
	TwitchUID          string       `gorethink:"tid"`
	DiscordStatusPosts []MessageRef `gorethink:"posts,omitempty"`
	IsLive             bool         `gorethink:"is_live"`
}

//MessageRef contains the details needed to specify a single discord message
//made to announce a stream going live
type MessageRef struct {
	GuildID   string `gorethink:"gid"`
	ChannelID string `gorethink:"cid"`
	MessageID string `gorethink:"mid"`
}
