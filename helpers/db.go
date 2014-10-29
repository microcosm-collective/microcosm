package helpers

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang/glog"
)

var db *sql.DB

// DBConfig stores the connection information used by InitDBConnection to
// establish a connection to the database
type DBConfig struct {
	Host     string
	Port     int64
	Database string
	Username string
	Password string
}

// InitDBConnection will establish the connection to the database or die trying
func InitDBConnection(c DBConfig) {
	var err error
	db, err = sql.Open(
		"postgres",
		fmt.Sprintf(
			"user=%s dbname=%s host=%s port=%d password=%s sslmode=%s",
			c.Username,
			c.Database,
			c.Host,
			c.Port,
			c.Password,
			"disable",
		),
	)
	if err != nil {
		glog.Fatal(fmt.Sprintf("Database connection failed: %v", err.Error()))
	}

	err = db.Ping()
	if err != nil {
		glog.Fatal(err)
	}

	// PostgreSQL max is 100, we need to be below that limit as there may be
	// connections from monitoring apps, migrations in process or active
	// debugging by staff
	db.SetMaxOpenConns(90)

	// Any connections above this number that are in the pool and idle will be
	// returned to the database. We should keep this number low enough to help
	// recycle connections over time and to provide spare capacity when needed
	// whilst operating at a lower memory use. But we should keep it high
	// enough that we aren't constantly waiting for connections to be opened.
	// db.SetMaxIdleConns(50)
}

// GetConnection returns a connection from the connection pool of the already
// instantiated db object
func GetConnection() (*sql.DB, error) {
	return db, nil
}

// GetTransaction will begin and then return a transaction on the already
// instantiated db object
func GetTransaction() (*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not start a transaction: %v", err.Error()))
	}

	return tx, err
}
