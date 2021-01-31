package guildmodels

//ManagedRoleRule represents a role which may be assigned automatically by the bot
type ManagedRoleRule struct {
	RoleID         string         `gorethink:"role_id"`
	GuildID        string         `gorethink:"guild_id"`
	RoleAssignment RoleAssignment `gorethink:"role_assignment"`
}

//RoleAssignment represents how a role should be assigned
type RoleAssignment struct {
	AssignmentType   string              `gorethink:"type"`
	ReactionRoleData *ReactionRoleAssign `gorethink:"reaction_opts"`
}

//ReactionRoleAssign represents a role assignment prompted by reacting to a post
type ReactionRoleAssign struct {
	MsgID       string `gorethink:"message_id"`
	ChanID      string `gorethink:"channel_id"`
	EmojiID     string `gorethink:"emoji_id"`
	ShouldClear bool   `gorethink:"should_clear_after"`
}
