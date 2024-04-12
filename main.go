package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
	"gopkg.in/yaml.v3"
)

var ErrRepositoryConfigDoesNotExist = errors.New("no config exists for repository")

type Config struct {
	Repositories map[string][]string `yaml:"repositories"`
	Path         string              `yaml:""`
}

func parseConfig(file string) (Config, error) {
	config := Config{Path: file}

	out, err := os.ReadFile(file)

	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(out, &config)

	if err != nil {
		return config, err
	}

	return config, nil
}

func loadConfigFile(dir string) (Config, error) {
	file := filepath.Join(dir, "gh-rr.yml")

	if _, err := os.Stat(file); err != nil {
		return Config{Path: file}, err
	}

	return parseConfig(file)
}

var ErrRepositoryNotConfigured = errors.New("no reviewers are configured for repository")

func determineReviewers(config Config, repository string) ([]string, error) {
	reviewers, ok := config.Repositories[repository]

	if !ok {
		return []string{}, ErrRepositoryNotConfigured
	}

	return reviewers, nil
}

func buildPullRequestURL(repository string, pr string) string {
	return fmt.Sprintf("https://github.com/%s/pull/%s", repository, pr)
}

func buildAddReviewersArgs(repository string, pr string, reviewers []string) []string {
	args := []string{"pr", "edit", pr, "--repo", repository}

	for _, reviewer := range reviewers {
		args = append(args, "--add-reviewer", reviewer)
	}

	return args
}

func addReviewers(repository string, pr string, reviewers []string) string {
	_, stderr, _ := gh.Exec(buildAddReviewersArgs(repository, pr, reviewers)...)

	return stderr.String()
}

func mustGetUserHomeDir() string {
	dir, err := os.UserHomeDir()

	// would be seriously surprised if this happens for a regular user,
	// so for now we're just going to burst into flames unless someone
	// actually opens an issue, at which point we'll deal with this :)
	if err != nil {
		panic(fmt.Sprintf("failed to get user home dir: %v", err))
	}

	return dir
}

func run(args []string, stdout, stderr io.Writer) int {
	cli := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	configDir := cli.String("config-dir", mustGetUserHomeDir(), "directory to search for the configuration file")
	isDryRun := cli.Bool("dry-run", false, "")

	// cli is set for ExitOnError so this will never return an error
	_ = cli.Parse(args)

	repository := cli.Arg(0)
	pr := cli.Arg(1)
	url := buildPullRequestURL(repository, pr)

	config, err := loadConfigFile(*configDir)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// todo: this could probably be worded better
			fmt.Fprintf(stderr, "please create %s to configure your repositories\n", config.Path)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}

		return 1
	}

	reviewers, err := determineReviewers(config, repository)

	if err != nil {
		if errors.Is(err, ErrRepositoryConfigDoesNotExist) {
			fmt.Fprintf(stderr, "no reviewers are configured for %s\n", repository)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}

		return 1
	}

	fmt.Fprintf(stdout, "adding the following as reviewers to %s\n", url)

	for _, reviewer := range reviewers {
		fmt.Fprintf(stdout, "  - %s\n", reviewer)
	}

	if !*isDryRun {
		out := addReviewers(repository, pr, reviewers)

		if out != "" {
			fmt.Fprintf(stdout, "\ncould not add reviewers: %s\n", strings.TrimSpace(out))

			return 1
		}
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
