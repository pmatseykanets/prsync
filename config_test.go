package main

import (
	"strings"
	"testing"

	"github.com/pmatseykanets/prsync/github"
)

func TestParseConfigValid(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
  - owner2/repo2
authors:
  include:
    users:
      - alice
      - bob
    teams:
      - myorg/team1
    orgs:
      - myorg
  exclude:
    users:
      - charlie
    teams:
      - myorg/team2
    orgs:
      - otherorg
pullRequests:
  add:
    states:
      - OPEN
    assignAuthor: true
    drafts: true
  delete:
    states:
      - MERGED
      - CLOSED
    drafts: false
    allAuthors: true
`

	cfg, err := parseConfig(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.project.owner != "myorg" || cfg.project.number != 123 {
		t.Errorf("Expected project myorg/123, got %s/%d", cfg.project.owner, cfg.project.number)
	}

	if len(cfg.repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(cfg.repos))
	}
	if cfg.repos[0].owner != "owner1" || cfg.repos[0].name != "repo1" {
		t.Errorf("Expected repo owner1/repo1, got %s/%s", cfg.repos[0].owner, cfg.repos[0].name)
	}

	if len(cfg.authors.include.users) != 2 {
		t.Errorf("Expected 2 included users, got %d", len(cfg.authors.include.users))
	}
	if len(cfg.authors.include.teams) != 1 {
		t.Errorf("Expected 1 included team, got %d", len(cfg.authors.include.teams))
	}
	if len(cfg.authors.include.orgs) != 1 {
		t.Errorf("Expected 1 included org, got %d", len(cfg.authors.include.orgs))
	}

	if len(cfg.authors.exclude.users) != 1 {
		t.Errorf("Expected 1 excluded user, got %d", len(cfg.authors.exclude.users))
	}

	if !cfg.pullRequests.add.assignAuthor {
		t.Error("Expected assignAuthor to be true")
	}
	if !cfg.pullRequests.add.drafts {
		t.Error("Expected drafts to be true")
	}

	if len(cfg.pullRequests.add.states) != 1 || cfg.pullRequests.add.states[0] != github.PullRequestStateOpen {
		t.Error("Expected add state to be OPEN")
	}
	if len(cfg.pullRequests.delete.states) != 2 {
		t.Errorf("Expected 2 delete states, got %d", len(cfg.pullRequests.delete.states))
	}
	if !cfg.pullRequests.delete.allAuthors {
		t.Error("Expected allAuthors to be true")
	}
}

func TestParseConfigMinimal(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
`

	cfg, err := parseConfig(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.githubURL != github.APIEndpoint {
		t.Errorf("Expected default GitHub URL %s, got %s", github.APIEndpoint, cfg.githubURL)
	}

	if len(cfg.pullRequests.add.states) != 1 || cfg.pullRequests.add.states[0] != github.PullRequestStateOpen {
		t.Error("Expected default add state to be OPEN")
	}

	if cfg.authors.include.empty() != true {
		t.Error("Expected empty include rules")
	}
	if cfg.authors.exclude.empty() != true {
		t.Error("Expected empty exclude rules")
	}
}

func TestParseConfigCustomGitHubURL(t *testing.T) {
	yaml := `
github:
  url: https://github.company.com/api/v3
project: myorg/123
repos:
  - owner1/repo1
`

	cfg, err := parseConfig(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "https://github.company.com/api/v3"
	if cfg.githubURL != expected {
		t.Errorf("Expected GitHub URL %s, got %s", expected, cfg.githubURL)
	}
}

func TestParseConfigInvalidProject(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing slash",
			yaml: `
project: myorg123
repos:
  - owner1/repo1
`,
		},
		{
			name: "missing owner",
			yaml: `
project: /123
repos:
  - owner1/repo1
`,
		},
		{
			name: "missing number",
			yaml: `
project: myorg/
repos:
  - owner1/repo1
`,
		},
		{
			name: "non-numeric number",
			yaml: `
project: myorg/abc
repos:
  - owner1/repo1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(strings.NewReader(tt.yaml))
			if err == nil {
				t.Error("Expected error for invalid project format")
			}
		})
	}
}

func TestParseConfigInvalidRepo(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing slash",
			yaml: `
project: myorg/123
repos:
  - owner1repo1
`,
		},
		{
			name: "missing owner",
			yaml: `
project: myorg/123
repos:
  - /repo1
`,
		},
		{
			name: "missing name",
			yaml: `
project: myorg/123
repos:
  - owner1/
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(strings.NewReader(tt.yaml))
			if err == nil {
				t.Error("Expected error for invalid repo format")
			}
		})
	}
}

