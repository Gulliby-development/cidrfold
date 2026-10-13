// Package ipset parses IPv4 CIDR lists and does set arithmetic on them.
package ipset

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// Interval is an inclusive range of IPv4 addresses.
type Interval struct {
	First, Last uint64
}

// Block is a CIDR block.
type Block struct {
	Addr uint32
	Bits int
}

// String returns the a.b.c.d/n form.
func (b Block) String() string {
	return fmt.Sprintf("%s/%d", FormatAddr(b.Addr), b.Bits)
}

// Interval returns the inclusive interval covered by the block.
func (b Block) Interval() Interval {
	size := uint64(1) << uint(32-b.Bits)
	return Interval{uint64(b.Addr), uint64(b.Addr) + size - 1}
}

// ParseError describes the first invalid entry of an input.
type ParseError struct {
	Line int
	Text string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d: invalid entry %q", e.Line, e.Text)
}

// FormatAddr formats an address as a.b.c.d.
func FormatAddr(a uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", a>>24, (a>>16)&255, (a>>8)&255, a&255)
}

// FormatRange formats an interval as first-last.
func FormatRange(iv Interval) string {
	return FormatAddr(uint32(iv.First)) + "-" + FormatAddr(uint32(iv.Last))
}

func parseNum(s string, max int) (int, bool) {
	if s == "" || len(s) > 3 || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	if n > max {
		return 0, false
	}
	return n, true
}

// ParseAddr parses a strict bare dotted-quad IPv4 address.
func ParseAddr(s string) (uint32, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, false
	}
	var a uint32
	for _, p := range parts {
		n, ok := parseNum(p, 255)
		if !ok {
			return 0, false
		}
		a = a<<8 | uint32(n)
	}
	return a, true
}

func parseEntry(s string) (Interval, bool) {
	addrText := s
	n := 32
	if i := strings.IndexByte(s, '/'); i >= 0 {
		addrText = s[:i]
		var ok bool
		n, ok = parseNum(s[i+1:], 32)
		if !ok {
			return Interval{}, false
		}
	}
	a, ok := ParseAddr(addrText)
	if !ok {
		return Interval{}, false
	}
	size := uint64(1) << uint(32-n)
	first := uint64(a) &^ (size - 1)
	return Interval{first, first + size - 1}, true
}

// Parse reads a whole input and returns its entries as unmerged intervals.
func Parse(data string) ([]Interval, error) {
	data = strings.TrimPrefix(data, "\uFEFF")
	var out []Interval
	for i, line := range strings.Split(data, "\n") {
		line = strings.TrimSuffix(line, "\r")
		line = strings.Trim(line, " \t")
		if line == "" || line[0] == '#' {
			continue
		}
		for j := 1; j < len(line); j++ {
			if line[j] == '#' && (line[j-1] == ' ' || line[j-1] == '\t') {
				line = strings.Trim(line[:j], " \t")
				break
			}
		}
		iv, ok := parseEntry(line)
		if !ok {
			return nil, &ParseError{Line: i + 1, Text: line}
		}
		out = append(out, iv)
	}
	return out, nil
}

// Merge returns the sorted union with overlapping and adjacent intervals combined.
func Merge(in []Interval) []Interval {
	s := append([]Interval(nil), in...)
	sort.Slice(s, func(i, j int) bool { return s[i].First < s[j].First })
	var out []Interval
	for _, iv := range s {
		if n := len(out); n > 0 && iv.First <= out[n-1].Last+1 {
			if iv.Last > out[n-1].Last {
				out[n-1].Last = iv.Last
			}
			continue
		}
		out = append(out, iv)
	}
	return out
}

// Subtract returns the addresses in a and not in b.
func Subtract(a, b []Interval) []Interval {
	a, b = Merge(a), Merge(b)
	var out []Interval
	for _, ia := range a {
		cur := ia.First
		for _, ib := range b {
			if ib.Last < cur {
				continue
			}
			if ib.First > ia.Last {
				break
			}
			if ib.First > cur {
				out = append(out, Interval{cur, ib.First - 1})
			}
			cur = ib.Last + 1
			if cur > ia.Last {
				break
			}
		}
		if cur <= ia.Last {
			out = append(out, Interval{cur, ia.Last})
		}
	}
	return out
}

// Intersect returns the addresses present in both a and b.
func Intersect(a, b []Interval) []Interval {
	a, b = Merge(a), Merge(b)
	var out []Interval
	for i, j := 0, 0; i < len(a) && j < len(b); {
		first := a[i].First
		if b[j].First > first {
			first = b[j].First
		}
		last := a[i].Last
		if b[j].Last < last {
			last = b[j].Last
		}
		if first <= last {
			out = append(out, Interval{first, last})
		}

		switch {
		case a[i].Last < b[j].Last:
			i++
		case b[j].Last < a[i].Last:
			j++
		default:
			i++
			j++
		}
	}
	return out
}

// CIDRs returns the minimal CIDR cover of the merged intervals.
func CIDRs(in []Interval) []Block {
	var out []Block
	for _, iv := range Merge(in) {
		cur := iv.First
		for cur <= iv.Last {
			maxSize := uint64(1) << 32
			if cur != 0 {
				maxSize = cur & -cur
			}
			remaining := iv.Last - cur + 1
			for maxSize > remaining {
				maxSize >>= 1
			}
			n := 32 - (bits.Len64(maxSize) - 1)
			out = append(out, Block{uint32(cur), n})
			cur += maxSize
		}
	}
	return out
}

// Ranges returns the maximal contiguous ranges of the set.
func Ranges(in []Interval) []Interval {
	return Merge(in)
}

// Lookup finds the block of a sorted, disjoint cover that holds addr.
func Lookup(blocks []Block, addr uint32) (Block, bool) {
	i := sort.Search(len(blocks), func(i int) bool {
		return blocks[i].Interval().Last >= uint64(addr)
	})
	if i < len(blocks) && blocks[i].Interval().First <= uint64(addr) {
		return blocks[i], true
	}
	return Block{}, false
}
