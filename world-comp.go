package ecs

import (
	"slices"

	"github.com/weakpixel/ecs/internal/storage"
)

// checkRegistered panics with "ecs: component not registered: ComponentID(
// <id>)" if id was never registered.
//
// Called only on the miss path of GetComp, GetCompOrNil, HasComp and
// RemoveComp, after the archetype has already reported it doesn't store id:
// an ID present in some archetype's column set is necessarily registered (an
// archetype's columns come from the registry), so a hit never pays for this
// check.
func (w *World) checkRegistered(id storage.CompID) {
	componentRegistry.Info(id) // panics with the right message for an unknown id
}

// GetComp returns a pointer to e's live component id, as *T boxed in any.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, with
// "ecs: component not registered: ComponentID(<id>)" if id is unknown, and
// with "ecs: entity does not have component: <type>" if e doesn't currently
// have it.
//
// Cost: 0 allocations. The pointer is valid until the next structural
// change to e's archetype (any SetComp that adds a component, RemoveComp,
// NewEntity or DeleteEntity); re-fetch it after one.
func (w *World) GetComp(e Entity, id ComponentID) any {
	rec := w.record(e)
	cid := storage.CompID(id)
	ci := rec.Arch.ColumnIndex(cid)
	if ci < 0 {
		w.checkRegistered(cid)
		panicMissingComp(cid)
	}
	return rec.Arch.Column(ci).PtrAny(int(rec.Row))
}

// GetCompOrNil is GetComp but returns a literal untyped nil instead of
// panicking when e doesn't currently have the component. It still panics on
// a stale e or an unregistered id.
//
// Cost: 0 allocations.
func (w *World) GetCompOrNil(e Entity, id ComponentID) any {
	rec := w.record(e)
	cid := storage.CompID(id)
	ci := rec.Arch.ColumnIndex(cid)
	if ci < 0 {
		w.checkRegistered(cid)
		return nil
	}
	return rec.Arch.Column(ci).PtrAny(int(rec.Row))
}

// HasComp reports whether e currently has component id.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, and with
// "ecs: component not registered: ComponentID(<id>)" if id is unknown.
func (w *World) HasComp(e Entity, id ComponentID) bool {
	rec := w.record(e)
	cid := storage.CompID(id)
	if rec.Arch.Has(cid) {
		return true
	}
	w.checkRegistered(cid)
	return false
}

// SetComp adds or replaces components on e, choosing the component by each
// value's dynamic type. A component e already has is replaced; others are
// added. An empty call is a no-op.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale (checked
// first, even for an empty call), "ecs: component must be a non-pointer
// struct: <type>" for a pointer or non-struct value, "ecs: component not
// registered: <type>" for an unregistered struct type, and "ecs: duplicate
// component: <type>" if two values share a type. Everything is validated
// before any mutation; values is never modified.
//
// Replacing every given value in place (e already has all of them) leaves
// the revision unchanged. Otherwise e moves to a new archetype exactly
// once, however many components are added, and the revision is bumped
// once. A move invalidates pointers GetComp previously returned for e.
func (w *World) SetComp(e Entity, values ...any) {
	rec := w.record(e)
	if len(values) == 0 {
		return
	}
	ids := w.validateValues(values)
	sorted := w.scratch.sorted

	arch := rec.Arch
	moving := false
	for _, id := range sorted {
		if !arch.Has(id) {
			moving = true
			break
		}
	}

	if moving {
		var dst *storage.Archetype
		if len(sorted) == 1 {
			// A single value can only mean one new component (otherwise
			// moving would be false), so the cached single-edge path
			// applies directly.
			dst = w.index.WithAdded(arch, sorted[0])
		} else {
			w.scratch.target = storage.Union(w.scratch.target, arch.Comps(), sorted)
			dst = w.index.Get(w.scratch.target)
		}
		w.moveTo(rec, dst)
		arch = dst
		w.revision++
	}

	for i, v := range values {
		arch.Column(arch.ColumnIndex(ids[i])).SetAny(int(rec.Row), v)
	}
}

// RemoveComp removes components from e; ids it doesn't have are ignored, and
// repeating an id is tolerated (unlike SetComp/NewEntity, this is not a
// duplicate-component panic). Removing nothing, because ids is empty or
// none of it is present, is a no-op and leaves the revision unchanged.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, and with
// "ecs: component not registered: ComponentID(<id>)" for an unknown id.
//
// Removing anything moves e to a new archetype exactly once; pointers
// GetComp previously returned for e are invalidated.
func (w *World) RemoveComp(e Entity, ids ...ComponentID) {
	rec := w.record(e)
	arch := rec.Arch

	present := w.scratch.sorted[:0]
	for _, id := range ids {
		cid := storage.CompID(id)
		if arch.Has(cid) {
			present = append(present, cid)
			continue
		}
		w.checkRegistered(cid)
	}
	w.scratch.sorted = present
	if len(present) == 0 {
		return
	}
	slices.Sort(present)
	present = slices.Compact(present)
	w.scratch.sorted = present

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
