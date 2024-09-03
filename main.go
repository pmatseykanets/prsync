package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/machinebox/graphql"
	"github.com/pmatseykanets/prsync/github"
	"github.com/pmatseykanets/prsync/version"
	"golang.org/x/oauth2"
)

const httpTimeout = 15 * time.Second

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var (
		configPath          string
		dryRun, showVersion bool
	)
	flag.StringVar(&configPath, "config", "config.yaml", "Path to the config file")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run")
	flag.BoolVar(&showVersion, "version", showVersion, "Print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("prsync version %s\n", version.Version)
		return nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	cfgRaw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config %s: %w", configPath, err)
	}

	cfg, err := parseConfig(bytes.NewReader(cfgRaw))
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	cfg.path = configPath
	cfg.dryRun = dryRun

	fmt.Printf("Config file: %s\n", cfg.path)
	fmt.Printf("Dry run: %t\n", cfg.dryRun)

	if err := checkGitHubURL(ctx, cfg.githubURL, token); err != nil {
		return fmt.Errorf("error checking API endpoint: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	httpClient.Timeout = httpTimeout
	client := graphql.NewClient(cfg.githubURL, graphql.WithHTTPClient(httpClient))

	startedAt := time.Now()

	authors := make(map[string]struct{})
	{
		if cfg.team.name != "" {
			members, err := getTeamMembers(ctx, client, cfg.team.owner, cfg.team.name)
			if err != nil {
				return fmt.Errorf("error fetching team members: %w", err)
			}

			for _, member := range members {
				authors[member.Login] = struct{}{}
			}
		}
		for _, author := range cfg.authors.include {
			authors[author] = struct{}{}
		}
		for _, author := range cfg.authors.exclude {
			delete(authors, author)
		}
	}

	var project *github.Project

	type prKey struct {
		owner  string
		repo   string
		number int
	}
	projectPRs := make(map[prKey]github.PullRequest)
	{
		prj, prs, err := getProjectPullRequests(ctx, client, cfg.project.owner, cfg.project.number)
		if err != nil {
			return fmt.Errorf("error fetching project pull requests: %w", err)
		}
		project = prj

		fmt.Printf("Project: %d %s\n", project.Number, project.Title)

		for _, pr := range prs {
			key := prKey{owner: pr.Repository.Owner.Login, repo: pr.Repository.Name, number: pr.Number}
			projectPRs[key] = pr
		}
	}

	fmt.Printf("Authors: %d\n", len(authors))

	fmt.Println("Repositories:")
	var teamPRs []github.PullRequest
	for _, repository := range cfg.repos {
		prs, err := getRepositoryPullRequests(ctx, client, repository.owner, repository.name, authors)
		if err != nil {
			return fmt.Errorf("error fetching repository pull requests: %w", err)
		}

		teamPRs = append(teamPRs, prs...)
		fmt.Printf("  - %s/%s (%d)\n", repository.owner, repository.name, len(prs))
	}

	var newCount int
	for _, pr := range teamPRs {
		key := prKey{owner: pr.Repository.Owner.Login, repo: pr.Repository.Name, number: pr.Number}
		if _, ok := projectPRs[key]; ok {
			continue
		}

		newCount++
		if newCount == 1 {
			fmt.Println("New Pull Requests:")
		}
		fmt.Printf("  - %s %s %s\n", pr.URL, pr.Author.Login, pr.Title)
		if !cfg.dryRun {
			if err := addPullRequestToProject(ctx, client, project.ID, pr.ID); err != nil {
				fmt.Printf("Error adding PR %s to the project: %v\n", pr.URL, err)
			}
		}
	}
	fmt.Printf("Added %d new pull requests in %f sec\n", newCount, time.Since(startedAt).Seconds())

	return nil
}

func checkGitHubURL(ctx context.Context, url, token string) error {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	httpClient.Timeout = httpTimeout

	body, err := json.Marshal(struct {
		Query string `json:"query"`
	}{
		Query: github.NewViewerQuery(),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

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

func getRepositoryPullRequests(
	ctx context.Context,
	client *graphql.Client,
	owner string,
	name string,
	authors map[string]struct{},
) ([]github.PullRequest, error) {
	var (
		prs   []github.PullRequest
		after string
	)

	for {
		var resp github.PullRequestResponse

		req := graphql.NewRequest(github.NewPullRequestQuery(owner, name, 100, after))
		if err := client.Run(ctx, req, &resp); err != nil {
			return nil, err
		}
		if resp.Errors != nil {
			return nil, resp.Errors
		}

		if resp.Repository == nil {
			return nil, fmt.Errorf("repository not found")
		}

		for _, pr := range resp.Repository.PullRequests.Nodes {
			// Filter PRs by the author.
			if _, ok := authors[pr.Author.Login]; !ok {
				continue
			}
			// Skip draft PRs.
			if pr.IsDraft {
				continue
			}

			prs = append(prs, pr)
		}

		if !resp.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}

		after = resp.Repository.PullRequests.PageInfo.EndCursor
	}

	return prs, nil
}

func getProjectPullRequests(
	ctx context.Context,
	client *graphql.Client,
	owner string,
	number int,
) (*github.Project, []github.PullRequest, error) {
	var (
		prs     []github.PullRequest
		after   string
		project *github.Project
	)

	for {
		var resp github.ProjectItemsResponse

		req := graphql.NewRequest(github.NewProjectItemsQuery(owner, number, 100, after))
		if err := client.Run(ctx, req, &resp); err != nil {
			return nil, nil, err
		}
		if resp.Errors != nil {
			return nil, nil, resp.Errors
		}

		if resp.Organization == nil || resp.Organization.Project == nil {
			return nil, nil, fmt.Errorf("project not found")
		}

		if project == nil {
			project = &github.Project{
				ID:     resp.Organization.Project.ID,
				Number: resp.Organization.Project.Number,
				Title:  resp.Organization.Project.Title,
			}
		}

		for _, item := range resp.Organization.Project.Items.Nodes {
			if item.Type != github.ProjectItemTypePullRequest {
				continue
			}

			prs = append(prs, *item.PullRequest)
		}

		if !resp.Organization.Project.Items.PageInfo.HasNextPage {
			break
		}

		after = resp.Organization.Project.Items.PageInfo.EndCursor
	}

	return project, prs, nil
}

func getTeamMembers(ctx context.Context, client *graphql.Client, teamOrg, teamName string) ([]github.User, error) {
	var resp github.TeamMembersResponse

	req := graphql.NewRequest(github.NewTeamMembersQuery(teamOrg, teamName, 100, ""))
	if err := client.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, resp.Errors
	}
	if resp.Organization == nil || resp.Organization.Team == nil {
		return nil, fmt.Errorf("team not found")
	}

	return resp.Organization.Team.Members.Nodes, nil
}

func addPullRequestToProject(ctx context.Context, client *graphql.Client, projectID, pullRequestID string) error {
	var resp github.AddPullRequestToProjectResponse

	req := graphql.NewRequest(github.NewAddPullRequestToProjectMutation(projectID, pullRequestID))
	if err := client.Run(ctx, req, &resp); err != nil {
		return err
	}
	if resp.Errors != nil {
		return resp.Errors
	}

	return nil
}
