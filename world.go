package ecs

import (
	"reflect"

	"github.com/weakpixel/ecs/internal/entities"
	"github.com/weakpixel/ecs/internal/storage"
)

// World owns all entities, archetypes and resources for one simulation.
// Worlds are independent from each other; they share only the process-global
// component registry (see RegisterComp). A World is not safe for concurrent
// use.
type World struct {
	index    *storage.Index
	entities entities.Table
	revision uint64

	// scratch holds buffers reused across calls to avoid allocating on every
	// NewEntity/SetComp/RemoveComp. A World is single-goroutine and these
	// methods don't recurse into each other, so a single set of buffers is
	// safe to reuse; each call overwrites the slice contents before reading
	// them.
	scratch scratchBuffers

	// resources holds Set/GetResource values, keyed by type. Left nil until
	// the first SetResource call so NewWorld stays allocation-free; reads
	// and deletes against a nil map are well-defined in Go.
	resources map[reflect.Type]any
}

// scratchBuffers are the reusable []storage.CompID slices behind value and
// ID validation. Keeping them on World instead of allocating fresh slices
// per call is what makes NewEntity/SetComp/RemoveComp 0-alloc in steady
// state once their capacity has grown to fit.
type scratchBuffers struct {
	ids    []storage.CompID // component IDs in the caller's order
	sorted []storage.CompID // sorted, duplicate-checked copy of ids
	target []storage.CompID // Union/Diff destination for SetComp/RemoveComp
}

// NewWorld returns an empty World.
func NewWorld() *World {
	return &World{index: storage.NewIndex(newColumn)}
}

// IsAlive reports whether e is a current live handle. It never panics; the
// zero Entity{} and any stale or out-of-range handle report false.
func (w *World) IsAlive(e Entity) bool {
	_, ok := w.entities.Get(e.toStorage())
	return ok
}

// record returns the live Record for e. Every method that needs e's current
// location calls this first, so the stale-entity panic always happens before
// any mutation.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is not currently
// alive.
func (w *World) record(e Entity) *entities.Record {
	rec, ok := w.entities.Get(e.toStorage())
	if !ok {
		panicStale(e)
	}
	return rec
}

// spawnIn allocates a new entity directly into arch and returns its handle
// and row. The caller still has to write the real component values: the row
// starts zeroed, matching storage.Archetype.AppendRow.
func (w *World) spawnIn(arch *storage.Archetype) (Entity, int) {
	se, rec := w.entities.Alloc()
	row := arch.AppendRow(se)
	rec.Arch = arch
	rec.Row = int32(row)
	return entityFromStorage(se), row
}

// moveTo moves the entity currently described by rec into dst and updates
// rec in place to its new location. It also fixes up the record of
// whatever entity was swapped into the row vacated in rec.Arch. It returns
// the entity's new row in dst.
//
// Precondition, not checked: dst != rec.Arch. Pointers previously returned
// into rec.Arch or dst are invalidated.
func (w *World) moveTo(rec *entities.Record, dst *storage.Archetype) int {
	src, row := rec.Arch, int(rec.Row)
	dstRow, moved, ok := storage.MoveRow(src, row, dst)
	if ok {
		movedRec, _ := w.entities.Get(moved)
		movedRec.Row = int32(row)
	}
	rec.Arch, rec.Row = dst, int32(dstRow)
	return dstRow
}

// NewEntity creates an entity holding values, stored inline in its
// archetype. Each value's dynamic type selects the component; order doesn't
// matter.
//
// Panics with "ecs: component must be a non-pointer struct: <type>" for a
// pointer or non-struct value, "ecs: component not registered: <type>" for a
// struct type nothing has registered yet, and "ecs: duplicate component:
// <type>" if two values share a type. Validation happens before any entity
// is created.
func (w *World) NewEntity(values ...any) Entity {
	ids := w.validateValues(values)

	arch := w.index.Empty()
	if sorted := w.scratch.sorted; len(sorted) > 0 {
		arch = w.index.Get(sorted)
	}

	e, row := w.spawnIn(arch)
	for i, v := range values {
		arch.Column(arch.ColumnIndex(ids[i])).SetAny(row, v)
	}
	w.revision++
	return e
}

// DeleteEntity removes e and invalidates every handle to it, including e
// itself.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is already stale.
func (w *World) DeleteEntity(e Entity) {
	rec := w.record(e)
	moved, ok := rec.Arch.RemoveRow(int(rec.Row))
	if ok {
		movedRec, _ := w.entities.Get(moved)
		movedRec.Row = rec.Row
	}
	w.entities.Free(e.toStorage())
	w.revision++
}
