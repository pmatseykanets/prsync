package github

import (
	"strings"
	"time"
)

const (
	APIEndpoint                = "https://api.github.com/graphql"
	ProjectItemTypeIssue       = "ISSUE"
	ProjectItemTypePullRequest = "PULL_REQUEST"
)

type PageInfo struct {
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

type ReviewRequest struct {
	ID                string `json:"id"`
	RequestedReviewer User   `json:"requestedReviewer"`
}

type Review struct {
	ID        string `json:"id"`
	Author    User   `json:"author"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
}

type Issue struct {
	ID string `json:"id"`
}

type ProjectItem struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	DatabaseID  int          `json:"databaseId"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	IsArchived  bool         `json:"isArchived"`
	PullRequest *PullRequest `json:"pullRequest"`
	Issue       *Issue       `json:"issue"`
}

type Project struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Owner  struct {
		Login string `json:"login"`
	} `json:"owner"`
	Items struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []ProjectItem `json:"nodes"`
		PageInfo   PageInfo      `json:"pageInfo"`
	} `json:"items"`
}

type PullRequestState string

const (
	PullRequestStateClosed PullRequestState = "CLOSED"
	PullRequestStateMerged PullRequestState = "MERGED"
	PullRequestStateOpen   PullRequestState = "OPEN"
)

type PullRequest struct {
	ID         string           `json:"id"`
	Number     int              `json:"number"`
	Title      string           `json:"title"`
	IsDraft    bool             `json:"isDraft"`
	Author     User             `json:"author"`
	Repository Repository       `json:"repository"`
	URL        string           `json:"url"`
	State      PullRequestState `json:"state"`
	Assignees  struct {
		TotalCount int      `json:"totalCount"`
		Nodes      []User   `json:"nodes"`
		PageInfo   PageInfo `json:"pageInfo"`
	} `json:"assignees"`
	Projects struct {
		TotalCount int       `json:"totalCount"`
		Nodes      []Project `json:"nodes"`
		PageInfo   PageInfo  `json:"pageInfo"`
	} `json:"projects"`
}

func (r *PullRequest) IsAuthorAssigned() bool {
	for _, a := range r.Assignees.Nodes {
		if a.Login == r.Author.Login {
			return true
		}
	}
	return false
}

type RepositoryOwner struct {
	Login string `json:"login"`
}

type Repository struct {
	Name         string          `json:"name"`
	Owner        RepositoryOwner `json:"owner"`
	PullRequests struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []PullRequest `json:"nodes"`
		PageInfo   PageInfo      `json:"pageInfo"`
	} `json:"pullRequests"`
}

type Error struct {
	Type    string   `json:"type"`
	Path    []string `json:"path"`
	Message string   `json:"message"`
}

func (e Error) Error() string {
	return e.Type + ": " + e.Message
}

type Errors []Error

func (e Errors) Error() string {
	var msg string
	for _, v := range e {
		msg += v.Error() + "\n"
	}
	return strings.TrimSuffix(msg, "\n")
}

type PullRequestResponse struct {
	Repository *Repository `json:"repository"`
	Errors     Errors      `json:"errors"`
}

type Team struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members struct {
		TotalCount int      `json:"totalCount"`
		Nodes      []User   `json:"nodes"`
		PageInfo   PageInfo `json:"pageInfo"`
	} `json:"members"`
}

type Organization struct {
	Team    *Team    `json:"team"`
	Project *Project `json:"projectV2"`
}

type TeamMembersResponse struct {
	Organization *Organization `json:"organization"`
	Errors       Errors        `json:"errors"`
}

type ProjectItemsResponse struct {
	Organization *Organization `json:"organization"`
	Errors       Errors        `json:"errors"`
}

type AddPullRequestToProjectResponse struct {
	AddProjectV2ItemByID struct {
		Item *ProjectItem `json:"item"`
	} `json:"addProjectV2ItemById"`
	Errors Errors `json:"errors"`
}

type AddAssigneeToPullRequestResponse struct {
	AddAssigneesToAssignable struct {
		Assignable struct {
			Assignees struct {
				TotalCount int    `json:"totalCount"`
				Nodes      []User `json:"nodes"`
			} `json:"assignees"`
		} `json:"item"`
	} `json:"addAssigneesToAssignable"`
	Errors Errors `json:"errors"`
}

type Viewer struct {
	Login string `json:"login"`
}

type ViewerResponse struct {
	Viewer *Viewer `json:"viewer"`
	Errors Errors  `json:"errors"`
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type LookUpUserResponse struct {
	User   *User  `json:"user"`
	Errors Errors `json:"errors"`
}
