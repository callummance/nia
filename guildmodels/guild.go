package guildmodels

//DiscordGuild contains configuration for a discord guild managed by this bot
type DiscordGuild struct {
	DiscordGID string   `gorethink:"id"`
	AdminRoles []string `gorethink:"admin_roles"`
}

//DefaultGuild returns an otherwise-empty guild struct with a given ID
func DefaultGuild(gid string) DiscordGuild {
	return DiscordGuild{
		DiscordGID: gid,
		AdminRoles: nil,
	}
}
