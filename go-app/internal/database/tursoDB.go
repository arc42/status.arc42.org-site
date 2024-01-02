package database

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
)

// database code depends on Turso database, see https://turso.tech.
// see https://github.com/tursodatabase/libsql-client-go

const TursoDBName = "arc42-statistics"
const TursoURLPlain = "libsql://" + TursoDBName + "-gernotstarke.turso.io"

const TableTimeOfStatusRequest = "TimeOfStatusRequest"

const TableTimeOfPlausibleCall = "TimeOfPlausibleCall"

const TableTimeOfGitHubCall = "TimeOfGitHubCall"

// to ensure
var (
	once       sync.Once
	dbInstance *sql.DB
)

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
