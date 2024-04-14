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
gh rr
```

As a thin wrapper around
[`gh pr edit`](https://cli.github.com/manual/gh_pr_edit), the repository and
pull request are inferred automatically based on the current directory and
checked out branch when called without any flags or arguments.

Like with `gh pr edit`, you can pass either a pull request number, url, or
branch as the first argument, and can use the `--repo` flag to specify the
repository the pull request you're targeting belongs to:

```shell
# targeting a specific pull request in the current repository
gh rr 123

# targeting the pull request associated with a specific branch in another repository
gh --repo octocat/hello-world my-feature
```

> [!NOTE]
>
> Currently, unlike the `gh` cli, the `-R` short-flag is not supported

You can also use the `--from` flag to target alternative reviewer groups:

```shell
gh rr --from infra
```
