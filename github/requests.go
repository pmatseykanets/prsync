package github

import (
	"fmt"

	"github.com/machinebox/graphql"
)

func NewPullRequestsRequest(owner, name string, first int, after string) *graphql.Request {
	pullRequestQuery := `
  query pulls {
      repository(owner: "%s", name: "%s") {
          id
          nameWithOwner
          pullRequests(states: OPEN, first: %d, after: "%s") {
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

	return graphql.NewRequest(fmt.Sprintf(pullRequestQuery, owner, name, first, after))
}

func NewTeamMembersRequest(org, team string, first int, after string) *graphql.Request {
	teamMembersQuery := `
  query teams{
    organization(login: "%s"){
      team(slug: "%s"){
        members (first: %d, after: "%s") {
          totalCount
          nodes{
            id
            name
            login
          }
          pageInfo{
            endCursor
            hasNextPage
            hasPreviousPage
            startCursor
          }
        }
      }
    }
  }`

	return graphql.NewRequest(fmt.Sprintf(teamMembersQuery, org, team, first, after))
}

func NewProjectItemsRequest(owner string, number int, first int, after string) *graphql.Request {
	projectItemsQuery := `
  query projectPullRequests{
    organization(login: "%s"){
      projectV2(number: %d){
        id
        title
        number
        items(first: %d, after: "%s") {
          totalCount
          nodes{
            id
            type
            databaseId
            createdAt
            updatedAt
            isArchived
            pullRequest: content{
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
                  owner{
                    login
                  }
                  name
                }
                url
                state
              }
            }
            issue: content{
              ... on Issue {
                id
              }
            }
          }
          pageInfo{
            endCursor
            hasNextPage
            hasPreviousPage
            startCursor
          }
        }
      }
    }
  }`

	return graphql.NewRequest(fmt.Sprintf(projectItemsQuery, owner, number, first, after))
}

func NewAddPullRequestToProjectRequest(projectId, pullRequestId string) *graphql.Request {
	addPullRequestToProjectMutation := `
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

	req := graphql.NewRequest(addPullRequestToProjectMutation)
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
