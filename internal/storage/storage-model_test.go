package storage

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// modelRec is the test's stand-in for the World's entity record.
type modelRec struct {
	arch *Archetype
	row  int
}

// modelValue encodes the entity ID and component ID into a value, so a row
// that ends up with another entity's data is detected.
func modelValue(e Entity, id CompID) int { return int(e.ID)*100 + int(id) + 1 }

// TestModel runs random spawn, delete, add and remove operations against the
// storage and checks it against a reference map after every step.
func TestModel(t *testing.T) {
	const comps = 6
	for seed := uint64(1); seed <= 3; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed*7919))
		ix := NewIndex(intColumn)
		live := map[uint32]*modelRec{}
		var nextID uint32

		write := func(e Entity, a *Archetype, row int, ids []CompID) {
			for _, id := range ids {
				a.Column(a.ColumnIndex(id)).SetAny(row, modelValue(e, id))
			}
		}
		fixMoved := func(a *Archetype, row int, moved Entity, ok bool) {
			if ok {
				live[moved.ID].arch, live[moved.ID].row = a, row
			}
		}
		pick := func() (uint32, *modelRec) {
			keys := make([]uint32, 0, len(live))
			for k := range live {
				keys = append(keys, k)
			}
			slices.Sort(keys) // deterministic given the seed
			k := keys[rng.IntN(len(keys))]
			return k, live[k]
		}

		for step := range 3000 {
			op := rng.IntN(4)
			if len(live) == 0 {
				op = 0
			}
			switch op {
			case 0: // spawn with a random set
				var ids []CompID
				for id := range CompID(comps) {
					if rng.IntN(2) == 0 {
						ids = append(ids, id)
					}
				}
				e := Entity{ID: nextID, Generation: 1}
				nextID++
				a := ix.Get(ids)
				row := a.AppendRow(e)
				write(e, a, row, ids)
				live[e.ID] = &modelRec{a, row}
			case 1: // delete
				id, r := pick()
				moved, ok := r.arch.RemoveRow(r.row)
				fixMoved(r.arch, r.row, moved, ok)
				delete(live, id)
			case 2: // add one component
				id, r := pick()
				c := CompID(rng.IntN(comps))
				dst := ix.WithAdded(r.arch, c)
				if dst == r.arch {
					break
				}
				e := Entity{ID: id, Generation: 1}
				src, srcRow := r.arch, r.row
				dstRow, moved, ok := MoveRow(src, srcRow, dst)
				fixMoved(src, srcRow, moved, ok)
				r.arch, r.row = dst, dstRow
				write(e, dst, dstRow, []CompID{c})
			case 3: // remove one component
				_, r := pick()
				c := CompID(rng.IntN(comps))
				dst := ix.WithRemoved(r.arch, c)
				if dst == r.arch {
					break
				}
				src, srcRow := r.arch, r.row
				dstRow, moved, ok := MoveRow(src, srcRow, dst)
				fixMoved(src, srcRow, moved, ok)
				r.arch, r.row = dst, dstRow
			}

			// Invariants: aligned columns, every live entity where its record
			// says with the right values, and the row count adds up.
			total := 0
			for _, a := range ix.Archetypes() {
				checkAligned(t, a)
				total += a.Len()
			}
			if total != len(live) {
				t.Fatalf("seed %d step %d: %d rows, %d live entities", seed, step, total, len(live))
			}
			for id, r := range live {
				e := Entity{ID: id, Generation: 1}
				if r.arch.Entity(r.row) != e {
					t.Fatalf("seed %d step %d: entity %d not at its record", seed, step, id)
				}
				for _, c := range r.arch.Comps() {
					got := *r.arch.Column(r.arch.ColumnIndex(c)).(*TypedColumn[int]).Ptr(r.row)
					if got != modelValue(e, c) {
						t.Fatalf("seed %d step %d: entity %d comp %d = %d, want %d", seed, step, id, c, got, modelValue(e, c))
					}
				}
			}
		}
	}
}
