package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"iter"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pmatseykanets/prsync/github"
	"github.com/pmatseykanets/prsync/version"
	"golang.org/x/oauth2"
)

const httpTimeout = 15 * time.Second

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("Operation canceled")
			os.Exit(130)
		}

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

// run executes the main application logic: parses config, fetches PRs, and syncs the project.
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
