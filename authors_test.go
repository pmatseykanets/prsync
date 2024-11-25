package main

import (
	"context"
	"testing"

	"github.com/pmatseykanets/prsync/github"
)

func TestAuthorsIsNoRules(t *testing.T) {
	ctx := context.Background()
	cfg := config{}
	client := &fakeGithubClient{}

	authors, err := NewAuthors(ctx, client, cfg)
	if err != nil {
		t.Fatal(err)
	}

	isAuthor, err := authors.Resolve(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := true, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}
}

func TestAuthorsIsExplicitlyIncluded(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		authors: configAuthors{
			include: configAuthorRules{
				users: []string{"user"},
			},
			exclude: configAuthorRules{
				teams: []configTeam{{"org1", "team1"}},
			},
		},
	}
	client := &fakeGithubClient{
		GetTeamMembersFunc: func(ctx context.Context, owner, name string) ([]github.User, error) {
			switch name {
			case "team1":
				return []github.User{
					{ID: "user", Login: "user"},
				}, nil
			default:
				return nil, nil
			}
		},
		GetUserOrganizationsFunc: func(ctx context.Context, login string) ([]github.Organization, error) {
			t.Errorf("Unexpected call to GetUserOrganizations for %s", login)
			return nil, nil
		},
	}

	authors, err := NewAuthors(ctx, client, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// A user that is explicitly included.
	isAuthor, err := authors.Resolve(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := true, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}

	// A user that is not included.
	isAuthor, err = authors.Resolve(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := false, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}
}

func TestAuthorsIsIncludedByATeam(t *testing.T) {
	ctx := context.Background()
	cfg := config{
		authors: configAuthors{
			include: configAuthorRules{
				teams: []configTeam{
					{"org1", "team1"},
					{"org1", "team2"},
				},
			},
			exclude: configAuthorRules{
				users: []string{"user1"},
				orgs:  []string{"org2"},
			},
		},
	}
	client := &fakeGithubClient{
		GetTeamMembersFunc: func(ctx context.Context, owner, name string) ([]github.User, error) {
			switch name {
			case "team1":
				return []github.User{
					{ID: "user", Login: "user"},
					{ID: "user1", Login: "user1"},
				}, nil
			case "team2":
				return []github.User{
					{ID: "user2", Login: "user2"},
					{ID: "user3", Login: "user3"},
				}, nil
			default:
				return nil, nil
			}
		},
		LookupUserFunc: func(ctx context.Context, login string) (*github.User, error) {
			return &github.User{ID: login, Login: login}, nil
		},
		GetUserOrganizationsFunc: func(ctx context.Context, login string) ([]github.Organization, error) {
			if login != "user4" {
				t.Errorf("Unexpected call to GetUserOrganizations for %s", login)
			}
			return nil, nil
		},
		IsOrganizationMemberFunc: func(ctx context.Context, login, org string) (bool, error) {
			if login != "user4" {
				t.Errorf("Unexpected call to IsOrganizationMember for %s", login)
			}
			return false, nil
		},
	}

	authors, err := NewAuthors(ctx, client, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// A user included in one of the teams.
	isAuthor, err := authors.Resolve(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := true, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}

	// A user not included in any of the teams.
	isAuthor, err = authors.Resolve(ctx, "user4")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := false, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}

	// A user not included in any of the teams.
	isAuthor, err = authors.Resolve(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := false, isAuthor; want != got {
		t.Fatalf("Expected %t, got %t", want, got)
	}
}
