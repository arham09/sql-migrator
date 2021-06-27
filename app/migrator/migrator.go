package migrator

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"
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

var Agent = &Migrator{
	Versions:   []string{},
	Migrations: map[string]*Migration{},
}

func Init(db *sql.DB) (*Migrator, error) {
	Agent.db = db

	// get past migration
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version varchar(255));`); err != nil {
		fmt.Println("Unable to create `schema_migrations` table", err)

		return Agent, err
	}

	// Find out past migrations
	rows, err := db.Query("SELECT version FROM `schema_migrations`;")

	if err != nil {
		return Agent, err
	}

	defer rows.Close()

	// Mark the migrations as Done if it is already executed
	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			return Agent, err
		}

		if Agent.Migrations[version] != nil {
			Agent.Migrations[version].done = true
		}
	}

	return Agent, err
}

func Create(name string) error {
	version := time.Now().Format("20060102150405")

	in := struct {
		Version string
		Name    string
	}{
		Version: version,
		Name:    name,
	}

	var out bytes.Buffer

	t := template.Must(template.ParseFiles("./migrations/template.txt"))

	err := t.Execute(&out, in)

	if err != nil {
		return errors.New("Unable to execute template:" + err.Error())
	}

	f, err := os.Create(fmt.Sprintf("./migrations/%s_%s.go", version, name))

	if err != nil {
		return errors.New("Unable to create migration file:" + err.Error())
	}

	defer f.Close()

	if _, err := f.WriteString(out.String()); err != nil {
		return errors.New("Unable to write to migration file:" + err.Error())
	}

	fmt.Println("Generated new migration files...", f.Name())

	return nil
}

func (m *Migrator) AddMigration(mg *Migration) {
	//add migration to hash
	m.Migrations[mg.Version] = mg

	// Insert version into versions array using insertion sort
	index := 0
	for index < len(m.Versions) {
		if m.Versions[index] > mg.Version {
			break
		}
		index++
	}

	m.Versions = append(m.Versions, mg.Version)

	copy(m.Versions[index+1:], m.Versions[index:])

	m.Versions[index] = mg.Version
}

func (m *Migrator) Up(step int) error {
	tx, err := m.db.BeginTx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	count := 0
	for _, v := range m.Versions {
		if step > 0 && count == step {
			break
		}

		mg := m.Migrations[v]

		if mg.done {
			continue
		}

		fmt.Println("Running migration", mg.Version)
		if err := mg.Up(tx); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec("INSERT INTO `schema_migrations` VALUES(?)", mg.Version); err != nil {
			tx.Rollback()
			return err
		}
		fmt.Println("Finished running migration", mg.Version)

		count++
	}

	tx.Commit()

	return nil
}

func (m *Migrator) Down(step int) error {
	tx, err := m.db.BeginTx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	count := 0
	for _, v := range reverse(m.Versions) {
		if step > 0 && count == step {
			break
		}

		mg := m.Migrations[v]

		if !mg.done {
			continue
		}

		fmt.Println("Reverting Migration", mg.Version)
		if err := mg.Down(tx); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec("DELETE FROM `schema_migrations` WHERE version = ?", mg.Version); err != nil {
			tx.Rollback()
			return err
		}
		fmt.Println("Finished reverting migration", mg.Version)

		count++
	}

	tx.Commit()

	return nil
}

func (m *Migrator) MigrationStatus() error {
	for _, v := range m.Versions {
		mg := m.Migrations[v]

		if mg.done {
			fmt.Println(fmt.Sprintf("Migration %s... completed", v))
		} else {
			fmt.Println(fmt.Sprintf("Migration %s... pending", v))
		}
	}

	return nil
}

func reverse(arr []string) []string {
	for i := 0; i < len(arr)/2; i++ {
		j := len(arr) - i - 1
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
