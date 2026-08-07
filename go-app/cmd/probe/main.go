package main

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/database"
	"arc42-status/internal/probe"
)

func main() {
	vantage := os.Getenv("PROBE_VANTAGE")
	if vantage == "" {
		vantage = "cli"
	}

	db := database.GetDB()
	start := time.Now()

	recorded, total, err := probe.RunAll(db, vantage)
	if err != nil {
		log.Fatal().Msgf("probe failed: %v", err)
	}

	log.Info().Msgf("probe: recorded %d/%d sites in %dms", recorded, total, time.Since(start).Milliseconds())
}
