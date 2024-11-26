# prsync

A tool to add and delete pull requests to and from a GitHub project.

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

## Authentication

The tool expects `GITHUB_TOKEN` environment variable to be set with a token that has the following scopes:

- `repo`
- `read:org`
- `read:user`
- `project`

## Configuration file

```yaml
github:
  # GitHub API endpoint. Optional. Default is https://api.github.com.
  url: https://api.github.com

# A project to sync pull requests to. Required.
project: <owner>/<number>

# A list of repositories to sync pull requests from. Required.
repos:
#   - <owner>/<name>

# A list of specific authors to include or exclude. Optional.
authors:
  include:
    users:
      # - <login>
    teams:
      # - <owner>/<name>
    orgs:
      # - <organization>
  exclude:
    users:
      # - <login>
    teams:
      # - <owner>/<name>
    orgs:
      # - <organization>
  

include:
    # - <login>
  exclude:
    # - <login>

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
