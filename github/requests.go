package github

import (
	"fmt"

	"github.com/machinebox/graphql"
)

func NewPullRequestQuery(owner, name string, first int, after string) string {
	return fmt.Sprintf(PullRequestQuery, owner, name, first, after)
}

var PullRequestQuery = `
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

func NewTeamMembersQuery(org, team string, first int, after string) string {
	return fmt.Sprintf(TeamMembersQuery, org, team, first, after)
}

var TeamMembersQuery = `
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

func NewProjectItemsQuery(owner string, number int, first int, after string) string {
	return fmt.Sprintf(projectItemsQuery, owner, number, first, after)
}

var projectItemsQuery = `
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

func NewAddPullRequestToProjectRequest(projectId, pullRequestId string) *graphql.Request {
	req := graphql.NewRequest(addPullRequestToProjectMutation)
	req.Var("projectId", projectId)
	req.Var("pullRequestId", pullRequestId)

	return req
}

var addPullRequestToProjectMutation = `
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

var getViewerQuery = `
query viewer{
    viewer {
        login
    }
}`

func NewViewerQuery() string {
	return getViewerQuery
}
