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
		target     string
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
				target:     "",
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
				target:     "123",
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
				target:     "123",
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
				target:     "123",
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

			if got := buildAddReviewersArgs(tt.args.repository, tt.args.target, tt.args.reviewers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildAddReviewersArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_determineReviewers(t *testing.T) {
	t.Parallel()

	type args struct {
		config     Config
		repository string
		group      string
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
					Repositories: map[string]map[string][]string{
						"octocat/hello-world": {"default": []string{"octocat"}},
					},
				},
				repository: "octocat/hello-sunshine",
				group:      "default",
			},
			want:    []string{},
			wantErr: ErrRepositoryNotConfigured,
		},
		{
			name: "reviewers when the repository is present",
			args: args{
				config: Config{
					Repositories: map[string]map[string][]string{
						"octocat/hello-world": {"default": []string{"octocat"}},
					},
				},
				repository: "octocat/hello-world",
				group:      "default",
			},
			want:    []string{"octocat"},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := determineReviewers(tt.args.config, tt.args.repository, tt.args.group)
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
							default:
								- octocat
				`,
			},
			want: Config{
				Repositories: map[string]map[string][]string{
					"octocat/hello-world": {"default": []string{"octocat"}},
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
							default:
							  - octocat
						octocat/hello-sunshine:
							default:
							  - octodog
							  - octopus
				`,
			},
			want: Config{
				Repositories: map[string]map[string][]string{
					"octocat/hello-world":    {"default": []string{"octocat"}},
					"octocat/hello-sunshine": {"default": []string{"octodog", "octopus"}},
				},
			},
			wantErr: false,
		},
		{
			name: "with multiple repositories (compat)",
			args: args{
				content: `
					repositories:
						octocat/hello-world: { default: ['octocat'] }
						octocat/hello-sunshine: { default: ['octodog', 'octopus'] }
				`,
			},
			want: Config{
				Repositories: map[string]map[string][]string{
					"octocat/hello-world":    {"default": []string{"octocat"}},
					"octocat/hello-sunshine": {"default": []string{"octodog", "octopus"}},
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
						octocat/hello-world:
							default: *shared
						octocat/hello-sunshine:
							default: *shared
				`,
			},
			want: Config{
				Repositories: map[string]map[string][]string{
					"octocat/hello-world":    {"default": []string{"octocat", "octodog"}},
					"octocat/hello-sunshine": {"default": []string{"octocat", "octodog"}},
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
		ghExec func(t *testing.T, args ...string) (string, string)
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
				ghExec: expectNoCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "repository must be prefixed with owner",
			args: args{
				args:   []string{"hello-world"},
				ghExec: expectNoCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "repository should not be a url",
			args: args{
				args:   []string{"https://github.com/octocat/hello-world"},
				ghExec: expectNoCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "target is not required",
			args: args{
				args:   []string{"octocat/hello-world", "abc"},
				ghExec: expectCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "target does not have to be a number",
			args: args{
				args:   []string{"octocat/hello-world"},
				ghExec: expectCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "config does not exist",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				ghExec: expectNoCallToGh,
				config: "",
			},
			exit: 1,
		},
		{
			name: "invalid config",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				ghExec: expectNoCallToGh,
				config: "!!!",
			},
			exit: 1,
		},
		{
			name: "repository does not exist in config",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				ghExec: expectNoCallToGh,
				config: `
					repositories:
						octocat/hello-sunshine:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 1,
		},
		{
			name: "group does not exist in config",
			args: args{
				args:   []string{"--from", "does-not-exist", "octocat/hello-world", "123"},
				ghExec: expectNoCallToGh,
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 1,
		},
		{
			name: "fulsome case",
			args: args{
				args:   []string{"octocat/hello-world", "123"},
				ghExec: expectCallToGh,
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octocat
						octocat/hello-sunshine:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "dry run",
			args: args{
				args:   []string{"--dry-run", "octocat/hello-world", "123"},
				ghExec: expectNoCallToGh,
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octocat
						octocat/hello-sunshine:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "explicit group",
			args: args{
				args:   []string{"--from", "infra", "octocat/hello-world", "123"},
				ghExec: expectCallToGh,
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octocat
							infra:
								- octodog
								- octopus
						octocat/hello-sunshine:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when ghExec fails",
			args: args{
				args: []string{"octocat/hello-world"},
				ghExec: func(t *testing.T, args ...string) (string, string) {
					t.Helper()

					return "", "no pull requests found for branch \"update-readme\""
				},
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octocat
							infra:
								- octodog
								- octopus
						octocat/hello-sunshine:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configDir := writeConfigFileInTempDir(t, dedent(t, tt.args.config))

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			a := []string{"--config-dir", configDir}
			a = append(a, tt.args.args...)

			var ghExecArgs []string
			ghExecCalled := false

			got := run(a, stdout, stderr, func(args ...string) (stdout, stderr string) {
				t.Helper()

				ghExecArgs = args
				ghExecCalled = true

				return tt.args.ghExec(t, args...)
			})

			if got != tt.exit {
				t.Errorf("run() = %v, want %v", got, tt.exit)
			}

			snaps.MatchSnapshot(t, normalizeStdStream(t, stdout))
			snaps.MatchSnapshot(t, normalizeStdStream(t, stderr))

			if ghExecCalled {
				snaps.MatchJSON(t, ghExecArgs)
			}
		})
	}
}

func Test_run_WithNoHomeVar(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOME", "")

	defer func() { _ = recover() }()

	run([]string{}, &bytes.Buffer{}, &bytes.Buffer{}, func(args ...string) (stdout, stderr string) {
		t.Helper()

		return expectNoCallToGh(t, args...)
	})

	t.Errorf("function did not panic when home directory could not be found")
}
