package ecs

import (
	"iter"
	"slices"

	"github.com/weakpixel/ecs/internal/storage"
)

// Query is a reusable view over every entity whose archetype contains all of
// a fixed set of components. Copies of a Query share the same underlying
// state (the archetype-match cache), so it is cheap to pass around and
// store; build one per World and iterate it as many times as needed.
//
// The zero Query is not bound to any world; calling Each on it panics.
type Query struct {
	state *queryState
}

// queryState is the mutable state shared by every copy of a Query. Kept
// behind a pointer so copies of the small Query value all see the same
// archetype-match cache.
type queryState struct {
	world   *World
	ids     []storage.CompID // request order; also Each's output order
	matched []*storage.Archetype
	scanned int // len(world.index.Archetypes()) already scanned for matches

	// buf and cols are sized len(ids) once, at construction, and reused by
	// every non-nested Each call: ids never changes for this Query's life, so
	// neither buffer ever needs to grow.
	buf  []any
	cols []storage.Column

	iterating bool // true while a non-nested Each is yielding
}

// Query returns a Query over every entity whose archetype contains all of
// ids. Order matters: it is the order Each's []any slice uses. Query() with
// no ids matches every archetype.
//
// Panics with "ecs: component not registered: ComponentID(<id>)" if any id
// is unknown, and with "ecs: duplicate component: <type>" if the same id is
// given twice.
func (w *World) Query(ids ...ComponentID) Query {
	cids := make([]storage.CompID, len(ids))
	for i, id := range ids {
		cid := storage.CompID(id)
		w.checkRegistered(cid)
		cids[i] = cid
	}

	// Duplicate check via a sorted scratch copy, mirroring validateValues.
	// cids itself (request order) is what the Query keeps.
	sorted := append(w.scratch.sorted[:0], cids...)
	slices.Sort(sorted)
	w.scratch.sorted = sorted
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1] == sorted[i] {
			panicDuplicateComp(sorted[i])
		}
	}

	return Query{state: &queryState{
		world: w,
		ids:   cids,
		buf:   make([]any, len(cids)),
		cols:  make([]storage.Column, len(cids)),
	}}
}

// archMatches reports whether arch has every component in ids.
func archMatches(arch *storage.Archetype, ids []storage.CompID) bool {
	for _, id := range ids {
		if !arch.Has(id) {
			return false
		}
	}
	return true
}

// scan extends matched with every archetype created since the last scan that
// has every component in ids. It never rescans from the start, so a Query
// built before new archetypes appear still sees entities created in them
// later.
func (st *queryState) scan() {
	archetypes := st.world.index.Archetypes()
	for _, arch := range archetypes[st.scanned:] {
		if archMatches(arch, st.ids) {
			st.matched = append(st.matched, arch)
		}
	}
	st.scanned = len(archetypes)
}

// Each yields every entity currently matching the query and, for each, a
// []any holding one pointer per requested component, in request order.
//
// Do not retain the []any past the loop body; it is reused between
// iterations (nested iteration of the same Query gets its own, freshly
// allocated). Replacing a component's value, or writing through a yielded
// pointer, is safe during iteration. A structural change (NewEntity,
// DeleteEntity, or SetComp/RemoveComp that adds or removes a component)
// panics with "ecs: query invalidated by world mutation" once the loop body
// that caused it returns true to continue; breaking out of the loop (the
// body returning false) is always safe, even right after such a mutation.
//
// Panics with "ecs: not bound to a world" if Query is the zero value.
func (q Query) Each() iter.Seq2[Entity, []any] {
	st := q.state
	if st == nil {
		panicNotBound()
	}
	return func(yield func(Entity, []any) bool) {
		st.scan()

		buf, cols := st.buf, st.cols
		if st.iterating {
			// A nested iteration of this same Query: the shared buffers are
			// already in use by the outer loop. This is the one allocation
			// Each can make, kept off the (far more common) non-nested path.
			buf = make([]any, len(st.ids))
			cols = make([]storage.Column, len(st.ids))
		} else {
			st.iterating = true
			defer func() { st.iterating = false }()
		}

		rev := st.world.revision
		for _, arch := range st.matched {
			n := arch.Len()
			if n == 0 {
				continue
			}
			for i, id := range st.ids {
				cols[i] = arch.Column(arch.ColumnIndex(id))
			}
			for row := range n {
				for i, c := range cols {
					buf[i] = c.PtrAny(row)
				}
				e := entityFromStorage(arch.Entity(row))
				if !yield(e, buf) {
					return
				}
				if st.world.revision != rev {
					panicQueryInvalidated()
				}
			}
		}
	}
}
