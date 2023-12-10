package fly

import (
	"os"
	"strings"
)

// thanx and credits to https://fly.io for providing
// an affordable hosting plan for this service.

// RegionAndLocation returns both region and location
// where the service is running (on fly.io).

func RegionAndLocation() (string, string) {
	region := flyRegion()
	return region, flyRegionCodeToLocation(region)
}

// FlyRegion retrieves the fly.io region from the environment variable FLY_REGION
func flyRegion() string {
	region := os.Getenv("FLY_REGION")
	if region == "" {
		return ""
	} else {
		return region
	}
}

// flyRegionCodeToLocation converts a 3-letter fly.io region
// code to a location name, hopefully being compatible
// with https://fly.io/docs/reference/regions/
// e.g. ams -> Amsterdam
func flyRegionCodeToLocation(regionCode string) string {
	switch strings.ToUpper(regionCode) {
	case "":
		return "on premise (likely: localhost)"
	case "AMS":
		return "Amsterdam, Netherlands"
	case "ARN":
		return "Stockholm, Sweden"
	case "ATL":
		return "Atlanta, Georgia (US)"
	case "BOG":
		return "Bogotá, Colombia"
	case "BOM":
		return "Mumbai, India"
	case "BOS":
		return "Boston, Massachusetts (US)"
	case "CDG":
		return "Paris, France"
	case "DEN":
		return "Denver, Colorado (US)"
	case "DFW":
		return "Dallas, Texas (US)"
	case "EWR":
		return "Secaucus, NJ (US)"
	case "EZE":
		return "Ezeiza, Argentina"
	case "FRA":
		return "Frankfurt, Germany"
	case "GDL":
		return "Guadalajara, Mexico"
	case "GIG":
		return "Rio de Janeiro, Brazil"
	case "GRU":
		return "Sao Paulo, Brazil"

	default:
		return "unknown location"
	}
}
