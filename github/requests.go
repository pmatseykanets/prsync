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
                  reviewRequests(first: 100) {
                      totalCount
                      nodes {
                          id
                          requestedReviewer {
                              ... on User {
                                  id
                                  login
                              }
                          }
                      }
                  }
                  reviews(states: [APPROVED, COMMENTED, CHANGES_REQUESTED], first: 100) {
                      totalCount
                      nodes {
                          createdAt
                          id
                          state
                          author {
                              login
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
              }
            }
            issue: content{
              ... on PullRequest {
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
  mutation addPullRequestToTheProject($projectId: ID!, $pullRequestId: ID!) {
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

func NewViewerQuery() string {
	return `
  query viewer{
      viewer {
          login
      }
  }`
}
