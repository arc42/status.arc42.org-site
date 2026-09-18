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
