package cli

import (
	"testing"
)

func TestFormatAnywhereSuccess(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "10.0.0.0/25\n")
	b := writeFile(t, dir, "b.txt", "10.0.0.128/25\n10.0.2.0/24\n")
	subA := writeFile(t, dir, "sub-a.txt", "10.0.0.0/24\n")
	subB := writeFile(t, dir, "sub-b.txt", "10.0.0.64/26\n")

	rangeMerge := lines("10.0.0.0-10.0.0.255", "10.0.2.0-10.0.2.255")
	cidrMerge := lines("10.0.0.0/24", "10.0.2.0/24")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"worked example after", []string{"merge", a, b, "--format", "range"}, rangeMerge},
		{"separate before", []string{"merge", "--format", "range", a, b}, rangeMerge},
		{"separate between", []string{"merge", a, "--format", "range", b}, rangeMerge},
		{"equals after", []string{"merge", a, b, "--format=range"}, rangeMerge},
		{"equals before", []string{"merge", "--format=range", a, b}, rangeMerge},
		{"equals between", []string{"merge", a, "--format=range", b}, rangeMerge},
		{"explicit cidr", []string{"merge", "--format=cidr", a, b}, cidrMerge},
		{"last format wins", []string{"merge", a, "--format=range", "--format=cidr"}, lines("10.0.0.0/25")},
		{"invalid then valid", []string{"merge", a, "--format=xml", "--format=range"}, lines("10.0.0.0-10.0.0.127")},
		{"subtract after", []string{"subtract", subA, subB, "--format", "range"}, lines("10.0.0.0-10.0.0.63", "10.0.0.128-10.0.0.255")},
		{"subtract between", []string{"subtract", subA, "--format=cidr", subB}, lines("10.0.0.0/26", "10.0.0.128/25")},
		{"intersect between", []string{"intersect", subA, "--format", "range", subB}, lines("10.0.0.64-10.0.0.127")},
		{"intersect after", []string{"intersect", subA, subB, "--format=cidr"}, lines("10.0.0.64/26")},
	}
	for _, c := range cases {
		code, out, errs := run(t, "", c.args...)
		if code != 0 || out != c.want || errs != "" {
			t.Errorf("%s: code=%d out=%q err=%q want %q", c.name, code, out, errs, c.want)
		}
	}
}

func TestFormatAnywhereErrors(t *testing.T) {
	f := writeFile(t, t.TempDir(), "f.txt", "10.0.0.1\n")
	usage := "cidrfold: usage: cidrfold subtract [--format cidr|range|count] A B\n"
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing trailing value", []string{"merge", f, "--format"}, "cidrfold: --format requires a value\n"},
		{"empty equals value", []string{"merge", f, "--format="}, "cidrfold: invalid --format \"\"\n"},
		{"invalid value", []string{"merge", f, "--format", "xml"}, "cidrfold: invalid --format \"xml\"\n"},
		{"consumed dash-prefixed value", []string{"merge", f, "--format", "-x"}, "cidrfold: invalid --format \"-x\"\n"},
		{"unknown option", []string{"merge", f, "-x"}, "cidrfold: unknown option \"-x\"\n"},
		{"scan error after invalid format", []string{"merge", "--format=xml", "-x"}, "cidrfold: unknown option \"-x\"\n"},
		{"unknown before missing value", []string{"merge", "-x", "--format"}, "cidrfold: unknown option \"-x\"\n"},
		{"value consumed even if option-shaped", []string{"merge", "--format", "--bogus"}, "cidrfold: invalid --format \"--bogus\"\n"},
		{"later missing value", []string{"merge", "--format=xml", "--format"}, "cidrfold: --format requires a value\n"},
		{"last invalid value wins", []string{"merge", f, "--format=range", "--format=xml"}, "cidrfold: invalid --format \"xml\"\n"},
		{"invalid format before operand count", []string{"subtract", f, "--format=xml"}, "cidrfold: invalid --format \"xml\"\n"},
		{"operand count after valid format", []string{"subtract", f, "--format=range"}, usage},
	}
	for _, c := range cases {
		code, out, errs := run(t, "", c.args...)
		if code != 2 || out != "" || errs != c.want {
			t.Errorf("%s: code=%d out=%q err=%q want %q", c.name, code, out, errs, c.want)
		}
	}
}

func TestContainsFormatOptionUnchanged(t *testing.T) {
	list := writeFile(t, t.TempDir(), "list.txt", "10.0.0.0/24\n")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"equals option remains unknown", []string{"contains", list, "10.0.0.1", "--format=range"}, "cidrfold: unknown option \"--format=range\"\n"},
		{"separate option remains unknown", []string{"contains", list, "10.0.0.1", "--format", "range"}, "cidrfold: unknown option \"--format\"\n"},
	}
	for _, c := range cases {
		code, out, errs := run(t, "", c.args...)
		if code != 2 || out != "" || errs != c.want {
			t.Errorf("%s: code=%d out=%q err=%q want %q", c.name, code, out, errs, c.want)
		}
	}

	code, out, errs := run(t, "", "contains", list, "10.0.0.1")
	if code != 0 || out != lines("10.0.0.1 yes 10.0.0.0/24") || errs != "" {
		t.Fatalf("without option: code=%d out=%q err=%q", code, out, errs)
	}
}

func TestStdinWithFormatAfterOperand(t *testing.T) {
	dir := t.TempDir()
	b := writeFile(t, dir, "b.txt", "10.0.0.64/26\n")
	cases := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"merge stdin before trailing option", lines("10.0.0.1"), []string{"merge", "-", "--format", "range"}, lines("10.0.0.1-10.0.0.1")},
		{"subtract stdin before trailing option", lines("10.0.0.0/24"), []string{"subtract", "-", b, "--format=range"}, lines("10.0.0.0-10.0.0.63", "10.0.0.128-10.0.0.255")},
		{"intersect stdin before trailing option", lines("10.0.0.0/24"), []string{"intersect", b, "-", "--format", "range"}, lines("10.0.0.64-10.0.0.127")},
	}
	for _, c := range cases {
		code, out, errs := run(t, c.stdin, c.args...)
		if code != 0 || out != c.want || errs != "" {
			t.Errorf("%s: code=%d out=%q err=%q want %q", c.name, code, out, errs, c.want)
		}
	}
}
