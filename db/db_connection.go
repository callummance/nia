package db

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	rethink "gopkg.in/gorethink/gorethink.v3"
)

const dbAddrEnvVar string = "NIA_DB_ADDR"
const dbName string = "test"
const baseDbPoolConnections int = 2
const maxDbPoolConnections int = 20

type DBConnection struct {
	session *rethink.Session
}

//Init creates a new connection pool for the database at the address provided by the relevant environment variable
func Init() (*DBConnection, error) {
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

	res := DBConnection{
		session: session,
	}
	err = res.CreateTables()
	if err != nil {
		return nil, err
	}

	return &res, nil
}

//Close cleanly terminates the database connection
func (db *DBConnection) Close() {
	logrus.Info("Terminating DB connection...")
	_ = db.session.Close()
}

//CreateTables ensures all tables needed exist.
func (db *DBConnection) CreateTables() error {
	//guilds table
	_, err := rethink.TableCreate(guildsTable, rethink.TableCreateOpts{
		PrimaryKey: "id",
	}).RunWrite(db.session)
	if err != nil {
		logrus.Errorf("Failed to create guilds table due to error %v", err)
	}
	return nil
}
