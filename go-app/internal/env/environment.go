package env

import (
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"sync"
)

// Singleton-pattern to ensure the environment is set only once
var (
	once        sync.Once
	environment string
)

// GetEnv is a singleton function that returns an Environment.
// It ensures that only one instance is created.
func GetEnv() string {
	once.Do(func() {
		envi := strings.ToUpper(os.Getenv("ENVIRONMENT"))
		switch {
		case strings.HasPrefix(envi, "DEV"):
			{
				envi = "DEV"
				log.Info().Msg("Running on localhost")
			}
		case strings.HasPrefix(envi, "TEST"):
			{
				envi = "TEST"
				log.Info().Msg("Running as TEST on localhost")
			}
		case strings.HasPrefix(envi, "PROD"):
			{
				envi = "PROD"
				log.Info().Msg("Running on fly.io")
			}
		default:
			envi = "DEV" // Default to Development
			log.Info().Msg("No ENVIRONMENT set, presumably running on localhost")
		}
		environment = envi
	})
	return environment
}
