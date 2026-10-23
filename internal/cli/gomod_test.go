package cli

import (
	"strings"
	"testing"
)

func TestGoModDeclaresGo122(t *testing.T) {
	_, lines := readLines(t, "../../go.mod")
	if !lines["go 1.22.0"] {
		t.Errorf("go.mod lacks the line %q", "go 1.22.0")
	}
	if lines["go 1.27.0"] {
		t.Errorf("go.mod still has the line %q", "go 1.27.0")
	}
}

func TestGoModHoldsOnlyModuleAndGo(t *testing.T) {
	text, _ := readLines(t, "../../go.mod")
	want := "module github.com/Gulliby-development/cidrfold\n\ngo 1.22.0\n"
	if got := strings.ReplaceAll(text, "\r\n", "\n"); got != want {
		t.Errorf("go.mod is %q, want %q", got, want)
	}
}

func TestLoopVariableIsPerIteration(t *testing.T) {
	var funcs []func() int
	for i := 0; i < 3; i++ {
		funcs = append(funcs, func() int { return i })
	}
	got := []int{funcs[0](), funcs[1](), funcs[2]()}
	if got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Errorf("closures saw %v, want [0 1 2]: the go directive must be 1.22 or newer", got)
	}
}

func TestREADMEStatesGoVersion(t *testing.T) {
	_, lines := readLines(t, "../../README.md")
	if !lines["cidrfold needs Go 1.22 or newer."] {
		t.Errorf("README.md lacks the line %q", "cidrfold needs Go 1.22 or newer.")
	}
}
