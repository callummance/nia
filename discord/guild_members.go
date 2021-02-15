package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const memberPageSize int = 512

//GuildMemberResult represents an item fetched using the a GuildMembersIter.
type GuildMemberResult struct {
	Member *discordgo.Member
	Error  error
}

//GuildMembersIter returns a new iterator through the members in a given discord guild
func (e *EventSource) GuildMembersIter(guildID string) chan GuildMemberResult {
	s := e.Session()
	ch := make(chan GuildMemberResult)
	go func(guildID string, s *discordgo.Session) {
		isDone := false
		lastLoc := 0
		var currentPage []*discordgo.Member
		for {
			if isDone {
				ch <- GuildMemberResult{
					Member: nil,
					Error:  nil,
				}
			} else if lastLoc+1 < len(currentPage) {
				//We still have members in the current page that have not been returned
				lastLoc++
				ch <- GuildMemberResult{
					Member: currentPage[lastLoc],
					Error:  nil,
				}
			} else {
				//Need to fetch more members from the API
				afterUID := maxUID(currentPage)
				newMembers, err := s.GuildMembers(guildID, afterUID, memberPageSize)
				if err != nil {
					logrus.Warnf("Failed to fetch page of guild members from discord api: %v", err)
					ch <- GuildMemberResult{
						Member: nil,
						Error:  err,
					}
				}
				//If new page of members is empty, close the iterator.
				if len(newMembers) == 0 {
					isDone = true
					close(ch)
					return
				}
				currentPage = newMembers
				lastLoc = 0
				ch <- GuildMemberResult{
					Member: currentPage[0],
					Error:  nil,
				}
			}
		}
	}(guildID, s)

	return ch
}

func maxUID(members []*discordgo.Member) string {
	maxuid := "0"
	for _, member := range members {
		if member.User.ID > maxuid {
			maxuid = member.User.ID
		}
	}
	return maxuid
}
