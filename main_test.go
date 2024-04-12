package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestMain(m *testing.M) {
	code := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(code)
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
			name: "with no file",
			args: args{
				content: "",
			},
			want: Config{
				Repositories: nil,
			},
			wantErr: true,
		},
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
		{
			name: "with aliases",
			args: args{
				content: `
					shared: &shared
						- octocat
						- octodog
					repositories:
						octocat/hello-world: *shared
						octocat/hello-sunshine: *shared
				`,
			},
			want: Config{
				Repositories: map[string][]string{
					"octocat/hello-world":    {"octocat", "octodog"},
					"octocat/hello-sunshine": {"octocat", "octodog"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := "none"

			if tt.name != "with no file" {
				f = writeTempConfigFile(t, dedent(t, tt.args.content))
			}

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
			name: "repository must be provided as the first argument",
			args: args{
				args:   []string{},
				config: "",
			},
			exit: 1,
		},
		{
			name: "repository must be prefixed with owner",
			args: args{
				args:   []string{"hello-world"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "repository should not be a url",
			args: args{
				args:   []string{"https://github.com/octocat/hello-world"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "pull request must be provided as the second argument",
			args: args{
				args:   []string{"octocat/hello-world"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "pull request must be a number",
			args: args{
				args:   []string{"octocat/hello-world", "abc"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "config does not exist",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				config: "",
			},
			exit: 1,
		},
		{
			name: "invalid config",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				config: "!!!",
			},
			exit: 1,
		},
		{
			name: "repository does not exist in config",
			args: args{
				args: []string{"octocat/hello-world", "123"},
				config: `
					repositories:
						octocat/hello-sunshine:
							- octodog
							- octopus
				`,
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

func Test_run_WithNoHomeVar(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOME", "")

	defer func() { _ = recover() }()

	run([]string{}, &bytes.Buffer{}, &bytes.Buffer{})

	t.Errorf("function did not panic when home directory could not be found")
}
