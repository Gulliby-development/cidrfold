package cli

import (
	"strings"
	"testing"
)

func TestComplement(t *testing.T) {
	dir := t.TempDir()
	halves := writeFile(t, dir, "halves.txt", lines("0.0.0.0/1", "128.0.0.0/2", "192.0.0.0/3"))
	ten := writeFile(t, dir, "ten.txt", lines("10.0.0.0/8"))
	other := writeFile(t, dir, "other.txt", lines("11.0.0.0/8", "10.0.0.0/9"))
	all := writeFile(t, dir, "all.txt", lines("0.0.0.0/0"))
	empty := writeFile(t, dir, "empty.txt", "")

	cases := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"worked cidr", "", []string{"complement", halves}, lines("224.0.0.0/3")},
		{"worked range", "", []string{"complement", "--format", "range", ten}, lines("0.0.0.0-9.255.255.255", "11.0.0.0-255.255.255.255")},
		{"worked cidr cover", "", []string{"complement", ten}, lines("0.0.0.0/5", "8.0.0.0/7", "11.0.0.0/8", "12.0.0.0/6", "16.0.0.0/4", "32.0.0.0/3", "64.0.0.0/2", "128.0.0.0/1")},
		{"worked count", "", []string{"complement", "--format=count", ten}, lines("4278190080")},
		{"option after operand", "", []string{"complement", ten, "--format", "range"}, lines("0.0.0.0-9.255.255.255", "11.0.0.0-255.255.255.255")},
		{"several files", "", []string{"complement", "--format=range", ten, other}, lines("0.0.0.0-9.255.255.255", "12.0.0.0-255.255.255.255")},
		{"whole space leaves nothing", "", []string{"complement", all}, ""},
		{"whole space count", "", []string{"complement", "--format", "count", all}, lines("0")},
		{"empty file is the whole space", "", []string{"complement", empty}, lines("0.0.0.0/0")},
		{"empty file count", "", []string{"complement", "--format=count", empty}, lines("4294967296")},
		{"implicit stdin", lines("10.0.0.0/8"), []string{"complement", "--format=count"}, lines("4278190080")},
		{"explicit stdin", lines("10.0.0.0/8"), []string{"complement", "-", "--format=count"}, lines("4278190080")},
		{"last address missing", lines("0.0.0.0/1", "128.0.0.0/2", "192.0.0.0/3", "224.0.0.0/4", "240.0.0.0/5", "248.0.0.0/6", "252.0.0.0/7", "254.0.0.0/8", "255.0.0.0/9", "255.128.0.0/10", "255.192.0.0/11", "255.224.0.0/12", "255.240.0.0/13", "255.248.0.0/14", "255.252.0.0/15", "255.254.0.0/16", "255.255.0.0/17", "255.255.128.0/18", "255.255.192.0/19", "255.255.224.0/20", "255.255.240.0/21", "255.255.248.0/22", "255.255.252.0/23", "255.255.254.0/24", "255.255.255.0/25", "255.255.255.128/26", "255.255.255.192/27", "255.255.255.224/28", "255.255.255.240/29", "255.255.255.248/30", "255.255.255.252/31", "255.255.255.254"), []string{"complement", "-"}, lines("255.255.255.255/32")},
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

func TestComplementErrors(t *testing.T) {
	dir := t.TempDir()
	bad := writeFile(t, dir, "bad.txt", lines("10.0.0.0/8", "10.0.0.256"))
	cases := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"invalid entry", "", []string{"complement", bad}, "cidrfold: " + bad + ":2: invalid entry \"10.0.0.256\"\n"},
		{"invalid entry on stdin", lines("10.0.0.0/8", "x"), []string{"complement"}, "cidrfold: stdin:2: invalid entry \"x\"\n"},
		{"invalid format", "", []string{"complement", "--format", "hex"}, "cidrfold: invalid --format \"hex\"\n"},
		{"missing format value", "", []string{"complement", "--format"}, "cidrfold: --format requires a value\n"},
		{"unknown option", "", []string{"complement", "--all"}, "cidrfold: unknown option \"--all\"\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, c.stdin, c.args...)
			if code != 2 || out != "" || errs != c.want {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}
}

func TestUsageListsComplement(t *testing.T) {
	code, out, errs := run(t, "", "--help")
	if code != 0 || errs != "" || !strings.Contains(out, "  cidrfold complement [--format cidr|range|count] [FILE...]\n") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}
