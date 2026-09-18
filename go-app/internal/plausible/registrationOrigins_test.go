package plausible

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[2,2]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", "2026-09-15")

	if len(bodies) != 3 {
		t.Fatalf("want three queries, got %d", len(bodies))
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

func TestRegistrationOriginsSmallSampleThreshold(t *testing.T) {
	if types.SmallSampleVisits != 20 {
		t.Fatalf("types.SmallSampleVisits = %d, want 20 (the decision this test guards)", types.SmallSampleVisits)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"dimensions":["arc42.org"],"metrics":[20,20]}],"meta":{}}`))
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "tok", "2026-09-15")
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

	got := registrationOriginsFrom(srv.URL, "tok", "2026-09-15")
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
}

// R3: with no token, the spec requires the section be omitted entirely - no
// request may be sent, and the result carries no cuts.
func TestRegistrationOriginsWithNoTokenSendsNoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no request should be sent when the token is empty, got %s %s", r.Method, r.URL)
	}))
	defer srv.Close()

	got := registrationOriginsFrom(srv.URL, "", "2026-09-15")

	if got.Cuts != nil {
		t.Errorf("Cuts = %+v, want nil when the token is missing", got.Cuts)
	}
	if got.JoinedOn != "2026-09-15" {
		t.Errorf("JoinedOn = %q, want the joined-on date to still be carried through", got.JoinedOn)
	}
	if got.TotalVisits != 0 {
		t.Errorf("TotalVisits = %d, want 0", got.TotalVisits)
	}
	if got.SmallSample {
		t.Error("SmallSample should not be set when there is no data at all")
	}
}
