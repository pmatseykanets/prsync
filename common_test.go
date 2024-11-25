package main

import (
	"context"
	"iter"

	"github.com/pmatseykanets/prsync/github"
)

type fakeGithubClient struct {
	AddAssigneeToPullRequestFunc     func(ctx context.Context, prID, userID string) error
	AddPullRequestToProjectFunc      func(ctx context.Context, projectID, prID string) error
	DeletePullRequestFromProjectFunc func(ctx context.Context, projectID, projectItemID string) error
	GetProjectFunc                   func(ctx context.Context, owner string, number int) (*github.Project, error)
	GetProjectPullRequestsFunc       func(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error]
	GetRepositoryPullRequestsFunc    func(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error]
	GetTeamMembersFunc               func(ctx context.Context, owner, name string) ([]github.User, error)
	GetUserOrganizationsFunc         func(ctx context.Context, login string) ([]github.Organization, error)
	LookupUserFunc                   func(ctx context.Context, login string) (*github.User, error)
	IsOrganizationMemberFunc         func(ctx context.Context, login, org string) (bool, error)
}

func (c *fakeGithubClient) AddAssigneeToPullRequest(ctx context.Context, prID, userID string) error {
	if c.AddAssigneeToPullRequestFunc != nil {
		return c.AddAssigneeToPullRequestFunc(ctx, prID, userID)
	}
	return nil
}
func (c *fakeGithubClient) AddPullRequestToProject(ctx context.Context, projectID, prID string) error {
	if c.AddPullRequestToProjectFunc != nil {
		return c.AddPullRequestToProjectFunc(ctx, projectID, prID)
	}
	return nil
}
func (c *fakeGithubClient) DeletePullRequestFromProject(ctx context.Context, projectID, projectItemID string) error {
	if c.DeletePullRequestFromProjectFunc != nil {
		return c.DeletePullRequestFromProjectFunc(ctx, projectID, projectItemID)
	}
	return nil
}
func (c *fakeGithubClient) GetProject(ctx context.Context, owner string, number int) (*github.Project, error) {
	if c.GetProjectFunc != nil {
		return c.GetProjectFunc(ctx, owner, number)
	}
	return nil, nil
}
func (c *fakeGithubClient) GetProjectPullRequests(ctx context.Context, owner string, number int) iter.Seq2[*github.PullRequest, error] {
	if c.GetProjectPullRequestsFunc != nil {
		return c.GetProjectPullRequestsFunc(ctx, owner, number)
	}
	return nil
}
func (c *fakeGithubClient) GetRepositoryPullRequests(ctx context.Context, owner string, name string, states []github.PullRequestState) iter.Seq2[*github.PullRequest, error] {
	if c.GetRepositoryPullRequestsFunc != nil {
		return c.GetRepositoryPullRequestsFunc(ctx, owner, name, states)
	}
	return nil
}
func (c *fakeGithubClient) GetTeamMembers(ctx context.Context, owner, name string) ([]github.User, error) {
	if c.GetTeamMembersFunc != nil {
		return c.GetTeamMembersFunc(ctx, owner, name)
	}
	return nil, nil
}
func (c *fakeGithubClient) GetUserOrganizations(ctx context.Context, login string) ([]github.Organization, error) {
	if c.GetUserOrganizationsFunc != nil {
		return c.GetUserOrganizationsFunc(ctx, login)
	}
	return nil, nil
}
func (c *fakeGithubClient) LookupUser(ctx context.Context, login string) (*github.User, error) {
	if c.LookupUserFunc != nil {
		return c.LookupUserFunc(ctx, login)
	}
	return nil, nil
}
func (c *fakeGithubClient) IsOrganizationMember(ctx context.Context, login, org string) (bool, error) {
	if c.IsOrganizationMemberFunc != nil {
		return c.IsOrganizationMemberFunc(ctx, login, org)
	}
	return false, nil
}
