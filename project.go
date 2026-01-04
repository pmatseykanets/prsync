package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pmatseykanets/prsync/github"
)

// getProjectPullRequests fetches all pull requests currently in the project.
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
