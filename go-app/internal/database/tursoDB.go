package database

import (
	"database/sql"
	"fmt"
	"github.com/rs/zerolog/log"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"os"
	"sync"
	"time"
)

// database code depends on Turso database, see https://turso.tech.
// see https://github.com/tursodatabase/libsql-client-go

const TursoDBName = "arc42-statistics"
const TursoURLPlain = "libsql://" + TursoDBName + "-gernotstarke.turso.io"

const TableTimeOfSystemStart = "TimeOfSystemStart"
const TableTimeOfInvocation = "TimeOfInvocation"
const TableTimeOfPlausibleCall = "TimeOfPlausibleCall"
const TableTimeOfGitHubCall = "TimeOfGitHubCall"

// column names for TableTimeOfSystemStart
const (
	SysStartupColumnStartup    = "startup"
	SysStartupColumnAppVersion = "app_version"
	SysStartupColumnEnv        = "environment"
)

// DateTimeLayout is used to format DateTime values
const DateTimeLayout = "2006-01-02 15:04:05"

// Singleton-pattern to ensure the DB is initialized only once
var (
	once       sync.Once
	dbInstance *sql.DB
)

// initAuthToken should not be called directly, it is only used by the Singleton GetDB()
func initAuthToken() string {
	tursoAuthToken := os.Getenv("TURSO_AUTH_TOKEN")
	if tursoAuthToken == "" {
		// no value, no DB calls
		// we exit here, as we have no chance of recovery
		log.Error().Msgf("CRITICAL ERROR: required Turso Auth token not set.\n" +
			"You need to set the 'TURSO_AUTH_TOKEN' environment variable prior to launching this application.\n")

		os.Exit(13)
	}
	return tursoAuthToken
}

// GetDB is a singleton function that returns a pointer to a sql.DB object.
// It ensures that only one instance of the database connection is created.
func GetDB() *sql.DB {
	once.Do(func() {

		var dbUrl = TursoURLPlain + "?authToken=" + initAuthToken()

		db, err := sql.Open("libsql", dbUrl)
		if err != nil {
			log.Error().Msgf("failed to open db %s: %s", dbUrl, err)
			os.Exit(13)
		}
		dbInstance = db
	})

	return dbInstance

}

// SaveStartupTime saves the startup time of the application to the database.
// It inserts a new record into the TableTimeOfSystemStart table with the current time, app version, and environment.
func SaveStartupTime(now time.Time, appVersion string, environment string) {
	// language-SQL
	insertStatement := fmt.Sprintf(
		`INSERT INTO %s ( Startup, AppVersion, Environment ) 
				 VALUES ("%s", "%s", "%s"); `,
		TableTimeOfSystemStart, now.Format(DateTimeLayout), appVersion, environment)

	_, err := GetDB().Exec(insertStatement)
	if err != nil {
		log.Error().Msgf("Error inserting startup metadata %s:%s:%s\n ", TableTimeOfSystemStart, err, environment)
	} else {
		log.Info().Msg("wrote startup time to database")
	}
}

func SaveInvocationParams(requestIP string, route string) {

	insertStatement := fmt.Sprintf(
		`INSERT INTO %s ( invocation_time, request_ip, route ) 
				 VALUES ("%s", "%s", "%s"); `,
		TableTimeOfInvocation, time.Now().Format(DateTimeLayout), requestIP, route)

	_, err := GetDB().Exec(insertStatement)
	if err != nil {
		log.Error().Msgf("Error inserting startup metadata %s:%s:%s\n ", TableTimeOfSystemStart, err)
	} else {
		log.Info().Msg("wrote startup time to database")
	}
}
