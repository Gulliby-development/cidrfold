package ipset

import (
	"reflect"
	"testing"
)

func TestComplement(t *testing.T) {
	cases := []struct {
		name string
		in   []Interval
		want []Interval
	}{
		{"empty list", nil, []Interval{iv(0, 0xFFFFFFFF)}},
		{"whole space", []Interval{iv(0, 0xFFFFFFFF)}, nil},
		{"first address", []Interval{iv(0, 0)}, []Interval{iv(1, 0xFFFFFFFF)}},
		{"last address", []Interval{iv(0xFFFFFFFF, 0xFFFFFFFF)}, []Interval{iv(0, 0xFFFFFFFE)}},
		{"middle block", []Interval{iv(0x0A000000, 0x0AFFFFFF)}, []Interval{iv(0, 0x09FFFFFF), iv(0x0B000000, 0xFFFFFFFF)}},
		{"unsorted overlapping", []Interval{iv(20, 30), iv(10, 25), iv(40, 40)}, []Interval{iv(0, 9), iv(31, 39), iv(41, 0xFFFFFFFF)}},
		{"adjacent blocks", []Interval{iv(10, 19), iv(20, 29)}, []Interval{iv(0, 9), iv(30, 0xFFFFFFFF)}},
		{"both halves", []Interval{iv(0, 0x7FFFFFFF), iv(0x80000000, 0xFFFFFFFF)}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Complement(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Complement(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestComplementTwiceIsMerge(t *testing.T) {
	in := []Interval{iv(50, 60), iv(10, 20), iv(21, 30), iv(0xFFFFFF00, 0xFFFFFFFF)}
	want := Merge(in)
	if got := Complement(Complement(in)); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
