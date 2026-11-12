package ecs

import (
	"slices"

	"github.com/weakpixel/ecs/internal/storage"
)

// Filter narrows which entities a view selects, beyond having all of the
// view's components. Pass filters to a view constructor:
//
//	moving := ecs.NewView2[Position, Velocity](w, ecs.Without[Body]())
//
// A filter is checked once per archetype, when the view's match cache takes
// the archetype in, so it costs nothing per entity during Each. Make one
// with [Without]; the zero Filter makes the view constructor panic with
// "ecs: zero Filter".
type Filter struct {
	without storage.CompID
	ok      bool // false for the zero Filter, which no constructor accepts
}

// Without returns a filter that drops every entity that has a T: ViewN.Each
// skips it and ViewN.Has reports false for it. Get, Set, Remove and
// NewEntity ignore filters. Without registers T like the other generic
// functions do.
//
// A view constructor panics with "ecs: component both required and
// excluded: <type>" if T is also one of its type parameters, and with "ecs:
// duplicate component: <type>" if two filters exclude the same T.
func Without[T any]() Filter {
	return Filter{without: storage.CompID(RegisterComp[T]()), ok: true}
}

// excludedIDs validates filters against the view's sorted component set and
// returns the excluded IDs, sorted and duplicate-free, or nil for none.
//
// Panics with "ecs: zero Filter", "ecs: component both required and
// excluded: <type>" or "ecs: duplicate component: <type>", for the first
// offending filter in argument order.
func excludedIDs(sorted []storage.CompID, filters []Filter) []storage.CompID {
	if len(filters) == 0 {
		return nil
	}
	without := make([]storage.CompID, 0, len(filters))
	for _, f := range filters {
		if !f.ok {
			panicZeroFilter()
		}
		if _, found := slices.BinarySearch(sorted, f.without); found {
			panicRequiredAndExcluded(f.without)
		}
		if slices.Contains(without, f.without) {
			panicDuplicateComp(f.without)
		}
		without = append(without, f.without)
	}
	slices.Sort(without)
	return without
}

// archHasNone reports whether arch has none of ids.
func archHasNone(arch *storage.Archetype, ids []storage.CompID) bool {
	for _, id := range ids {
		if arch.Has(id) {
			return false
		}
	}
	return true
}
