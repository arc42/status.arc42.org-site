// Command probe is the availability prober of ADR-0019: a batch job run
// by a scheduled GitHub Actions workflow. It measures every monitored
// arc42 property once, records transitions and daily rollups in Turso,
// and exits. It is deliberately not a service - nothing to keep running,
// nothing to pay for.
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"arc42-status/internal/types"
)

const (
	// RequestTimeoutSeconds bounds one attempt; a site slower than this
	// is down for practical purposes.
	RequestTimeoutSeconds = 10

	// SlowMs is the degraded threshold: reachable, but not healthy.
	SlowMs = 2000

	// A failure must be confirmed before it is recorded: confirmNeeded
	// of confirmAttempts attempts, retryPauseSeconds apart, must fail.
	// One dropped packet must never write an incident.
	confirmAttempts   = 3
	confirmNeeded     = 2
	retryPauseSeconds = 5
)

// pause is time.Sleep, injectable so the confirmation tests don't wait.
var pause = time.Sleep

// Result is one site's confirmed measurement.
type Result struct {
	Site       string
	State      string // "up", "degraded", "down"
	Detail     string
	ResponseMs int
}

// classify maps one attempt's raw outcome onto a state, per the table in
// ADR-0019.
func classify(statusCode, elapsedMs int, body, expected string, err error) (string, string) {
	if err != nil {
		detail := "error"
		msg := err.Error()
		switch {
		case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
			detail = "timeout"
		case strings.Contains(msg, "no such host"):
			detail = "dns"
		case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls"):
			detail = "tls"
		}
		return "down", detail
	}
	if statusCode < 200 || statusCode > 299 {
		return "down", fmt.Sprintf("%d", statusCode)
	}
	if elapsedMs > SlowMs {
		return "degraded", fmt.Sprintf("slow: %dms", elapsedMs)
	}
	if expected != "" && !strings.Contains(body, expected) {
		// serves 200 but the build broke - the case pure ping
		// monitoring misses
		return "degraded", "content"
	}
	return "up", ""
}

func probeOnce(client *http.Client, url, expected string) (string, string, int) {
	start := time.Now()
	resp, err := client.Get(url)
	elapsedMs := int(time.Since(start).Milliseconds())
	if err != nil {
		state, detail := classify(0, elapsedMs, "", expected, err)
		return state, detail, elapsedMs
	}
	defer resp.Body.Close()
	// 1 MB is plenty to find the expected substring; the pages are small
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	state, detail := classify(resp.StatusCode, elapsedMs, string(body), expected, nil)
	return state, detail, elapsedMs
}

// probeURL measures one URL with confirmation. Split from ProbeSite so
// tests can point it at an httptest server.
func probeURL(client *http.Client, p types.Property, url string) Result {
	state, detail, ms := probeOnce(client, url, p.ExpectedContent)
	if state == "up" {
		return Result{Site: p.Key, State: state, ResponseMs: ms}
	}
	// not up: confirm before declaring. The first attempt already
	// counts as one vote.
	failVotes := 1
	lastState, lastDetail, lastMs := state, detail, ms
	for attempt := 1; attempt < confirmAttempts; attempt++ {
		pause(retryPauseSeconds * time.Second)
		s, d, m := probeOnce(client, url, p.ExpectedContent)
		if s == "up" {
			continue
		}
		failVotes++
		lastState, lastDetail, lastMs = s, d, m
	}
	if failVotes >= confirmNeeded {
		return Result{Site: p.Key, State: lastState, Detail: lastDetail, ResponseMs: lastMs}
	}
	return Result{Site: p.Key, State: "up", ResponseMs: ms}
}

// ProbeSite measures one property at its public URL.
func ProbeSite(client *http.Client, p types.Property) Result {
	return probeURL(client, p, "https://"+p.Host+"/")
}
