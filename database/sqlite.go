package database

import (
	"fmt"

	waLog "go.mau.fi/whatsmeow/util/log"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

var conn *sqlstore.Container

func NewSqlite() (*sqlstore.Container, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		fmt.Errorf("err sqlstore.New : %v ", err)
		return nil, err
	}

	SetConnection(container)

	return container, nil
}

// GetConnection : Get Available Connection
func GetConnection() *sqlstore.Container {
	return conn
}

// SetConnection : Set Available Connection
func SetConnection(connection *sqlstore.Container) {
	conn = connection
}
