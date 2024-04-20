package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type config struct {
	Repositories repositories `yaml:"repositories"`
}

type repositories map[string]map[string][]string
type repositoryGroups struct {
	Groups map[string][]string
}

func (rg *repositoryGroups) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var group []string

	// allow an array to be provided as a shorthand for the default group
	if err := unmarshal(&group); err == nil {
		rg.Groups = map[string][]string{"default": group}

		return nil
	}

	if err := unmarshal(&rg.Groups); err != nil {
		return err
	}

	return nil
}

func (r *repositories) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var repos map[string]repositoryGroups

	if err := unmarshal(&repos); err != nil {
		return err
	}

	for s, v := range repos {
		(*r)[strings.ToLower(s)] = v.Groups
	}

	return nil
}

func parseConfig(file string) (config, error) {
	conf := config{Repositories: map[string]map[string][]string{}}

	out, err := os.ReadFile(file)

	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(out, &conf)

	if err != nil {
		return conf, err
	}

	return conf, nil
}

var errRepositoryNotConfigured = errors.New("no reviewers are configured for repository")
var errGroupNotConfigured = errors.New("repository is not configured with group")

func determineReviewers(conf config, repository string, group string) ([]string, error) {
	if _, ok := conf.Repositories[repository]; !ok {
		return []string{}, errRepositoryNotConfigured
	}

	reviewers, ok := conf.Repositories[repository][group]

	if !ok {
		return []string{}, errGroupNotConfigured
	}

	return reviewers, nil
}

func buildAddReviewersArgs(repository string, target string, reviewers []string) []string {
	args := []string{"pr", "edit", target, "--repo", repository}

	for _, reviewer := range reviewers {
		args = append(args, "--add-reviewer", reviewer)
	}

	return args
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

// ghExecutor invokes a gh command in a subprocess and captures the output and error streams
type ghExecutor = func(args ...string) (stdout, stderr string)

func run(args []string, stdout, stderr io.Writer, ghExec ghExecutor) int {
	cli := flag.NewFlagSet("gh rr", flag.ContinueOnError)

	repoF := cli.StringP("repo", "R", "", "select another repository using the [HOST/]OWNER/REPO format")
	group := cli.StringP("from", "f", "default", "group of users to request review from")
	globalGroups := cli.BoolP("global", "g", false, "use the global reviewer groups")
	configDir := cli.String("config-dir", mustGetUserHomeDir(), "directory to search for the configuration file")
	isDryRun := cli.Bool("dry-run", false, "outputs instead of executing gh")

	cli.SetOutput(stderr)

	err := cli.Parse(args)

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		fmt.Fprintln(stderr, err)

		return 1
	}

	target := cli.Arg(0)

	repo := *repoF

	if repo == "" {
		currentRepo, err := repository.Current()

		if err != nil {
			fmt.Fprintf(os.Stderr, "could not determine repository: %v\n", err)

			return 1
		}

		repo = fmt.Sprintf("%s/%s", currentRepo.Owner, currentRepo.Name)
	}

	if _, _, found := strings.Cut(repo, "/"); !found || strings.HasPrefix(repo, "http") {
		fmt.Fprintln(stderr, "repository should be in the format of <owner>/<repository>")

		return 1
	}

	confPath := filepath.Join(*configDir, "gh-rr.yml")
	conf, err := parseConfig(confPath)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// todo: this could probably be worded better
			fmt.Fprintf(stderr, "please create %s to configure your repositories\n", confPath)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}

		return 1
	}

	repo2 := repo

	if *globalGroups {
		repo2 = "*"
	}
	reviewers, err := determineReviewers(conf, strings.ToLower(repo2), *group)

	if err != nil {
		if errors.Is(err, errRepositoryNotConfigured) {
			fmt.Fprintf(stderr, "no reviewers are configured for %s\n", repo)
		} else if errors.Is(err, errGroupNotConfigured) {
			fmt.Fprintf(stderr, "%s does not have a group named %s\n", repo, *group)
		} else {
			fmt.Fprintf(stderr, "%v\n", err)
		}

		return 1
	}

	if *isDryRun {
		fmt.Fprintf(stdout, "would have used `gh pr edit --repo %s` to request reviews from:\n", repo)
	} else {
		url, errMsg := ghExec(buildAddReviewersArgs(repo, target, reviewers)...)

		if errMsg != "" {
			fmt.Fprintf(stdout, "\ncould not add reviewers: %s\n", strings.TrimSpace(errMsg))

			return 1
		}

		fmt.Fprintf(stdout, "requested reviews on %s from:\n", url)
	}

	for _, reviewer := range reviewers {
		fmt.Fprintf(stdout, "  - %s\n", reviewer)
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, func(args ...string) (string, string) {
		stdout, stderr, _ := gh.Exec(args...)

		return strings.TrimSpace(stdout.String()), stderr.String()
	}))
}
