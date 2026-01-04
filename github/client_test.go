package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	httpClient := &http.Client{}
	githubURL := "https://api.github.com"

	client := NewClient(httpClient, githubURL)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.githubURL != githubURL {
		t.Errorf("Expected githubURL = %q, got %q", githubURL, client.githubURL)
	}
	if client.http != httpClient {
		t.Error("Expected http client to match")
	}
	if client.graphql == nil {
		t.Error("Expected graphql client to be initialized")
	}
}

// NOTE: Most GraphQL methods (GetRepositoryPullRequests, GetProject, GetProjectPullRequests,
// GetTeamMembers, AddPullRequestToProject, DeletePullRequestFromProject, AddAssigneeToPullRequest,
// LookupUser, GetUserOrganizations) cannot be easily unit tested without refactoring the Client
// to accept a GraphQL client interface. These methods are better tested through integration tests
// or by refactoring Client to use dependency injection for the graphql client.
//
// The IsOrganizationMember method uses the REST API and can be tested with httptest.

func TestIsOrganizationMemberTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/orgs/myorg/members/alice") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	isMember, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isMember {
		t.Error("Expected user to be organization member")
	}
}

func TestIsOrganizationMemberFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	isMember, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isMember {
		t.Error("Expected user not to be organization member")
	}
}

func TestIsOrganizationMemberFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	isMember, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isMember {
		t.Error("Expected user not to be organization member when requester is not a member")
	}
}

func TestIsOrganizationMemberError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(Error{
			Message: "Bad credentials",
		})
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	isMember, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err == nil {
		t.Fatal("Expected error for unauthorized request")
	}
	if isMember {
		t.Error("Expected isMember to be false on error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected error to mention status code, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Bad credentials") {
		t.Errorf("Expected error to include GitHub message, got: %v", err)
	}
}

func TestIsOrganizationMemberErrorNoMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	isMember, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err == nil {
		t.Fatal("Expected error for server error")
	}
	if isMember {
		t.Error("Expected isMember to be false on error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to mention status code, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("Expected error to include status text, got: %v", err)
	}
}

func TestIsOrganizationMemberURLEscaping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// httptest automatically decodes URL paths, so we see the decoded version
		// The important thing is that special characters are properly handled
		if !strings.Contains(r.URL.Path, "/orgs/") {
			t.Errorf("Expected path to contain /orgs/, got %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "/members/") {
			t.Errorf("Expected path to contain /members/, got %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "user@domain") {
			t.Errorf("Expected path to contain user@domain, got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	_, err := client.IsOrganizationMember(context.Background(), "user@domain", "my-org/test")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestIsOrganizationMemberContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not reach here
		t.Error("Request should not be made with cancelled context")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client := NewClient(server.Client(), server.URL)
	_, err := client.IsOrganizationMember(ctx, "alice", "myorg")

	if err == nil {
		t.Fatal("Expected error with cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestIsOrganizationMemberNetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	client := NewClient(&http.Client{}, "http://invalid-host-that-does-not-exist.local")
	_, err := client.IsOrganizationMember(context.Background(), "alice", "myorg")

	if err == nil {
		t.Fatal("Expected network error")
	}
	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Expected 'error making request' in error, got: %v", err)
	}
}
