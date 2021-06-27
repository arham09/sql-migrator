package main

import (
	"github.com/arham09/sql-migrator/cmd"
	_ "github.com/arham09/sql-migrator/migrations"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cmd.Execute()
}
