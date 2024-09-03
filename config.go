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

type configProject struct {
	owner  string
	number int
}

type configTeam struct {
	owner string
	name  string
}

type configRepo struct {
	owner string
	name  string
}

type configAuthors struct {
	include []string
	exclude []string
}

type config struct {
	path      string
	githubURL string
	project   configProject
	team      configTeam
	repos     []configRepo
	authors   configAuthors
	dryRun    bool
}

type configFile struct {
	GitHub struct {
		URL string `yaml:"url"`
	}
	Project string   `yaml:"project"`
	Team    string   `yaml:"team"`
	Repos   []string `yaml:"repos"`
	Authors struct {
		Include []string `yaml:"include"`
		Exclude []string `yaml:"exclude"`
	} `yaml:"authors"`
}

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
	if cfgFile.GitHub.URL != "" {
		if _, err := url.Parse(cfgFile.GitHub.URL); err != nil {
			return config{}, fmt.Errorf("invalid GitHub URL: %s: %w", cfgFile.GitHub.URL, err)
		}
		cfg.githubURL = cfgFile.GitHub.URL
	}

	owner, number, ok := strings.Cut(cfgFile.Project, "/")
	if !ok || owner == "" || number == "" {
		return config{}, fmt.Errorf("invalid project: %s", cfgFile.Project)
	}
	cfg.project.owner = owner
	if cfg.project.number, err = strconv.Atoi(number); err != nil {
		return config{}, fmt.Errorf("invalid project number: %s: %w", cfgFile.Project, err)
	}

	if cfgFile.Team != "" {
		owner, name, ok := strings.Cut(cfgFile.Team, "/")
		if !ok || owner == "" || name == "" {
			return config{}, fmt.Errorf("invalid team: %s", cfgFile.Team)
		}
		cfg.team.owner = owner
		cfg.team.name = name
	}

	for _, repo := range cfgFile.Repos {
		owner, name, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || name == "" {
			return config{}, fmt.Errorf("invalid repo: %s", repo)
		}
		cfg.repos = append(cfg.repos, configRepo{owner, name})
	}
	if len(cfg.repos) == 0 {
		return config{}, fmt.Errorf("no repositories specified")
	}

	if cfg.team.name == "" && len(cfgFile.Authors.Include) == 0 {
		return config{}, fmt.Errorf("neither team or authors specified")
	}

	cfg.authors.include = cfgFile.Authors.Include
	cfg.authors.exclude = cfgFile.Authors.Exclude

	return cfg, nil
}
