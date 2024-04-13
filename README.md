# gh-rr

GitHub CLI extension to request reviews from defined groups of reviewers.

## Installation

```shell
gh extension install g-rath/gh-rr
```

## Usage

Create a `gh-rr.yml` file in your home directory for configuring groups of
reviewers:

```yaml
# this is a map of repositories to groups of GitHub usernames
repositories:
  g-rath/my-awesome-app:
    default:
      - g-rath
      - octocat
    infra:
      - octodog
      - octopus
  g-rath/dotfiles:
    default:
      - g-rath
```

Then start requesting reviewers on your pull requests:

```shell
# will infer the pull request to target based on the current branch
gh rr g-rath/my-awesome-app

# will infer the pull request based on the named branch
gh rr g-rath/my-awesome-app my-feature

# will target the specific pull request
gh rr g-rath/my-awesome-app 123
```

Under the hood this extension uses
[`gh pr edit`](https://cli.github.com/manual/gh_pr_edit) to add reviewers, with
the second argument being provided as that commands first argument.

You can specify specific groups using `--from`:

```shell
gh rr --from infra g-rath/my-awesome-app 123
```
