package storage

import (
	"runtime"
	"strings"
	"testing"
)

type pos struct{ X, Y float32 }

type named struct {
	Name string
	Ref  *int
}

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", want)
		}
		var msg string
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, want) {
			t.Fatalf("panic %q does not contain %q", msg, want)
		}
	}()
	f()
}

func fill(c *TypedColumn[int], n int) {
	for i := range n {
		c.Append(i * 10)
	}
}

func TestColumn(t *testing.T) {
	t.Run("TypedAndAny", func(t *testing.T) {
		c := NewTypedColumn[pos]()
		c.Append(pos{1, 2})
		c.AppendAny(pos{3, 4})
		c.AppendZero()
		if c.Len() != 3 {
			t.Fatalf("Len = %d", c.Len())
		}
		c.Set(2, pos{5, 6})
		c.SetAny(0, pos{7, 8})
		want := []pos{{7, 8}, {3, 4}, {5, 6}}
		for i, w := range want {
			if *c.Ptr(i) != w {
				t.Errorf("row %d = %v, want %v", i, *c.Ptr(i), w)
			}
			if *c.PtrAny(i).(*pos) != w {
				t.Errorf("PtrAny row %d = %v, want %v", i, *c.PtrAny(i).(*pos), w)
			}
		}
		c.Ptr(1).X = 9
		if c.data[1].X != 9 {
			t.Error("Ptr does not alias storage")
		}
	})
	t.Run("AppendZeroIsZero", func(t *testing.T) {
		c := NewTypedColumn[named]()
		c.AppendZero()
		if c.data[0] != (named{}) {
			t.Errorf("got %v", c.data[0])
		}
	})
	t.Run("WrongTypePanics", func(t *testing.T) {
		c := NewTypedColumn[pos]()
		mustPanic(t, "interface conversion", func() { c.AppendAny(42) })
	})
	t.Run("SwapRemove", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			n    int
			row  int
			want []int
		}{
			{"first", 3, 0, []int{20, 10}},
			{"middle", 4, 1, []int{0, 30, 20}},
			{"last", 3, 2, []int{0, 10}},
			{"only", 1, 0, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := NewTypedColumn[int]()
				fill(c, tc.n)
				c.SwapRemove(tc.row)
				if c.Len() != len(tc.want) {
					t.Fatalf("Len = %d, want %d", c.Len(), len(tc.want))
				}
				for i, w := range tc.want {
					if c.data[i] != w {
						t.Errorf("row %d = %d, want %d", i, c.data[i], w)
					}
				}
				// The vacated slots in the backing array must be zero.
				for i, v := range c.data[len(c.data):cap(c.data)][:tc.n-len(c.data)] {
					if v != 0 {
						t.Errorf("vacated slot %d = %d, want 0", i, v)
					}
				}
			})
		}
	})
	t.Run("SwapRemoveZeroesPointers", func(t *testing.T) {
		c := NewTypedColumn[named]()
		x := 1
		c.Append(named{"a", &x})
		c.Append(named{"b", &x})
		c.SwapRemove(0)
		if got := c.data[:2][1]; got != (named{}) {
			t.Errorf("vacated slot = %v, want zero", got)
		}
		runtime.KeepAlive(&x)
	})
	t.Run("MoveRowTo", func(t *testing.T) {
		src, dst := NewTypedColumn[int](), NewTypedColumn[int]()
		fill(src, 3)
		src.MoveRowTo(dst, 1)
		if src.Len() != 3 || dst.Len() != 1 || dst.data[0] != 10 {
			t.Errorf("src=%v dst=%v", src.data, dst.data)
		}
		mustPanic(t, "interface conversion", func() { src.MoveRowTo(NewTypedColumn[pos](), 0) })
	})
	t.Run("PtrAnyAllocs", func(t *testing.T) {
		var c Column = NewTypedColumn[pos]()
		c.AppendZero()
		var sink any
		if n := testing.AllocsPerRun(100, func() { sink = c.PtrAny(0) }); n != 0 {
			t.Errorf("allocs = %v, want 0", n)
		}
		_ = sink
	})
}
