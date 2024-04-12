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
# this is a map of repositories to a list of GitHub usernames
repositories:
  g-rath/my-awesome-app:
    - g-rath
    - octocat
  g-rath/dotfiles:
    - g-rath
```

Then start requesting reviewers on your pull requests:

```shell
gh rr g-rath/my-awesome-app 123
```
