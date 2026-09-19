package statsv2

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"arc42-status/internal/types"
)

func TestRegistrationOriginsBuildsTheFilter(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		bodies = append(bodies, string(b))
		// Asymmetric metrics: visitors and visits must not be swappable
		// without a test noticing.
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[2,3]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", trainingsTarget("2026-09-15"))

	if len(bodies) != 3 {
		t.Fatalf("want three queries, got %d", len(bodies))
	}

	wantDimensions := map[string]bool{
		"visit:entry_page_hostname": false,
		"visit:entry_page":          false,
		"visit:source":              false,
	}
	for _, b := range bodies {
		if !strings.Contains(b, `"has_done"`) || !strings.Contains(b, `"/registration/"`) {
			t.Errorf("every query must select visits with has_done, got %s", b)
		}
		if !strings.Contains(b, `"date_range":"all"`) {
			t.Errorf("the window must be all time, got %s", b)
		}
		if !strings.Contains(b, `"site_id":"rollup.arc42.com"`) {
			t.Errorf("the site must be the shared rollup dashboard, got %s", b)
		}

		var decoded struct {
			Dimensions []string `json:"dimensions"`
			Metrics    []string `json:"metrics"`
		}
		if err := json.Unmarshal([]byte(b), &decoded); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if len(decoded.Dimensions) != 1 {
			t.Fatalf("want exactly one dimension per query, got %v", decoded.Dimensions)
		}
		dim := decoded.Dimensions[0]
		seen, known := wantDimensions[dim]
		if !known {
			t.Errorf("unexpected dimension %q", dim)
		} else if seen {
			t.Errorf("dimension %q was sent more than once", dim)
		}
		wantDimensions[dim] = true

		if len(decoded.Metrics) != 2 || decoded.Metrics[0] != "visitors" || decoded.Metrics[1] != "visits" {
			t.Errorf("metrics = %v, want [visitors visits] in that order", decoded.Metrics)
		}
	}
	for dim, seen := range wantDimensions {
		if !seen {
			t.Errorf("dimension %q was never sent", dim)
		}
	}

	if got.TotalVisits != 3 {
		t.Errorf("TotalVisits = %d, want 3 (visits, not visitors, from the asymmetric metrics)", got.TotalVisits)
	}
	for _, c := range got.Cuts {
		if len(c.Rows) != 1 {
			t.Fatalf("cut %q rows = %+v, want one row", c.Title, c.Rows)
		}
		if c.Rows[0].Visitors != 2 {
			t.Errorf("cut %q Visitors = %d, want 2", c.Title, c.Rows[0].Visitors)
		}
		if c.Rows[0].Visits != 3 {
			t.Errorf("cut %q Visits = %d, want 3", c.Title, c.Rows[0].Visits)
		}
	}
	if !got.SmallSample {
		t.Error("3 visits is below the threshold and must be flagged")
	}
	if got.JoinedOn != "2026-09-15" {
		t.Errorf("JoinedOn = %q", got.JoinedOn)
	}
	if !got.InRollup || got.Site != "trainings.arc42.org" || got.Page != "/registration/" {
		t.Errorf("trainings block = InRollup %v, Site %q, Page %q; want true, trainings.arc42.org, /registration/",
			got.InRollup, got.Site, got.Page)
	}
}

// The German courses register on arc42.de, which reports only to its own
// dashboard. So its block asks arc42.de, not the rollup, over twelve months,
// and has no entry-hostname cut: every visit there enters on arc42.de.
func TestRegistrationOriginsGermanTarget(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		bodies = append(bodies, string(b))
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["/termine/"],"metrics":[64,67]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", germanTarget())

	if len(bodies) != 2 {
		t.Fatalf("want two queries for arc42.de, got %d", len(bodies))
	}
	wantDimensions := map[string]bool{"visit:entry_page": false, "visit:source": false}
	for _, b := range bodies {
		if !strings.Contains(b, `"site_id":"arc42.de"`) {
			t.Errorf("the German block must ask arc42.de's own dashboard, got %s", b)
		}
		if !strings.Contains(b, `"has_done"`) || !strings.Contains(b, `"/anmeldung/"`) {
			t.Errorf("every query must select visits that did /anmeldung/, got %s", b)
		}
		if !strings.Contains(b, `"date_range":"12mo"`) {
			t.Errorf("the German window must be twelve months, got %s", b)
		}
		var decoded struct {
			Dimensions []string `json:"dimensions"`
		}
		if err := json.Unmarshal([]byte(b), &decoded); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if len(decoded.Dimensions) != 1 {
			t.Fatalf("want exactly one dimension per query, got %v", decoded.Dimensions)
		}
		if _, known := wantDimensions[decoded.Dimensions[0]]; !known {
			t.Errorf("unexpected dimension %q for arc42.de", decoded.Dimensions[0])
		}
		wantDimensions[decoded.Dimensions[0]] = true
	}
	for dim, seen := range wantDimensions {
		if !seen {
			t.Errorf("dimension %q was never sent", dim)
		}
	}

	if got.InRollup {
		t.Error("arc42.de is not in the rollup; its block must say so")
	}
	if got.Site != "arc42.de" || got.Page != "/anmeldung/" {
		t.Errorf("Site %q, Page %q; want arc42.de, /anmeldung/", got.Site, got.Page)
	}
	if got.JoinedOn != "" {
		t.Errorf("JoinedOn = %q; arc42.de never joined the rollup, want empty", got.JoinedOn)
	}
	if got.Heading == "" || got.Window == "" {
		t.Errorf("Heading %q, Window %q; both must name what the block reports", got.Heading, got.Window)
	}
	if got.TotalVisits != 67 || got.SmallSample {
		t.Errorf("TotalVisits %d, SmallSample %v; want 67 and not small", got.TotalVisits, got.SmallSample)
	}
}

