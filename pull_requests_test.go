package main

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/pmatseykanets/prsync/github"
)

func TestDraftState(t *testing.T) {
	tests := []struct {
		name  string
		draft bool
		want  string
	}{
		{name: "draft PR", draft: true, want: "DRAFT"},
		{name: "regular PR", draft: false, want: "PR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := draftState(tt.draft); got != tt.want {
				t.Errorf("draftState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAuthorsPullRequestsSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}
	cfg.pullRequests.add.drafts = false

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
		{
			ID:      "pr2",
			IsDraft: false,
			Author:  github.Author{Login: "bob", Type: github.AuthorTypeUser},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return true, nil // All authors included
		},
	}

	var results []*github.PullRequest
	for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, pr)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 PRs, got %d", len(results))
	}
}

func TestGetAuthorsPullRequestsSkipDrafts(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}
	cfg.pullRequests.add.drafts = false // Don't include drafts

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
		{
			ID:      "pr2",
			IsDraft: true, // Draft should be skipped
			Author:  github.Author{Login: "bob", Type: github.AuthorTypeUser},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return true, nil
		},
	}

	var results []*github.PullRequest
	for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, pr)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 PR (draft skipped), got %d", len(results))
	}
	if results[0].IsDraft {
		t.Error("Result should not include draft PR")
	}
}

func TestGetAuthorsPullRequestsIncludeDrafts(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}
	cfg.pullRequests.add.drafts = true // Include drafts

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
		{
			ID:      "pr2",
			IsDraft: true,
			Author:  github.Author{Login: "bob", Type: github.AuthorTypeUser},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return true, nil
		},
	}

	var results []*github.PullRequest
	for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, pr)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 PRs (including draft), got %d", len(results))
	}
}

func TestGetAuthorsPullRequestsSkipBots(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
		{
			ID:      "pr2",
			IsDraft: false,
			Author:  github.Author{Login: "dependabot", Type: github.AuthorTypeBot},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return true, nil
		},
	}

	var results []*github.PullRequest
	for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, pr)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 PR (bot skipped), got %d", len(results))
	}
	if results[0].Author.Type != github.AuthorTypeUser {
		t.Error("Result should only include user PRs")
	}
}

func TestGetAuthorsPullRequestsFilterByAuthor(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
		{
			ID:      "pr2",
			IsDraft: false,
			Author:  github.Author{Login: "bob", Type: github.AuthorTypeUser},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return login == "alice", nil // Only alice included
		},
	}

	var results []*github.PullRequest
	for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		results = append(results, pr)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 PR (filtered by author), got %d", len(results))
	}
	if results[0].Author.Login != "alice" {
		t.Errorf("Expected alice's PR, got %s", results[0].Author.Login)
	}
}

func TestGetAuthorsPullRequestsError(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}

	expectedErr := fmt.Errorf("API error")
	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				yield(nil, expectedErr)
			}
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return true, nil
		},
	}

	for _, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "error fetching repository pull requests") {
			t.Errorf("Expected wrapped error, got: %v", err)
		}
		break
	}
}

func TestGetAuthorsPullRequestsAuthorResolveError(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}

	prs := []*github.PullRequest{
		{
			ID:      "pr1",
			IsDraft: false,
			Author:  github.Author{Login: "alice", Type: github.AuthorTypeUser},
		},
	}

	client := &fakeGithubClient{
		GetRepositoryPullRequestsFunc: func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
			return func(yield func(*github.PullRequest, error) bool) {
				for _, pr := range prs {
					if !yield(pr, nil) {
						return
					}
				}
			}
		},
	}

	expectedErr := fmt.Errorf("author resolve error")
	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return false, expectedErr
		},
	}

	for _, err := range getAuthorsPullRequests(ctx, client, cfg, authors, "owner", "repo") {
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "error evaluating author filter") {
			t.Errorf("Expected author filter error, got: %v", err)
		}
		break
	}
}

