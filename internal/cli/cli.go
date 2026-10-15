// Package cli implements the cidrfold command line.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Gulliby-development/cidrfold/internal/ipset"
)

const usageText = `usage:
  cidrfold merge [--format cidr|range] [FILE...]
  cidrfold subtract [--format cidr|range] A B
  cidrfold intersect [--format cidr|range] A B
  cidrfold contains LIST ADDR...
`

type runner struct {
	stdin     io.Reader
	stdinUsed bool
}

// Run executes cidrfold and returns the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	if len(args) == 1 && args[0] == "--help" {
		fmt.Fprint(stdout, usageText)
		return 0
	}
	cmd := args[0]
	if cmd != "merge" && cmd != "subtract" && cmd != "intersect" && cmd != "contains" {
		fmt.Fprintf(stderr, "cidrfold: unknown command %q\n", cmd)
		return 2
	}
	format := "cidr"
	rest := args[1:]
	if cmd != "contains" {
		operands := make([]string, 0, len(rest))
		for i := 0; i < len(rest); i++ {
			a := rest[i]
			switch {
			case a == "--format":
				if i+1 >= len(rest) {
					fmt.Fprintln(stderr, "cidrfold: --format requires a value")
					return 2
				}
				i++
				format = rest[i]
			case strings.HasPrefix(a, "--format="):
				format = strings.TrimPrefix(a, "--format=")
			case strings.HasPrefix(a, "-") && a != "-":
				fmt.Fprintf(stderr, "cidrfold: unknown option %q\n", a)
				return 2
			default:
				operands = append(operands, a)
			}
		}
		rest = operands
	} else {
		for _, a := range rest {
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, "cidrfold: unknown option %q\n", a)
				return 2
			}
		}
	}
	if format != "cidr" && format != "range" {
		fmt.Fprintf(stderr, "cidrfold: invalid --format %q\n", format)
		return 2
	}
	r := &runner{stdin: stdin}
	switch cmd {
	case "merge":
		return r.merge(rest, format, stdout, stderr)
	case "subtract":
		if len(rest) != 2 {
			fmt.Fprint(stderr, "cidrfold: usage: cidrfold subtract [--format cidr|range] A B\n")
			return 2
		}
		return r.subtract(rest, format, stdout, stderr)
	case "intersect":
		if len(rest) != 2 {
			fmt.Fprint(stderr, "cidrfold: usage: cidrfold intersect [--format cidr|range] A B\n")
			return 2
		}
		return r.intersect(rest, format, stdout, stderr)
	default:
		if len(rest) < 2 {
			fmt.Fprint(stderr, "cidrfold: usage: cidrfold contains LIST ADDR...\n")
			return 2
		}
		return r.contains(rest, stdout, stderr)
	}
}

func (r *runner) load(name string) ([]ipset.Interval, error) {
	var data string
	label := name
	if name == "-" {
		label = "stdin"
		if !r.stdinUsed {
			r.stdinUsed = true
			b, err := io.ReadAll(r.stdin)
			if err != nil {
				return nil, fmt.Errorf("%s: %v", label, err)
			}
			data = string(b)
		}
	} else {
		b, err := os.ReadFile(name)
		if err != nil {
			var pe *os.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			return nil, fmt.Errorf("%s: %v", name, err)
		}
		data = string(b)
	}
	ivs, err := ipset.Parse(data)
	if err != nil {
		var pe *ipset.ParseError
		if errors.As(err, &pe) {
			return nil, fmt.Errorf("%s:%s", label, pe.Error())
		}
		return nil, fmt.Errorf("%s: %v", label, err)
	}
	return ivs, nil
}

func render(ivs []ipset.Interval, format string) string {
	var sb strings.Builder
	if format == "range" {
		for _, iv := range ipset.Ranges(ivs) {
			sb.WriteString(ipset.FormatRange(iv))
			sb.WriteByte('\n')
		}
	} else {
		for _, b := range ipset.CIDRs(ivs) {
			sb.WriteString(b.String())
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (r *runner) merge(files []string, format string, stdout, stderr io.Writer) int {
	if len(files) == 0 {
		files = []string{"-"}
	}
	var all []ipset.Interval
	for _, f := range files {
		ivs, err := r.load(f)
		if err != nil {
			fmt.Fprintf(stderr, "cidrfold: %v\n", err)
			return 2
		}
		all = append(all, ivs...)
	}
	io.WriteString(stdout, render(all, format))
	return 0
}

func (r *runner) subtract(ops []string, format string, stdout, stderr io.Writer) int {
	a, b, ok := r.loadPair(ops, stderr)
	if !ok {
		return 2
	}
	io.WriteString(stdout, render(ipset.Subtract(a, b), format))
	return 0
}

func (r *runner) intersect(ops []string, format string, stdout, stderr io.Writer) int {
	a, b, ok := r.loadPair(ops, stderr)
	if !ok {
		return 2
	}
	io.WriteString(stdout, render(ipset.Intersect(a, b), format))
	return 0
}

func (r *runner) loadPair(ops []string, stderr io.Writer) ([]ipset.Interval, []ipset.Interval, bool) {
	if ops[0] == "-" && ops[1] == "-" {
		fmt.Fprintln(stderr, "cidrfold: only one input may be -")
		return nil, nil, false
	}
	a, err := r.load(ops[0])
	if err != nil {
		fmt.Fprintf(stderr, "cidrfold: %v\n", err)
		return nil, nil, false
	}
	b, err := r.load(ops[1])
	if err != nil {
		fmt.Fprintf(stderr, "cidrfold: %v\n", err)
		return nil, nil, false
	}
	return a, b, true
}

func (r *runner) contains(args []string, stdout, stderr io.Writer) int {
	list, err := r.load(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "cidrfold: %v\n", err)
		return 2
	}
	addrs := make([]uint32, 0, len(args)-1)
	for _, t := range args[1:] {
		a, ok := ipset.ParseAddr(t)
		if !ok {
			fmt.Fprintf(stderr, "cidrfold: invalid address %q\n", t)
			return 2
		}
		addrs = append(addrs, a)
	}
	blocks := ipset.CIDRs(list)
	var sb strings.Builder
	code := 0
	for i, a := range addrs {
		if b, ok := ipset.Lookup(blocks, a); ok {
			fmt.Fprintf(&sb, "%s yes %s\n", args[i+1], b.String())
		} else {
			fmt.Fprintf(&sb, "%s no\n", args[i+1])
			code = 1
		}
	}
	io.WriteString(stdout, sb.String())
	return code
}
