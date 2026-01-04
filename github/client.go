package github

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/machinebox/graphql"
)

// Client wraps HTTP and GraphQL clients for GitHub API access.
type Client struct {
	githubURL string
	http      *http.Client
	graphql   *graphql.Client
}

// NewClient creates a GitHub API client with the given HTTP client and API endpoint.
func NewClient(httpClient *http.Client, githubURL string) *Client {
	return &Client{
		githubURL: githubURL,
		http:      httpClient,
		graphql:   graphql.NewClient(githubURL+"/graphql", graphql.WithHTTPClient(httpClient)),
	}
}

// GetRepositoryPullRequests returns an iterator over all PRs in a repository matching the given states.
func (c *Client) GetRepositoryPullRequests(ctx context.Context, owner string, name string, states []PullRequestState) iter.Seq2[*PullRequest, error] {
	return func(yield func(*PullRequest, error) bool) {
		var after string
		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			var resp PullRequestResponse
			req := NewPullRequestsRequest(owner, name, states, 100, after)
			if err := c.graphql.Run(ctx, req, &resp); err != nil {
				yield(nil, err)
				return
			}
			if resp.Errors != nil {
				yield(nil, resp.Errors)
				return
			}

			if resp.Repository == nil {
				yield(nil, fmt.Errorf("repository not found"))
				return
			}

			for _, pr := range resp.Repository.PullRequests.Nodes {
				if !yield(&pr, nil) {
					return
				}
			}

			if !resp.Repository.PullRequests.PageInfo.HasNextPage {
				break
			}

			after = resp.Repository.PullRequests.PageInfo.EndCursor
		}
	}
}

// GetProject fetches project metadata by owner and number.
func (c *Client) GetProject(ctx context.Context, owner string, number int) (*Project, error) {
	var resp ProjectResponse

	req := NewProjectRequest(owner, number)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, resp.Errors
	}

	if resp.Organization == nil || resp.Organization.Project == nil {
		return nil, fmt.Errorf("project not found")
	}

	return &Project{
		ID:     resp.Organization.Project.ID,
		Number: resp.Organization.Project.Number,
		Title:  resp.Organization.Project.Title,
	}, nil
}

// GetProjectPullRequests returns an iterator over all PRs in a project.
func (c *Client) GetProjectPullRequests(ctx context.Context, owner string, number int) iter.Seq2[*PullRequest, error] {
	return func(yield func(*PullRequest, error) bool) {
		var after string
		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			var resp ProjectItemsResponse
			req := NewProjectItemsRequest(owner, number, 100, after)
			if err := c.graphql.Run(ctx, req, &resp); err != nil {
				yield(nil, err)
				return
			}
			if resp.Errors != nil {
				yield(nil, resp.Errors)
				return
			}

			if resp.Organization == nil || resp.Organization.Project == nil {
				yield(nil, fmt.Errorf("project not found"))
				return
			}

			for _, item := range resp.Organization.Project.Items.Nodes {
				if item.Type != ProjectItemTypePullRequest {
					continue
				}

				pr := *item.PullRequest
				pr.ProjectItemID = item.ID

				if !yield(&pr, nil) {
					return
				}
			}

			if !resp.Organization.Project.Items.PageInfo.HasNextPage {
				break
			}

			after = resp.Organization.Project.Items.PageInfo.EndCursor
		}
	}
}

// GetTeamMembers fetches all members of a GitHub team.
func (c *Client) GetTeamMembers(ctx context.Context, teamOrg, teamName string) ([]User, error) {
	var allMembers []User
	var after string

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var resp TeamMembersResponse
		req := NewTeamMembersRequest(teamOrg, teamName, 100, after)
		if err := c.graphql.Run(ctx, req, &resp); err != nil {
			return nil, err
		}
		if resp.Errors != nil {
			return nil, resp.Errors
		}
		if resp.Organization == nil || resp.Organization.Team == nil {
			return nil, fmt.Errorf("team not found")
		}

		allMembers = append(allMembers, resp.Organization.Team.Members.Nodes...)

		if !resp.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}
		after = resp.Organization.Team.Members.PageInfo.EndCursor
	}

	return allMembers, nil
}

// AddPullRequestToProject adds a PR to a project board.
func (c *Client) AddPullRequestToProject(ctx context.Context, projectID, pullRequestID string) error {
	var resp AddPullRequestToProjectResponse

	req := NewAddPullRequestToProjectRequest(projectID, pullRequestID)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return err
	}
	if resp.Errors != nil {
		return resp.Errors
	}

	return nil
}

// DeletePullRequestFromProject removes a PR from a project board.
func (c *Client) DeletePullRequestFromProject(ctx context.Context, projectID, itemID string) error {
	var resp DeletePullRequestFromProjectResponse

	req := NewDeletePullRequestFromProjectRequest(projectID, itemID)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return err
	}
	if resp.Errors != nil {
		return resp.Errors
	}

	return nil
}

// AddAssigneeToPullRequest assigns a user to a PR.
func (c *Client) AddAssigneeToPullRequest(ctx context.Context, pullRequestID, userID string) error {
	var resp AddAssigneeToPullRequestResponse

	req := NewAddAssigneeToPullRequestRequest(pullRequestID, userID)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return err
	}
	if resp.Errors != nil {
		return resp.Errors
	}

	return nil
}

// LookupUser fetches user information by login.
func (c *Client) LookupUser(ctx context.Context, login string) (*User, error) {
	var resp LookupUserResponse

	req := NewLookupUserRequest(login)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, resp.Errors
	}

	if resp.User == nil {
		return nil, fmt.Errorf("user not found")
	}

	return resp.User, nil
}

// GetUserOrganizations fetches all organizations a user belongs to.
func (c *Client) GetUserOrganizations(ctx context.Context, login string) ([]Organization, error) {
	var resp LookupUserMembershipResponse

	req := NewLookupUserMembershipRequest(login)
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	if resp.Errors != nil {
		return nil, resp.Errors
	}

	return resp.User.Organizations.Nodes, nil
}

// IsOrganizationMember checks if a user is a member of an organization.
func (c *Client) IsOrganizationMember(ctx context.Context, login, org string) (bool, error) {
	url := c.githubURL + "/orgs/" + url.PathEscape(org) + "/members/" + url.PathEscape(login)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		// 204	If requester is an organization member and user is a member
		return true, nil
	case http.StatusFound, http.StatusNotFound:
		// 302	If requester is not an organization member
		// 404	If requester is an organization member and user is not a member
		return false, nil
	default:
		message := http.StatusText(resp.StatusCode)

		githubError := &Error{}
		err = json.NewDecoder(resp.Body).Decode(githubError)
		if err == nil && githubError.Message != "" {
			message = githubError.Message
		}

		return false, fmt.Errorf("%d %s", resp.StatusCode, message)
	}
}
