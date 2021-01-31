package db

import (
	"fmt"

	"github.com/callummance/nia/guildmodels"
	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
)

//AddManagedRoleRule inserts a new managed role rule struct into the database
func (db *DBConnection) AddManagedRoleRule(rule guildmodels.ManagedRoleRule) error {
	resp, err := rethink.Table(guildRolesTable).Insert(rule).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Encountered error inserting managed role  rule %v into database: %v.", rule, err)
	} else if resp.Errors > 0 {
		err := fmt.Errorf("%v", resp.FirstError)
		logrus.Warnf("Encountered error appending admin role to DB: %v", err)
		return err
	}
	return nil
}
