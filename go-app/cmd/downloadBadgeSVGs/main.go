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
	"arc42-status/internal/badge"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strconv"
)

// SVGBadgePath is the constant path to the directory where SVG files for badges are stored.
// It points to /docs, the directory of the static Jekyll site!
const SVGBadgePath = "../../../docs/images/badges/"

const issuesColor = "CEA41E"
const bugsColor = "DC143C"

const badgeDownloadURLPrefix = "https://img.shields.io/badge/"

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
	log.Info().Msgf("try to create directory %s", dirName)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		log.Info().Msgf("directory %s did not exist, creating...", dirName)
		errDir := os.MkdirAll(dirName, 0755)
		if errDir != nil {
			log.Fatal().Msg(errDir.Error())
		} else {
			log.Info().Msgf("directory %s created", dirName)
		}
	} else {
		log.Info().Msg("directory seems to be present...")
	}
}

// Function to download and save SVG file
func downloadSVG(url string, count int, kindOf string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fileName := badge.SVGFileNameForKindOf(kindOf, count)
	log.Info().Msgf("filename is %s%s", SVGBadgePath, fileName)

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

func downloadSVGBadgesForKindOf(kindOf string) {
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

	downloadSVGBadgesForKindOf(badge.IssueName)
	downloadSVGBadgesForKindOf(badge.BugName)

}
