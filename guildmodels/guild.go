package guildmodels

//DiscordGuild contains configuration for a discord guild managed by this bot
type DiscordGuild struct {
	DiscordGID           string                `gorethink:"id"`
	AdminRoles           []string              `gorethink:"admin_roles"`
	NotificationChannels *NotificationChannels `gorethink:"notification_channels"`
}

//NotificationChannels contains details on which channel each type of alert should be
//posted onto within a discord guild
type NotificationChannels struct {
	StreamNotificationsChannel *string `gorethink:"stream_notification_channel,omitempty"`
}

//DefaultGuild returns an otherwise-empty guild struct with a given ID
func DefaultGuild(gid string) DiscordGuild {
	return DiscordGuild{
		DiscordGID: gid,
		AdminRoles: nil,
	}
}
