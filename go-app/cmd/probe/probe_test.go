package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"arc42-status/internal/types"
)

func init() {
	pause = func(time.Duration) {} // no real sleeping in tests
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		elapsedMs  int
		body       string
		err        error
		wantState  string
		wantDetail string
	}{
		{"fast 200 with content", 200, 150, "<html>arc42 rocks</html>", nil, "up", ""},
		{"slow 200", 200, 3480, "<html>arc42</html>", nil, "degraded", "slow: 3480ms"},
		{"content missing", 200, 150, "<html>empty</html>", nil, "degraded", "content"},
		{"http error status", 502, 80, "", nil, "down", "502"},
		{"transport error", 0, 0, "", errors.New("dial tcp: timeout"), "down", "timeout"},
	}
	for _, c := range cases {
		state, detail := classify(c.statusCode, c.elapsedMs, c.body, "arc42", c.err)
		if state != c.wantState || detail != c.wantDetail {
			t.Errorf("%s: got %s/%q, want %s/%q", c.name, state, detail, c.wantState, c.wantDetail)
		}
	}
}

func TestProbeSiteUpNeedsOneAttempt(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("arc42"))
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", Host: srv.Listener.Addr().String(), ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "up" {
		t.Errorf("state = %s", res.State)
	}
	if hits.Load() != 1 {
		t.Errorf("an up verdict must need exactly one attempt, got %d", hits.Load())
	}
}

func TestProbeSiteDownNeedsConfirmation(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "down" || res.Detail != "502" {
		t.Errorf("got %s/%q", res.State, res.Detail)
	}
	if hits.Load() < 2 {
		t.Errorf("down needs 2-of-3 confirmation, saw only %d attempts", hits.Load())
	}
}

func TestProbeSiteFlickerIsNotDown(t *testing.T) {
	// first attempt fails, the two confirmations succeed: one dropped
	// packet must not write an incident
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("arc42"))
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "up" {
		t.Errorf("state = %s, want up (failure not confirmed)", res.State)
	}
}
