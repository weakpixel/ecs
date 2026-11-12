package ecs

import (
	"github.com/weakpixel/ecs/internal/registry"
	"github.com/weakpixel/ecs/internal/storage"
)

// ComponentID identifies a registered component type. IDs are dense, start at
// 0 and are assigned in registration order, so they are not stable across
// processes: don't persist them.
type ComponentID uint16

// componentRegistry is the process-global registry shared by all worlds. It is
// safe for concurrent use, so package inits and parallel tests may register.
var componentRegistry = registry.New(registry.MaxComponents)

// RegisterComp registers T if needed and returns its ID. It is idempotent, so
// it is also how to look up the ID of a type. It is safe for concurrent use and
// for package-level initializers (var id = ecs.RegisterComp[T]()).
//
// Cost: a registered type costs one atomic load and a map lookup, with no
// allocations. A new type copies the registry once (O(registered types)).
//
// It panics with "ecs: component must be a non-pointer struct: <type>" when T
// is not a struct (pointers included), and with "ecs: component registry
// capacity exceeded: max 4096" when the registry is full.
func RegisterComp[T any]() ComponentID {
	return ComponentID(registry.Register[T](componentRegistry))
}

// newColumn is the column factory handed to storage.NewIndex. It panics with
// "ecs: component not registered: ComponentID(<id>)" for an unknown ID.
func newColumn(id storage.CompID) storage.Column {
	return componentRegistry.Info(id).NewColumn()
}
