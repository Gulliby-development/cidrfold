package cli

import (
	"path/filepath"
	"testing"
)

func TestCountExamples(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", lines("10.0.0.0/25", "10.0.0.128/25", "10.0.1.0/24", "192.168.1.5", "192.168.1.4"))
	b := writeFile(t, dir, "b.txt", lines("10.0.0.64/26"))
	c := writeFile(t, dir, "c.txt", lines("10.0.0.0/24"))
	d := writeFile(t, dir, "d.txt", lines("10.0.0.0/23"))
	e := writeFile(t, dir, "e.txt", lines("10.0.0.128/25", "10.0.1.0/25", "10.0.3.0/24"))
	all := writeFile(t, dir, "all.txt", lines("0.0.0.0/1", "128.0.0.0/1"))
	overlapping := writeFile(t, dir, "overlapping.txt", lines("10.0.0.0/25", "10.0.0.64/26"))
	adjacent := writeFile(t, dir, "adjacent.txt", lines("10.0.0.128/25"))
	single := writeFile(t, dir, "single.txt", lines("192.0.2.17"))
	empty := writeFile(t, dir, "empty.txt", "")

	cases := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"worked merge", "", []string{"merge", "--format", "count", a}, lines("514")},
		{"worked range after count format", "", []string{"merge", "--format", "count", "--format", "range", a}, lines("10.0.0.0-10.0.1.255", "192.168.1.4-192.168.1.5")},
		{"worked subtract equals", "", []string{"subtract", "--format=count", c, b}, lines("192")},
		{"worked intersect after operands", "", []string{"intersect", d, e, "--format", "count"}, lines("256")},
		{"worked empty subtraction", "", []string{"subtract", "--format", "count", b, c}, lines("0")},
		{"whole address space", "", []string{"merge", "--format=count", all}, lines("4294967296")},
		{"repeated overlapping and adjacent input", "", []string{"merge", "--format", "count", overlapping, adjacent, overlapping}, lines("256")},
		{"single address", "", []string{"merge", "--format=count", single}, lines("1")},
		{"empty list from implicit stdin", "", []string{"merge", "--format", "count"}, lines("0")},
		{"empty list from file", "", []string{"merge", "--format=count", empty}, lines("0")},
		{"empty subtraction result", "", []string{"subtract", "--format=count", c, c}, lines("0")},
		{"empty intersection result", "", []string{"intersect", "--format", "count", empty, b}, lines("0")},
		{"implicit stdin", lines("192.0.2.1"), []string{"merge", "--format", "count"}, lines("1")},
		{"explicit stdin", lines("10.0.0.0/24"), []string{"merge", "-", "--format=count"}, lines("256")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, c.stdin, c.args...)
			if code != 0 || out != c.want || errs != "" {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}
}

func TestCountOptionPlacementAndPrecedence(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", lines("10.0.0.0/25"))
	b := writeFile(t, dir, "b.txt", lines("10.0.0.128/25"))
	count := lines("256")
	ranges := lines("10.0.0.0-10.0.0.255")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"separate before operands", []string{"merge", "--format", "count", a, b}, count},
		{"equals before operands", []string{"merge", "--format=count", a, b}, count},
		{"separate among operands", []string{"merge", a, "--format", "count", b}, count},
		{"equals among operands", []string{"merge", a, "--format=count", b}, count},
		{"separate after operands", []string{"merge", a, b, "--format", "count"}, count},
		{"equals after operands", []string{"merge", a, b, "--format=count"}, count},
		{"count then range", []string{"merge", "--format", "count", a, b, "--format", "range"}, ranges},
		{"range then count", []string{"merge", "--format", "range", a, b, "--format=count"}, count},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, "", c.args...)
			if code != 0 || out != c.want || errs != "" {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}
}

func TestCountErrorsAndUsage(t *testing.T) {
	badInput := lines("10.0.0.0/24", "invalid")
	countCode, countOut, countErr := run(t, badInput, "merge", "--format", "count")
	rangeCode, rangeOut, rangeErr := run(t, badInput, "merge", "--format", "range")
	wantBadEntry := "cidrfold: stdin:2: invalid entry \"invalid\"\n"
	if countCode != 2 || countOut != "" || countErr != wantBadEntry ||
		rangeCode != 2 || rangeOut != "" || rangeErr != wantBadEntry {
		t.Fatalf("count error=(%d, %q, %q), range error=(%d, %q, %q)", countCode, countOut, countErr, rangeCode, rangeOut, rangeErr)
	}

	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"uppercase count", []string{"merge", "--format", "COUNT"}, "cidrfold: invalid --format \"COUNT\"\n"},
		{"plural counts", []string{"merge", "--format=counts"}, "cidrfold: invalid --format \"counts\"\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, "", c.args...)
			if code != 2 || out != "" || errs != c.want {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}

	missing := filepath.Join(t.TempDir(), "missing.txt")
	countCode, countOut, countErr = run(t, "", "merge", "--format=count", missing)
	rangeCode, rangeOut, rangeErr = run(t, "", "merge", "--format=range", missing)
	if countCode != 2 || countOut != "" || countErr == "" ||
		rangeCode != 2 || rangeOut != "" || rangeErr != countErr {
		t.Fatalf("count missing-file error=(%d, %q, %q), range error=(%d, %q, %q)", countCode, countOut, countErr, rangeCode, rangeOut, rangeErr)
	}

	file := writeFile(t, t.TempDir(), "input.txt", lines("10.0.0.1"))
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"subtract operand count", []string{"subtract", "--format", "count", file}, "cidrfold: usage: cidrfold subtract [--format cidr|range|count] A B\n"},
		{"intersect operand count", []string{"intersect", file, "--format=count"}, "cidrfold: usage: cidrfold intersect [--format cidr|range|count] A B\n"},
		{"contains rejects format", []string{"contains", file, "10.0.0.1", "--format=count"}, "cidrfold: unknown option \"--format=count\"\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, "", c.args...)
			if code != 2 || out != "" || errs != c.want {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}

	wantUsage := "usage:\n" +
		"  cidrfold merge [--format cidr|range|count] [FILE...]\n" +
		"  cidrfold subtract [--format cidr|range|count] A B\n" +
		"  cidrfold intersect [--format cidr|range|count] A B\n" +
		"  cidrfold contains LIST ADDR...\n"
	code, out, errs := run(t, "", "--help")
	if code != 0 || out != wantUsage || errs != "" {
		t.Fatalf("help code=%d out=%q err=%q want %q", code, out, errs, wantUsage)
	}
}
