package cart

/* Handle SQLite database connection */
// Reference: https://gosamples.dev/sqlite-intro/

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const databaseFilename = "sqlite.db"

var Repo *SQLiteDatabase

func InitDatabase() {
	db, err := sql.Open("sqlite3", databaseFilename)
	if err != nil {
		log.Printf("Error in InitDatabase(): %v\n", err)
	}

	Repo = NewSQLiteDatabase(db)

	if err := Repo.Migrate(); err != nil {
		log.Printf("Migrate failed:\n")
		log.Fatal(err)
	}
}
