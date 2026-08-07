package probe

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arc42-status/internal/types"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		elapsedMs  int
		body       string
		expected   string
		err        error
		wantState  string
		wantDetail string
	}{
		{"healthy", 200, 150, "hello arc42 world", "arc42", nil, "up", ""},
		{"slow", 200, 2500, "hello arc42 world", "arc42", nil, "degraded", "slow: 2500ms"},
		{"content missing", 200, 150, "hello world", "arc42", nil, "degraded", "content"},
		{"http 500", 500, 150, "", "", nil, "down", "500"},
		{"http 404", 404, 150, "", "", nil, "down", "404"},
		{"no expected string required", 200, 150, "anything", "", nil, "up", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotDetail := classify(tt.status, tt.elapsedMs, tt.body, tt.expected, tt.err)
			if gotState != tt.wantState || gotDetail != tt.wantDetail {
				t.Errorf("classify() = (%q, %q), want (%q, %q)",
					gotState, gotDetail, tt.wantState, tt.wantDetail)
			}
		})
	}
}

func TestProbeURLConfirmation(t *testing.T) {
	// Inject instantaneous pause for tests
	origPause := pause
	pause = func(time.Duration) {}
	defer func() { pause = origPause }()

	t.Run("first try passes -> no retries", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(200)
			w.Write([]byte("arc42 ok"))
		}))
		defer srv.Close()

		p := types.Property{Key: "test", ExpectedContent: "arc42"}
		res := probeURL(srv.Client(), p, srv.URL)
		if res.State != "up" || calls != 1 {
			t.Errorf("got (%s, calls=%d), want (up, calls=1)", res.State, calls)
		}
	})

	t.Run("flaky site passes on retry -> confirmed up", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(200)
			w.Write([]byte("arc42 ok"))
		}))
		defer srv.Close()

		p := types.Property{Key: "test", ExpectedContent: "arc42"}
		res := probeURL(srv.Client(), p, srv.URL)
		if res.State != "up" {
			t.Errorf("got %s, want up", res.State)
		}
	})

	t.Run("persistently down -> confirmed down", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(502)
		}))
		defer srv.Close()

		p := types.Property{Key: "test", ExpectedContent: "arc42"}
		res := probeURL(srv.Client(), p, srv.URL)
		if res.State != "down" || res.Detail != "502" || calls != 3 {
			t.Errorf("got (%s, detail=%s, calls=%d), want (down, detail=502, calls=3)",
				res.State, res.Detail, calls)
		}
	})
}
