package db

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
)

const dbAddrEnvVar string = "NIA_DB_ADDR"
const dbNameDefault string = "nia"
const dbNameEnvVar string = "NIA_DB_NAME"
const baseDbPoolConnections int = 2
const maxDbPoolConnections int = 20

//Connection contains a handle to the database
type Connection struct {
	session *rethink.Session
}

//Init creates a new connection pool for the database at the address provided by the relevant environment variable
func Init() (*Connection, error) {
	rethink.SetVerbose(true)
	//Get DB name from env
	dbName, exists := os.LookupEnv(dbNameEnvVar)
	if !exists {
		logrus.Warnf("DB name was not provided, falling back to default `%v`", dbNameDefault)
		dbName = dbNameDefault
	}
	//Get DB address from env
	rethinkDBAddr, exists := os.LookupEnv(dbAddrEnvVar)
	if !exists {
		logrus.Errorf("`%v` env variable was not set.", dbAddrEnvVar)
		return nil, fmt.Errorf("`%v` env variable was not set", dbAddrEnvVar)
	}
	//Create new connection pool to db
	session, err := rethink.Connect(rethink.ConnectOpts{
		Address:    rethinkDBAddr,
		Database:   dbName,
		InitialCap: baseDbPoolConnections,
		MaxOpen:    maxDbPoolConnections,
	})
	if err != nil {
		logrus.Errorf("Failed to create connection to rethinkdb instance at address %v because %v.", rethinkDBAddr, err)
		return nil, fmt.Errorf("failed to create connection to rethinkdb instance at address %v because %v", rethinkDBAddr, err)
	}

	res := Connection{
		session: session,
	}

	//Ensure database and required tables exist, and wait for it all to be ready
	res.CreateDatabase(dbName)
	res.CreateTables()

	return &res, nil
}

//Close cleanly terminates the database connection
func (db *Connection) Close() {
	logrus.Info("Terminating DB connection...")
	_ = db.session.Close()
}

//CreateTables ensures all tables needed exist.
func (db *Connection) CreateTables() {
	//guilds table
	_, err := rethink.TableCreate(guildsTable, rethink.TableCreateOpts{
		PrimaryKey: "id",
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to create guilds table due to error %v", err)
	}
	//managed role rules table
	_, err = rethink.TableCreate(guildRolesTable, rethink.TableCreateOpts{
		PrimaryKey: "id",
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to create role rules table due to error %v", err)
	}
	//member data table
	_, err = rethink.TableCreate(membersTable, rethink.TableCreateOpts{
		PrimaryKey: "id",
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to create members table due to error %v", err)
	}
	//twitch data table
	_, err = rethink.TableCreate(twitchTable, rethink.TableCreateOpts{
		PrimaryKey: "tid",
	}).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to create twitch streams table due to error %v", err)
	}
	//Wait for all tables
	rethink.Table(guildsTable).Wait()
	rethink.Table(guildRolesTable).Wait()
	rethink.Table(membersTable).Wait()
	rethink.Table(twitchTable).Wait()
}

func (db *Connection) WaitTablesRead() {
	waitOpts := rethink.WaitOpts{
		WaitFor: "ready_for_reads",
	}
	rethink.Table(guildsTable).Wait(waitOpts)
	rethink.Table(guildRolesTable).Wait(waitOpts)
	rethink.Table(membersTable).Wait(waitOpts)
	rethink.Table(twitchTable).Wait(waitOpts)

}

//CreateDatabase ensures the nia database exists
func (db *Connection) CreateDatabase(dbName string) {
	_, err := rethink.DBCreate(dbName).RunWrite(db.session)
	if err != nil {
		logrus.Warnf("Failed to create %v DB due to error %v", dbName, err)
	}
	rethink.DB(dbName).Wait()
}
