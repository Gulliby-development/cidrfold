package ipset

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

func iv(first, last uint64) Interval { return Interval{first, last} }

func TestParseAccepted(t *testing.T) {
	one := []Interval{iv(0x0A000001, 0x0A000001)}
	two := []Interval{iv(0x0A000001, 0x0A000001), iv(0x0A000002, 0x0A000002)}
	cases := []struct {
		name, in string
		want     []Interval
	}{
		{"cidr", "10.0.0.0/24", []Interval{iv(0x0A000000, 0x0A0000FF)}},
		{"host bits", "10.0.0.5/24", []Interval{iv(0x0A000000, 0x0A0000FF)}},
		{"bare", "10.0.0.1", one},
		{"spaces tabs", "  10.0.0.1 \t", one},
		{"trailing comment", "10.0.0.1 # c", one},
		{"tab comment", "10.0.0.1\t#c", one},
		{"crlf", "10.0.0.1\r\n10.0.0.2\r\n", two},
		{"bom", "\uFEFF10.0.0.1\n", one},
		{"no final newline", "10.0.0.1\n10.0.0.2", two},
		{"blank and comment", "\n# x\n  # y\n10.0.0.1\n", one},
		{"slash zero", "0.0.0.0/0", []Interval{iv(0, 0xFFFFFFFF)}},
		{"slash zero host bits", "1.2.3.4/0", []Interval{iv(0, 0xFFFFFFFF)}},
		{"max host", "255.255.255.255", []Interval{iv(0xFFFFFFFF, 0xFFFFFFFF)}},
		{"slash 32", "1.2.3.4/32", []Interval{iv(0x01020304, 0x01020304)}},
		{"empty", "", nil},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestParseRejected(t *testing.T) {
	cases := []struct {
		in   string
		line int
		text string
	}{
		{"::1", 1, "::1"},
		{"10.0.0.0/33", 1, "10.0.0.0/33"},
		{"10.0.0.0/08", 1, "10.0.0.0/08"},
		{"010.0.0.1", 1, "010.0.0.1"},
		{"10.0.0.256", 1, "10.0.0.256"},
		{"host.example", 1, "host.example"},
		{"10.0.0.1 10.0.0.2", 1, "10.0.0.1 10.0.0.2"},
		{"10.0.0.1#x", 1, "10.0.0.1#x"},
		{"+1/24", 1, "+1/24"},
		{"10.0.0.0/+4", 1, "10.0.0.0/+4"},
		{"10.0.0.0/", 1, "10.0.0.0/"},
		{"10.0.0/24", 1, "10.0.0/24"},
		{"10.0.0.0.1", 1, "10.0.0.0.1"},
		{"10.0.0.1/-1", 1, "10.0.0.1/-1"},
		{"10.0.0.0/24\n\n# c\n10.0.0.256", 4, "10.0.0.256"},
		{"\uFEFF10.0.0.0/24\r\n\r\n# note\r\n10.0.0.256\r\n", 4, "10.0.0.256"},
		{"\uFEFF::1", 1, "::1"},
		{"10.0.0.1\n10.0.0.2 x # c\n", 2, "10.0.0.2 x"},
		{"10.0.0.1\nbad\nworse", 2, "bad"},
	}
	for _, c := range cases {
		_, err := Parse(c.in)
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("%q: expected ParseError, got %v", c.in, err)
			continue
		}
		if pe.Line != c.line || pe.Text != c.text {
			t.Errorf("%q: got line %d text %q, want %d %q", c.in, pe.Line, pe.Text, c.line, c.text)
		}
	}
}

func TestParseAddr(t *testing.T) {
	if a, ok := ParseAddr("10.0.1.77"); !ok || a != 0x0A00014D {
		t.Errorf("got %x %v", a, ok)
	}
	for _, s := range []string{"10.0.0.1/32", "", "1.2.3", "01.2.3.4", "1.2.3.256", " 1.2.3.4", "a.b.c.d"} {
		if _, ok := ParseAddr(s); ok {
			t.Errorf("%q should be rejected", s)
		}
	}
}

func blocksStr(bs []Block) []string {
	var out []string
	for _, b := range bs {
		out = append(out, b.String())
	}
	return out
}

func mustParse(t *testing.T, s string) []Interval {
	t.Helper()
	ivs, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return ivs
}

func TestEdges(t *testing.T) {
	in := mustParse(t, "0.0.0.0/1\n128.0.0.0/1\n")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"0.0.0.0/0"}) {
		t.Errorf("halves: %v", got)
	}
	in = mustParse(t, "0.0.0.0/0")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"0.0.0.0/0"}) {
		t.Errorf("all: %v", got)
	}
	if got := FormatRange(Ranges(in)[0]); got != "0.0.0.0-255.255.255.255" {
		t.Errorf("range all: %s", got)
	}
	in = mustParse(t, "255.255.255.255\n255.255.255.254\n")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"255.255.255.254/31"}) {
		t.Errorf("top: %v", got)
	}
	in = mustParse(t, "255.255.255.255")
	if got := FormatRange(Ranges(in)[0]); got != "255.255.255.255-255.255.255.255" {
		t.Errorf("single: %s", got)
	}
	in = mustParse(t, "0.0.0.0")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"0.0.0.0/32"}) {
		t.Errorf("zero: %v", got)
	}
	all := mustParse(t, "0.0.0.0/0")
	if got := Subtract(all, all); len(got) != 0 {
		t.Errorf("all-all: %v", got)
	}
	got := blocksStr(CIDRs(Subtract(all, mustParse(t, "0.0.0.0\n255.255.255.255"))))
	if len(got) != 62 || got[0] != "0.0.0.1/32" || got[len(got)-1] != "255.255.255.254/32" {
		t.Errorf("all minus ends: %v", got)
	}
}

