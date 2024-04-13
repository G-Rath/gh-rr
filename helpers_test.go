package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

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

func expectNoCallToGh(t *testing.T, _ ...string) (string, string) {
	t.Helper()

	t.Errorf("unexpected call to gh")

	return "", ""
}

func expectCallToGh(t *testing.T, _ ...string) (string, string) {
	t.Helper()

	return "https://github.com/octocat/hello-world", ""
}
