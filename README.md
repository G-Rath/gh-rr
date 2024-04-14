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
branch as the first argument, and can use the `-R|--repo` flag to specify the
repository the pull request you're targeting belongs to:

```shell
# targeting a specific pull request in the current repository
gh rr 123

# targeting the pull request associated with a specific branch in another repository
gh --repo octocat/hello-world my-feature
```

You can also use the `-f|--from` flag to target alternative reviewer groups:

```shell
gh rr --from infra
```

## Why not use [CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners) or [GitHub teams](https://docs.github.com/en/organizations/organizing-members-into-teams/managing-code-review-settings-for-your-team)?

Both of these can be used to achieve a similar result as this extension, but
they're not entirely equivalent: team-focused reviews generally dismiss the team
review when a single member of that team submits their review, and automatic
review assignment has a limit of up to 7 people (plus, it skips people who set
their status to "busy" - while well intended, it assumes everyone diligently
updates their status).

While you can use CODEOWNERS to work around this by specifying users instead of
teams, that means you then need to do a commit to update the file (which is
particularly annoying when you're using branches for deployments), and it only
applies at the file level rather than the change level (e.g. you might make an
update to a section of your readme that relates to infrastructure, so should be
reviewed by your platforms team)
