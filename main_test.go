package main

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestMain(m *testing.M) {
	code := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(code)
}

func Test_run(t *testing.T) {
	t.Parallel()

	type args struct {
		args   []string
		config string
		ghExec ghExecutor
	}
	tests := []struct {
		name string
		args args
		exit int
	}{
		{
			name: "when help is requested",
			args: args{
				args:   []string{"--help"},
				ghExec: expectNoCallToGh(t),
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when an unknown flag is requested",
			args: args{
				args:   []string{"--blah"},
				ghExec: expectNoCallToGh(t),
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
			name: "when no arguments are provided",
			args: args{
				args:   []string{},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when the target is not a number",
			args: args{
				args:   []string{"abc"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "abc"),
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when an explicit repository is provided using the longhand flag",
			args: args{
				args:   []string{"--repo", "octocat/hello-sunshine", "123"},
				ghExec: expectCallToGh(t, "octocat/hello-sunshine", "123"),
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
			name: "when an explicit repository is provided using the shorthand flag",
			args: args{
				args:   []string{"-R", "octocat/hello-sunshine", "123"},
				ghExec: expectCallToGh(t, "octocat/hello-sunshine", "123"),
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
			name: "when the explicit repository is not prefixed with the owner",
			args: args{
				args:   []string{"--repo", "hello-world"},
				ghExec: expectNoCallToGh(t),
				config: "",
			},
			exit: 1,
		},
		{
			name: "when the explicit repository is a url",
			args: args{
				args:   []string{"--repo", "https://github.com/octocat/hello-world"},
				ghExec: expectNoCallToGh(t),
				config: "",
			},
			exit: 1,
		},
		{
			name: "when the config file does not exist",
			args: args{
				args:   []string{"123"},
				ghExec: expectNoCallToGh(t),
				config: "",
			},
			exit: 1,
		},
		{
			name: "when the config file is invalid",
			args: args{
				args:   []string{"123"},
				ghExec: expectNoCallToGh(t),
				config: "!!!",
			},
			exit: 1,
		},
		{
			name: "when the config file is invalid (in a different way)",
			args: args{
				args:   []string{"123"},
				ghExec: expectNoCallToGh(t),
				config: "repositories: 1",
			},
			exit: 1,
		},
		{
			name: "when the repository does not exist in config",
			args: args{
				args:   []string{"123"},
				ghExec: expectNoCallToGh(t),
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
			name: "when the group does not exist in config",
			args: args{
				args:   []string{"--from", "does-not-exist", "123"},
				ghExec: expectNoCallToGh(t),
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
			name: "when an array is provided instead of a map of groups",
			args: args{
				args:   []string{},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						octocat/hello-world:
							- octodog
							- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when doing a dry-run",
			args: args{
				args:   []string{"--dry-run", "123"},
				ghExec: expectNoCallToGh(t),
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
			name: "when an explicit group is provided using the longhand flag",
			args: args{
				args:   []string{"--from", "infra", "123"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "123"),
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
			name: "when an explicit group is provided using the shorthand flag",
			args: args{
				args:   []string{"-f", "infra", "123"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "123"),
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
				args: []string{},
				ghExec: func(_ ...string) (string, string) {
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
		{
			name: "when repo case is different to whats in the config",
			args: args{
				args:   []string{"-R", "OctoCat/hello-sunshine", "123"},
				ghExec: expectCallToGh(t, "OctoCat/hello-sunshine", "123"),
				config: `
					repositories:
						octocat/hello-world:
							default:
								- octocat
						octocat/Hello-Sunshine:
							default:
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

			a := []string{"--config-dir", configDir}

			// quietly explicitly set the repo, since otherwise it'll be inferred from
			// the actual repo using git which is most likely going to be G-Rath/gh-rr
			if !slices.Contains(tt.args.args, "--repo") {
				a = append(a, "--repo", "octocat/hello-world")
			}

			a = append(a, tt.args.args...)

			var ghExecArgs []string

			got := run(a, stdout, stderr, func(args ...string) (stdout, stderr string) {
				t.Helper()

				ghExecArgs = args

				return tt.args.ghExec(args...)
			})

			if got != tt.exit {
				t.Errorf("run() = %v, want %v", got, tt.exit)
			}

			snaps.MatchSnapshot(t, normalizeStdStream(t, stdout))
			snaps.MatchSnapshot(t, normalizeStdStream(t, stderr))
			snaps.MatchJSON(t, ghExecArgs)
		})
	}
}

func Test_run_GlobalGroups(t *testing.T) {
	t.Parallel()

	type args struct {
		args   []string
		config string
		ghExec ghExecutor
	}
	tests := []struct {
		name string
		args args
		exit int
	}{
		{
			name: "when using the longhand flag",
			args: args{
				args:   []string{"--global", "--from", "security"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						'*':
							security:
								- octodog
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when using the shorthand flag",
			args: args{
				args:   []string{"-g", "--from", "security"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						'*':
							security:
								- octodog
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when combining shorthand flags",
			args: args{
				args:   []string{"-gf", "security"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						'*':
							security:
								- octodog
						octocat/hello-world:
							default:
								- octodog
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when the repo has a group with the same name",
			args: args{
				args:   []string{"-gf", "security"},
				ghExec: expectCallToGh(t, "octocat/hello-world", "1"),
				config: `
					repositories:
						'*':
							security:
								- octodog
						octocat/hello-world:
							security:
								- octopus
				`,
			},
			exit: 0,
		},
		{
			name: "when the repo has a group with the same name but the global one does not exist",
			args: args{
				args:   []string{"-gf", "security"},
				ghExec: expectNoCallToGh(t),
				config: `
					repositories:
						'*':
							platforms:
								- octodog
						octocat/hello-world:
							security:
								- octopus
				`,
			},
			exit: 1,
		},
		{
			name: "when a specific repository is given that is not in the config",
			args: args{
				args:   []string{"-gf", "security", "-R", "octocat/hello-sunshine"},
				ghExec: expectCallToGh(t, "octocat/hello-sunshine", "1"),
				config: `
					repositories:
						'*':
							security:
								- octodog
						octocat/hello-world:
							default:
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

			a := []string{"--config-dir", configDir}

			// quietly explicitly set the repo, since otherwise it'll be inferred from
			// the actual repo using git which is most likely going to be G-Rath/gh-rr
			if !slices.Contains(tt.args.args, "--repo") {
				a = append(a, "--repo", "octocat/hello-world")
			}

			a = append(a, tt.args.args...)

			var ghExecArgs []string

			got := run(a, stdout, stderr, func(args ...string) (stdout, stderr string) {
				t.Helper()

				ghExecArgs = args

				return tt.args.ghExec(args...)
			})

			if got != tt.exit {
				t.Errorf("run() = %v, want %v", got, tt.exit)
			}

			snaps.MatchSnapshot(t, normalizeStdStream(t, stdout))
			snaps.MatchSnapshot(t, normalizeStdStream(t, stderr))
			snaps.MatchJSON(t, ghExecArgs)
		})
	}
}

func Test_run_WithoutRepoFlag(t *testing.T) {
	t.Parallel()

	configDir := writeConfigFileInTempDir(t, dedent(t, `
		repositories:
			G-Rath/gh-rr:
				default:
					- octocat
	`))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	var ghExecArgs []string
	ghExecCalled := false

	got := run([]string{"--config-dir", configDir}, stdout, stderr, func(args ...string) (stdout, stderr string) {
		t.Helper()

		ghExecArgs = args
		ghExecCalled = true

		return "https://github.com/G-Rath/gh-rr", ""
	})

	if got != 0 {
		t.Errorf("run() = %v, want %v", got, 0)
	}

	snaps.MatchSnapshot(t, normalizeStdStream(t, stdout))
	snaps.MatchSnapshot(t, normalizeStdStream(t, stderr))

	if ghExecCalled {
		snaps.MatchJSON(t, ghExecArgs)
	}
}

func Test_run_WithNoHomeVar(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOME", "")

	defer func() { _ = recover() }()

	run([]string{}, &bytes.Buffer{}, &bytes.Buffer{}, expectNoCallToGh(t))

	t.Errorf("function did not panic when home directory could not be found")
}
