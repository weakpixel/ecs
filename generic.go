package ecs

import (
	"slices"

	"github.com/weakpixel/ecs/internal/storage"
)

//go:generate go run ./internal/gen

// columnOf returns a's column for id as a *storage.TypedColumn[T], or nil
// when a doesn't store id. The type assertion on the Column interface is a
// data-word extraction, not a conversion, so it never allocates.
func columnOf[T any](a *storage.Archetype, id storage.CompID) *storage.TypedColumn[T] {
	ci := a.ColumnIndex(id)
	if ci < 0 {
		return nil
	}
	return a.Column(ci).(*storage.TypedColumn[T])
}

// newEntityArchetype resolves the archetype for exactly ids, the typed-ID
// analogue of validateValues: it sorts a scratch copy and panics with
// "ecs: duplicate component: <type>" on a repeat. Generated NewEntityN
// already has concrete ComponentIDs, so this skips compIDOf's reflection.
// ids is read-only to the World; only w.scratch.sorted is mutated.
func (w *World) newEntityArchetype(ids []storage.CompID) *storage.Archetype {
	sorted := append(w.scratch.sorted[:0], ids...)
	slices.Sort(sorted)
	w.scratch.sorted = sorted
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1] == sorted[i] {
			panicDuplicateComp(sorted[i])
		}
	}
	return w.index.Get(sorted)
}

// GetComp returns a pointer to e's live T. It registers T on first use, like
// every function in this file.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, and with
// "ecs: entity does not have component: <type>" if e doesn't currently have
// T.
//
// Cost: 0 allocations. The pointer is valid until the next structural
// change to e's archetype (an add, a remove, or a move); re-fetch it after
// one.
func GetComp[T any](w *World, e Entity) *T {
	id := storage.CompID(RegisterComp[T]())
	rec := w.record(e)
	col := columnOf[T](rec.Arch, id)
	if col == nil {
		panicMissingComp(id)
	}
	return col.Ptr(int(rec.Row))
}

// GetCompOrNil is GetComp but returns nil instead of panicking when e
// doesn't currently have T. It still panics on a stale e.
//
// Cost: 0 allocations.
func GetCompOrNil[T any](w *World, e Entity) *T {
	id := storage.CompID(RegisterComp[T]())
	rec := w.record(e)
	col := columnOf[T](rec.Arch, id)
	if col == nil {
		return nil
	}
	return col.Ptr(int(rec.Row))
}

// HasComp reports whether e currently has T.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
func HasComp[T any](w *World, e Entity) bool {
	id := storage.CompID(RegisterComp[T]())
	rec := w.record(e)
	return rec.Arch.Has(id)
}

// SetComp adds T to e or replaces it. Replacing a T e already has is in
// place and leaves the revision unchanged; adding moves e to a new
// archetype exactly once and bumps the revision once.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
//
// Cost: 0 allocations, in place and when moving.
func SetComp[T any](w *World, e Entity, value T) {
	id := storage.CompID(RegisterComp[T]())
	rec := w.record(e)
	arch := rec.Arch
	if !arch.Has(id) {
		dst := w.index.WithAdded(arch, id)
		w.moveTo(rec, dst)
		arch = dst
		w.revision++
	}
	columnOf[T](arch, id).Set(int(rec.Row), value)
}

// RemoveComp removes e's T; a no-op, leaving the revision unchanged, if e
// doesn't have it. Removing moves e to a new archetype exactly once.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
func RemoveComp[T any](w *World, e Entity) {
	id := storage.CompID(RegisterComp[T]())
	rec := w.record(e)
	arch := rec.Arch
	if !arch.Has(id) {
		return
	}
	dst := w.index.WithRemoved(arch, id)
	w.moveTo(rec, dst)
	w.revision++
}
