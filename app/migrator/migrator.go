package migrator

import (
	"database/sql"
	"fmt"
)

type Migration struct {
	Version string
	Up      func(*sql.Tx) error
	Down    func(*sql.Tx) error

	done bool
}

type Migrator struct {
	db         *sql.DB
	Versions   []string
	Migrations map[string]*Migration
}

var migrator = &Migrator{
	Versions:   []string{},
	Migrations: map[string]*Migration{},
}

func Init(db *sql.DB) (*Migrator, error) {
	migrator.db = db

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations ( version varchar(255));`); err != nil {
		fmt.Println("Unable to create `schema_migrations` table", err)

		return migrator, err
	}

	// Find out past migrations
	rows, err := db.Query("SELECT version FROM `schema_migrations`;")
	if err != nil {
		return migrator, err
	}

	defer rows.Close()

	// Mark the migrations as Done if it is already executed
	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			return migrator, err
		}

		if migrator.Migrations[version] != nil {
			migrator.Migrations[version].done = true
		}
	}

	return migrator, err
}
