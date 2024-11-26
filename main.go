package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"iter"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

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

type githubClient interface {
	AddAssigneeToPullRequest(ctx context.Context, prID, userID string) error
	AddPullRequestToProject(ctx context.Context, projectID, prID string) error
	DeletePullRequestFromProject(ctx context.Context, projectID, projectItemID string) error
	GetProject(ctx context.Context, owner string, number int) (*github.Project, error)
	GetProjectPullRequests(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error]
	GetRepositoryPullRequests(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error]
	GetTeamMembers(ctx context.Context, owner, name string) ([]github.User, error)
	GetUserOrganizations(ctx context.Context, login string) ([]github.Organization, error)
	LookupUser(ctx context.Context, login string) (*github.User, error)
	IsOrganizationMember(ctx context.Context, login, org string) (bool, error)
}

type authorResolver interface {
	Resolve(ctx context.Context, login string) (bool, error)
	GetID(ctx context.Context, login string) (string, error)
}

func run(ctx context.Context) error {
	var (
		configPath          string
		dryRun, showVersion bool
		verbose             bool
	)
	flag.StringVar(&configPath, "config", "config.yaml", "Path to the config file")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
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
	cfg.verbose = verbose

	fmt.Printf("Config file: %s\n", cfg.path)
	fmt.Printf("  Dry run: %t\n", cfg.dryRun)

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	httpClient.Timeout = httpTimeout

	if err := checkGitHubURL(ctx, cfg.githubURL, httpClient); err != nil {
		return fmt.Errorf("error checking API endpoint: %w", err)
	}

	client := github.NewClient(httpClient, cfg.githubURL)

	startedAt := time.Now()

	authors, err := NewAuthors(ctx, client, cfg)
	if err != nil {
		return err
	}

	project, err := client.GetProject(ctx, cfg.project.owner, cfg.project.number)
	if err != nil {
		return err
	}

	projectPRs, err := getProjectPullRequests(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("error fetching project pull requests: %w", err)
	}

	fmt.Printf("Project: %d %s (%d pull requests)\n", project.Number, project.Title, len(projectPRs))

	err = addNewPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		return err
	}
	err = deleteCompletedPullRequests(ctx, client, cfg, authors, project, projectPRs)
	if err != nil {
		return err
	}

	fmt.Printf("Took %f sec\n", time.Since(startedAt).Seconds())

	return nil
}

