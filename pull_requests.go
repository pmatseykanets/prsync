package main

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/pmatseykanets/prsync/github"
)

// draftState returns the string representation of the draft state of the pull request.
func draftState(draft bool) string {
	if draft {
		return "DRAFT"
	}
	return "PR"
}

// prKey uniquely identifies a pull request by repository and number.
type prKey struct {
	owner  string
	repo   string
	number int
}

// getAuthorsPullRequests returns repository PRs filtered by author and draft status.
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
	PR:
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
					continue PR // PR is already linked to the project.
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
			addCount++
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

			if !ourAuthor {
				if cfg.verbose {
					fmt.Printf("  - %s %s %s %s %s SKIP\n", pr.URL, pr.Author.Login, pr.Title, pr.State, draftState(pr.IsDraft))
				}
				continue
			}
		}

		delete := pr.IsDraft && cfg.pullRequests.delete.drafts ||
			slices.Contains(cfg.pullRequests.delete.states, pr.State)

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
