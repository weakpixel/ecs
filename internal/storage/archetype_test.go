package storage

import (
	"runtime"
	"testing"
)

// Test component IDs.
const (
	cPos CompID = iota
	cVel
	cName
	cInt
)

// testColumn is the column factory used by the tests.
func testColumn(id CompID) Column {
	switch id {
	case cPos, cVel:
		return NewTypedColumn[pos]()
	case cName:
		return NewTypedColumn[named]()
	case cInt:
		return NewTypedColumn[int]()
	}
	panic("unknown test component")
}

func ent(id uint32) Entity { return Entity{ID: id, Generation: 1} }

// addEntity appends a row for entity id and returns its row.
func addEntity(a *Archetype, id uint32) int { return a.AppendRow(ent(id)) }

func checkAligned(t *testing.T, a *Archetype) {
	t.Helper()
	for i, c := range a.cols {
		if c.Len() != a.Len() {
			t.Fatalf("column %d (comp %d) len = %d, want %d", i, a.comps[i], c.Len(), a.Len())
		}
	}
}

func TestArchetypeAccessors(t *testing.T) {
	a := newArchetype([]CompID{cVel, cInt}, testColumn)
	if !a.Has(cVel) || !a.Has(cInt) || a.Has(cPos) || a.Has(cName) || a.Has(4000) {
		t.Error("Has wrong")
	}
	if a.ColumnIndex(cVel) != 0 || a.ColumnIndex(cInt) != 1 || a.ColumnIndex(cPos) != -1 || a.ColumnIndex(4000) != -1 {
		t.Error("ColumnIndex wrong")
	}
	if _, ok := a.Column(a.ColumnIndex(cInt)).(*TypedColumn[int]); !ok {
		t.Error("Column type wrong")
	}
	row := addEntity(a, 7)
	if row != 0 || a.Len() != 1 || a.Entity(0) != ent(7) || len(a.Entities()) != 1 {
		t.Error("AppendRow bookkeeping wrong")
	}
	checkAligned(t, a)

	empty := newArchetype(nil, testColumn)
	if empty.Has(0) || empty.ColumnIndex(0) != -1 {
		t.Error("empty archetype reports components")
	}
	addEntity(empty, 1)
	if empty.Len() != 1 {
		t.Error("empty archetype can't hold rows")
	}
}

func TestArchetypeRemoveRow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		n, row int
	}{
		{"first", 3, 0},
		{"middle", 4, 1},
		{"last", 3, 2},
		{"only", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newArchetype([]CompID{cInt}, testColumn)
			col := a.Column(0).(*TypedColumn[int])
			for i := range tc.n {
				r := addEntity(a, uint32(i))
				col.Set(r, i*10)
			}
			moved, ok := a.RemoveRow(tc.row)
			if ok != (tc.row != tc.n-1) {
				t.Fatalf("ok = %v", ok)
			}
			if ok {
				if moved != ent(uint32(tc.n-1)) {
					t.Errorf("moved = %v, want entity %d", moved, tc.n-1)
				}
				if a.Entity(tc.row) != moved || *col.Ptr(tc.row) != (tc.n-1)*10 {
					t.Error("moved entity's row is wrong")
				}
			}
			if a.Len() != tc.n-1 {
				t.Errorf("Len = %d", a.Len())
			}
			checkAligned(t, a)
			// The vacated entity slot is cleared.
			if tail := a.entities[:tc.n][tc.n-1]; tail != (Entity{}) {
				t.Errorf("vacated entity slot = %v", tail)
			}
		})
	}
}

func TestMoveRow(t *testing.T) {
	setup := func(comps []CompID, n int) *Archetype {
		a := newArchetype(comps, testColumn)
		for i := range n {
			r := addEntity(a, uint32(i))
			if a.Has(cInt) {
				a.Column(a.ColumnIndex(cInt)).SetAny(r, i*10)
			}
			if a.Has(cName) {
				a.Column(a.ColumnIndex(cName)).SetAny(r, named{Name: "n", Ref: new(int)})
			}
		}
		return a
	}
	for _, tc := range []struct {
		name     string
		src, dst []CompID
	}{
		{"add", []CompID{cInt}, []CompID{cName, cInt}},
		{"remove", []CompID{cName, cInt}, []CompID{cInt}},
		{"overlap", []CompID{cPos, cInt}, []CompID{cName, cInt}},
		{"fromEmpty", nil, []CompID{cInt}},
		{"toEmpty", []CompID{cInt}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src, dst := setup(tc.src, 3), newArchetype(tc.dst, testColumn)
			dstRow, moved, ok := MoveRow(src, 0, dst)
			if dstRow != 0 || !ok || moved != ent(2) {
				t.Fatalf("dstRow=%d moved=%v ok=%v", dstRow, moved, ok)
			}
			if dst.Entity(0) != ent(0) || src.Entity(0) != ent(2) || src.Len() != 2 || dst.Len() != 1 {
				t.Error("entity tables wrong")
			}
			checkAligned(t, src)
			checkAligned(t, dst)
			if dst.Has(cInt) {
				want := 0
				if got := *dst.Column(dst.ColumnIndex(cInt)).(*TypedColumn[int]).Ptr(0); got != want {
					t.Errorf("moved int = %d", got)
				}
				if src.Has(cInt) {
					if got := *src.Column(src.ColumnIndex(cInt)).(*TypedColumn[int]).Ptr(0); got != 20 {
						t.Errorf("swapped int = %d, want 20", got)
					}
				}
			}
			if dst.Has(cName) {
				got := *dst.Column(dst.ColumnIndex(cName)).(*TypedColumn[named]).Ptr(0)
				if src.Has(cName) {
					if got.Name != "n" || got.Ref == nil {
						t.Errorf("shared name lost: %v", got)
					}
				} else if got != (named{}) {
					t.Errorf("new component not zero: %v", got)
				}
			}
			// Moving the last row reports no swap.
			if _, _, ok := MoveRow(src, src.Len()-1, dst); ok {
				t.Error("last-row move reported a swap")
			}
		})
	}

	t.Run("SurvivesGC", func(t *testing.T) {
		a, b := setup([]CompID{cName}, 50), newArchetype([]CompID{cName, cInt}, testColumn)
		for range 50 {
			MoveRow(a, 0, b)
			runtime.GC()
		}
		runtime.GC()
		col := b.Column(b.ColumnIndex(cName)).(*TypedColumn[named])
		for i := range 50 {
			if v := *col.Ptr(i); v.Name != "n" || v.Ref == nil {
				t.Fatalf("row %d corrupted: %v", i, v)
			}
		}
	})
}

func TestMoveRowAllocs(t *testing.T) {
	a := newArchetype([]CompID{cPos, cInt}, testColumn)
	b := newArchetype([]CompID{cPos, cName, cInt}, testColumn)
	for i := range 4 {
		addEntity(a, uint32(i))
	}
	// Warm up so the destination columns have capacity.
	MoveRow(a, 0, b)
	MoveRow(b, 0, a)
	if n := testing.AllocsPerRun(100, func() {
		MoveRow(a, 0, b)
		MoveRow(b, b.Len()-1, a)
	}); n != 0 {
		t.Errorf("allocs = %v, want 0", n)
	}
}
