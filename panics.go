package ecs

import (
	"fmt"
	"reflect"

	"github.com/weakpixel/ecs/internal/storage"
)

// Stable panic texts from doc/api-low-level.md#panics. Each panic value is
// this text, optionally followed by ": <detail>". Tests match the stable
// text with strings.Contains, so these must not change without updating the
// doc.
const (
	panicNonPointerStructMsg = "ecs: component must be a non-pointer struct"
	panicNotRegisteredMsg    = "ecs: component not registered"
	panicDuplicateMsg        = "ecs: duplicate component"
	panicStaleEntityMsg      = "ecs: invalid or stale entity"
	panicMissingComponentMsg = "ecs: entity does not have component"
	panicQueryInvalidatedMsg = "ecs: query invalidated by world mutation"
	panicNotBoundMsg         = "ecs: not bound to a world"
	panicResourceNotSetMsg   = "ecs: resource not set"
	panicRequiredExcludedMsg = "ecs: component both required and excluded"
	panicZeroFilterMsg       = "ecs: zero Filter"
)

// panicStale panics with the stale-entity message, detailed with e's value
// in %+v form (e.g. "ecs: invalid or stale entity: {ID:3 Generation:2}").
func panicStale(e Entity) {
	panic(fmt.Sprintf("%s: %+v", panicStaleEntityMsg, e))
}

// panicMissingComp panics with the missing-component message, detailed with
// id's registered type.
func panicMissingComp(id storage.CompID) {
	panic(fmt.Sprintf("%s: %v", panicMissingComponentMsg, componentRegistry.Info(id).Type))
}

// panicDuplicateComp panics with the duplicate-component message, detailed
// with id's registered type.
func panicDuplicateComp(id storage.CompID) {
	panic(fmt.Sprintf("%s: %v", panicDuplicateMsg, componentRegistry.Info(id).Type))
}

// panicBadValue panics with the non-pointer-struct message for a value
// passed to NewEntity/SetComp whose reflect.Type is t (nil for a nil
// interface, per compIDOf).
func panicBadValue(t reflect.Type) {
	if t == nil {
		panic(panicNonPointerStructMsg + ": <nil>")
	}
	panic(fmt.Sprintf("%s: %v", panicNonPointerStructMsg, t))
}

// panicUnregisteredValue panics with the not-registered message, detailed
// with the value's type (compIDOf: a struct type that was never passed to
// RegisterComp or an earlier generic call).
func panicUnregisteredValue(t reflect.Type) {
	panic(fmt.Sprintf("%s: %v", panicNotRegisteredMsg, t))
}

// panicQueryInvalidated panics because a structural change happened while a
// Query was mid-iteration (Query.Each, after yield returned true).
func panicQueryInvalidated() {
	panic(panicQueryInvalidatedMsg)
}

// panicNotBound panics because a zero Query or zero View was used.
func panicNotBound() {
	panic(panicNotBoundMsg)
}

// panicResourceNotSet panics with the resource-not-set message, detailed
// with the resource's type.
func panicResourceNotSet(t reflect.Type) {
	panic(fmt.Sprintf("%s: %v", panicResourceNotSetMsg, t))
}

// panicRequiredAndExcluded panics because a view constructor got a Without
// filter for one of its own component types, detailed with that type.
func panicRequiredAndExcluded(id storage.CompID) {
	panic(fmt.Sprintf("%s: %v", panicRequiredExcludedMsg, componentRegistry.Info(id).Type))
}

// panicZeroFilter panics because a view constructor got a Filter that was
// not made by Without.
func panicZeroFilter() {
	panic(panicZeroFilterMsg)
}
