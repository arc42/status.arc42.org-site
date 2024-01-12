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

// @deprecated as of ADR-0013 (use Atlas for schema management)
// code left here to demo how a table can be created in Go.

func main() {

	db := database.GetDB()
	fmt.Printf("Create DB table on database %v\n", db.Driver())

	createDemoTable := `CREATE TABLE IF NOT EXISTS demo_table (
			current_time  DATETIME PRIMARY KEY DEFAULT CURRENT_TIMESTAMP,
            service_version VARCHAR(15)
         )`

	createTable("demo_table", database.GetDB(), createDemoTable)

}
