package cli

import (
	"os"
	"strings"
	"testing"
)

func readLines(t *testing.T, path string) (string, map[string]bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(data)
	lines := map[string]bool{}
	for _, l := range strings.Split(text, "\n") {
		lines[strings.TrimRight(l, "\r")] = true
	}
	return text, lines
}

func TestREADMEWorkedExamples(t *testing.T) {
	text, lines := readLines(t, "../../README.md")
	want := []string{
		"10.0.0.0/23",
		"192.168.1.4/31",
		"10.0.0.0-10.0.1.255",
		"192.168.1.4-192.168.1.5",
		"10.0.0.0/24",
		"172.16.0.0/12",
		"0.0.0.0/0",
		"255.255.255.254/31",
		"10.0.0.0/26",
		"10.0.0.128/25",
		"10.0.0.0-10.0.0.63",
		"10.0.0.128-10.0.0.255",
		"10.0.1.77 yes 10.0.0.0/23",
		"10.0.2.1 no",
		`cidrfold: stdin:4: invalid entry "10.0.0.256"`,
	}
	for _, w := range want {
		if !lines[w] {
			t.Errorf("README.md lacks line %q", w)
		}
	}
	for _, h := range []string{"## Install", "## Commands", "## Input format", "## Exit codes", "## Development"} {
		if !lines[h] {
			t.Errorf("README.md lacks section %q", h)
		}
	}
	for _, s := range []string{"go install", "go build", "go vet ./...", "go test ./..."} {
		if !strings.Contains(text, s) {
			t.Errorf("README.md does not mention %q", s)
		}
	}
}

func TestLicenseMIT(t *testing.T) {
	data, err := os.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("read LICENSE: %v", err)
	}
	text := string(data)
	for _, s := range []string{"MIT License", "Permission is hereby granted, free of charge"} {
		if !strings.Contains(text, s) {
			t.Errorf("LICENSE lacks %q", s)
		}
	}
}
