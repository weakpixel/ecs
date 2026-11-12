package storage

import (
	"slices"
	"testing"
)

func intColumn(CompID) Column { return NewTypedColumn[int]() }

func TestIndex(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		ix := NewIndex(testColumn)
		if len(ix.Archetypes()) != 1 || ix.Archetypes()[0] != ix.Empty() || ix.Empty().Len() != 0 || len(ix.Empty().Comps()) != 0 {
			t.Fatal("empty archetype missing")
		}
		if ix.Get(nil) != ix.Empty() || ix.Get([]CompID{}) != ix.Empty() {
			t.Error("empty set doesn't return Empty()")
		}
	})
	t.Run("SameSetSamePointer", func(t *testing.T) {
		ix := NewIndex(testColumn)
		a := ix.Get([]CompID{cPos, cInt})
		if b := ix.Get([]CompID{cPos, cInt}); a != b {
			t.Error("different pointers for the same set")
		}
		if c := ix.Get([]CompID{cPos}); c == a {
			t.Error("different sets share an archetype")
		}
		if !slices.Equal(a.Comps(), []CompID{cPos, cInt}) {
			t.Errorf("Comps = %v", a.Comps())
		}
	})
	t.Run("AppendOnly", func(t *testing.T) {
		ix := NewIndex(testColumn)
		sets := [][]CompID{{cPos}, {cVel}, {cPos, cVel}, {cPos}}
		for _, s := range sets {
			ix.Get(s)
		}
		all := ix.Archetypes()
		if len(all) != 4 { // empty + 3 distinct
			t.Fatalf("len = %d", len(all))
		}
		for i, want := range [][]CompID{nil, {cPos}, {cVel}, {cPos, cVel}} {
			if !slices.Equal(all[i].Comps(), want) {
				t.Errorf("archetype %d = %v, want %v", i, all[i].Comps(), want)
			}
		}
	})
	t.Run("LargeSets", func(t *testing.T) {
		ix := NewIndex(intColumn)
		for _, n := range []int{32, 33, 100} {
			ids := make([]CompID, n)
			for i := range ids {
				ids[i] = CompID(i * 2)
			}
			a := ix.Get(ids)
			if ix.Get(ids) != a || len(a.Comps()) != n || a.ColumnIndex(ids[n-1]) != n-1 {
				t.Errorf("n=%d broken", n)
			}
		}
	})
	t.Run("HighIDs", func(t *testing.T) {
		ix := NewIndex(intColumn)
		a := ix.Get([]CompID{0, 4094, 4095})
		if a.ColumnIndex(4095) != 2 || a.ColumnIndex(4094) != 1 || a.ColumnIndex(1) != -1 || a.ColumnIndex(4096) != -1 {
			t.Error("ColumnIndex wrong near the cap")
		}
	})
	t.Run("InvalidSetPanics", func(t *testing.T) {
		ix := NewIndex(testColumn)
		mustPanic(t, "ecs: internal error", func() { ix.Get([]CompID{cVel, cPos}) })
		mustPanic(t, "ecs: internal error", func() { ix.Get([]CompID{cPos, cPos}) })
	})
}

func TestIndexGetAllocs(t *testing.T) {
	ix := NewIndex(testColumn)
	ids := []CompID{cPos, cVel, cName}
	ix.Get(ids)
	if n := testing.AllocsPerRun(100, func() { ix.Get(ids) }); n != 0 {
		t.Errorf("allocs = %v, want 0", n)
	}
}

func TestEdge(t *testing.T) {
	t.Run("MatchesGet", func(t *testing.T) {
		ix := NewIndex(testColumn)
		a := ix.Get([]CompID{cPos, cName})
		if got, want := ix.WithAdded(a, cVel), ix.Get([]CompID{cPos, cVel, cName}); got != want {
			t.Error("WithAdded != Get(union)")
		}
		if got, want := ix.WithRemoved(a, cPos), ix.Get([]CompID{cName}); got != want {
			t.Error("WithRemoved != Get(diff)")
		}
		if ix.WithAdded(ix.Empty(), cInt) != ix.Get([]CompID{cInt}) || ix.WithRemoved(ix.Get([]CompID{cInt}), cInt) != ix.Empty() {
			t.Error("edges to and from the empty set are wrong")
		}
	})
	t.Run("Cached", func(t *testing.T) {
		ix := NewIndex(testColumn)
		a := ix.Get([]CompID{cPos})
		n1 := ix.WithAdded(a, cVel)
		count := len(ix.Archetypes())
		if ix.WithAdded(a, cVel) != n1 || len(ix.Archetypes()) != count {
			t.Error("second WithAdded not cached")
		}
		r1 := ix.WithRemoved(n1, cVel)
		if r1 != a {
			t.Error("remove edge didn't return the original archetype")
		}
		if ix.WithRemoved(n1, cVel) != r1 || len(ix.Archetypes()) != count {
			t.Error("second WithRemoved not cached")
		}
	})
	t.Run("NoOp", func(t *testing.T) {
		ix := NewIndex(testColumn)
		a := ix.Get([]CompID{cPos})
		count := len(ix.Archetypes())
		if ix.WithAdded(a, cPos) != a || ix.WithRemoved(a, cVel) != a || len(ix.Archetypes()) != count {
			t.Error("no-op edges changed the archetype")
		}
	})
	t.Run("CachedAllocs", func(t *testing.T) {
		ix := NewIndex(testColumn)
		a := ix.Get([]CompID{cPos})
		b := ix.WithAdded(a, cVel)
		ix.WithRemoved(b, cVel)
		if n := testing.AllocsPerRun(100, func() {
			ix.WithAdded(a, cVel)
			ix.WithRemoved(b, cVel)
		}); n != 0 {
			t.Errorf("allocs = %v, want 0", n)
		}
	})
}
