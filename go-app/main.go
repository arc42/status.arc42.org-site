package main

import (
	"arc42-status/internal/api"
	"arc42-status/internal/database"
	"arc42-status/internal/domain"
	"arc42-status/internal/env"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

const appVersion = "0.5.5"

// version history
// 0.5.x rate limit: limit amount of queries to external APIs
//       0.5.2: distinct env package, distinct DB for DEV, handle OPTIONS request
//		 0.5.3: BUG and BUGS are both recognized
// 		 0.5.4: start with empty table on homepage
// 		 0.5.5: caching with zcache
// 0.4.7 replace most inline styles by css
// 0.4.6 sortable table (a: initial, b...e: fix layout issues), f: fix #94
// 0.4.5 fix missing separators in large numbers
// 0.4.4 fix bad hyperlink to GitHub issues
// 0.4.3 fix #57 (local svg images for issues and badges)
// 0.4.2 merge repositories (site-statistics and status.arc42.org-site) into one!
// 0.4.1 added links to issue & bug badges
// 0.4.0 first version with Goroutines
// 0.3.4b take log level from environment variable LOGLEVEL
// 0.3.4 removed all fmt.print*, migrated to zerolog
// 0.3.3 fixed issue #46
// 0.3.1 slight refactoring
// 0.2.0 integrated GitHub issues
// 0.1.0 made it work

// init is called AFTER all imported packages have been initialized.
func init() {
	// Configure the global logger to write to console/stdout and add file and line number
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: zerolog.TimeFormatUnix}
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	// check if LOGLEVEL is configured in environment

	// zerolog allows for logging at the following levels (from highest to lowest):
	// panic (zerolog.PanicLevel, 5)
	// fatal (zerolog.FatalLevel, 4)
	// error (zerolog.ErrorLevel, 3)
	// warn (zerolog.WarnLevel, 2)
	// info (zerolog.InfoLevel, 1)
	// debug (zerolog.DebugLevel, 0)
	// trace (zerolog.TraceLevel, -1)

	loglevel := os.Getenv("LOGLEVEL")
	switch loglevel {
	case "":
		// no loglevel has been set, default to warning
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		loglevel = "WARN"
	case "TRACE":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		loglevel = "WARN"
	}

	log.Info().Msgf("log level set to %s", loglevel)
}

func main() {
	// As the main package cannot be imported, constants defined here
	// cannot directly be used in internal/* packages.
	// Therefore, we set the appVersion via a func.
	domain.SetAppVersion(appVersion)

	// Save the startup metadata persistently, see ADR-0012
	database.SaveStartupTime(time.Now(), appVersion, env.GetEnv())

	// log the server details
	api.LogServerDetails(appVersion)

	// load statistics and add results to cache

	// Start a server which runs in the background, and waits for http requests
	// to arrive at predefined routes.
	// THIS IS A BLOCKING CALL, therefore server details are printed prior to starting the server

	api.StartAPIServer()
}