// addNewPullRequests adds new pull requests to the project
// based on the author, state, and draft status of the pull request.
func addNewPullRequests(
	ctx context.Context,
	client githubClient,
	cfg config,
	authors authorResolver,
	project *github.Project,
	projectPRs map[prKey]*github.PullRequest,
) error {
	var addCount int
	fmt.Println("Checking for pull requests to add:")
	for _, repository := range cfg.repos {
		fmt.Printf("  - %s/%s\n", repository.owner, repository.name)
		for pr, err := range getAuthorsPullRequests(ctx, client, cfg, authors, repository.owner, repository.name) {
			if err != nil {
				return fmt.Errorf("error fetching authors' pull requests: %w", err)
			}

			key := prKey{owner: pr.Repository.Owner.Login, repo: pr.Repository.Name, number: pr.Number}
			if _, ok := projectPRs[key]; ok {
				if cfg.verbose {
					fmt.Printf("    - %s %s %s %s %s EXISTS\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
				}
				continue
			}

			if cfg.verbose {
				fmt.Printf("    - %s %s %s %s %s NEW \n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
			} else {
				fmt.Printf("    - %s %s %s %s %s\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))

			}

			if !pr.IsAuthorAssigned() && cfg.pullRequests.add.assignAuthor {
				userID, err := authors.GetID(ctx, pr.Author.Login)
				if err != nil {
					return fmt.Errorf("error looking up user %s: %w", pr.Author.Login, err)
				}

				if cfg.verbose {
					fmt.Println("        Assigning author")
				}

				if userID != "" && !cfg.dryRun {
					if err := client.AddAssigneeToPullRequest(ctx, pr.ID, userID); err != nil {
						return fmt.Errorf("error adding assignee %s to the PR %s: %w", pr.Author.Login, pr.URL, err)
					}
				}
			}

			// Sanity check.
			for _, prj := range pr.Projects.Nodes {
				if prj.Owner.Login == cfg.project.owner && prj.Number == cfg.project.number {
					continue // PR is already linked to the project.
				}
			}

			if cfg.verbose {
				fmt.Println("        Adding to project")
			}
			if !cfg.dryRun {
				if err := client.AddPullRequestToProject(ctx, project.ID, pr.ID); err != nil {
					return fmt.Errorf("error adding PR %s to the project: %w", pr.URL, err)
				}
			}
		}
	}

	if addCount > 0 {
		fmt.Printf("Added %d pull requests\n", addCount)
	} else {
		fmt.Println("No pull requests to add")
	}

	return nil
}

// deleteCompletedPullRequests deletes pull requests from the project
// that match the state or draft status.
// It takes authors into consideration if cfg.pullRequests.delete.allAuthors is false.
func deleteCompletedPullRequests(
	ctx context.Context,
	client githubClient,
	cfg config,
	authors authorResolver,
	project *github.Project,
	projectPRs map[prKey]*github.PullRequest,
) error {
	if len(cfg.pullRequests.delete.states) == 0 && !cfg.pullRequests.delete.drafts {
		return nil // Nothing else to do.
	}

	fmt.Println("Checking for pull requests to delete:")

	var deleteCount int
	for _, pr := range projectPRs {
		if !cfg.pullRequests.delete.allAuthors {
			ourAuthor, err := authors.Resolve(ctx, pr.Author.Login)
			if err != nil {
				return fmt.Errorf("error checking if %s is our author: %w", pr.Author.Login, err)
			}

			if cfg.verbose {
				fmt.Printf("  - %s %s %s %s %s SKIP\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
			}

			if !ourAuthor {
				continue
			}
		}

		delete := pr.IsDraft && cfg.pullRequests.delete.drafts
		if !delete {
			for _, state := range cfg.pullRequests.delete.states {
				if pr.State == state {
					delete = true
					break
				}
			}
		}

		if cfg.verbose {
			fmt.Printf("  - %s %s %s %s %s", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
		}

		if !delete {
			if cfg.verbose {
				fmt.Println(" KEEP")
			}
			continue
		}

		deleteCount++

		if cfg.verbose {
			fmt.Println(" DELETE")
		} else {
			fmt.Printf("  - %s %s %s %s %s\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
		}

		if !cfg.dryRun {
			if err := client.DeletePullRequestFromProject(ctx, project.ID, pr.ProjectItemID); err != nil {
				return fmt.Errorf("error deleting PR %s from the project: %w", pr.URL, err)
			}
		}
	}

	if deleteCount > 0 {
		fmt.Printf("Deleted %d pull requests\n", deleteCount)
	} else {
		fmt.Println("No pull requests to delete")
	}

	return nil
}

// draftState returns the string representation of the draft state of the pull request.
func draftState(draft bool) string {
	if draft {
		return "DRAFT"
	}
	return "PR"
}

// checkGitHubURL checks if the provided URL is a valid GitHub API endpoint
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

type prKey struct {
	owner  string
	repo   string
	number int
}

// getProjectPullRequests returns the project information and all of its pull requests.
func getProjectPullRequests(
	ctx context.Context,
	client githubClient,
	cfg config,
) (map[prKey]*github.PullRequest, error) {
	projectPRs := make(map[prKey]*github.PullRequest)

	if cfg.verbose {
		fmt.Println("Fetching project info and pull requests")
	}

	for pr, err := range client.GetProjectPullRequests(ctx, cfg.project.owner, cfg.project.number) {
		if err != nil {
			return nil, fmt.Errorf("error fetching project pull requests: %w", err)
		}

		key := prKey{owner: pr.Repository.Owner.Login, repo: pr.Repository.Name, number: pr.Number}
		projectPRs[key] = pr
	}

	if cfg.verbose {
		keys := slices.Collect(maps.Keys(projectPRs))
		slices.SortFunc(keys, func(i, j prKey) int {
			if i.owner != j.owner {
				return strings.Compare(i.owner, j.owner)
			}
			if i.repo != j.repo {
				return strings.Compare(i.repo, j.repo)
			}
			return i.number - j.number
		})

		var repo string
		for _, key := range keys {
			currentRepo := key.owner + "/" + key.repo
			if repo != currentRepo {
				repo = currentRepo
				fmt.Printf("  - %s\n", repo)
			}

			pr := projectPRs[key]
			fmt.Printf("    - %s %s %s %s %s\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
		}
	}

	return projectPRs, nil
}

// getAuthorsPullRequests returns an iterator that yields pull requests
// for a repository filtered according to the draft status and authors.
// Filtering by the state is done by client.GetRepositoryPullRequests.
func getAuthorsPullRequests(
	ctx context.Context,
	client githubClient,
	cfg config,
	authors authorResolver,
	owner string,
	repo string,
) iter.Seq2[*github.PullRequest, error] {
	return func(yield func(*github.PullRequest, error) bool) {
		for pr, err := range client.GetRepositoryPullRequests(ctx, owner, repo, cfg.pullRequests.add.states) {
			if err != nil {
				yield(nil, fmt.Errorf("error fetching repository pull requests: %w", err))
				return
			}

			// Skip draft PRs.
			if pr.IsDraft && !cfg.pullRequests.add.drafts {
				continue
			}

			// Skip PRs from non-users (e.g. bots).
			if pr.Author.Type != github.AuthorTypeUser {
				continue
			}

			includedAuthor, err := authors.Resolve(ctx, pr.Author.Login)
			if err != nil {
				yield(nil, fmt.Errorf("error evaluating author filter for %s: %w", pr.Author.Login, err))
				return
			}

			if !includedAuthor {
				continue
			}

			if !yield(pr, nil) {
				return
			}
		}
	}
}
