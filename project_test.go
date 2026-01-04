package main

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/pmatseykanets/prsync/github"
)

func TestGetProjectPullRequestsSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	expectedPRs := []*github.PullRequest{
		{
			ID:     "pr1",
			Number: 1,
			Title:  "PR 1",
			URL:    "https://github.com/myorg/repo1/pull/1",
			State:  github.PullRequestStateOpen,
			Author: github.Author{Login: "alice"},
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
			ProjectItemID: "item1",
		},
		{
			ID:     "pr2",
			Number: 2,
			Title:  "PR 2",
			URL:    "https://github.com/myorg/repo2/pull/2",
			State:  github.PullRequestStateOpen,
			Author: github.Author{Login: "bob"},
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo2",
			},
			ProjectItemID: "item2",
		},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			if owner != "myorg" || number != 123 {
				t.Errorf("Expected myorg/123, got %s/%d", owner, number)
			}
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range expectedPRs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	result, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 PRs, got %d", len(result))
	}

	key1 := prKey{owner: "myorg", repo: "repo1", number: 1}
	if pr, ok := result[key1]; !ok {
		t.Error("Expected PR myorg/repo1#1 in result")
	} else if pr.ID != "pr1" {
		t.Errorf("Expected PR ID pr1, got %s", pr.ID)
	}

	key2 := prKey{owner: "myorg", repo: "repo2", number: 2}
	if pr, ok := result[key2]; !ok {
		t.Error("Expected PR myorg/repo2#2 in result")
	} else if pr.ID != "pr2" {
		t.Errorf("Expected PR ID pr2, got %s", pr.ID)
	}
}

func TestGetProjectPullRequestsEmpty(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				// Return empty sequence
			}
		},
	}

	result, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d PRs", len(result))
	}
}

func TestGetProjectPullRequestsError(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	expectedErr := fmt.Errorf("API error")
	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				yield(nil, expectedErr)
			}
		},
	}

	_, err := getProjectPullRequests(ctx, client, cfg)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error fetching project pull requests") {
		t.Errorf("Expected wrapped error message, got: %v", err)
	}
}

func TestGetProjectPullRequestsMultipleRepos(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	prs := []*github.PullRequest{
		{
			ID:     "pr1",
			Number: 10,
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
		},
		{
			ID:     "pr2",
			Number: 5,
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo2",
			},
		},
		{
			ID:     "pr3",
			Number: 20,
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
		},
		{
			ID:     "pr4",
			Number: 3,
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "otherorg"},
				Name:  "repo3",
			},
		},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	result, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("Expected 4 PRs, got %d", len(result))
	}

	// Verify all PRs are indexed correctly
	expectedKeys := []prKey{
		{owner: "myorg", repo: "repo1", number: 10},
		{owner: "myorg", repo: "repo1", number: 20},
		{owner: "myorg", repo: "repo2", number: 5},
		{owner: "otherorg", repo: "repo3", number: 3},
	}

	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("Expected PR %s/%s#%d in result", key.owner, key.repo, key.number)
		}
	}
}

func TestGetProjectPullRequestsDuplicateKeys(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	// Simulate duplicate PRs with same key (should overwrite)
	prs := []*github.PullRequest{
		{
			ID:     "pr1-old",
			Number: 10,
			Title:  "Old PR",
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
		},
		{
			ID:     "pr1-new",
			Number: 10,
			Title:  "New PR",
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
		},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	result, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 PR (deduplicated), got %d", len(result))
	}

	key := prKey{owner: "myorg", repo: "repo1", number: 10}
	if pr, ok := result[key]; !ok {
		t.Error("Expected PR in result")
	} else if pr.ID != "pr1-new" {
		t.Errorf("Expected latest PR (pr1-new), got %s", pr.ID)
	}
}

func TestGetProjectPullRequestsVerboseMode(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
		verbose: true,
	}

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			Number:  1,
			Title:   "Test PR",
			URL:     "https://github.com/myorg/repo1/pull/1",
			State:   github.PullRequestStateOpen,
			IsDraft: false,
			Author:  github.Author{Login: "alice"},
			Repository: github.Repository{
				Owner: github.RepositoryOwner{Login: "myorg"},
				Name:  "repo1",
			},
		},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	// Verbose mode prints output, but shouldn't error
	result, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		t.Fatalf("Expected no error in verbose mode, got: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 PR, got %d", len(result))
	}
}

func TestGetProjectPullRequestsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config{
		project: configProject{owner: "myorg", number: 123},
	}

	client := &fakeGithubClient{
		GetProjectPullRequestsFunc: func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				// Cancel context before yielding
				cancel()
				// The iterator should handle context cancellation gracefully
				select {
				case <-ctx.Done():
					yield(nil, ctx.Err())
				default:
				}
			}
		},
	}

	_, err := getProjectPullRequests(ctx, client, cfg)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}
