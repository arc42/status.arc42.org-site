package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCheckPushGranted(t *testing.T) {
	f := newFakeGitHub(t)

	login, push, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test")
	if err != nil {
		t.Fatalf("checkPush: %v", err)
	}
	if login != "octocat" || !push {
		t.Errorf("got login %q push %t, want octocat true", login, push)
	}
	if _, authorization := f.recorded(); authorization != "Bearer gho_test" {
		t.Errorf("Authorization header = %q", authorization)
	}
}

func TestCheckPushWithoutPermission(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.push = false })

	login, push, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test")
	if err != nil {
		t.Fatalf("checkPush: %v", err)
	}
	if login != "octocat" || push {
		t.Errorf("got login %q push %t, want octocat false", login, push)
	}
}

func TestCheckPushAPIError(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.apiStatus = http.StatusInternalServerError })

	if _, _, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test"); err == nil {
		t.Error("a 500 from GitHub was not reported")
	}
}

func TestCheckPushTimeout(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.apiDelay = 200 * time.Millisecond })
	client := &http.Client{Timeout: 50 * time.Millisecond}

	if _, _, err := checkPush(context.Background(), client, f.server.URL+"/api", "gho_test"); err == nil {
		t.Error("a GitHub slower than the timeout was not reported")
	}
}
