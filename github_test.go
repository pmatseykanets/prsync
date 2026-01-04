package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pmatseykanets/prsync/github"
)

func TestCheckGitHubURLSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		writeString(w, `{"data":{}}`)
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestCheckGitHubURLGraphQLEndpoint(t *testing.T) {
	graphqlCalled := false
	restCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST for GraphQL, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("Accept") != "application/json" {
				t.Errorf("Expected Accept application/json, got %s", r.Header.Get("Accept"))
			}

			// Verify query is sent in body
			var body struct {
				Query string `json:"query"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
			}
			if body.Query == "" {
				t.Error("Expected query in request body")
			}

			graphqlCalled = true
			w.WriteHeader(http.StatusOK)
			writeString(w, `{"data":{}}`)
		} else if strings.HasSuffix(r.URL.Path, "/user") {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET for REST, got %s", r.Method)
			}
			restCalled = true
			w.WriteHeader(http.StatusOK)
			writeString(w, `{"login":"test"}`)
		} else {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !graphqlCalled {
		t.Error("GraphQL endpoint was not called")
	}
	if !restCalled {
		t.Error("REST endpoint was not called")
	}
}

func TestCheckGitHubURLGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, github.Error{
				Message: "Bad credentials",
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected error for unauthorized GraphQL request")
	}
	if !strings.Contains(err.Error(), "error checking GraphQL endpoint") {
		t.Errorf("Expected GraphQL error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 status in error, got: %v", err)
	}
}

func TestCheckGitHubURLRESTError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusOK)
			writeString(w, `{"data":{}}`)
		} else if strings.HasSuffix(r.URL.Path, "/user") {
			w.WriteHeader(http.StatusForbidden)
			writeJSON(w, github.Error{
				Message: "Forbidden",
			})
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected error for forbidden REST request")
	}
	if !strings.Contains(err.Error(), "error checking REST endpoint") {
		t.Errorf("Expected REST error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("Expected 403 status in error, got: %v", err)
	}
}

func TestCheckGitHubURLGenericHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusInternalServerError)
			// No JSON body, should use status text
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected 500 status in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("Expected status text in error, got: %v", err)
	}
}

func TestCheckGitHubURLNetworkError(t *testing.T) {
	ctx := context.Background()
	// Use an invalid URL that will cause network error
	err := checkGitHubURL(ctx, "http://localhost:1", http.DefaultClient)
	if err == nil {
		t.Fatal("Expected network error")
	}
	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Expected network error message, got: %v", err)
	}
}

func TestCheckGitHubURLContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server that never responds
		select {}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}
	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Expected request error from cancelled context, got: %v", err)
	}
}

func TestCheckGitHubURLInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusBadRequest)
			writeString(w, `invalid json`)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected error for bad request")
	}
	// Should still get an error with status code even if JSON parsing fails
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected 400 status in error, got: %v", err)
	}
}

func TestCheckGitHubURLCustomErrorMessage(t *testing.T) {
	customMessage := "Rate limit exceeded"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusTooManyRequests)
			writeJSON(w, github.Error{
				Type:    "RATE_LIMITED",
				Message: customMessage,
			})
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected rate limit error")
	}
	if !strings.Contains(err.Error(), customMessage) {
		t.Errorf("Expected custom error message '%s', got: %v", customMessage, err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("Expected 429 status, got: %v", err)
	}
}

func TestCheckGitHubURLBothEndpointsChecked(t *testing.T) {
	// Test that if GraphQL succeeds but REST fails, we still get an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			w.WriteHeader(http.StatusOK)
			writeString(w, `{"data":{}}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	err := checkGitHubURL(ctx, server.URL, server.Client())
	if err == nil {
		t.Fatal("Expected error from REST endpoint check")
	}
	if !strings.Contains(err.Error(), "REST endpoint") {
		t.Errorf("Expected REST endpoint error, got: %v", err)
	}
}

func writeString(w http.ResponseWriter, s string) {
	_, _ = w.Write([]byte(s))
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}
