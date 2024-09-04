package github

import (
	"github.com/machinebox/graphql"
)

func NewPullRequestsRequest(owner, name string, states []PullRequestState, first int, after string) *graphql.Request {
	query := `
  query repositoryPullRequests($owner: String!, $name: String!, $states: [PullRequestState!], $first: Int!, $after: String!) {
      repository(owner: $owner, name: $name) {
          id
          nameWithOwner
          pullRequests(states: $states, first: $first, after: $after) {
              totalCount
              nodes {
                  id
                  number
                  isDraft
                  title
                  createdAt
                  updatedAt
                  author {
                    login
                  }
                  repository {
                    id
                    owner{
                      login
                    }
                    name
                  }
                  url
                  state
                  assignees(first:100) {
                    totalCount
                    nodes {
                      login
                    }
                  }
                  projects: projectsV2(first: 100) {
                    totalCount
                    nodes {
                      id
                      number
                      title
                      owner {
                        ...on Organization {
                            login
                        }
                        ...on User {
                            login
                        }
                      }
                    }
                }
              }
              pageInfo {
                  endCursor
                  hasNextPage
                  hasPreviousPage
                  startCursor
              }
          }
      }
  }`

	req := graphql.NewRequest(query)
	req.Var("owner", owner)
	req.Var("name", name)
	req.Var("states", states)
	req.Var("first", first)
	req.Var("after", after)

	return req
}

func NewTeamMembersRequest(org, team string, first int, after string) *graphql.Request {
	query := `
  query teamMembers($org: String!, $team: String!, $first: Int!, $after: String!) {
    organization(login: $org) {
      team(slug: $team) {
        members (first: $first, after: $after) {
          totalCount
          nodes {
            id
            name
            login
          }
          pageInfo {
            endCursor
            hasNextPage
            hasPreviousPage
            startCursor
          }
        }
      }
    }
  }`

	req := graphql.NewRequest(query)
	req.Var("org", org)
	req.Var("team", team)
	req.Var("first", first)
	req.Var("after", after)

	return req
}

func NewProjectItemsRequest(owner string, number int, first int, after string) *graphql.Request {
	query := `
  query projectPullRequests ($owner: String!, $number: Int!, $first: Int!, $after: String!) {
    organization(login: $owner) {
      projectV2(number: $number) {
        id
        title
        number
        items(first: $first, after: $after) {
          totalCount
          nodes {
            id
            type
            databaseId
            createdAt
            updatedAt
            isArchived
            pullRequest: content {
              ... on PullRequest {
                id
                number
                isDraft
                title
                createdAt
                updatedAt
                author {
                  login
                }
                repository {
                  id
                  owner {
                    login
                  }
                  name
                }
                url
                state
              }
            }
            issue: content {
              ... on Issue {
                id
              }
            }
          }
          pageInfo {
            endCursor
            hasNextPage
            hasPreviousPage
            startCursor
          }
        }
      }
    }
  }`

	req := graphql.NewRequest(query)
	req.Var("owner", owner)
	req.Var("number", number)
	req.Var("first", first)
	req.Var("after", after)

	return req
}

func NewAddPullRequestToProjectRequest(projectId, pullRequestId string) *graphql.Request {
	mutation := `
  mutation addPullRequestToProject($projectId: ID!, $pullRequestId: ID!) {
    addProjectV2ItemById(input: {projectId: $projectId, contentId: $pullRequestId}) {
      item{
        id
        type
        databaseId
        createdAt
        updatedAt
        isArchived
      }
    }
  }`

	req := graphql.NewRequest(mutation)
	req.Var("projectId", projectId)
	req.Var("pullRequestId", pullRequestId)

	return req
}

func NewDeletePullRequestFromProjectRequest(projectId, pullRequestId string) *graphql.Request {
	mutation := `
  mutation deletePullRequestFromProject($projectId: ID!, $pullRequestId: ID!) {
    deleteProjectV2Item(input: {projectId: $projectId, itemId: $pullRequestId}) {
      deletedItemId
    }
  }`

	req := graphql.NewRequest(mutation)
	req.Var("projectId", projectId)
	req.Var("pullRequestId", pullRequestId)

	return req
}

func NewViewerQuery() string {
	return `
  query viewer{
    viewer {
      login
    }
  }`
}

func NewAddAssigneeToPullRequestRequest(pullRequestID, userID string) *graphql.Request {
	mutation := `
  mutation addAssigneeToPullRequest($pullRequestId: ID!, $userId: ID!) {
    addAssigneesToAssignable(input: {assignableId: $pullRequestId, assigneeIds: [$userId]}) {
      assignable {
        assignees(first: 100) {
          totalCount
          nodes {
              login
          }
        }
      }
    }
  }`

	req := graphql.NewRequest(mutation)
	req.Var("pullRequestId", pullRequestID)
	req.Var("userId", userID)

	return req
}

func NewLookupUserRequest(login string) *graphql.Request {
	query := `
  query user($login: String!) {
    user(login: $login) {
      id
      login
      name
    }
  }`

	req := graphql.NewRequest(query)
	req.Var("login", login)

	return req
}