func TestDeleteCompletedPullRequestsNothingToDelete(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	// No delete states or drafts configured
	cfg.pullRequests.delete.states = []github.PullRequestState{}
	cfg.pullRequests.delete.drafts = false

	client := &fakeGithubClient{}
	authors := &fakeAuthorResolver{}
	project := &github.Project{ID: "proj1"}
	projectPRs := make(map[prKey]*github.PullRequest)

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestDeleteCompletedPullRequestsByState(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.delete.states = []github.PullRequestState{github.PullRequestStateMerged}
	cfg.pullRequests.delete.drafts = false
	cfg.pullRequests.delete.allAuthors = true

	deletedIDs := []string{}
	client := &fakeGithubClient{
		DeletePullRequestFromProjectFunc: func(ctx context.Context, projectID, projectItemID string) error {
			deletedIDs = append(deletedIDs, projectItemID)
			return nil
		},
	}

	authors := &fakeAuthorResolver{}
	project := &github.Project{ID: "proj1"}
	projectPRs := map[prKey]*github.PullRequest{
		{owner: "org", repo: "repo1", number: 1}: {
			ID:            "pr1",
			State:         github.PullRequestStateMerged,
			ProjectItemID: "item1",
			Author:        github.Author{Login: "alice"},
		},
		{owner: "org", repo: "repo1", number: 2}: {
			ID:            "pr2",
			State:         github.PullRequestStateOpen,
			ProjectItemID: "item2",
			Author:        github.Author{Login: "bob"},
		},
	}

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(deletedIDs) != 1 {
		t.Errorf("Expected 1 PR deleted, got %d", len(deletedIDs))
	}
	if len(deletedIDs) > 0 && deletedIDs[0] != "item1" {
		t.Errorf("Expected item1 deleted, got %s", deletedIDs[0])
	}
}

func TestDeleteCompletedPullRequestsDrafts(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.delete.states = []github.PullRequestState{}
	cfg.pullRequests.delete.drafts = true
	cfg.pullRequests.delete.allAuthors = true

	deletedIDs := []string{}
	client := &fakeGithubClient{
		DeletePullRequestFromProjectFunc: func(ctx context.Context, projectID, projectItemID string) error {
			deletedIDs = append(deletedIDs, projectItemID)
			return nil
		},
	}

	authors := &fakeAuthorResolver{}
	project := &github.Project{ID: "proj1"}
	projectPRs := map[prKey]*github.PullRequest{
		{owner: "org", repo: "repo1", number: 1}: {
			ID:            "pr1",
			IsDraft:       true,
			ProjectItemID: "item1",
			Author:        github.Author{Login: "alice"},
		},
		{owner: "org", repo: "repo1", number: 2}: {
			ID:            "pr2",
			IsDraft:       false,
			ProjectItemID: "item2",
			Author:        github.Author{Login: "bob"},
		},
	}

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(deletedIDs) != 1 {
		t.Errorf("Expected 1 draft deleted, got %d", len(deletedIDs))
	}
	if len(deletedIDs) > 0 && deletedIDs[0] != "item1" {
		t.Errorf("Expected item1 deleted, got %s", deletedIDs[0])
	}
}

func TestDeleteCompletedPullRequestsFilterByAuthor(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.delete.states = []github.PullRequestState{github.PullRequestStateMerged}
	cfg.pullRequests.delete.drafts = false
	cfg.pullRequests.delete.allAuthors = false // Filter by author

	deletedIDs := []string{}
	client := &fakeGithubClient{
		DeletePullRequestFromProjectFunc: func(ctx context.Context, projectID, projectItemID string) error {
			deletedIDs = append(deletedIDs, projectItemID)
			return nil
		},
	}

	authors := &fakeAuthorResolver{
		resolveFunc: func(ctx context.Context, login string) (bool, error) {
			return login == "alice", nil // Only alice
		},
	}

	project := &github.Project{ID: "proj1"}
	projectPRs := map[prKey]*github.PullRequest{
		{owner: "org", repo: "repo1", number: 1}: {
			ID:            "pr1",
			State:         github.PullRequestStateMerged,
			ProjectItemID: "item1",
			Author:        github.Author{Login: "alice"},
		},
		{owner: "org", repo: "repo1", number: 2}: {
			ID:            "pr2",
			State:         github.PullRequestStateMerged,
			ProjectItemID: "item2",
			Author:        github.Author{Login: "bob"},
		},
	}

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(deletedIDs) != 1 {
		t.Errorf("Expected 1 PR deleted (alice only), got %d", len(deletedIDs))
	}
	if len(deletedIDs) > 0 && deletedIDs[0] != "item1" {
		t.Errorf("Expected item1 deleted, got %s", deletedIDs[0])
	}
}

func TestDeleteCompletedPullRequestsDryRun(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		dryRun: true,
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.delete.states = []github.PullRequestState{github.PullRequestStateMerged}
	cfg.pullRequests.delete.allAuthors = true

	deleteCalled := false
	client := &fakeGithubClient{
		DeletePullRequestFromProjectFunc: func(ctx context.Context, projectID, projectItemID string) error {
			deleteCalled = true
			return nil
		},
	}

	authors := &fakeAuthorResolver{}
	project := &github.Project{ID: "proj1"}
	projectPRs := map[prKey]*github.PullRequest{
		{owner: "org", repo: "repo1", number: 1}: {
			ID:            "pr1",
			State:         github.PullRequestStateMerged,
			ProjectItemID: "item1",
			Author:        github.Author{Login: "alice"},
		},
	}

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if deleteCalled {
		t.Error("Delete should not be called in dry-run mode")
	}
}

func TestDeleteCompletedPullRequestsError(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		pullRequests: struct {
			add struct {
				states       []github.PullRequestState
				assignAuthor bool
				drafts       bool
			}
			delete struct {
				states     []github.PullRequestState
				drafts     bool
				allAuthors bool
			}
		}{},
	}
	cfg.pullRequests.delete.states = []github.PullRequestState{github.PullRequestStateMerged}
	cfg.pullRequests.delete.allAuthors = true

	expectedErr := fmt.Errorf("delete error")
	client := &fakeGithubClient{
		DeletePullRequestFromProjectFunc: func(ctx context.Context, projectID, projectItemID string) error {
			return expectedErr
		},
	}

	authors := &fakeAuthorResolver{}
	project := &github.Project{ID: "proj1"}
	projectPRs := map[prKey]*github.PullRequest{
		{owner: "org", repo: "repo1", number: 1}: {
			ID:            "pr1",
			State:         github.PullRequestStateMerged,
			ProjectItemID: "item1",
			Author:        github.Author{Login: "alice"},
		},
	}

	err := deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error deleting PR") {
		t.Errorf("Expected delete error, got: %v", err)
	}
}

// Fake author resolver for testing
type fakeAuthorResolver struct {
	resolveFunc func(ctx context.Context, login string) (bool, error)
	getIDFunc   func(ctx context.Context, login string) (string, error)
}

func (f *fakeAuthorResolver) Resolve(ctx context.Context, login string) (bool, error) {
	if f.resolveFunc != nil {
		return f.resolveFunc(ctx, login)
	}
	return true, nil
}

func (f *fakeAuthorResolver) GetID(ctx context.Context, login string) (string, error) {
	if f.getIDFunc != nil {
		return f.getIDFunc(ctx, login)
	}
	return "user-" + login, nil
}
