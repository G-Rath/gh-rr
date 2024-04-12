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
gh rr g-rath/my-awesome-app 123
```

You can specify specific groups using `--from`:

```shell
gh rr --from infra g-rath/my-awesome-app 123
```