func TestParseConfigNoRepos(t *testing.T) {
	yaml := `
project: myorg/123
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "no repositories") {
		t.Errorf("Expected 'no repositories' error, got: %v", err)
	}
}

func TestParseConfigInvalidTeam(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "include team missing slash",
			yaml: `
project: myorg/123
repos:
  - owner1/repo1
authors:
  include:
    teams:
      - myorgteam1
`,
		},
		{
			name: "exclude team missing owner",
			yaml: `
project: myorg/123
repos:
  - owner1/repo1
authors:
  exclude:
    teams:
      - /team1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(strings.NewReader(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), "invalid team") {
				t.Errorf("Expected 'invalid team' error, got: %v", err)
			}
		})
	}
}

func TestParseConfigConflictingUsers(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
authors:
  include:
    users:
      - alice
  exclude:
    users:
      - alice
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "can't include and exclude the same user") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
}

func TestParseConfigConflictingTeams(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
authors:
  include:
    teams:
      - myorg/team1
  exclude:
    teams:
      - myorg/team1
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "can't include and exclude the same team") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
}

func TestParseConfigConflictingOrgs(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
authors:
  include:
    orgs:
      - myorg
  exclude:
    orgs:
      - myorg
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "can't include and exclude the same organization") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
}

func TestParseConfigInvalidPRState(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "invalid add state",
			yaml: `
project: myorg/123
repos:
  - owner1/repo1
pullRequests:
  add:
    states:
      - INVALID
`,
		},
		{
			name: "invalid delete state",
			yaml: `
project: myorg/123
repos:
  - owner1/repo1
pullRequests:
  delete:
    states:
      - WRONG
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(strings.NewReader(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), "invalid pullRequest") {
				t.Errorf("Expected invalid state error, got: %v", err)
			}
		})
	}
}

func TestParseConfigConflictingPRStates(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
pullRequests:
  add:
    states:
      - OPEN
      - MERGED
  delete:
    states:
      - MERGED
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "can't add and delete pull requests") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
}

func TestParseConfigInvalidGitHubURL(t *testing.T) {
	yaml := `
github:
  url: "://invalid-url"
project: myorg/123
repos:
  - owner1/repo1
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil || !strings.Contains(err.Error(), "invalid GitHub URL") {
		t.Errorf("Expected invalid URL error, got: %v", err)
	}
}

func TestParseConfigInvalidYAML(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
  invalid yaml structure
    bad indentation
`

	_, err := parseConfig(strings.NewReader(yaml))
	if err == nil {
		t.Error("Expected YAML parsing error")
	}
}

func TestConfigAuthorRulesEmpty(t *testing.T) {
	tests := []struct {
		name  string
		rules configAuthorRules
		want  bool
	}{
		{
			name:  "all empty",
			rules: configAuthorRules{},
			want:  true,
		},
		{
			name: "has users",
			rules: configAuthorRules{
				users: []string{"alice"},
			},
			want: false,
		},
		{
			name: "has teams",
			rules: configAuthorRules{
				teams: []configTeam{{"org", "team"}},
			},
			want: false,
		},
		{
			name: "has orgs",
			rules: configAuthorRules{
				orgs: []string{"myorg"},
			},
			want: false,
		},
		{
			name: "has all",
			rules: configAuthorRules{
				users: []string{"alice"},
				teams: []configTeam{{"org", "team"}},
				orgs:  []string{"myorg"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rules.empty(); got != tt.want {
				t.Errorf("empty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigTeamString(t *testing.T) {
	team := configTeam{owner: "myorg", name: "team1"}
	expected := "myorg/team1"
	if got := team.String(); got != expected {
		t.Errorf("String() = %v, want %v", got, expected)
	}
}

func TestParseConfigCaseInsensitivePRStates(t *testing.T) {
	yaml := `
project: myorg/123
repos:
  - owner1/repo1
pullRequests:
  add:
    states:
      - open
      - Open
      - OPEN
  delete:
    states:
      - merged
      - CLOSED
`

	cfg, err := parseConfig(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(cfg.pullRequests.add.states) != 3 {
		t.Errorf("Expected 3 add states, got %d", len(cfg.pullRequests.add.states))
	}
	for _, state := range cfg.pullRequests.add.states {
		if state != github.PullRequestStateOpen {
			t.Errorf("Expected all states to be OPEN, got %s", state)
		}
	}

	if len(cfg.pullRequests.delete.states) != 2 {
		t.Errorf("Expected 2 delete states, got %d", len(cfg.pullRequests.delete.states))
	}
}

func TestParseConfigGitHubURLTrailingSlash(t *testing.T) {
	yaml := `
github:
  url: https://github.company.com/api/v3/
project: myorg/123
repos:
  - owner1/repo1
`

	cfg, err := parseConfig(strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "https://github.company.com/api/v3"
	if cfg.githubURL != expected {
		t.Errorf("Expected GitHub URL %s (trailing slash removed), got %s", expected, cfg.githubURL)
	}
}
