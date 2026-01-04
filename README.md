# prsync

A tool to automatically sync pull requests between GitHub repositories and GitHub Projects. Keep your project board up-to-date by automatically adding new PRs and removing completed ones based on configurable rules.

## Features

- 🔄 **Automatic Syncing**: Add PRs from multiple repositories to a project board
- 🎯 **Smart Filtering**: Include/exclude PRs by author, team, or organization
- 🧹 **Auto-cleanup**: Remove merged, closed, or draft PRs automatically
- 👤 **Auto-assignment**: Optionally assign PR authors as assignees
- 🔒 **Safe Testing**: Dry-run mode to preview changes before applying
- 🌐 **GitHub Enterprise**: Support for custom GitHub API endpoints

## Usage

```bash
prsync -h
Usage of prsync:
  -config string
        Path to the config file (default "config.yaml")
  -dry-run
        Dry run
  -verbose
        Verbose output
  -version
        Print version and exit
```

## Quick Start

1. **Create a GitHub token** with required scopes (see Authentication below)
2. **Create a config file** (see Configuration examples below)
3. **Run the tool**:
   ```bash
   export GITHUB_TOKEN=your_token_here
   prsync -config config.yaml
   ```

## Authentication

The tool expects `GITHUB_TOKEN` environment variable to be set with a token that has the following scopes:

- `repo` - Access to repository data and PRs
- `read:org` - Read organization teams and membership
- `read:user` - Read user profile information
- `project` - Manage GitHub Projects

**Creating a token:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select the scopes listed above
4. Generate and copy the token

## Configuration Examples

### Basic: Sync all open PRs from multiple repos

```yaml
project: myorg/123

repos:
  - myorg/backend
  - myorg/frontend
  - myorg/mobile
```

This adds all open PRs from the specified repositories to project #123.

### Sync PRs from specific team members

```yaml
project: myorg/123

repos:
  - myorg/backend
  - myorg/api

authors:
  include:
    teams:
      - myorg/backend-team
```

Only adds PRs authored by members of the `backend-team`.

### Exclude PRs from bots and contractors

```yaml
project: myorg/123

repos:
  - myorg/repo

authors:
  exclude:
    users:
      - dependabot
      - renovate-bot
    orgs:
      - external-contractors
```

Excludes PRs from specific users and members of specified organizations.

### Cleanup merged and closed PRs

```yaml
project: myorg/123

repos:
  - myorg/backend

pullRequests:
  delete:
    states:
      - MERGED
      - CLOSED
```

Automatically removes PRs from the project once they're merged or closed.

### Include drafts and assign authors

```yaml
project: myorg/123

repos:
  - myorg/backend

pullRequests:
  add:
    drafts: true
    assignAuthor: true
```

Adds draft PRs to the project and automatically assigns the PR author.

### Comprehensive example with cleanup

```yaml
project: myorg/456

repos:
  - myorg/backend
  - myorg/frontend

authors:
  include:
    teams:
      - myorg/core-team
  exclude:
    users:
      - dependabot

pullRequests:
  add:
    states:
      - OPEN
    assignAuthor: true
  delete:
    states:
      - MERGED
      - CLOSED
    forAllAuthors: true  # Clean up PRs from any author, not just included ones
```

## Configuration Reference

```yaml
github:
  # GitHub API endpoint. Optional. Default is https://api.github.com.
  url: https://api.github.com

# A project to sync pull requests to. Required.
project: <owner>/<number>

# A list of repositories to sync pull requests from. Required.
repos:
#   - <owner>/<name>

# PR author filtering rules. The rules are additive.
authors:
  include:
    users:
      # A list of specific users whose PRs to include.
      # - <login>
    teams:
      # A list of teams whose members' PRs to include.
      # - <owner>/<name>
    orgs:
      # A list of organizations whose members' PRs to include.
      # - <organization>
  exclude:
    users:
      # A list of specific users whose PRs to exclude.
      # - <login>
    teams:
      # A list of teams whose members' PRs to exclude.
      # - <owner>/<name>
    orgs:
      # A list of organizations whose members' PRs to exclude.
      # - <organization>

# Pull request syncing rules.
pullRequests:
  add:
    # Add pull requests only in the following states. Default is [OPEN].
    # Mutually exclusive with delete.states.
    states:
      - OPEN
    # Add draft pull requests. Default is false.
    drafts: true
    # Add the author of the pull request to assignees. Default is false.
    assignAuthor: true
  delete:
    # Delete pull requests only in the following states. Default is none.
    # Mutually exclusive with add.states.
    states:
      - CLOSED
      - MERGED
    # Delete draft pull requests from the project. Default is false.
    drafts: false
    # Delete pull requests from the project from all authors 
    # or only matching rules in the authors section. Default is false.
    forAllAuthors: false
```

## Common Use Cases

### Run as a scheduled GitHub Action

Create `.github/workflows/sync-project.yml`:

```yaml
name: Sync Project Board
on:
  schedule:
    - cron: '*/30 * * * *'  # Every 30 minutes
  workflow_dispatch:  # Allow manual trigger

jobs:
  sync:
    name: "Run prsync"
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0
      - name: Install prsync
        env:
          VERSION: 0.1.2
        run: |
          curl -L "https://github.com/pmatseykanets/prsync/releases/download/v${VERSION}/prsync_${VERSION}_linux_amd64.tar.gz" > "prsync_${VERSION}_linux_amd64.tar.gz"
          tar -xvzf "prsync_${VERSION}_linux_amd64.tar.gz" prsync
          ./prsync -version
      - name: Run
        env:
          GITHUB_TOKEN: ${{ secrets.GH_TOKEN }}
        run: ./prsync -config config.yaml```

### Testing configuration changes

Always test with dry-run mode first:

```bash
prsync -config config.yaml -dry-run -verbose
```

This shows what would be added or removed without making any changes.

### Using with GitHub Enterprise

```yaml
github:
  url: https://github.company.com/api/v3

project: enterprise-org/123
repos:
  - enterprise-org/repo1
```

## Troubleshooting

**"GITHUB_TOKEN is required"**
- Ensure the environment variable is set: `export GITHUB_TOKEN=your_token`
- Verify the token hasn't expired and has the required scopes

**"project not found" or "repository not found"**
- Check that the project/repo format is correct: `owner/number` for projects, `owner/name` for repos
- Verify your token has access to these resources
- For organization projects, ensure you have the correct organization name

**"team not found"**
- Verify team format is `org/team-slug` (use the team's slug, not display name)
- Ensure your token has `read:org` scope

**PRs not being added**
- Check author filters - PRs from bots (non-user types) are automatically excluded
- Use `-verbose` flag to see which PRs are being evaluated
- Verify PR states match your configuration

**Rate limiting**
- GitHub has rate limits (5000 requests/hour for authenticated requests)
- Consider reducing sync frequency if hitting limits
- Use `-verbose` to see how many API calls are being made

## License

MIT
