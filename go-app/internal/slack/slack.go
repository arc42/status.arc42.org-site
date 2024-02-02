package slack

import (
	"arc42-status/internal/env"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"os"
	"sync"
)

const SlackTokenName = "SLACK_AUTH_TOKEN"

// SlackChannelID could be different for PROD and DEV envirionments,
// but is currently identical.
const SlackChannelID = "C06FY002V1D"

var (
	once     sync.Once
	slackAPI *slack.Client
)

// getSlackAuthToken should not be called directly, it is only used by the Singleton GetSlack()
func getSlackAuthToken() string {
	slackAuthToken := os.Getenv(SlackTokenName)
	if slackAuthToken == "" {
		// no value, no slack messages
		// we exit here, as we have no chance of recovery
		log.Error().Msgf("CRITICAL ERROR: required Slack Auth token not set.\n" +
			"You need to set the 'SLACK_AUTH_TOKEN' environment variable prior to launching this application.\n")
		//os.Exit(14)
	}
	return slackAuthToken
}

// checkIfTokenValid checks if the slackAuthToken used to create slackClient
// is valid and has not been revoked
func checkIfTokenValid(api *slack.Client) error {

	_, err := api.AuthTest()
	if err != nil {
		log.Info().Msgf("Token validation failed: %v\n", err)
		return err
	}
	return nil
}

// getSlackAPI() returns a Slack API connection if there is a valid auth token available.
// In case of errors, it returns nil
func getSlackAPI() *slack.Client {
	once.Do(func() {
		var (
			slackClient *slack.Client
			slackError  error
		)

		switch env.GetEnv() {
		case "PROD":
			{
				slackClient = slack.New(getSlackAuthToken())
				break
			}
		case "DEV", "TEST":
			{
				slackClient = slack.New(getSlackAuthToken(), slack.OptionDebug(true))

				break
			}
		default:
			{
				// this should never happen, as env.GetEnv() needs to care for valid environments
				log.Error().Msgf("Invalid environment %s  specified for slack messages", env.GetEnv())
				// os.Exit(14)
			}
		}

		slackError = checkIfTokenValid(slackClient)

		if slackError != nil {
			log.Error().Msgf("Failed to init communication with Slack: %s", slackError)
			slackClient = nil
		}
		slackAPI = slackClient
	})

	return slackAPI

}

// SendSlackMessage sends the given message to the configured slack channel
// Different environments (PROD, DEV, TEST) might have different channels configured
func SendSlackMessage(msg string) {
	api := getSlackAPI()

	if api != nil {
		channelID, timestamp, err := api.PostMessage(
			SlackChannelID,
			slack.MsgOptionText(msg, false),
		)
		if err != nil {
			log.Error().Msgf("Error sending message to Slack: %s", err)
		} else {

			fmt.Printf("Message successfully sent to channel %s at %s", channelID, timestamp)
		}
	} else { // api == nil
		log.Error().Msgf("Could not sent message %s to slack, invalid api token", msg)
	}

}
