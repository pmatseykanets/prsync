package main

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/pmatseykanets/prsync/github"
	"gopkg.in/yaml.v3"
)

// configProject represents a GitHub project by owner and number.
type configProject struct {
	owner  string
	number int
}

// configTeam represents a GitHub team by owner and name.
type configTeam struct {
	owner string
	name  string
}

func (t configTeam) String() string {
	return fmt.Sprintf("%s/%s", t.owner, t.name)
}

// configRepo represents a GitHub repository by owner and name.
type configRepo struct {
	owner string
	name  string
}

// configAuthorRules defines include/exclude rules for users, teams, and orgs.
type configAuthorRules struct {
	users []string
	teams []configTeam
	orgs  []string
}

// configAuthors contains both include and exclude author rules.
type configAuthors struct {
	include configAuthorRules
	exclude configAuthorRules
}

// empty returns true if no rules are defined.
func (r *configAuthorRules) empty() bool {
	return len(r.users) == 0 && len(r.teams) == 0 && len(r.orgs) == 0
}

// config holds the parsed and validated configuration.
type config struct {
	path         string
	githubURL    string
	project      configProject
	repos        []configRepo
	authors      configAuthors
	pullRequests struct {
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
	}
	dryRun  bool
	verbose bool
}

// configFile matches the YAML structure for unmarshaling.
type configFile struct {
	GitHub struct {
		URL string `yaml:"url"`
	}
	Project string   `yaml:"project"`
	Repos   []string `yaml:"repos"`
	Authors struct {
		Include struct {
			Users []string `yaml:"users"`
			Teams []string `yaml:"teams"`
			Orgs  []string `yaml:"orgs"`
		} `yaml:"include"`
		Exclude struct {
			Users []string `yaml:"users"`
			Teams []string `yaml:"teams"`
			Orgs  []string `yaml:"orgs"`
		} `yaml:"exclude"`
	} `yaml:"authors"`
	PullRequests struct {
		AssignAuthor        bool `yaml:"assignAuthor"`
		IncludeDrafts       bool `yaml:"includeDrafts"`
		DeleteMerged        bool `yaml:"deleteMerged"`
		DeleteClosed        bool `yaml:"deleteClosed"`
		DeleteForAllAuthors bool `yaml:"deleteForAllAuthors"`
		Add                 struct {
			States       []string `yaml:"states"`
			AssignAuthor bool     `yaml:"assignAuthor"`
			Drafts       bool     `yaml:"drafts"`
		} `yaml:"add"`
		Delete struct {
			States     []string `yaml:"states"`
			Drafts     bool     `yaml:"drafts"`
			AllAuthors bool     `yaml:"allAuthors"`
		} `yaml:"delete"`
	} `yaml:"pullRequests"`
}

// parseConfig reads and validates the YAML configuration.
func parseConfig(r io.Reader) (config, error) {
	var (
		cfgFile configFile
		cfg     config
		err     error
	)
	if err = yaml.NewDecoder(r).Decode(&cfgFile); err != nil {
		return config{}, err
	}

	cfg.githubURL = github.APIEndpoint
	configuredURL := strings.TrimSuffix(cfgFile.GitHub.URL, "/")
	if configuredURL != "" {
		if _, err := url.Parse(configuredURL); err != nil {
			return config{}, fmt.Errorf("invalid GitHub URL: %s: %w", cfgFile.GitHub.URL, err)
		}
		cfg.githubURL = configuredURL
	}

	owner, number, ok := strings.Cut(cfgFile.Project, "/")
	if !ok || owner == "" || number == "" {
		return config{}, fmt.Errorf("invalid project: %s", cfgFile.Project)
	}
	cfg.project.owner = owner
	if cfg.project.number, err = strconv.Atoi(number); err != nil {
		return config{}, fmt.Errorf("invalid project number: %s: %w", cfgFile.Project, err)
	}

	for _, repo := range cfgFile.Repos {
		owner, name, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || name == "" {
			return config{}, fmt.Errorf("invalid repository: %s", repo)
		}
		cfg.repos = append(cfg.repos, configRepo{owner, name})
	}
	if len(cfg.repos) == 0 {
		return config{}, fmt.Errorf("no repositories specified")
	}

	for _, teamName := range cfgFile.Authors.Include.Teams {
		owner, name, ok := strings.Cut(teamName, "/")
		if !ok || owner == "" || name == "" {
			return config{}, fmt.Errorf("invalid team: %s", teamName)
		}
		cfg.authors.include.teams = append(cfg.authors.include.teams, configTeam{owner, name})
	}

	for _, teamName := range cfgFile.Authors.Exclude.Teams {
		owner, name, ok := strings.Cut(teamName, "/")
		if !ok || owner == "" || name == "" {
			return config{}, fmt.Errorf("invalid team: %s", teamName)
		}

		for _, included := range cfg.authors.include.teams {
			if included.owner == owner && included.name == name {
				return config{}, fmt.Errorf("can't include and exclude the same team: %s", teamName)
			}
		}

		cfg.authors.exclude.teams = append(cfg.authors.exclude.teams, configTeam{owner, name})
	}

	cfg.authors.include.users = cfgFile.Authors.Include.Users
	cfg.authors.exclude.users = cfgFile.Authors.Exclude.Users
	for _, included := range cfg.authors.include.users {
		for _, excluded := range cfg.authors.exclude.users {
			if included == excluded {
				return config{}, fmt.Errorf("can't include and exclude the same user: %s", included)
			}
		}
	}

	cfg.authors.include.orgs = cfgFile.Authors.Include.Orgs
	cfg.authors.exclude.orgs = cfgFile.Authors.Exclude.Orgs
	for _, included := range cfg.authors.include.orgs {
		for _, excluded := range cfg.authors.exclude.orgs {
			if included == excluded {
				return config{}, fmt.Errorf("can't include and exclude the same organization: %s", included)
			}
		}
	}

	cfg.pullRequests.add.assignAuthor = cfgFile.PullRequests.Add.AssignAuthor
	cfg.pullRequests.add.drafts = cfgFile.PullRequests.Add.Drafts

	cfg.pullRequests.delete.drafts = cfgFile.PullRequests.Delete.Drafts
	cfg.pullRequests.delete.allAuthors = cfgFile.PullRequests.Delete.AllAuthors

	for _, state := range cfgFile.PullRequests.Add.States {
		prState := github.PullRequestState(strings.ToUpper(state))
		if !prState.IsValid() {
			return config{}, fmt.Errorf("invalid pullRequest.add state: %s", state)
		}
		cfg.pullRequests.add.states = append(cfg.pullRequests.add.states, prState)
	}
	for _, state := range cfgFile.PullRequests.Delete.States {
		prState := github.PullRequestState(strings.ToUpper(state))
		if !prState.IsValid() {
			return config{}, fmt.Errorf("invalid pullRequest.delete state: %s", state)
		}
		for _, addState := range cfg.pullRequests.add.states {
			if prState == addState {
				return config{}, fmt.Errorf("can't add and delete pull requests in %s state", state)
			}
		}
		cfg.pullRequests.delete.states = append(cfg.pullRequests.delete.states, prState)
	}

	if len(cfgFile.PullRequests.Add.States) == 0 {
		// By default, add pull requests in OPEN state.
		cfg.pullRequests.add.states = []github.PullRequestState{github.PullRequestStateOpen}
	}

	return cfg, nil
}
