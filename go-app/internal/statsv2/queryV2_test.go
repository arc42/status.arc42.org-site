package statsv2

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

// TestRunV2QueryMissingResultsKey: a 200 that is valid JSON but carries no
// "results" key at all (e.g. "{}") must not read as an empty result set - it
// is a malformed response and must error.
func TestRunV2QueryMissingResultsKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := RunV2Query(srv.URL, "s3cret", V2Query{SiteID: "x", Metrics: []string{"visitors"}, DateRange: "all"})
	if err == nil {
		t.Fatal("want an error when the response has no results key, got nil")
	}
	if !strings.Contains(err.Error(), "no results") {
		t.Errorf("error should say the response had no results, got %q", err)
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatal("the token must never appear in an error")
	}
}
