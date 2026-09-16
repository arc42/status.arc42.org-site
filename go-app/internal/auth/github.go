package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// checkPush asks GitHub, with the visitor's own token, who they are and
// whether they may push to GateRepo. GateRepo is public, so the token needs
// no scopes for this.
func checkPush(ctx context.Context, client *http.Client, apiBaseURL, token string) (login string, push bool, err error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := getJSON(ctx, client, apiBaseURL+"/user", token, &user); err != nil {
		return "", false, err
	}
	if user.Login == "" {
		return "", false, errors.New("GitHub /user returned no login")
	}

	var repo struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := getJSON(ctx, client, apiBaseURL+"/repos/"+GateRepo, token, &repo); err != nil {
		return user.Login, false, err
	}
	return user.Login, repo.Permissions.Push, nil
}

func getJSON(ctx context.Context, client *http.Client, url, token string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(into); err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	return nil
}
