package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pmatseykanets/prsync/github"
)

// checkGitHubURL verifies the GitHub API endpoint by testing both GraphQL and REST endpoints.
// by exercising the GraphQL and REST endpoints.
func checkGitHubURL(ctx context.Context, url string, httpClient *http.Client) error {
	doRequest := func(method, url string, body io.Reader) error {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("error making request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK {
			return nil
		}

		message := http.StatusText(resp.StatusCode)

		githubError := &github.Error{}
		err = json.NewDecoder(resp.Body).Decode(githubError)
		if err == nil && githubError.Message != "" {
			message = githubError.Message
		}

		return fmt.Errorf("%d %s", resp.StatusCode, message)
	}

	// Check GraphQL endpoint.
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(struct {
		Query string `json:"query"`
	}{
		Query: github.NewViewerQuery(),
	})
	if err != nil {
		return fmt.Errorf("error encoding request body: %w", err)
	}

	if err := doRequest(http.MethodPost, url+"/graphql", &body); err != nil {
		return fmt.Errorf("error checking GraphQL endpoint: %w", err)
	}

	// Check REST endpoint.
	if err := doRequest(http.MethodGet, url+"/user", nil); err != nil {
		return fmt.Errorf("error checking REST endpoint: %w", err)
	}

	return nil
}
