package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	if stdout.String() != "Dotbot-Go version 0.2.1\n" {
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

func TestExecuteHelpUsesStyledSections(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	got := stdout.String()
	for _, expected := range []string{
		"dotbot-go",
		"Usage",
		"Examples",
		"Built-In Directives",
		"Flags",
		"Output",
		"--config-file <file>",
		"--dry-run",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q in help:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "\033[") {
		t.Fatalf("unexpected color for buffer output: %q", got)
	}
	if strings.Contains(got, "Compatibility") || strings.Contains(got, "--plugin") {
		t.Fatalf("help includes plugin support: %q", got)
	}
	if strings.Contains(got, "--super-quiet") {
		t.Fatalf("help includes hidden deprecated flag: %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteHelpExamplesByCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		examples []string
	}{
		{
			name: "root",
			args: []string{"--help"},
			examples: []string{
				"dotbot-go -c install.conf.yaml",
				"dotbot-go validate -c install.conf.yaml",
				"dotbot-go plan -c install.conf.yaml --output json",
				"dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run",
				"dotbot-go -c install.conf.yaml --only link -vv",
			},
		},
		{
			name: "validate",
			args: []string{"validate", "--help"},
			examples: []string{
				"dotbot-go validate -c install.conf.yaml",
				"dotbot-go validate -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml",
				"dotbot-go validate -c install.conf.yaml --only link",
			},
		},
		{
			name: "plan",
			args: []string{"plan", "--help"},
			examples: []string{
				"dotbot-go plan -c install.conf.yaml",
				"dotbot-go plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml",
				"dotbot-go plan -c install.conf.yaml --output json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Execute(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, want 0", code)
			}
			got := stdout.String()
			var want strings.Builder
			want.WriteString("Examples\n")
			for _, example := range tt.examples {
				want.WriteString("  ")
				want.WriteString(example)
				want.WriteString("\n")
			}
			want.WriteString("\n")
			if !strings.Contains(got, want.String()) {
				t.Fatalf("missing examples block:\n%s\nhelp:\n%s", want.String(), got)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestExecuteSuperQuietIsHiddenCompatibilityFlag(t *testing.T) {
	dir := t.TempDir()
	createdPath := filepath.Join(dir, "created")
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte("- create:\n  - "+createdPath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--super-quiet", "-c", configPath, "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want quiet output", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("dry run created path or unexpected stat error: %v", err)
	}
}

func TestExecuteValidateDoesNotApplyConfig(t *testing.T) {
	dir := t.TempDir()
	createdPath := filepath.Join(dir, "created")
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte("- create:\n  - "+createdPath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"validate", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Configuration is valid") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("validate created path or unexpected stat error: %v", err)
	}
}

func TestExecuteValidateRejectsMalformedDirective(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte("- shell:\n  - description: missing command\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"validate", "-c", configPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "shell directive item must include a command") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteValidateHelpOmitsApplyOnlyFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"validate", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	got := stdout.String()
	if strings.Contains(got, "--dry-run") || strings.Contains(got, "--exit-on-failure") {
		t.Fatalf("validate help includes apply-only flags:\n%s", got)
	}
	if !strings.Contains(got, "--config-file <file>") {
		t.Fatalf("validate help missing config flag:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanPrintsOperationsWithoutApplyingConfig(t *testing.T) {
	dir := t.TempDir()
	createdPath := filepath.Join(dir, "created")
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte("- create:\n  - "+createdPath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Plan: 1 operation(s)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "create  "+createdPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("plan created path or unexpected stat error: %v", err)
	}
}

func TestExecutePlanPreservesYAMLDirectiveOrder(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte(`
- shell:
    - [echo before, before]
  create:
    - tmp
  clean:
    - "~"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stdout.String()
	shellIndex := strings.Index(got, "shell   echo before [before]")
	createIndex := strings.Index(got, "create  tmp")
	cleanIndex := strings.Index(got, "clean   ~")
	if shellIndex < 0 || createIndex < 0 || cleanIndex < 0 {
		t.Fatalf("missing expected operations:\n%s", got)
	}
	if !(shellIndex < createIndex && createIndex < cleanIndex) {
		t.Fatalf("operations are not in source order:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanPreservesJSONDirectiveOrder(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.json")
	if err := os.WriteFile(configPath, []byte(`[
  {
    "shell": [["echo before", "before"]],
    "create": ["tmp"],
    "clean": ["~"]
  }
]`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stdout.String()
	shellIndex := strings.Index(got, "shell   echo before [before]")
	createIndex := strings.Index(got, "create  tmp")
	cleanIndex := strings.Index(got, "clean   ~")
	if shellIndex < 0 || createIndex < 0 || cleanIndex < 0 {
		t.Fatalf("missing expected operations:\n%s", got)
	}
	if !(shellIndex < createIndex && createIndex < cleanIndex) {
		t.Fatalf("operations are not in source order:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanPreservesJSON5DirectiveOrder(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.json5")
	if err := os.WriteFile(configPath, []byte(`[
  {
    // JSON5 allows comments, unquoted keys, and trailing commas.
    shell: [["echo before", "before"]],
    create: ["tmp"],
    clean: ["~"],
  },
]`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stdout.String()
	shellIndex := strings.Index(got, "shell   echo before [before]")
	createIndex := strings.Index(got, "create  tmp")
	cleanIndex := strings.Index(got, "clean   ~")
	if shellIndex < 0 || createIndex < 0 || cleanIndex < 0 {
		t.Fatalf("missing expected operations:\n%s", got)
	}
	if !(shellIndex < createIndex && createIndex < cleanIndex) {
		t.Fatalf("operations are not in source order:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanPreservesTOMLDirectiveOrder(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.toml")
	if err := os.WriteFile(configPath, []byte(`
tasks = [
  { shell = [["echo before", "before"]], create = ["tmp"], clean = ["~"] },
]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	got := stdout.String()
	shellIndex := strings.Index(got, "shell   echo before [before]")
	createIndex := strings.Index(got, "create  tmp")
	cleanIndex := strings.Index(got, "clean   ~")
	if shellIndex < 0 || createIndex < 0 || cleanIndex < 0 {
		t.Fatalf("missing expected operations:\n%s", got)
	}
	if !(shellIndex < createIndex && createIndex < cleanIndex) {
		t.Fatalf("operations are not in source order:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	if err := os.WriteFile(configPath, []byte("- shell:\n  - [\"echo hi >/tmp/out && true\", say hi]\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "-c", configPath, "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var got struct {
		OperationCount int `json:"operation_count"`
		Operations     []struct {
			Directive string `json:"directive"`
			Target    string `json:"target"`
			Detail    string `json:"detail"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout json: %v\n%s", err, stdout.String())
	}
	if strings.Contains(stdout.String(), `\u003e`) || strings.Contains(stdout.String(), `\u0026`) {
		t.Fatalf("json output escaped shell operators:\n%s", stdout.String())
	}
	if got.OperationCount != 1 || len(got.Operations) != 1 {
		t.Fatalf("plan = %#v", got)
	}
	if got.Operations[0].Directive != "shell" || got.Operations[0].Target != "echo hi >/tmp/out && true" || got.Operations[0].Detail != "say hi" {
		t.Fatalf("operation = %#v", got.Operations[0])
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlanHelpShowsOutputFlagAndOmitsApplyOnlyFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"plan", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	got := stdout.String()
	if !strings.Contains(got, "--output <format>") {
		t.Fatalf("plan help missing output flag:\n%s", got)
	}
	if strings.Contains(got, "--dry-run") || strings.Contains(got, "--exit-on-failure") {
		t.Fatalf("plan help includes apply-only flags:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteRejectsPluginFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--plugin", "example.py"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unknown flag: --plugin") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteHelpForceColor(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"--force-color", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "\033[1;36mdotbot-go\033[0m") {
		t.Fatalf("missing colored title: %q", stdout.String())
	}
}
