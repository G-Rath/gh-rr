package main

import (
	"errors"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func dedent(t *testing.T, str string) string {
	t.Helper()

	// 0. replace all tabs with spaces
	str = strings.ReplaceAll(str, "\t", "  ")

	// 1. remove trailing whitespace
	re := regexp.MustCompile(`\r?\n([\t ]*)$`)
	str = re.ReplaceAllString(str, "")

	// 2. if any of the lines are not indented, return as we're already dedent-ed
	re = regexp.MustCompile(`(^|\r?\n)[^\t \n]`)
	if re.MatchString(str) {
		return str
	}

	// 3. find all line breaks to determine the highest common indentation level
	re = regexp.MustCompile(`\n[\t ]+`)
	matches := re.FindAllString(str, -1)

	// 4. remove the common indentation from all strings
	if matches != nil {
		size := len(matches[0]) - 1

		for _, match := range matches {
			if len(match)-1 < size {
				size = len(match) - 1
			}
		}

		re := regexp.MustCompile(`\n[\t ]{` + strconv.Itoa(size) + `}`)
		str = re.ReplaceAllString(str, "\n")
	}

	// 5. Remove leading whitespace.
	re = regexp.MustCompile(`^\r?\n`)
	str = re.ReplaceAllString(str, "")

	return str
}

func Test_buildAddReviewersArgs(t *testing.T) {
	t.Parallel()

	type args struct {
		repository string
		pr         string
		reviewers  []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "with everything empty",
			args: args{
				repository: "",
				pr:         "",
				reviewers:  nil,
			},
			want: []string{
				"pr", "edit", "",
				"--repo", "",
			},
		},
		{
			name: "with no reviewers",
			args: args{
				repository: "octocat/hello-world",
				pr:         "123",
				reviewers:  nil,
			},
			want: []string{
				"pr", "edit", "123",
				"--repo", "octocat/hello-world",
			},
		},
		{
			name: "with one reviewer",
			args: args{
				repository: "octocat/hello-world",
				pr:         "123",
				reviewers:  []string{"octocat"},
			},
			want: []string{
				"pr", "edit", "123",
				"--repo", "octocat/hello-world",
				"--add-reviewer", "octocat",
			},
		},
		{
			name: "with some reviewers",
			args: args{
				repository: "octocat/hello-world",
				pr:         "123",
				reviewers:  []string{"octocat", "octodog", "octopus"},
			},
			want: []string{
				"pr", "edit", "123",
				"--repo", "octocat/hello-world",
				"--add-reviewer", "octocat",
				"--add-reviewer", "octodog",
				"--add-reviewer", "octopus",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildAddReviewersArgs(tt.args.repository, tt.args.pr, tt.args.reviewers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildAddReviewersArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildPullRequestURL(t *testing.T) {
	t.Parallel()

	type args struct {
		repository string
		pr         string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "with everything empty",
			args: args{
				repository: "",
				pr:         "",
			},
			want: "https://github.com//pull/",
		},
		{
			name: "with a repository",
			args: args{
				repository: "octocat/hello-world",
				pr:         "",
			},
			want: "https://github.com/octocat/hello-world/pull/",
		},
		{
			name: "with everything provided",
			args: args{
				repository: "octocat/hello-world",
				pr:         "123",
			},
			want: "https://github.com/octocat/hello-world/pull/123",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildPullRequestURL(tt.args.repository, tt.args.pr); got != tt.want {
				t.Errorf("buildPullRequestURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determineReviewers(t *testing.T) {
	t.Parallel()

	type args struct {
		config     Config
		repository string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr error
	}{
		{
			name: "ErrRepositoryNotConfigured when the repository is not present",
			args: args{
				config: Config{
					Repositories: map[string][]string{
						"octocat/hello-sunshine": {"octocat"},
					},
				},
				repository: "octocat/hello-world",
			},
			want:    []string{},
			wantErr: ErrRepositoryNotConfigured,
		},
		{
			name: "reviewers when the repository is present",
			args: args{
				config: Config{
					Repositories: map[string][]string{
						"octocat/hello-world": {"octocat"},
					},
				},
				repository: "octocat/hello-world",
			},
			want:    []string{"octocat"},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := determineReviewers(tt.args.config, tt.args.repository)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("determineReviewers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("determineReviewers() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// writeTempConfigFile makes a temporary configuration file with the given
// content for testing, which is automatically cleaned up when testing finishes
func writeTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "gh-rr-test-config-*.yml")
	if err != nil {
		t.Fatalf("could not create config file: %v", err)
	}

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("could not write to config file: %v", err)
	}

	// ensure the file is removed when we're done testing
	t.Cleanup(func() { _ = os.RemoveAll(f.Name()) })

	return f.Name()
}

func Test_parseConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		content string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "with an empty file",
			args: args{
				content: "",
			},
			want: Config{
				Repositories: nil,
			},
			wantErr: false,
		},
		{
			name: "with invalid yaml",
			args: args{
				content: "!!!",
			},
			want: Config{
				Repositories: nil,
			},
			wantErr: true,
		},
		{
			name: "with a single repository",
			args: args{
				content: `
					repositories:
						octocat/hello-world:
							- octocat
				`,
			},
			want: Config{
				Repositories: map[string][]string{
					"octocat/hello-world": {"octocat"},
				},
			},
			wantErr: false,
		},
		{
			name: "with multiple repositories",
			args: args{
				content: `
					repositories:
						octocat/hello-world:
							- octocat
						octocat/hello-sunshine:
							- octodog
							- octopus
				`,
			},
			want: Config{
				Repositories: map[string][]string{
					"octocat/hello-world":    {"octocat"},
					"octocat/hello-sunshine": {"octodog", "octopus"},
				},
			},
			wantErr: false,
		},
		{
			name: "with multiple repositories (compat)",
			args: args{
				content: `
					repositories:
						octocat/hello-world: ['octocat']
						octocat/hello-sunshine: ['octodog', 'octopus']
				`,
			},
			want: Config{
				Repositories: map[string][]string{
					"octocat/hello-world":    {"octocat"},
					"octocat/hello-sunshine": {"octodog", "octopus"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := writeTempConfigFile(t, dedent(t, tt.args.content))

			got, err := parseConfig(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
