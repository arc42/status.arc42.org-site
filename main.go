package main

import (
	"site-usage-statistics/internal/api"
	"site-usage-statistics/internal/domain"
)

const AppVersion = "0.3.3"

func main() {

	// as the main package cannot be imported, constants defined here
	// cannot directly be used in internal/* packages, therefore we
	// set the AppVersion via a func.
	domain.SetAppVersion(AppVersion)

	// Start a server which runs in background and waits for http requests to arrive
	// on predefined routes.
	// THIS IS A BLOCKING CALL!
	api.PrintServerDetails(AppVersion)
	api.StartAPIServer()
}
