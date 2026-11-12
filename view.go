package ecs

import (
	"slices"

	"github.com/weakpixel/ecs/internal/entities"
	"github.com/weakpixel/ecs/internal/storage"
)

// viewState is the state shared by every copy of one ViewN, N = 1..16 (see
// zz_gen_generic-api.go). It holds everything the generated ViewN methods
// need that doesn't depend on the concrete T1..TN: resolving e's components,
// deciding whether a Set moves it, and caching the archetypes that match
// Each.
type viewState struct {
	world  *World
	ids    []storage.CompID // type-parameter order: what Get/Set/NewEntity index into for column k
	sorted []storage.CompID // same set, ascending and duplicate-free: what archetype operations need

	// without holds the components excluded by Without filters, ascending;
	// nil without filters. Each and Has drop archetypes holding any of them.
	without []storage.CompID

	spawn *storage.Archetype // ViewN.NewEntity's target archetype; nil until first use

	matched []*storage.Archetype // Each's archetype-match cache, same shape as queryState
	scanned int                  // len(world.index.Archetypes()) already scanned for matches

	// cols is Each's per-archetype column buffer, sized len(ids) once here and
	// reused by every non-nested Each call, exactly like queryState.buf/cols.
	// A nested iteration of the same View (iterating already true) allocates
	// its own, same as Query.Each: the call that already owns cols is still
	// mid-loop, using it.
	cols      []storage.Column
	iterating bool

	presentBuf []storage.CompID // Remove's scratch: the subset of sorted that e currently has
}

// newViewState builds the shared state for a view constructed with ids in
// type-parameter order (what the generated NewViewN just resolved via
// RegisterComp) and the constructor's filters. ids is retained; the caller
// must not reuse its backing array.
//
// Panics with "ecs: duplicate component: <type>" if any type repeats, in
// ids or among the filters, and with the filter panics of excludedIDs.
func newViewState(w *World, ids []storage.CompID, filters []Filter) *viewState {
	sorted := append([]storage.CompID(nil), ids...)
	slices.Sort(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1] == sorted[i] {
			panicDuplicateComp(sorted[i])
		}
	}
	return &viewState{
		world:   w,
		ids:     ids,
		sorted:  sorted,
		without: excludedIDs(sorted, filters),
		cols:    make([]storage.Column, len(ids)),
	}
}

// spawnArchetype returns, resolving and caching it on first call, the
// archetype ViewN.NewEntity spawns into: the one holding exactly this
// view's components. Resolved lazily so a view used only for
// Get/Has/Set/Remove/Each never pays an Index.Get call.
func (st *viewState) spawnArchetype() *storage.Archetype {
	if st.spawn == nil {
		st.spawn = st.world.index.Get(st.sorted)
	}
	return st.spawn
}

// has reports whether e currently has every component in st and none that
// its filters exclude, which is whether Each would yield it. Panics with
// "ecs: invalid or stale entity: <e>" if e is stale. Arity-agnostic: this is
// the entire body of ViewN.Has for every N.
func (st *viewState) has(e Entity) bool {
	rec := st.world.record(e)
	return st.matches(rec.Arch)
}

// matches reports whether arch has every component in st.sorted and none in
// st.without: the per-archetype test behind Each and Has.
func (st *viewState) matches(arch *storage.Archetype) bool {
	return archMatches(arch, st.sorted) && archHasNone(arch, st.without)
}

// setTarget is ViewN.Set's structural half, mirroring World.SetComp's
// "moving" pre-pass: it checks whether e already has every component in st
// before deciding anything needs to move. If so, it returns e's current
// Record and archetype unchanged (no move, no revision bump). Otherwise e
// moves exactly once — through the single cached WithAdded edge when st has
// one component, else a Union into w.scratch.target followed by
// Index.Get — and the revision bumps once. The generated ViewN.Set writes
// the N typed values afterward, through the returned rec.Row and arch.
func (st *viewState) setTarget(e Entity) (rec *entities.Record, arch *storage.Archetype) {
	w := st.world
	rec = w.record(e)
	arch = rec.Arch

	moving := false
	for _, id := range st.sorted {
		if !arch.Has(id) {
			moving = true
			break
		}
	}
	if !moving {
		return rec, arch
	}

	var dst *storage.Archetype
	if len(st.sorted) == 1 {
		dst = w.index.WithAdded(arch, st.sorted[0])
	} else {
		w.scratch.target = storage.Union(w.scratch.target, arch.Comps(), st.sorted)
		dst = w.index.Get(w.scratch.target)
	}
	w.moveTo(rec, dst)
	w.revision++
	return rec, dst
}

// remove removes whichever of st's components e currently has, in one
// archetype move; a no-op, leaving the revision unchanged, if e has none of
// them. st.sorted is already ascending and duplicate-free, so the filtered
// subsequence of components e has is too — unlike World.RemoveComp there's
// no separate sort/compact pass needed. Arity-agnostic: this is the entire
// body of ViewN.Remove for every N.
func (st *viewState) remove(e Entity) {
	w := st.world
	rec := w.record(e)
	arch := rec.Arch

	present := st.presentBuf[:0]
	for _, id := range st.sorted {
		if arch.Has(id) {
			present = append(present, id)
		}
	}
	st.presentBuf = present
	if len(present) == 0 {
		return
	}

	var dst *storage.Archetype
	if len(present) == 1 {
		dst = w.index.WithRemoved(arch, present[0])
	} else {
		w.scratch.target = storage.Diff(w.scratch.target, arch.Comps(), present)
		dst = w.index.Get(w.scratch.target)
	}
	w.moveTo(rec, dst)
	w.revision++
}

// scan extends matched with every archetype created since the last scan
// that has every component in st.sorted and none in st.without. Filters are
// decided here, once per archetype, so Each pays nothing per entity for
// them. Same shape as queryState.scan
// (query.go) — kept as its own method rather than sharing one, since the two
// states aren't the same concrete type and this is only a handful of lines.
func (st *viewState) scan() {
	archetypes := st.world.index.Archetypes()
	for _, arch := range archetypes[st.scanned:] {
		if st.matches(arch) {
			st.matched = append(st.matched, arch)
		}
	}
	st.scanned = len(archetypes)
}
