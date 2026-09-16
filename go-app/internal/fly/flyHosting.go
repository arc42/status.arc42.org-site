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

// DeploymentID returns the deployment this process was started from, or "" when
// the service does not run on fly.io.
//
// fly.io injects no release number into the machine (FLY_RELEASE_VERSION is
// empty on Machines); what it does inject is the image reference the machine
// runs, e.g.
//
//	registry.fly.io/arc42-stats:deployment-01M2MMR6YW3JYJ9G10MBDRXAYA
//
// The tag after "deployment-" is that identifier, and it is what the footer
// shows. Anything else - no variable, a digest, a tag of another shape - is
// reported as "" rather than guessed at: a wrong deployment id is worse than
// none (ADR-0002: report what was measured).
func DeploymentID() string {
	imageRef := os.Getenv("FLY_IMAGE_REF")

	// the tag lives in the last path segment, so a registry host with a port
	// (host:5000/app:tag) cannot be mistaken for it
	lastSegment := imageRef[strings.LastIndex(imageRef, "/")+1:]

	colon := strings.LastIndex(lastSegment, ":")
	if colon < 0 {
		return ""
	}

	tag := lastSegment[colon+1:]
	if !strings.HasPrefix(tag, "deployment-") {
		return ""
	}

	return strings.TrimPrefix(tag, "deployment-")
}
