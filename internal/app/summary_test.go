package app

import "testing"

func TestRunSummary(t *testing.T) {
	got := runSummary(3, 2, "/tmp/dotfiles", true)
	want := "Starting dry-run with 3 task(s), 2 config file(s), base /tmp/dotfiles"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}
