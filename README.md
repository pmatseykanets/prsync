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
  # GitHub graphql API endpoint. Optional. Default is https://api.github.com/graphql.
  url: https://api.github.com/graphql

# A project to sync pull requests to. Required.
project: <owner>/<number>

# A list of repositories to sync pull requests from. Required.
repos:
#   - <owner>/<name>

# A team to get a list of pull request authors from. Optional.
team: <org>/<name>

# A list of specific authors to include or exclude. Optional.
authors:
  include:
    # - <login>
  exclude:
    # - <login>

pullRequests:
  # Add the author of the pull request to assignees. Default is false.
  assignAuthor: true
  # Add draft pull requests. Default is false.
  includeDrafts: false
  # Delete merged pull requests from the project. Default is false.
  deleteMerged: true
  # Delete closed pull requests from the project. Default is false.
  deleteClosed: true
  # A list of pull request states to add to the project. Default is [OPEN].
  # MERGED and CLOSED are mutually exclusive with deleteMerged: true and deleteClosed: true.
  states:
    - OPEN
    # - MERGED
    # - CLOSED
```
