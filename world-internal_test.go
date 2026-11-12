package ecs

import (
	"testing"

	"github.com/weakpixel/ecs/internal/storage"
)

type weTag struct{ N int }

// weComp returns a *weTag pointing into e's live storage, for tests that
// need to check component values before GetComp exists.
func weComp(t *testing.T, w *World, id ComponentID, e Entity) *weTag {
	t.Helper()
	rec := w.record(e)
	return rec.Arch.Column(rec.Arch.ColumnIndex(storage.CompID(id))).PtrAny(int(rec.Row)).(*weTag)
}

func TestSwapRemoveFixesMovedEntity(t *testing.T) {
	id := RegisterComp[weTag]()
	w := NewWorld()
	a := w.NewEntity(weTag{N: 1})
	b := w.NewEntity(weTag{N: 2})
	c := w.NewEntity(weTag{N: 3})

	w.DeleteEntity(a)

	if w.IsAlive(a) {
		t.Fatalf("a still alive after its own delete")
	}
	if !w.IsAlive(b) || !w.IsAlive(c) {
		t.Fatalf("b or c not alive after deleting a")
	}
	// c was the last row, so swap-remove should have moved it into a's old
	// row without changing its value.
	if got := weComp(t, w, id, c); got.N != 3 {
		t.Fatalf("c's component = %d, want 3", got.N)
	}
	if got := weComp(t, w, id, b); got.N != 2 {
		t.Fatalf("b's component = %d, want 2", got.N)
	}
}

func TestDeleteAnyOrderRollback(t *testing.T) {
	id := RegisterComp[weTag]()
	w := NewWorld()
	const n = 5
	es := make([]Entity, n)
	for i := range n {
		es[i] = w.NewEntity(weTag{N: i})
	}

	order := []int{2, 0, 4, 1, 3}
	deleted := make([]bool, n)
	for _, idx := range order {
		w.DeleteEntity(es[idx])
		deleted[idx] = true

		for j, e := range es {
			alive := w.IsAlive(e)
			if deleted[j] {
				if alive {
					t.Fatalf("entity %d still alive after its own delete", j)
				}
				continue
			}
			if !alive {
				t.Fatalf("entity %d not alive, but was never deleted", j)
			}
			if got := weComp(t, w, id, e); got.N != j {
				t.Fatalf("entity %d component = %d, want %d", j, got.N, j)
			}
		}
	}
	if w.entities.Len() != 0 {
		t.Fatalf("entities.Len() = %d after deleting all, want 0", w.entities.Len())
	}
}

type weDup struct{ N int }

func TestFailedNewEntityLeavesNoEntity(t *testing.T) {
	RegisterComp[weDup]()
	w := NewWorld()
	beforeRev, beforeLen := w.revision, w.entities.Len()

	func() {
		defer func() { recover() }()
		w.NewEntity(weDup{N: 1}, weDup{N: 2}) // duplicate type, panics
	}()

	if w.revision != beforeRev {
		t.Fatalf("revision changed from %d to %d after failed NewEntity", beforeRev, w.revision)
	}
	if w.entities.Len() != beforeLen {
		t.Fatalf("entity count changed from %d to %d after failed NewEntity", beforeLen, w.entities.Len())
	}
}

func TestSetCompInPlaceKeepsRevision(t *testing.T) {
	type wiPos struct{ X, Y int }
	posID := RegisterComp[wiPos]()
	w := NewWorld()
	e := w.NewEntity(wiPos{X: 1, Y: 2})
	before := w.revision

	w.SetComp(e, wiPos{X: 9, Y: 9})

	if w.revision != before {
		t.Fatalf("revision changed from %d to %d for in-place SetComp", before, w.revision)
	}
	got := w.GetComp(e, posID).(*wiPos)
	if got.X != 9 || got.Y != 9 {
		t.Fatalf("value after in-place SetComp = %+v", got)
	}
}

func TestSetCompMultiAddMovesOnce(t *testing.T) {
	type (
		wiA struct{ N int }
		wiB struct{ N int }
		wiC struct{ N int }
	)
	RegisterComp[wiA]()
	RegisterComp[wiB]()
	RegisterComp[wiC]()
	w := NewWorld()
	e := w.NewEntity(wiA{N: 1})
	beforeRev := w.revision
	beforeArchCount := len(w.index.Archetypes())

	w.SetComp(e, wiB{N: 2}, wiC{N: 3})

	if w.revision != beforeRev+1 {
		t.Fatalf("revision = %d, want %d (exactly one move)", w.revision, beforeRev+1)
	}
	if got := len(w.index.Archetypes()); got != beforeArchCount+1 {
		t.Fatalf("archetype count = %d, want %d (exactly one new archetype)", got, beforeArchCount+1)
	}
}

func TestRemoveCompNoRevisionBumpWhenNoop(t *testing.T) {
	type (
		wiPos struct{ X, Y int }
		wiVel struct{ X, Y int }
	)
	RegisterComp[wiPos]()
	velID := RegisterComp[wiVel]()
	w := NewWorld()
	e := w.NewEntity(wiPos{X: 1, Y: 2})
	before := w.revision

	w.RemoveComp(e)        // no ids
	w.RemoveComp(e, velID) // absent id

	if w.revision != before {
		t.Fatalf("revision changed from %d to %d for no-op RemoveComp", before, w.revision)
	}
}
