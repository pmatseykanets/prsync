package github

import (
	"testing"
)

func TestPullRequestStateIsValid(t *testing.T) {
	tests := []struct {
		name  string
		state PullRequestState
		want  bool
	}{
		{
			name:  "valid OPEN",
			state: PullRequestStateOpen,
			want:  true,
		},
		{
			name:  "valid CLOSED",
			state: PullRequestStateClosed,
			want:  true,
		},
		{
			name:  "valid MERGED",
			state: PullRequestStateMerged,
			want:  true,
		},
		{
			name:  "invalid state",
			state: PullRequestState("INVALID"),
			want:  false,
		},
		{
			name:  "empty state",
			state: PullRequestState(""),
			want:  false,
		},
		{
			name:  "lowercase state",
			state: PullRequestState("open"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Errorf("PullRequestState.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPullRequestIsAuthorAssignedTrue(t *testing.T) {
	pr := &PullRequest{
		Author: Author{Login: "alice"},
		Assignees: struct {
			TotalCount int      `json:"totalCount"`
			Nodes      []User   `json:"nodes"`
			PageInfo   PageInfo `json:"pageInfo"`
		}{
			Nodes: []User{
				{Login: "bob"},
				{Login: "alice"},
				{Login: "charlie"},
			},
		},
	}

	if !pr.IsAuthorAssigned() {
		t.Error("Expected author to be assigned")
	}
}

func TestPullRequestIsAuthorAssignedFalse(t *testing.T) {
	pr := &PullRequest{
		Author: Author{Login: "alice"},
		Assignees: struct {
			TotalCount int      `json:"totalCount"`
			Nodes      []User   `json:"nodes"`
			PageInfo   PageInfo `json:"pageInfo"`
		}{
			Nodes: []User{
				{Login: "bob"},
				{Login: "charlie"},
			},
		},
	}

	if pr.IsAuthorAssigned() {
		t.Error("Expected author not to be assigned")
	}
}

func TestPullRequestIsAuthorAssignedEmpty(t *testing.T) {
	pr := &PullRequest{
		Author: Author{Login: "alice"},
		Assignees: struct {
			TotalCount int      `json:"totalCount"`
			Nodes      []User   `json:"nodes"`
			PageInfo   PageInfo `json:"pageInfo"`
		}{
			Nodes: []User{},
		},
	}

	if pr.IsAuthorAssigned() {
		t.Error("Expected author not to be assigned when no assignees")
	}
}

func TestPullRequestIsAuthorAssignedCaseSensitive(t *testing.T) {
	pr := &PullRequest{
		Author: Author{Login: "Alice"},
		Assignees: struct {
			TotalCount int      `json:"totalCount"`
			Nodes      []User   `json:"nodes"`
			PageInfo   PageInfo `json:"pageInfo"`
		}{
			Nodes: []User{
				{Login: "alice"}, // Different case
			},
		},
	}

	if pr.IsAuthorAssigned() {
		t.Error("Expected case-sensitive match to fail")
	}
}

func TestErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "with type and message",
			err: Error{
				Type:    "RATE_LIMITED",
				Message: "API rate limit exceeded",
			},
			want: "RATE_LIMITED: API rate limit exceeded",
		},
		{
			name: "empty type",
			err: Error{
				Type:    "",
				Message: "Some error",
			},
			want: ": Some error",
		},
		{
			name: "empty message",
			err: Error{
				Type:    "ERROR",
				Message: "",
			},
			want: "ERROR: ",
		},
		{
			name: "both empty",
			err: Error{
				Type:    "",
				Message: "",
			},
			want: ": ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorsError(t *testing.T) {
	tests := []struct {
		name   string
		errors Errors
		want   string
	}{
		{
			name: "single error",
			errors: Errors{
				{Type: "ERROR", Message: "Something went wrong"},
			},
			want: "ERROR: Something went wrong",
		},
		{
			name: "multiple errors",
			errors: Errors{
				{Type: "ERROR1", Message: "First error"},
				{Type: "ERROR2", Message: "Second error"},
			},
			want: "ERROR1: First error\nERROR2: Second error",
		},
		{
			name:   "empty errors",
			errors: Errors{},
			want:   "",
		},
		{
			name: "three errors",
			errors: Errors{
				{Type: "A", Message: "Error A"},
				{Type: "B", Message: "Error B"},
				{Type: "C", Message: "Error C"},
			},
			want: "A: Error A\nB: Error B\nC: Error C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errors.Error(); got != tt.want {
				t.Errorf("Errors.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorsErrorNoTrailingNewline(t *testing.T) {
	errors := Errors{
		{Type: "ERROR", Message: "Test"},
	}

	result := errors.Error()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		t.Error("Errors.Error() should not have trailing newline")
	}
}

func TestAuthorTypeConstants(t *testing.T) {
	if AuthorTypeBot != "Bot" {
		t.Errorf("AuthorTypeBot = %q, want %q", AuthorTypeBot, "Bot")
	}
	if AuthorTypeUser != "User" {
		t.Errorf("AuthorTypeUser = %q, want %q", AuthorTypeUser, "User")
	}
}

func TestPullRequestStateConstants(t *testing.T) {
	if PullRequestStateOpen != "OPEN" {
		t.Errorf("PullRequestStateOpen = %q, want %q", PullRequestStateOpen, "OPEN")
	}
	if PullRequestStateClosed != "CLOSED" {
		t.Errorf("PullRequestStateClosed = %q, want %q", PullRequestStateClosed, "CLOSED")
	}
	if PullRequestStateMerged != "MERGED" {
		t.Errorf("PullRequestStateMerged = %q, want %q", PullRequestStateMerged, "MERGED")
	}
}

func TestProjectItemTypeConstants(t *testing.T) {
	if ProjectItemTypeIssue != "ISSUE" {
		t.Errorf("ProjectItemTypeIssue = %q, want %q", ProjectItemTypeIssue, "ISSUE")
	}
	if ProjectItemTypePullRequest != "PULL_REQUEST" {
		t.Errorf("ProjectItemTypePullRequest = %q, want %q", ProjectItemTypePullRequest, "PULL_REQUEST")
	}
}

func TestAPIEndpointConstant(t *testing.T) {
	expected := "https://api.github.com"
	if APIEndpoint != expected {
		t.Errorf("APIEndpoint = %q, want %q", APIEndpoint, expected)
	}
}
