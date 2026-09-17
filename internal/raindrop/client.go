// Package raindrop is a small client for the parts of the Raindrop.io REST
// API (https://developer.raindrop.io/) this project needs: finding tagged
// raindrops and updating their tags once they've been synced.
package raindrop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.raindrop.io/rest/v1"

// Client is a Raindrop.io API client authenticated with a test token
// (Settings -> Integrations -> "For Developers" -> Create test token).
type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Raindrop is the subset of the Raindrop.io raindrop object this tool uses.
type Raindrop struct {
	ID    int      `json:"_id"`
	Link  string   `json:"link"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

type listResponse struct {
	Result bool       `json:"result"`
	Items  []Raindrop `json:"items"`
}

func (c *Client) do(method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// ListByTag returns raindrops across all collections tagged with the given
// tag (without the leading '#').
func (c *Client) ListByTag(tag string) ([]Raindrop, error) {
	q := url.Values{}
	q.Set("search", "#"+tag)
	// Collection 0 means "all raindrops" across every collection.
	var resp listResponse
	if err := c.do(http.MethodGet, "/raindrops/0?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// MarkProcessed removes triggerTag from the raindrop and, if archiveTag is
// non-empty, adds it, so the item isn't picked up again on the next poll.
func (c *Client) MarkProcessed(item Raindrop, triggerTag, archiveTag string) error {
	tags := make([]string, 0, len(item.Tags)+1)
	for _, t := range item.Tags {
		if !strings.EqualFold(t, triggerTag) {
			tags = append(tags, t)
		}
	}
	if archiveTag != "" {
		tags = append(tags, archiveTag)
	}

	body := map[string]any{"tags": tags}
	return c.do(http.MethodPut, fmt.Sprintf("/raindrop/%d", item.ID), body, nil)
}
