package main

// Saves svg-badges from shields.io to local storage.
//
// These svg files follow this naming schema:
// <nr>-issues.svg (e.g. `13-issues.svg)
// <nr>-bugs.svg (`3-bugs.svg`)
// If there are more issues than `badge.LocalBadgeSvgThreshold`,
// the naming is `20+issues.svg`.

// we download these badges from shields.io, from URLs like this:
// https://img.shields.io/badge/open_issues-19-BDB76B
import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"site-usage-statistics/internal/badge"
	"strconv"
)

// SVGBadgePath is the constant path to the directory where SVG files for badges are stored.
const SVGBadgePath = "./svgs/"

const issuesColor = "CEA41E"
const bugsColor = "DC143C"

const badgeDownloadURLPrefix = "https://img.shields.io/badge/"

// svgFileNameForKindOf creates the filename for the downloaded issue-svg files.
// These are required both for the downloading process AND for creating the URLs in the final output HTML
func svgFileNameForKindOf(kindOf string, count int) string {
	switch kindOf {
	case badge.IssueName:
		return strconv.Itoa(count) + badge.IssueBadgeFileNameSuffix
	case badge.BugName:
		return strconv.Itoa(count) + badge.BugBadgeFileNameSuffix
	default:
		log.Error().Msgf("error creating filename for count %d and kindOf %s", count, kindOf)
		return "_error-" + strconv.Itoa(count)
	}
}

func badgeColorForKindOf(kindOf string) string {
	switch kindOf {
	case badge.IssueName:
		return issuesColor
	case badge.BugName:
		return bugsColor
	default:
		return issuesColor
	}
}

// nrOfBugsIssuesShown returns the infix needed to create the shields.io URL
func nrOfBugsIssuesShown(count int, kindOf string) string {
	if count == 1 {
		// note: Singular
		return "open_" + kindOf + "-1-"
	} else if (count == 0) || (count <= badge.LocalBadgeSvgThreshold) {
		// count + Plural
		return "open_" + kindOf + "s-" + strconv.Itoa(count) + "-"
	} else {
		// 20+
		return "open_" + kindOf + "s-" + strconv.Itoa(badge.LocalBadgeSvgThreshold) + "+-"
	}
}

func openIssueSVGBadgeDownloadURL(count int, kindOf string) string {
	var infix = nrOfBugsIssuesShown(count, kindOf)
	return badgeDownloadURLPrefix + infix + badgeColorForKindOf(kindOf)
}

func createSVGBadgeDirIfNotPresent(dirName string) {

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		errDir := os.MkdirAll(dirName, 0755)
		if errDir != nil {
			log.Fatal().Msg(errDir.Error())
		} else {
			log.Info().Msgf("directory %s created", dirName)
		}

	}
}

// Function to download and save SVG file
func downloadSVG(url string, count int, kindOf string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fileName := svgFileNameForKindOf(kindOf, count)
	log.Info().Msgf("filename is %s", fileName)

	file, err := os.Create(SVGBadgePath + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func init() {
	// Configure the global logger to write to console/stdout and add file and line number
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: zerolog.TimeFormatUnix}
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()
}

func downloadSVGBadgesForKindof(kindOf string) {
	log.Info().Msgf("starting to download %ss", kindOf)

	// Download and save each SVG file
	for count := 1; count <= badge.LocalBadgeSvgThreshold+1; count++ {

		// first, load badge for issue
		url := openIssueSVGBadgeDownloadURL(count, kindOf)
		log.Info().Msgf("loading badge-SVG for %d %ss from %s", count, kindOf, url)
		err := downloadSVG(url, count, kindOf)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	// create SVGBadge directory, if it does not exist,
	// so we have a filesystem location to store SVG files
	createSVGBadgeDirIfNotPresent(SVGBadgePath)

	downloadSVGBadgesForKindof(badge.IssueName)
	downloadSVGBadgesForKindof(badge.BugName)

}
