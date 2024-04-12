package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestMain(m *testing.M) {
	code := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(code)
}

// Attempts to normalize any file paths in the given `output` so that they can
// be compared reliably regardless of the file path separator being used.
//
// Namely, escaped forward slashes are replaced with backslashes.
func normalizeFilePaths(t *testing.T, output string) string {
	t.Helper()

	return strings.ReplaceAll(strings.ReplaceAll(output, "\\\\", "/"), "\\", "/")
}

// normalizeTempDirectory attempts to replace references to the temp directory
// with "<tempdir>", to ensure tests pass across different OSs
func normalizeTempDirectory(t *testing.T, str string) string {
	t.Helper()

	//nolint:gocritic // ensure that the directory doesn't end with a trailing slash
	tempDir := normalizeFilePaths(t, filepath.Join(os.TempDir()))
	re := regexp.MustCompile(tempDir + `/gh-rr-test-\d+`)

	return re.ReplaceAllString(str, "<tempdir>")
}

// normalizeErrors attempts to replace error messages on alternative OSs with their
// known linux equivalents, to ensure tests pass across different OSs
func normalizeErrors(t *testing.T, str string) string {
	t.Helper()

	str = strings.ReplaceAll(str, "The filename, directory name, or volume label syntax is incorrect.", "no such file or directory")
	str = strings.ReplaceAll(str, "The system cannot find the path specified.", "no such file or directory")
	str = strings.ReplaceAll(str, "The system cannot find the file specified.", "no such file or directory")

	return str
}

// normalizeStdStream applies a series of normalizes to the buffer from a std stream like stdout and stderr
func normalizeStdStream(t *testing.T, std *bytes.Buffer) string {
	t.Helper()

	str := std.String()

	for _, normalizer := range []func(t *testing.T, str string) string{
		normalizeFilePaths,
		normalizeTempDirectory,
		normalizeErrors,
	} {
		str = normalizer(t, str)
	}

	return str
}

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

			// these should be equal
			tt.want.Path = f

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// writeConfigFileInTempDir makes a `gh-rr.yml` configuration file with the given
// content for testing in a temporary directory, which is automatically cleaned up
func writeConfigFileInTempDir(t *testing.T, content string) string {
	t.Helper()

	p, err := os.MkdirTemp("", "gh-rr-test-*")
	if err != nil {
		t.Fatalf("could not create test directory: %v", err)
	}

	// only create the config if we've been given some content
	if content != "" {
		err = os.WriteFile(filepath.Join(p, "gh-rr.yml"), []byte(content), 0600)
		if err != nil {
			t.Fatalf("could not create test config: %v", err)
		}
	}

	// ensure the test directory is removed when we're done testing
	t.Cleanup(func() { _ = os.RemoveAll(p) })

	return p
}

func Test_run(t *testing.T) {
	t.Parallel()

	type args struct {
		args   []string
		config string
	}
	tests := []struct {
		name string
		args args
		exit int
	}{
		{
			name: "config does not exist",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "fulsome case",
			args: args{
				args: []string{"octocat/hello-world", "123"},
				config: `
					repositories:
						octocat/hello-world:
							- octocat
						octocat/hello-sunshine:
							- octodog
							- octopus
				`,
			},
			exit: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configDir := writeConfigFileInTempDir(t, dedent(t, tt.args.config))

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			a := []string{"--dry-run", "--config-dir", configDir}
			a = append(a, tt.args.args...)

			got := run(a, stdout, stderr)
			if got != tt.exit {
				t.Errorf("run() = %v, want %v", got, tt.exit)
			}

			snaps.MatchSnapshot(t, normalizeStdStream(t, stdout))
			snaps.MatchSnapshot(t, normalizeStdStream(t, stderr))
		})
	}
}
