package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_VersionFlag_DefaultDev(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-v")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected command to exit cleanly 0, got err: %v, out: %s", err, string(out))
	}
	output := strings.TrimSpace(string(out))
	if output != "sukami version dev" {
		t.Fatalf("expected 'sukami version dev', got %q", output)
	}
}

func TestCLI_VersionFlag_Long(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected command to exit cleanly 0, got err: %v, out: %s", err, string(out))
	}
	output := strings.TrimSpace(string(out))
	if output != "sukami version dev" {
		t.Fatalf("expected 'sukami version dev', got %q", output)
	}
}
