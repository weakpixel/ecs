package storage

import (
	"slices"
	"testing"
)

func TestCompSet(t *testing.T) {
	t.Run("IsSet", func(t *testing.T) {
		for _, tc := range []struct {
			ids  []CompID
			want bool
		}{
			{nil, true}, {[]CompID{3}, true}, {[]CompID{1, 2, 9}, true},
			{[]CompID{2, 1}, false}, {[]CompID{1, 1}, false},
		} {
			if got := IsSet(tc.ids); got != tc.want {
				t.Errorf("IsSet(%v) = %v, want %v", tc.ids, got, tc.want)
			}
		}
	})
	t.Run("Contains", func(t *testing.T) {
		s := []CompID{1, 4, 7}
		for id, want := range map[CompID]bool{0: false, 1: true, 4: true, 5: false, 7: true, 8: false} {
			if Contains(s, id) != want {
				t.Errorf("Contains(%v, %d) != %v", s, id, want)
			}
		}
	})
	t.Run("UnionDiff", func(t *testing.T) {
		for _, tc := range []struct{ a, b, union, diff []CompID }{
			{nil, nil, nil, nil},
			{[]CompID{1, 2}, nil, []CompID{1, 2}, []CompID{1, 2}},
			{nil, []CompID{1, 2}, []CompID{1, 2}, nil},
			{[]CompID{1, 3, 5}, []CompID{2, 3, 6}, []CompID{1, 2, 3, 5, 6}, []CompID{1, 5}},
			{[]CompID{1, 2}, []CompID{1, 2}, []CompID{1, 2}, nil},
		} {
			if got := Union(nil, tc.a, tc.b); !slices.Equal(got, tc.union) {
				t.Errorf("Union(%v,%v) = %v, want %v", tc.a, tc.b, got, tc.union)
			}
			if got := Diff(nil, tc.a, tc.b); !slices.Equal(got, tc.diff) {
				t.Errorf("Diff(%v,%v) = %v, want %v", tc.a, tc.b, got, tc.diff)
			}
		}
	})
	t.Run("NoAllocs", func(t *testing.T) {
		a, b := []CompID{1, 3, 5}, []CompID{2, 3}
		scratch := make([]CompID, 0, 8)
		if n := testing.AllocsPerRun(100, func() {
			scratch = Union(scratch, a, b)
			scratch = Diff(scratch, a, b)
		}); n != 0 {
			t.Errorf("allocs = %v, want 0", n)
		}
	})
}