func TestWorkedExamples(t *testing.T) {
	in := mustParse(t, "10.0.0.0/25\n10.0.0.128/25\n10.0.1.0/24\n192.168.1.5\n192.168.1.4\n")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"10.0.0.0/23", "192.168.1.4/31"}) {
		t.Errorf("cidr: %v", got)
	}
	var rs []string
	for _, r := range Ranges(in) {
		rs = append(rs, FormatRange(r))
	}
	if !reflect.DeepEqual(rs, []string{"10.0.0.0-10.0.1.255", "192.168.1.4-192.168.1.5"}) {
		t.Errorf("range: %v", rs)
	}
	in = mustParse(t, "10.0.0.5/24\n10.0.0.77\n172.16.0.0/12\n172.20.0.0/16\n")
	if got := blocksStr(CIDRs(in)); !reflect.DeepEqual(got, []string{"10.0.0.0/24", "172.16.0.0/12"}) {
		t.Errorf("second: %v", got)
	}
	sub := Subtract(mustParse(t, "10.0.0.0/24"), mustParse(t, "10.0.0.64/26"))
	if got := blocksStr(CIDRs(sub)); !reflect.DeepEqual(got, []string{"10.0.0.0/26", "10.0.0.128/25"}) {
		t.Errorf("subtract: %v", got)
	}
	if got := Subtract(mustParse(t, "10.0.0.0/24"), mustParse(t, "10.0.0.0/16")); len(got) != 0 {
		t.Errorf("subtract empty: %v", got)
	}
}

func TestLookup(t *testing.T) {
	bs := CIDRs(mustParse(t, "10.0.0.0/23\n255.255.255.255\n0.0.0.0"))
	cases := []struct {
		addr string
		want string
	}{
		{"10.0.1.77", "10.0.0.0/23"},
		{"10.0.2.1", ""},
		{"255.255.255.255", "255.255.255.255/32"},
		{"0.0.0.0", "0.0.0.0/32"},
		{"0.0.0.1", ""},
		{"255.255.255.254", ""},
	}
	for _, c := range cases {
		a, _ := ParseAddr(c.addr)
		b, ok := Lookup(bs, a)
		if c.want == "" {
			if ok {
				t.Errorf("%s: unexpected %v", c.addr, b)
			}
			continue
		}
		if !ok || b.String() != c.want {
			t.Errorf("%s: got %v %v want %s", c.addr, b, ok, c.want)
		}
	}
	if _, ok := Lookup(nil, 5); ok {
		t.Error("empty cover must hold nothing")
	}
}

