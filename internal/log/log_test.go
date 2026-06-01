package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerPrefixesLevels(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out)
	logger.SetLevel(Debug)

	logger.Debug("details")
	logger.Info("ready")
	logger.Action("created link")
	logger.Warning("check this")
	logger.Error("failed")

	got := out.String()
	for _, expected := range []string{
		"debug details",
		"info  ready",
		"step  created link",
		"warn  check this",
		"error failed",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q in %q", expected, got)
		}
	}
}

func TestLoggerForceColor(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out)
	logger.UseColor(true)

	logger.Error("failed")

	if !strings.Contains(out.String(), "\033[1;31merror\033[0m") {
		t.Fatalf("missing colored error label: %q", out.String())
	}
}

func TestLoggerNoColorForBuffersByDefault(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out)

	logger.Action("created")

	if strings.Contains(out.String(), "\033[") {
		t.Fatalf("unexpected color output: %q", out.String())
	}
}
