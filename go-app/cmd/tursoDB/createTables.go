package main

import (
	"database/sql"
	"fmt"
	"log"
)

import (
	"arc42-status/internal/database"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func createTable(tableName string, db *sql.DB, createTableSQL string) {
	// Execute the SQL statement.
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Error trying to create ", tableName, err)
	}

	fmt.Printf("Table %s created successfully\n", tableName)
}

func main() {

	db := database.GetDB()
	fmt.Printf("Trying to create DB tables on database %v\n", db.Driver())

	// SQL statement to create a table.
	// language=SQL
	createToSRSQL := "CREATE TABLE IF NOT EXISTS " + database.TableTimeOfStatusRequest +
		` (TimeCalled DATETIME PRIMARY KEY DEFAULT CURRENT_TIMESTAMP, 
         RequestIP VARCHAR(16),
		 Route VARCHAR(50))`

	createTable(database.TableTimeOfStatusRequest, db, createToSRSQL)

	// language=SQL
	createToPCSQL := `CREATE TABLE IF NOT EXISTS ` + database.TableTimeOfPlausibleCall +
		`(
			PlausibleCalledAt DATETIME PRIMARY KEY DEFAULT CURRENT_TIMESTAMP,
            ServiceVersion VARCHAR(15)
         )`
	createTable(database.TableTimeOfPlausibleCall, database.GetDB(), createToPCSQL)

	createToGHSQL := `CREATE TABLE IF NOT EXISTS ` + database.TableTimeOfGitHubCall +
		`(
			GitHubCalledAt DATETIME PRIMARY KEY DEFAULT CURRENT_TIMESTAMP,
            ServiceVersion VARCHAR(15)
         )`
	createTable(database.TableTimeOfGitHubCall, database.GetDB(), createToGHSQL)

	createToSysStart := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS  %s 
( Startup DATETIME PRIMARY KEY DEFAULT CURRENT_TIMESTAMP, 
AppVersion VARCHAR(15), 
Environment VARCHAR(15) ); `, database.TableTimeOfSystemStart)
	createTable(database.TableTimeOfSystemStart, database.GetDB(), createToSysStart)
}
