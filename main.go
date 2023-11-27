package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"site-usage-statistics/internal/api"
	"site-usage-statistics/internal/domain"
	"strings"
)

const AppVersion = "0.3.4b"

// version history
// 0.3.4b take log level from envirionment variable LOGLEVEL
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

	// find out runtime environment:
	// PROD or PRODUCTION -> fly.io, running in the cloud
	// DEV or DEVELOPMENT -> localhost, running on local machine
	environment := strings.ToUpper(os.Getenv("ENVIRONMENT"))
	if strings.HasPrefix(environment, "PROD") {
		log.Info().Msg("Running on fly.io")
	} else {
		log.Info().Msg("Running on localhost")
	}

	// as the main package cannot be imported, constants defined here
	// cannot directly be used in internal/* packages, therefore we
	// set the AppVersion via a func.
	domain.SetAppVersion(AppVersion)

	// Start a server which runs in background and waits for http requests to arrive
	// on predefined routes.
	// THIS IS A BLOCKING CALL, therefore server details are printed prior to starting the server
	api.LogServerDetails(AppVersion)
	api.StartAPIServer()
}
