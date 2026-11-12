package ecs

import (
	"reflect"
	"slices"

	"github.com/weakpixel/ecs/internal/storage"
)

// compIDOf returns the ComponentID of v's dynamic type.
//
// Panics with "ecs: component must be a non-pointer struct: <type>" (detail
// "<nil>" for a nil interface) if v isn't a non-pointer struct, and with
// "ecs: component not registered: <type>" if that struct type was never
// passed to RegisterComp (or an earlier generic call).
func compIDOf(v any) storage.CompID {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Struct {
		panicBadValue(t)
	}
	id, ok := componentRegistry.Lookup(t)
	if !ok {
		panicUnregisteredValue(t)
	}
	return id
}

// validateValues resolves the ComponentID of every value, in values' order,
// without modifying values. It also fills w.scratch.sorted with a sorted
// copy and panics with "ecs: duplicate component: <type>" if any two values
// share a component type.
//
// Used by NewEntity and SetComp, whose doc contracts require every value to
// be validated before any entity is created or moved.
func (w *World) validateValues(values []any) []storage.CompID {
	ids := w.scratch.ids[:0]
	for _, v := range values {
		ids = append(ids, compIDOf(v))
	}
	w.scratch.ids = ids

	sorted := append(w.scratch.sorted[:0], ids...)
	slices.Sort(sorted)
	w.scratch.sorted = sorted
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1] == sorted[i] {
			panicDuplicateComp(sorted[i])
		}
	}
	return ids
}
