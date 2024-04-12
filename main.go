package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cli/go-gh/v2"
	"gopkg.in/yaml.v3"
)

var ErrRepositoryConfigDoesNotExist = errors.New("no config exists for repository")

type Config struct {
	Repositories map[string][]string `yaml:"repositories"`
}

func parseConfig(file string) (Config, error) {
	var config Config

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

func loadConfigFile() (Config, error) {
	// dir, err := os.UserHomeDir()
	dir, err := os.Getwd()

	if err != nil {
		panic("oh noes, could not get home directory!")
	}

	file := filepath.Join(dir, "gh-rr.yml")

	if _, err := os.Stat(file); err != nil {
		return Config{}, err
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

func addReviewers(repository string, pr string, reviewers []string) (string, error) {
	stdOut, _, err := gh.Exec(buildAddReviewersArgs(repository, pr, reviewers)...)

	return stdOut.String(), err
}

func run(args []string, stdout, stderr io.Writer) int {
	cli := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// cli is set for ExitOnError so this will never return an error
	_ = cli.Parse(args)

	repository := cli.Arg(0)
	pr := cli.Arg(1)
	url := buildPullRequestURL(repository, pr)

	config, err := loadConfigFile()

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "please create config file\n")
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

	out, err := addReviewers(repository, pr, reviewers)

	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)

		return 1
	}
	fmt.Fprint(stdout, out)

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
