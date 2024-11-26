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

type Client struct {
	githubURL string
	http      *http.Client
	graphql   *graphql.Client
}

func NewClient(httpClient *http.Client, githubURL string) *Client {
	return &Client{
		githubURL: githubURL,
		http:      httpClient,
		graphql:   graphql.NewClient(githubURL+"/graphql", graphql.WithHTTPClient(httpClient)),
	}
}

func (c *Client) GetRepositoryPullRequests(ctx context.Context, owner string, name string, states []PullRequestState) iter.Seq2[*PullRequest, error] {
	return func(yield func(*PullRequest, error) bool) {
		var after string
		for {
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

func (c *Client) GetProjectPullRequests(ctx context.Context, owner string, number int) iter.Seq2[*PullRequest, error] {
	return func(yield func(*PullRequest, error) bool) {
		var after string
		for {
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

func (c *Client) GetTeamMembers(ctx context.Context, teamOrg, teamName string) ([]User, error) {
	var resp TeamMembersResponse

	req := NewTeamMembersRequest(teamOrg, teamName, 100, "")
	if err := c.graphql.Run(ctx, req, &resp); err != nil {
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
	defer resp.Body.Close()

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
