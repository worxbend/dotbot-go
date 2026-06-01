package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stdout.String() != "Dotbot-Go version 0.1.0\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteAppExitDoesNotDuplicateToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--force-color", "--no-color"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "`--force-color` and `--no-color` cannot both be provided") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