// Both course sites are reported, trainings first, and every block's queries
// are sent - five in all.
func TestRegistrationOriginsReportsBothCourseSites(t *testing.T) {
	var mu sync.Mutex
	sites := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var decoded struct {
			SiteID string `json:"site_id"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &decoded)
		mu.Lock()
		sites[decoded.SiteID]++
		mu.Unlock()
		_, _ = w.Write([]byte(`{"results":[],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsAll(srv.URL, "tok", "2026-09-15")

	if len(got) != 2 {
		t.Fatalf("want two blocks, got %d", len(got))
	}
	if got[0].Site != "trainings.arc42.org" || got[1].Site != "arc42.de" {
		t.Errorf("block order = %q, %q; want trainings.arc42.org, then arc42.de", got[0].Site, got[1].Site)
	}
	if sites["rollup.arc42.com"] != 3 || sites["arc42.de"] != 2 {
		t.Errorf("queries per site = %v; want rollup.arc42.com 3, arc42.de 2", sites)
	}
}

func TestRegistrationOriginsSmallSampleThreshold(t *testing.T) {
	if types.SmallSampleVisits != 20 {
		t.Fatalf("types.SmallSampleVisits = %d, want 20 (the decision this test guards)", types.SmallSampleVisits)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[20,20]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", trainingsTarget("2026-09-15"))
	if got.TotalVisits != types.SmallSampleVisits {
		t.Fatalf("TotalVisits = %d, want %d", got.TotalVisits, types.SmallSampleVisits)
	}
	if got.SmallSample {
		t.Error("TotalVisits at the threshold must not be flagged as a small sample")
	}
}

func TestRegistrationOriginsKeepsFailureSeparateFromEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", trainingsTarget("2026-09-15"))
	if len(got.Cuts) != 3 {
		t.Fatalf("want three cuts even when every query failed, got %d", len(got.Cuts))
	}
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
	if got.SmallSample {
		t.Error("SmallSample must not fire on a zero that means unknown, not measured-and-small")
	}
}

// TestRegistrationOriginsMixedOutcome: one cut fails while its siblings
// succeed. The surviving cuts must keep their rows and drive TotalVisits; the
// failed cut carries no rows and stays separate from an empty result.
func TestRegistrationOriginsMixedOutcome(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		var decoded struct {
			Dimensions []string `json:"dimensions"`
		}
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		dim := ""
		if len(decoded.Dimensions) > 0 {
			dim = decoded.Dimensions[0]
		}
		if dim == "visit:entry_page" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[5,7]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", trainingsTarget("2026-09-15"))

	if len(got.Cuts) != 3 {
		t.Fatalf("want three cuts, got %d", len(got.Cuts))
	}
	for _, c := range got.Cuts {
		if c.Title == "Which page they came in through" {
			if !c.Failed {
				t.Errorf("cut %q should be marked failed", c.Title)
			}
			if len(c.Rows) != 0 {
				t.Errorf("failed cut %q must carry no rows, got %+v", c.Title, c.Rows)
			}
			continue
		}
		if c.Failed {
			t.Errorf("cut %q should have succeeded, got FailureReason %q", c.Title, c.FailureReason)
		}
		if len(c.Rows) != 1 || c.Rows[0].Visitors != 5 || c.Rows[0].Visits != 7 {
			t.Errorf("cut %q rows = %+v, want one row with Visitors=5, Visits=7", c.Title, c.Rows)
		}
	}
	if got.TotalVisits != 7 {
		t.Errorf("TotalVisits = %d, want 7 from the surviving cuts", got.TotalVisits)
	}
	if !got.SmallSample {
		t.Error("7 visits is below the threshold and must be flagged - the surviving cuts did produce real data")
	}
}

// With no token the section is omitted entirely: no request may be sent, and
// there are no blocks at all.
func TestRegistrationOriginsWithNoTokenSendsNoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no request should be sent when the token is empty, got %s %s", r.Method, r.URL)
	}))
	defer srv.Close()

	got := registrationOriginsAll(srv.URL, "", "2026-09-15")

	if got != nil {
		t.Errorf("blocks = %+v, want nil when the token is missing", got)
	}
}
