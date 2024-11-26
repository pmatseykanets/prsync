package main

import (
	"context"
	"fmt"
)

type authors struct {
	client         githubClient
	cfg            config
	ids            map[string]string
	included       map[string]bool
	excluded       map[string]bool
	includedByTeam map[string]bool
	excludedByTeam map[string]bool
	includedByOrg  map[string]bool
	excludedByOrg  map[string]bool
	teams          map[configTeam]map[string]bool
	orgs           map[string]map[string]bool
}

func NewAuthors(ctx context.Context, client githubClient, cfg config) (*authors, error) {
	a := &authors{
		client:         client,
		cfg:            cfg,
		ids:            make(map[string]string),
		included:       make(map[string]bool),
		excluded:       make(map[string]bool),
		includedByTeam: make(map[string]bool),
		excludedByTeam: make(map[string]bool),
		includedByOrg:  make(map[string]bool),
		excludedByOrg:  make(map[string]bool),
		teams:          make(map[configTeam]map[string]bool),
		orgs:           make(map[string]map[string]bool),
	}

	for _, user := range cfg.authors.include.users {
		a.included[user] = true
	}

	for _, user := range cfg.authors.exclude.users {
		a.excluded[user] = true
	}

	for _, team := range append(cfg.authors.include.teams, cfg.authors.exclude.teams...) {
		if _, ok := a.teams[team]; ok {
			break
		}

		if cfg.verbose {
			fmt.Printf("Fetching team members for %s/%s:\n", team.owner, team.name)
		}

		a.teams[team] = make(map[string]bool)
		members, err := client.GetTeamMembers(ctx, team.owner, team.name)
		if err != nil {
			return nil, err
		}

		for _, m := range members {
			a.ids[m.Login] = m.ID
			a.teams[team][m.Login] = true

			if cfg.verbose {
				fmt.Printf("  - %s\n", m.Login)
			}
		}
	}

	return a, nil
}

func (a *authors) Resolve(ctx context.Context, login string) (bool, error) {
	// By default, all authors are included.
	if a.cfg.authors.include.empty() && a.cfg.authors.exclude.empty() {
		return true, nil
	}

	// Explicitly excluded.
	if a.excluded[login] {
		return false, nil
	}

	// Explicitly included.
	if a.included[login] {
		return true, nil
	}

	// Excluded by team.
	if len(a.cfg.authors.exclude.teams) > 0 {
		excluded, ok := a.excludedByTeam[login]
		if !ok {
			for _, t := range a.cfg.authors.exclude.teams {
				if a.teams[t][login] {
					excluded = true
					a.excludedByTeam[login] = true
					break
				}
			}
		}
		if excluded {
			return false, nil
		}
	}

	// Included by a team.
	if len(a.cfg.authors.include.teams) > 0 {
		included, ok := a.includedByTeam[login]
		if !ok {
			for _, t := range a.cfg.authors.include.teams {
				if a.teams[t][login] {
					included = true
					a.includedByTeam[login] = true
					break
				}
			}
		}
		if included {
			return true, nil
		}
	}

	getOrgs := func(ctx context.Context, login string, orgs []string) (map[string]bool, error) {
		userOrgs, ok := a.orgs[login]
		if !ok {
			if a.cfg.verbose {
				fmt.Printf("        Fetching organizations for %s\n", login)
			}
			ghOrgs, err := a.client.GetUserOrganizations(ctx, login)
			if err != nil {
				return nil, err
			}

			a.orgs[login] = make(map[string]bool)
			for _, org := range ghOrgs {
				a.orgs[login][org.Login] = true
			}

			// It's possible user profile is private so we can't get users orgs.
			// We'll try to explicitly check for membership in the requested orgs.
			if len(ghOrgs) == 0 {
				for _, name := range orgs {
					if a.cfg.verbose {
						fmt.Printf("        Checking membership in %s for %s\n", name, login)
					}
					isMember, err := a.client.IsOrganizationMember(ctx, login, name)
					if err != nil {
						return nil, err
					}
					if isMember {
						a.orgs[login][name] = true
					}
				}
			}

			return a.orgs[login], nil
		}

		return userOrgs, nil
	}

	// Excluded by an org.
	if len(a.cfg.authors.exclude.orgs) > 0 {
		excluded, ok := a.excludedByOrg[login]
		if !ok {
			orgs, err := getOrgs(ctx, login, a.cfg.authors.exclude.orgs)
			if err != nil {
				return false, err
			}

			a.excludedByOrg[login] = false
			for _, org := range a.cfg.authors.exclude.orgs {
				if orgs[org] {
					excluded = true
					a.excludedByOrg[login] = true
					break
				}
			}
		}
		if excluded {
			return false, nil
		}
	}

	// Included by an org.
	if len(a.cfg.authors.include.orgs) > 0 {
		included, ok := a.includedByOrg[login]
		if !ok {
			orgs, err := getOrgs(ctx, login, a.cfg.authors.include.orgs)
			if err != nil {
				return false, err
			}

			a.includedByOrg[login] = false
			for _, org := range a.cfg.authors.include.orgs {
				if orgs[org] {
					included = true
					a.includedByOrg[login] = true
					break
				}
			}
		}
		if included {
			return true, nil
		}
	}

	return a.cfg.authors.include.empty(), nil
}

func (a *authors) GetID(ctx context.Context, login string) (string, error) {
	id, ok := a.ids[login]
	if !ok {
		if a.cfg.verbose {
			fmt.Printf("        Fetching user ID for %s\n", login)
		}
		user, err := a.client.LookupUser(ctx, login)
		if err != nil {
			return "", err
		}

		id = user.ID
		a.ids[login] = id
	}

	return id, nil
}
