package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func lines(ss ...string) string {
	if len(ss) == 0 {
		return ""
	}
	return strings.Join(ss, "\n") + "\n"
}

func TestMergeExamples(t *testing.T) {
	basic := lines("10.0.0.0/25", "10.0.0.128/25", "10.0.1.0/24", "192.168.1.5", "192.168.1.4")
	cases := []struct {
		name string
		in   string
		args []string
		want string
	}{
		{"basic", basic, []string{"merge"}, lines("10.0.0.0/23", "192.168.1.4/31")},
		{"range", basic, []string{"merge", "--format", "range"}, lines("10.0.0.0-10.0.1.255", "192.168.1.4-192.168.1.5")},
		{"range eq", basic, []string{"merge", "--format=range"}, lines("10.0.0.0-10.0.1.255", "192.168.1.4-192.168.1.5")},
		{"cidr explicit", lines("10.0.0.0/25", "10.0.0.128/25"), []string{"merge", "--format=cidr"}, lines("10.0.0.0/24")},
		{"host bits", lines("10.0.0.5/24", "10.0.0.77", "172.16.0.0/12", "172.20.0.0/16"), []string{"merge"}, lines("10.0.0.0/24", "172.16.0.0/12")},
		{"halves", lines("0.0.0.0/1", "128.0.0.0/1"), []string{"merge"}, lines("0.0.0.0/0")},
		{"all", lines("0.0.0.0/0"), []string{"merge"}, lines("0.0.0.0/0")},
		{"all range", lines("0.0.0.0/0"), []string{"merge", "--format", "range"}, lines("0.0.0.0-255.255.255.255")},
		{"top", lines("255.255.255.255", "255.255.255.254"), []string{"merge"}, lines("255.255.255.254/31")},
		{"single range", lines("10.0.0.1"), []string{"merge", "--format", "range"}, lines("10.0.0.1-10.0.0.1")},
		{"empty", "", []string{"merge"}, ""},
		{"comments", "# c\n\n  10.0.0.1 # x\t\n10.0.0.0\t#y", []string{"merge"}, lines("10.0.0.0/31")},
		{"no final newline", "10.0.0.0/24", []string{"merge"}, lines("10.0.0.0/24")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := run(t, c.in, c.args...)
			if code != 0 || out != c.want || errs != "" {
				t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, c.want)
			}
		})
	}
}

func TestMergeFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "10.0.0.0/25\n")
	b := writeFile(t, dir, "b.txt", "10.0.0.128/25\n")
	code, out, errs := run(t, "", "merge", a, b)
	if code != 0 || out != lines("10.0.0.0/24") || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestStdinErrors(t *testing.T) {
	base := "10.0.0.0/24\n\n# note\n10.0.0.256\n"
	variants := map[string]string{
		"plain":    base,
		"bom crlf": "\xef\xbb\xbf" + strings.ReplaceAll(base, "\n", "\r\n"),
	}
	for name, in := range variants {
		t.Run(name, func(t *testing.T) {
			code, out, errs := run(t, in, "merge")
			want := "cidrfold: stdin:4: invalid entry \"10.0.0.256\"\n"
			if code != 2 || out != "" || errs != want {
				t.Fatalf("code=%d out=%q err=%q", code, out, errs)
			}
		})
	}
}

