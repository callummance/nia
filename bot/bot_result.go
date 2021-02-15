package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type BotResult interface {
	DiscordMessage() string
	WriteToLog()
}

//AdminRoleAlreadyExists indicates that no change was made as the role is already an admin
type AdminRoleAlreadyExists struct {
	roleID    string
	timeStamp time.Time
}

func (r AdminRoleAlreadyExists) DiscordMessage() string {
	return fmt.Sprintf("Could not add role %v as admin because it already has admin priveliges. %v", r.roleID, writeLogRef(r.timeStamp))
}

func (r AdminRoleAlreadyExists) WriteToLog() {
	logrus.Infof("%v Could not add role %v as admin because it already has admin priveliges.", logLineLabel(r.timeStamp), r.roleID)
}

//AdminRoleAdded represents a successful addition of an admin role
type AdminRoleAdded struct {
	timeStamp time.Time
}

func (r AdminRoleAdded) DiscordMessage() string {
	return fmt.Sprintf("All done!")
}

func (r AdminRoleAdded) WriteToLog() {
	logrus.Infof("%v Added role as admin.", logLineLabel(r.timeStamp))
}

//ManagedRoleAdded represents a successful addition of an managed role
type ManagedRoleAdded struct {
	timeStamp time.Time
}

func (r ManagedRoleAdded) DiscordMessage() string {
	return fmt.Sprintf("All done!")
}

func (r ManagedRoleAdded) WriteToLog() {
	logrus.Infof("%v Added managed role.", logLineLabel(r.timeStamp))
}

//RoleNotFound indicates that the named role was not found
type RoleNotFound struct {
	roleName  string
	timeStamp time.Time
}

func (r RoleNotFound) DiscordMessage() string {
	return fmt.Sprintf("Oops, I couldn't find a role called `%v`. It might be worth using an @mention in the command. %v", r.roleName, writeLogRef(r.timeStamp))
}

func (r RoleNotFound) WriteToLog() {
	logrus.Infof("%v Couldn't find role %v.", logLineLabel(r.timeStamp), r.roleName)
}

//InvalidMessageRef indicates that the provided message could not be found
type InvalidMessageRef struct {
	ref       string
	timeStamp time.Time
}

func (r InvalidMessageRef) DiscordMessage() string {
	return fmt.Sprintf("I couldn't find a message at `%v`. It might be worth using a message link in the command. %v", r.ref, writeLogRef(r.timeStamp))
}

func (r InvalidMessageRef) WriteToLog() {
	logrus.Infof("%v Couldn't find message %v.", logLineLabel(r.timeStamp), r.ref)
}

//InvalidEmote indicates that the provided emote could not be found
type InvalidEmote struct {
	emote     string
	timeStamp time.Time
}

func (r InvalidEmote) DiscordMessage() string {
	return fmt.Sprintf("I couldn't find the emote `%v`. %v", r.emote, writeLogRef(r.timeStamp))
}

func (r InvalidEmote) WriteToLog() {
	logrus.Infof("%v Couldn't find emote %v.", logLineLabel(r.timeStamp), r.emote)
}

//SyntaxError indicates that we didn't get the arguments we expected
type SyntaxError struct {
	args      string
	syntax    string
	timeStamp time.Time
}

func (r SyntaxError) DiscordMessage() string {
	return fmt.Sprintf("Sorry, `%v` doesn't make sense as arguments for that command. The correct syntax is %v. %v", r.args, r.syntax, writeLogRef(r.timeStamp))
}

func (r SyntaxError) WriteToLog() {
	logrus.Infof("%v Syntax error: %v should have been %v.", logLineLabel(r.timeStamp), r.args, r.syntax)
}

//InternalError indicates some kind of error whilst accessing the database
type InternalError struct {
	err       error
	timeStamp time.Time
}

func (r InternalError) DiscordMessage() string {
	return fmt.Sprintf("Uh-oh, something went wrong. Please message whoever is responsible for running the bot. %v", writeLogRef(r.timeStamp))
}

func (r InternalError) WriteToLog() {
	logrus.Warnf("%v Encountered critical interal error %v.", logLineLabel(r.timeStamp), r.err)
}

//CommandNeedsAdmin indicates an admin-restricted command was attempted by a non-admin
type CommandNeedsAdmin struct {
	command   string
	timeStamp time.Time
}

func (r CommandNeedsAdmin) DiscordMessage() string {
	return fmt.Sprintf("Sorry, only admins can run `%v`. If you think this is a mistake, please contact a developer. %v", r.command, writeLogRef(r.timeStamp))
}

func (r CommandNeedsAdmin) WriteToLog() {
	logrus.Infof("%v Rejected admin command %v.", logLineLabel(r.timeStamp), r.command)
}

//RoleReset indicates a managed role has been successfully reset
type RoleReset struct {
	roleID    string
	roleName  string
	timeStamp time.Time
}

func (r RoleReset) DiscordMessage() string {
	return fmt.Sprintf("All done!")
}

func (r RoleReset) WriteToLog() {
	logrus.Infof("%v Reset role %v.", logLineLabel(r.timeStamp), r.roleID)
}

type PartialRoleReset struct {
	failedMembers []failedRoleRemoval
	failedRules   []failedRoleRuleReset
	roleID        string
	roleName      string
	timeStamp     time.Time
}

func (r PartialRoleReset) DiscordMessage() string {
	var b strings.Builder
	fmt.Fprintf(&b, ">>>Role %v was reset, but a couple things went wrong.\n", r.roleName)
	if r.failedMembers != nil {
		fmt.Fprintf(&b, "%d members (", len(r.failedMembers))
		for i, failedMember := range r.failedMembers {
			fmt.Fprintf(&b, "%v", failedMember.member.Nick)
			if i < len(r.failedMembers) {
				fmt.Fprint(&b, ", ")
			}
		}
		fmt.Fprint(&b, ") could not have their roles removed.\n")
	}
	if r.failedRules != nil {
		fmt.Fprint(&b, "Also encountered an issue resetting some role assignment methods.\n")
	}
	fmt.Fprint(&b, writeLogRef(r.timeStamp))
	return b.String()
}

func (r PartialRoleReset) WriteToLog() {
	logrus.Infof("%v Reset role %v with issues. Failed members: %v, Failed rules: %v", logLineLabel(r.timeStamp), r.roleID, r.failedMembers, r.failedRules)
}

/////////////////////
//Utility Functions//
/////////////////////
func writeLogRef(t time.Time) string {
	return fmt.Sprintf("More details can be found on log line %v", t.UnixNano())
}

func logLineLabel(t time.Time) string {
	return fmt.Sprintf("#%v#", t.UnixNano())
}
