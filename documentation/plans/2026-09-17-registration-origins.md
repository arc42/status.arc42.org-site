# Registration Origins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show, on the maintainers-only `/rollup` page, where the visits that reach `trainings.arc42.org/registration/` begin — entry hostname, entry page and source — with the sample size stated honestly.

**Architecture:** A small typed client for Plausible's Stats API v2 (`POST /api/v2/query`), written against `net/http` because the vendored `go-plausible v0.3.1` has no `entry_page`, `exit_page` or `hostname` dimension. Three queries per render, differing only in `dimensions`, all filtered with the behavioural operator `has_done` so that whole visits are selected rather than the registration page views themselves. Results flow through the existing cached collection run into the rollup template.

**Tech Stack:** Go 1.21, stdlib `net/http` and `encoding/json`, zerolog, `html/template`, `httptest` for tests. No new dependencies.

**Spec:** `documentation/specs/2026-09-16-registration-origins-design.md`

## Global Constraints

- No new Go dependency. The v2 client is stdlib only.
- The API token is the existing `PLAUSIBLE_API_KEY`. No new secret, and the token is never logged — not in an error, not at debug level.
- Endpoint `https://plausible.io/api/v2/query`, `site_id` `rollup.arc42.com`, header `Authorization: Bearer <token>`.
- The filter is always `[["has_done", ["is", "event:page", ["/registration/"]]]]`. An event-only filter is wrong and must never be used here: it selects matching page views, not the visits containing them.
- Default window is **all time in the rollup** (`"date_range": "all"`), never a 30-day label over two days of membership.
- Every rendered table states the visits behind it. Below **20 visits** the section leads with a sentence saying the sample is too small to generalise from.
- A failed query reports as failed. Zero is never printed in place of unknown (ADR-0002).
- No test calls the live API. Every test serves canned JSON from `httptest`.
- Every href and src added to the template is absolute against `.SiteBaseURL`; `cmd/rendercheck` enforces this.
- Commits end with `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.

---

### Task 1: The Stats API v2 client

**Files:**
- Create: `go-app/internal/plausible/queryV2.go`
- Test: `go-app/internal/plausible/queryV2_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `type V2Query struct{ SiteID string; Metrics []string; DateRange any; Dimensions []string; Filters []any; OrderBy []any; Pagination *V2Pagination }`, `type V2Pagination struct{ Limit, Offset int }`, `type V2Row struct{ Dimensions []string; Metrics []int }`, `func RunV2Query(endpoint, token string, q V2Query) ([]V2Row, error)`.

- [ ] **Step 1: Write the failing test**