func TestBOMCRLFAccepted(t *testing.T) {
	code, out, _ := run(t, "\xef\xbb\xbf10.0.0.0/24\r\n10.0.1.0/24\r\n", "merge")
	if code != 0 || out != lines("10.0.0.0/23") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestInvalidEntries(t *testing.T) {
	bad := []string{"::1", "10.0.0.0/33", "10.0.0.0/08", "010.0.0.1", "host.example", "10.0.0.1 10.0.0.2", "10.0.0.1#x", "+1/24"}
	for _, e := range bad {
		code, out, errs := run(t, "# c\n"+e+"\n", "merge")
		want := "cidrfold: stdin:2: invalid entry \"" + e + "\"\n"
		if code != 2 || out != "" || errs != want {
			t.Errorf("%q: code=%d out=%q err=%q", e, code, out, errs)
		}
	}
}

func TestFileErrorName(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "x.txt", "10.0.0.1\nbad\n")
	code, out, errs := run(t, "", "merge", p)
	want := "cidrfold: " + p + ":2: invalid entry \"bad\"\n"
	if code != 2 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestNoOutputOnLaterError(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "10.0.0.1\n")
	b := writeFile(t, dir, "b.txt", "oops\n")
	code, out, _ := run(t, "", "merge", a, b)
	if code != 2 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestSubtract(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "10.0.0.0/24\n")
	b := writeFile(t, dir, "b.txt", "10.0.0.64/26\n")
	c := writeFile(t, dir, "c.txt", "10.0.0.0/16\n")
	d := writeFile(t, dir, "d.txt", "0.0.0.0/0\n")

	code, out, errs := run(t, "", "subtract", a, b)
	if code != 0 || out != lines("10.0.0.0/26", "10.0.0.128/25") || errs != "" {
		t.Fatalf("cidr code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = run(t, "", "subtract", "--format", "range", a, b)
	if code != 0 || out != lines("10.0.0.0-10.0.0.63", "10.0.0.128-10.0.0.255") {
		t.Fatalf("range code=%d out=%q", code, out)
	}
	code, out, errs = run(t, "", "subtract", a, c)
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("empty code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = run(t, "", "subtract", d, d)
	if code != 0 || out != "" {
		t.Fatalf("all code=%d out=%q", code, out)
	}
}

func TestSubtractStdin(t *testing.T) {
	dir := t.TempDir()
	b := writeFile(t, dir, "b.txt", "10.0.0.64/26\n")
	code, out, _ := run(t, "10.0.0.0/24\n", "subtract", "-", b)
	if code != 0 || out != lines("10.0.0.0/26", "10.0.0.128/25") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, errs := run(t, "10.0.0.0/24\n", "subtract", "-", "-")
	if code != 2 || out != "" || errs != "cidrfold: only one input may be -\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestStdinReadOnce(t *testing.T) {
	code, out, errs := run(t, "10.0.0.1\n", "merge", "-", "-")
	if code != 0 || out != lines("10.0.0.1/32") || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestContains(t *testing.T) {
	list := writeFile(t, t.TempDir(), "l.txt", "10.0.0.0/23\n")
	code, out, errs := run(t, "", "contains", list, "10.0.1.77")
	if code != 0 || out != lines("10.0.1.77 yes 10.0.0.0/23") || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = run(t, "", "contains", list, "10.0.1.77", "10.0.2.1")
	if code != 1 || out != lines("10.0.1.77 yes 10.0.0.0/23", "10.0.2.1 no") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, errs = run(t, "", "contains", list, "10.0.1.77", "10.0.0.1/32")
	if code != 2 || out != "" || errs != "cidrfold: invalid address \"10.0.0.1/32\"\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = run(t, "10.0.0.0/24\n10.0.1.0/24\n", "contains", "-", "10.0.1.5")
	if code != 0 || out != lines("10.0.1.5 yes 10.0.0.0/23") {
		t.Fatalf("stdin code=%d out=%q", code, out)
	}
	code, out, _ = run(t, "# nothing\n", "contains", "-", "1.2.3.4")
	if code != 1 || out != lines("1.2.3.4 no") {
		t.Fatalf("empty code=%d out=%q", code, out)
	}
	code, out, _ = run(t, "0.0.0.0/0\n", "contains", "-", "255.255.255.255", "0.0.0.0")
	if code != 0 || out != lines("255.255.255.255 yes 0.0.0.0/0", "0.0.0.0 yes 0.0.0.0/0") {
		t.Fatalf("edge code=%d out=%q", code, out)
	}
}

func TestOptionErrors(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "f.txt", "10.0.0.1\n")
	cases := [][]string{
		{"merge", f, "--format", "cidr"},
		{"merge", f, "-x"},
		{"merge", "-x"},
		{"merge", "--bogus", f},
		{"contains", "-x", f, "10.0.0.1"},
		{"contains", f, "-x"},
		{"bogus"},
		{"subtract", f},
		{"subtract", f, f, f},
		{"contains", f},
		{"contains"},
	}
	for _, args := range cases {
		code, out, errs := run(t, "", args...)
		if code != 2 || out != "" || strings.Count(errs, "\n") != 1 || !strings.HasPrefix(errs, "cidrfold: ") {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errs)
		}
	}
}

func TestInvalidFormat(t *testing.T) {
	for _, args := range [][]string{
		{"merge", "--format", "xml"},
		{"merge", "--format=xml"},
		{"subtract", "--format=xml", "a", "b"},
	} {
		code, out, errs := run(t, "", args...)
		if code != 2 || out != "" || errs != "cidrfold: invalid --format \"xml\"\n" {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errs)
		}
	}
}

func TestMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing.txt")
	code, out, errs := run(t, "", "merge", p)
	if code != 2 || out != "" || !strings.HasPrefix(errs, "cidrfold: "+p+": ") || strings.Count(errs, "\n") != 1 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestHelpAndUsage(t *testing.T) {
	code, out, errs := run(t, "", "--help")
	if code != 0 || errs != "" {
		t.Fatalf("help code=%d err=%q", code, errs)
	}
	for _, w := range []string{"merge", "subtract", "contains"} {
		if !strings.Contains(out, w) {
			t.Errorf("usage missing %q", w)
		}
	}
	code, out2, errs2 := run(t, "")
	if code != 2 || out2 != "" || errs2 != out {
		t.Fatalf("noargs code=%d out=%q err=%q", code, out2, errs2)
	}
}