const base = 0x0A000000

func genSet(r *rand.Rand) ([]Interval, [1024]bool) {
	var set [1024]bool
	var ivs []Interval
	n := 1 + r.Intn(8)
	for i := 0; i < n; i++ {
		bits := 22 + r.Intn(11)
		size := 1 << uint(32-bits)
		start := r.Intn(1024/size) * size
		ivs = append(ivs, Block{uint32(base + start), bits}.Interval())
		for j := start; j < start+size; j++ {
			set[j] = true
		}
	}
	return ivs, set
}

func coverOf(t *testing.T, bs []Block) [1024]bool {
	t.Helper()
	var set [1024]bool
	for _, b := range bs {
		x := b.Interval()
		if x.First < base || x.Last >= base+1024 {
			t.Fatalf("block %v outside domain", b)
		}
		for a := x.First; a <= x.Last; a++ {
			if set[a-base] {
				t.Fatalf("block %v overlaps another", b)
			}
			set[a-base] = true
		}
	}
	return set
}

func checkBlocks(t *testing.T, tag string, bs []Block, want [1024]bool) {
	t.Helper()
	if coverOf(t, bs) != want {
		t.Fatalf("%s: coverage differs from oracle: %v", tag, blocksStr(bs))
	}
	for i := 1; i < len(bs); i++ {
		p, c := bs[i-1], bs[i]
		pi, ci := p.Interval(), c.Interval()
		if pi.Last >= ci.First {
			t.Fatalf("%s: not sorted/disjoint: %v", tag, blocksStr(bs))
		}
		if p.Bits == c.Bits && p.Bits > 0 && pi.Last+1 == ci.First &&
			uint64(p.Addr)%(uint64(1)<<uint(33-p.Bits)) == 0 {
			t.Fatalf("%s: %v and %v are mergeable", tag, p, c)
		}
	}
}

func checkRanges(t *testing.T, tag string, rs []Interval, want [1024]bool) {
	t.Helper()
	var got [1024]bool
	for i, r := range rs {
		if r.First > r.Last {
			t.Fatalf("%s: bad range %v", tag, r)
		}
		if i > 0 && rs[i-1].Last+1 >= r.First {
			t.Fatalf("%s: ranges not maximal/sorted: %v", tag, rs)
		}
		for a := r.First; a <= r.Last; a++ {
			got[a-base] = true
		}
	}
	if got != want {
		t.Fatalf("%s: range coverage differs from oracle", tag)
	}
}

func TestOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20240607))
	for i := 0; i < 500; i++ {
		a, as := genSet(r)
		b, bs := genSet(r)

		checkBlocks(t, "merge", CIDRs(a), as)
		checkRanges(t, "merge ranges", Ranges(a), as)

		var diff, union [1024]bool
		for j := range diff {
			diff[j] = as[j] && !bs[j]
			union[j] = as[j] || bs[j]
		}
		d := Subtract(a, b)
		checkBlocks(t, "subtract", CIDRs(d), diff)
		checkRanges(t, "subtract ranges", Ranges(d), diff)
		checkBlocks(t, "union", CIDRs(append(append([]Interval{}, a...), b...)), union)

		cover := CIDRs(a)
		for j := 0; j < 1024; j += 37 {
			blk, ok := Lookup(cover, uint32(base+j))
			if ok != as[j] {
				t.Fatalf("lookup %d: got %v want %v", j, ok, as[j])
			}
			if ok {
				x := blk.Interval()
				if uint64(base+j) < x.First || uint64(base+j) > x.Last {
					t.Fatalf("lookup %d returned wrong block %v", j, blk)
				}
			}
		}
	}
}