```go
package plausible

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunV2QuerySendsTokenAndBody(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[2,2]}],"meta":{}}`))
	}))
	defer srv.Close()

	rows, err := RunV2Query(srv.URL, "s3cret", V2Query{
		SiteID:     "rollup.arc42.com",
		Metrics:    []string{"visitors", "visits"},
		DateRange:  "all",
		Dimensions: []string{"visit:entry_page_hostname"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer s3cret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer s3cret")
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if sent["site_id"] != "rollup.arc42.com" {
		t.Errorf("site_id = %v", sent["site_id"])
	}
	if len(rows) != 1 || rows[0].Dimensions[0] != "arc42.org" || rows[0].Metrics[0] != 2 {
		t.Errorf("rows = %+v", rows)
	}
}

func TestRunV2QueryReportsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"unknown operator has_done"}`))
	}))
	defer srv.Close()

	_, err := RunV2Query(srv.URL, "s3cret", V2Query{SiteID: "x", Metrics: []string{"visitors"}, DateRange: "all"})
	if err == nil {
		t.Fatal("want an error for HTTP 400, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should name the status code, got %q", err)
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatal("the token must never appear in an error")
	}
}

func TestRunV2QueryEmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[],"meta":{}}`))
	}))
	defer srv.Close()

	rows, err := RunV2Query(srv.URL, "t", V2Query{SiteID: "x", Metrics: []string{"visitors"}, DateRange: "all"})
	if err != nil {
		t.Fatalf("an empty result set is not an error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none", rows)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd go-app && go test ./internal/plausible/ -run TestRunV2Query -v`
Expected: FAIL — `undefined: RunV2Query`

- [ ] **Step 3: Write the implementation**

```go
package plausible

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StatsV2Endpoint is Plausible's query API. The vendored go-plausible client
// speaks v1 only, which has no entry_page, exit_page or hostname dimension -
// the three this report is made of - so these queries are sent directly.
const StatsV2Endpoint = "https://plausible.io/api/v2/query"

// V2Pagination limits a query's result rows.
type V2Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// V2Query is a request to the v2 query endpoint. DateRange is a named range
// such as "all" or "30d", or a two-element array of ISO dates.
type V2Query struct {
	SiteID     string        `json:"site_id"`
	Metrics    []string      `json:"metrics"`
	DateRange  any           `json:"date_range"`
	Dimensions []string      `json:"dimensions,omitempty"`
	Filters    []any         `json:"filters,omitempty"`
	OrderBy    []any         `json:"order_by,omitempty"`
	Pagination *V2Pagination `json:"pagination,omitempty"`
}

// V2Row is one result row. Dimensions and Metrics are positional: they follow
// the order of the query's Dimensions and Metrics fields.
type V2Row struct {
	Dimensions []string `json:"dimensions"`
	Metrics    []int    `json:"metrics"`
}

type v2Response struct {
	Results []V2Row `json:"results"`
	Error   string  `json:"error"`
}

// RunV2Query posts one query and returns its rows. An empty result set is a
// result, not an error: it means nothing matched, which is a statement about
// the traffic rather than about the query.
//
// The token is only ever a header value. It is never placed in an error, so
// that a failing query cannot leak it into the log (the spec forbids it).
func RunV2Query(endpoint, token string, q V2Query) ([]V2Row, error) {
	body, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("plausible v2: encoding the query: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("plausible v2: building the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plausible v2: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("plausible v2: reading the response: %w", err)
	}

	var parsed v2Response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("plausible v2: HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("plausible v2: response is not JSON: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != "" {
			return nil, fmt.Errorf("plausible v2: HTTP %d: %s", resp.StatusCode, parsed.Error)
		}
		return nil, fmt.Errorf("plausible v2: HTTP %d", resp.StatusCode)
	}

	return parsed.Results, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd go-app && go test ./internal/plausible/ -run TestRunV2Query -v`
Expected: PASS, three tests

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/plausible/queryV2.go go-app/internal/plausible/queryV2_test.go
git commit -m "plausible: a stdlib client for the v2 query API"
```

---

### Task 2: The three registration-origin queries

**Files:**
- Create: `go-app/internal/plausible/registrationOrigins.go`
- Test: `go-app/internal/plausible/registrationOrigins_test.go`
- Modify: `go-app/internal/types/types.go` (append the result types)

**Interfaces:**
- Consumes: `RunV2Query`, `V2Query`, `V2Row`, `V2Pagination` from Task 1.
- Produces: `types.OriginRow{Label string; Visitors, Visits int}`, `types.OriginCut{Title, Note string; Rows []types.OriginRow; Failed bool; FailureReason string}`, `types.RegistrationOrigins{Cuts []types.OriginCut; TotalVisits int; SmallSample bool; JoinedOn string}`, and `func RegistrationOriginsFor(token string) types.RegistrationOrigins`.

- [ ] **Step 1: Add the result types**

In `go-app/internal/types/types.go`, after the rollup types:

```go
// OriginRow is one row of a registration-origins table: a dimension value
// with the visitors and visits behind it.
type OriginRow struct {
	Label    string
	Visitors int
	Visits   int
}

// OriginCut is one way of cutting the same visits - by entry hostname, entry
// page or source. Failed is separate from an empty Rows: "the query failed"
// and "no visit matched" are different statements (ADR-0002).
type OriginCut struct {
	Title         string
	Note          string
	Rows          []OriginRow
	Failed        bool
	FailureReason string
}

// RegistrationOrigins is the whole section. SmallSample is set when fewer
// than SmallSampleVisits visits stand behind it, and makes the page say so
// before showing any table.
type RegistrationOrigins struct {
	Cuts        []OriginCut
	TotalVisits int
	SmallSample bool
	JoinedOn    string
}

// SmallSampleVisits is the line below which the section refuses to let a
// one-row table read as a finding.
const SmallSampleVisits = 20
```

- [ ] **Step 2: Write the failing test**

```go
package plausible

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"arc42-status/internal/types"
)

func TestRegistrationOriginsBuildsTheFilter(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		bodies = append(bodies, string(b))
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[2,2]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", "2026-09-15")

	if len(bodies) != 3 {
		t.Fatalf("want three queries, got %d", len(bodies))
	}
	for _, b := range bodies {
		if !contains(b, `"has_done"`) || !contains(b, `"/registration/"`) {
			t.Errorf("every query must select visits with has_done, got %s", b)
		}
		if !contains(b, `"date_range":"all"`) {
			t.Errorf("the window must be all time, got %s", b)
		}
	}
	if got.TotalVisits != 2 {
		t.Errorf("TotalVisits = %d, want 2", got.TotalVisits)
	}
	if !got.SmallSample {
		t.Error("2 visits is below the threshold and must be flagged")
	}
	if got.JoinedOn != "2026-09-15" {
		t.Errorf("JoinedOn = %q", got.JoinedOn)
	}
}

func TestRegistrationOriginsKeepsFailureSeparateFromEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", "2026-09-15")
	for _, c := range got.Cuts {
		if !c.Failed {
			t.Errorf("cut %q should be marked failed", c.Title)
		}
		if len(c.Rows) != 0 {
			t.Errorf("a failed cut must carry no rows, got %+v", c.Rows)
		}
	}
	if got.TotalVisits != 0 {
		t.Errorf("TotalVisits = %d, want 0 when every query failed", got.TotalVisits)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

var _ = types.SmallSampleVisits
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd go-app && go test ./internal/plausible/ -run TestRegistrationOrigins -v`
Expected: FAIL — `undefined: registrationOriginsFrom`

- [ ] **Step 4: Write the implementation**

```go
package plausible

import (
	"os"

	"arc42-status/internal/types"

	"github.com/rs/zerolog/log"
)

// registrationPath is the page whose visits this report is about. Plausible
// strips the query string, so this single path covers every ?kurs=... link.
const registrationPath = "/registration/"

// rollupSiteID is the shared dashboard. The question can only be answered
// here: in the trainings site's own dashboard a visit arriving from the docs
// looks like a fresh visit with a referrer, not one visit that began there.
const rollupSiteID = "rollup.arc42.com"

type originCutSpec struct {
	dimension string
	title     string
	note      string
}

var originCuts = []originCutSpec{
	{"visit:entry_page_hostname", "Which arc42 site they came in through", ""},
	{"visit:entry_page", "Which page they came in through",
		"Paths carry no host name here: \"/\" is the front page of whichever member site the visit started on."},
	{"visit:source", "Where they came from before arc42", ""},
}

// RegistrationOriginsFor answers: of the visits that opened the registration
// page, where did the visit begin? joinedOn is the day trainings started
// reporting into the rollup; windows reaching back further under-report, and
// the page says so.
func RegistrationOriginsFor(joinedOn string) types.RegistrationOrigins {
	return registrationOriginsFrom(StatsV2Endpoint, os.Getenv("PLAUSIBLE_API_KEY"), joinedOn)
}

func registrationOriginsFrom(endpoint, token, joinedOn string) types.RegistrationOrigins {
	out := types.RegistrationOrigins{JoinedOn: joinedOn}

	// Selects whole visits that contained a registration page view. An
	// event-level filter would select the page views themselves, whose entry
	// page is meaningless - see the spec.
	filter := []any{[]any{"has_done", []any{"is", "event:page", []string{registrationPath}}}}

	for _, spec := range originCuts {
		cut := types.OriginCut{Title: spec.title, Note: spec.note}

		rows, err := RunV2Query(endpoint, token, V2Query{
			SiteID:     rollupSiteID,
			Metrics:    []string{"visitors", "visits"},
			DateRange:  "all",
			Dimensions: []string{spec.dimension},
			Filters:    filter,
			OrderBy:    []any{[]any{"visitors", "desc"}},
			Pagination: &V2Pagination{Limit: 25},
		})
		if err != nil {
			log.Warn().Msgf("registration origins (%s): %v", spec.dimension, err)
			cut.Failed = true
			cut.FailureReason = err.Error()
			out.Cuts = append(out.Cuts, cut)
			continue
		}

		visits := 0
		for _, r := range rows {
			label := ""
			if len(r.Dimensions) > 0 {
				label = r.Dimensions[0]
			}
			visitors, v := 0, 0
			if len(r.Metrics) > 0 {
				visitors = r.Metrics[0]
			}
			if len(r.Metrics) > 1 {
				v = r.Metrics[1]
			}
			visits += v
			cut.Rows = append(cut.Rows, types.OriginRow{Label: label, Visitors: visitors, Visits: v})
		}
		// Every cut counts the same visits, so the total is one cut's worth,
		// never the sum of all three.
		if visits > out.TotalVisits {
			out.TotalVisits = visits
		}
		out.Cuts = append(out.Cuts, cut)
	}

	out.SmallSample = out.TotalVisits < types.SmallSampleVisits
	return out
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd go-app && go test ./internal/plausible/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go-app/internal/plausible/registrationOrigins.go go-app/internal/plausible/registrationOrigins_test.go go-app/internal/types/types.go
git commit -m "plausible: where the visits that reach the registration begin"
```

---

### Task 3: Carry the result to the page

**Files:**
- Modify: `go-app/internal/domain/domain.go`
- Modify: `go-app/internal/api/apiGateway.go` (the `/rollup` handler's template data)

**Interfaces:**
- Consumes: `plausible.RegistrationOriginsFor`, `types.RegistrationOrigins`.
- Produces: a `RegistrationOrigins` field on the rollup page's template data.

- [ ] **Step 1: Find the join date rather than hard-coding it**

The date already exists in the roster. In `domain.go`, add:

```go
// trainingsJoinedRollup returns the day trainings.arc42.org started reporting
// into the rollup, taken from the declared roster so that one edit there keeps
// the page honest.
func trainingsJoinedRollup() string {
	for _, p := range types.Arc42properties {
		if p.Key == "trainings.arc42.org" {
			return p.RollupSince
		}
	}
	return ""
}
```

- [ ] **Step 2: Collect it with the rest**

Where the rollup figures are built for the page, add:

```go
	origins := plausible.RegistrationOriginsFor(trainingsJoinedRollup())
```

and put `origins` on the struct handed to `rollupPage.gohtml` as `RegistrationOrigins`.

- [ ] **Step 3: Verify it builds and the suite still passes**

Run: `cd go-app && go build ./... && go vet ./... && go test ./...`
Expected: all packages ok

- [ ] **Step 4: Commit**

```bash
git add go-app/internal/domain/domain.go go-app/internal/api/apiGateway.go
git commit -m "rollup: carry the registration origins to the page"
```

---

### Task 4: The section on the page

**Files:**
- Modify: `go-app/internal/api/rollupPage.gohtml`
- Modify: `go-app/cmd/rendercheck/main.go` (fixture)

**Interfaces:**
- Consumes: `.RegistrationOrigins` from Task 3.

- [ ] **Step 1: Add the section after "The numbers"**

```gotemplate
<h2 id="registrations">Where registrations start</h2>

{{ with .RegistrationOrigins }}
<p>Of the visits that opened <code>/registration/</code> on trainings.arc42.org,
    this is where the visit began. Only the rollup can answer this: in the
    trainings dashboard alone, a reader arriving from the docs looks like a new
    visit with a referrer, not one visit that started there.
    {{ if .JoinedOn }}trainings.arc42.org has been reporting into the rollup
    since {{ .JoinedOn }}; nothing before that day is counted here.{{ end }}</p>

{{ if .SmallSample }}
<p class="rollup__note"><b>Too small to generalise from.</b>
    {{ .TotalVisits }} visit{{ if ne .TotalVisits 1 }}s{{ end }} stand behind
    the tables below. They are shown because they are what was measured, not
    because they show a pattern.</p>
{{ end }}

{{ range .Cuts }}
<h3 class="rollup__subhead">{{ .Title }}</h3>
{{ if .Failed }}
<p class="site-page__nodata">This query failed: {{ .FailureReason }}</p>
{{ else if not .Rows }}
<p class="site-page__nodata">No visit in the rollup has reached the registration page yet.</p>
{{ else }}
{{ if .Note }}<p class="rollup__note">{{ .Note }}</p>{{ end }}
<table class="rollup-figures">
    <thead>
    <tr><th scope="col">{{ .Title }}</th><th scope="col">Visitors</th><th scope="col">Visits</th></tr>
    </thead>
    <tbody>
    {{ range .Rows }}
    <tr><th scope="row">{{ .Label }}</th><td>{{ .Visitors }}</td><td>{{ .Visits }}</td></tr>
    {{ end }}
    </tbody>
</table>
{{ end }}
{{ end }}
{{ end }}
```

- [ ] **Step 2: Add the rendercheck fixture**

Give the rollup fixture a `RegistrationOrigins` with three cuts: one with rows, one empty, one failed — so every branch renders at least once.

- [ ] **Step 3: Run rendercheck**

Run: `cd go-app && go run ./cmd/rendercheck`
Expected: `every href and src is absolute, stylesheet present`

- [ ] **Step 4: Commit**

```bash
git add go-app/internal/api/rollupPage.gohtml go-app/cmd/rendercheck/main.go
git commit -m "rollup: show where registrations start"
```

---

### Task 5: Record the decision

**Files:**
- Create: `documentation/adrs/0023-report-where-registrations-start.md`
- Modify: `documentation/specs/2026-09-16-registration-origins-design.md` (status → implemented)
- Modify: `go-app/main.go` (version history, bump to 1.6.0)

- [ ] **Step 1: Write the ADR**

Context: the rollup is the only place the question is answerable; Plausible records no sequences. Decision: three cuts of the same visits, `has_done`, all-time window, sample size always stated. Consequences: three extra API calls per collection run; the report stays thin until the window fills; entry pages carry no host name.

- [ ] **Step 2: Bump the version and describe it**

```go
const appVersion = "1.6.0"

// 1.6.0: the rollup page reports where the visits that reach the trainings
//        registration begin - entry host, entry page and source (ADR-0023),
//        via Plausible's Stats API v2.
```

- [ ] **Step 3: Full verification**

Run: `cd go-app && go build ./... && go vet ./... && go test ./... && go run ./cmd/rendercheck`
Expected: green, and the rollup section renders

- [ ] **Step 4: Commit**

```bash
git add documentation/ go-app/main.go
git commit -m "docs: ADR-0023, where registrations start"
```

---

## Self-review

**Spec coverage:** the question (Tasks 2-4), the `has_done` filter (Task 2, asserted in tests), the three dimensions (Task 2), the honesty rules — failure separate from empty (Tasks 2, 4), sample size stated (Tasks 2, 4), join date from the roster rather than a second hard-coded constant (Task 3) — the failure modes table (Tasks 2, 4), the code shape (Tasks 1-4), and the testing section (Tasks 1, 2). The probe is absent on purpose: it was run on 2026-09-17 and the risk is retired.

**Placeholders:** none. Every code step carries the code to write.

**Type consistency:** `V2Query`, `V2Row`, `V2Pagination`, `RunV2Query` are defined in Task 1 and used with those names in Task 2; `OriginRow`, `OriginCut`, `RegistrationOrigins`, `SmallSampleVisits` are defined in Task 2 and used with those names in Tasks 3 and 4.
