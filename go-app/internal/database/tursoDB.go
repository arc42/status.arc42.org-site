package database

import (
	"arc42-status/internal/env"
	"database/sql"
	"fmt"
	"github.com/rs/zerolog/log"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"os"
	"sync"
	"time"
)

// database code depends on Turso database, see https://turso.tech.

// Access to turso depends on the libsql driver,
// see https://github.com/tursodatabase/libsql-client-go

// database schema (tables, columns) are defined in file "schema.hcl"
// and managed by Atlas.

const TursoPRODDBName = "arc42-statistics"
const TursoTESTDBName = "arc42-test"
const TursoPRODUrl = "libsql://" + TursoPRODDBName + "-gernotstarke.turso.io"
const LocalSQLiteURL = "sqlite://dev.db?_fk=1"

const TableTimeOfSystemStart = "system_startup"
const TableTimeOfInvocation = "time_of_invocation"
const TableTimeOfPlausibleCall = "time_of_plausible_call"
const TableTimeOfGitHubCall = "time_of_github_call"

// column names for TableTimeOfSystemStart
const (
	SysStartupColumnStartup    = "startup"
	SysStartupColumnAppVersion = "app_version"
	SysStartupColumnEnv        = "environment"
)

// colum names for TableTimeOfInvocation
const (
	InvocationTimeColumnInvocation = "invocation_time"
	InvocationTimeColumnRequestIP  = "request_ip"
	InvocationTimeColumnRoute      = "route"
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
// For PROD, this is always the Turso libSQL database.
// For DEV or TEST, this is a local instance of SQLite.
func GetDB() *sql.DB {
	once.Do(func() {
		var dbUrl string
		var driverName string

		switch env.GetEnv() {
		case "PROD":
			{
				dbUrl = TursoPRODUrl + "?authToken=" + initAuthToken()
				driverName = "libsql"
				break
			}
		case "DEV":
			{
				dbUrl = LocalSQLiteURL
				driverName = "sqlite"
				break
			}
		case "TEST":
			{
				dbUrl = ""
				driverName = ""
				break
			}
		default:
			{
				// this should never happen, as env.GetEnv() needs to care for valid environments
				log.Error().Msgf("Invalid environment %s  specified", env.GetEnv())
				os.Exit(13)
			}
		}

		// open the database
		db, err := sql.Open(driverName, dbUrl)
		if err != nil {
			log.Error().Msgf("Failed to open db %s: %s", dbUrl, err)
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
		`INSERT INTO %s ( %s, %s, %s ) 
				 VALUES ("%s", "%s", "%s"); `,
		TableTimeOfSystemStart,
		SysStartupColumnStartup, SysStartupColumnAppVersion, SysStartupColumnEnv,
		now.Format(DateTimeLayout), appVersion, environment)

	_, err := GetDB().Exec(insertStatement)
	if err != nil {
		log.Error().Msgf("Error inserting startup metadata %s:%s:%s\n ", TableTimeOfSystemStart, err, environment)
	} else {
		log.Info().Msg("wrote startup time to database")
	}
}

func SaveInvocationParams(requestIP string, route string) {

	insertStatement := fmt.Sprintf(
		`INSERT INTO %s ( %s, %s, %s ) 
				 VALUES ("%s", "%s", "%s"); `,
		TableTimeOfInvocation,
		InvocationTimeColumnInvocation, InvocationTimeColumnRequestIP,
		InvocationTimeColumnRoute,
		time.Now().Format(DateTimeLayout), requestIP, route)

	_, err := GetDB().Exec(insertStatement)
	if err != nil {
		log.Error().Msgf("Error inserting invocation parameters %s:%s:%s\n ", TableTimeOfSystemStart, err)
	} else {
		log.Info().Msg("wrote request parameters to database")
	}
}
