package registry

import (
	"fmt"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/weakpixel/ecs/internal/storage"
)

// MaxComponents is the capacity of a registry: IDs 0..MaxComponents-1. It stays
// far below the int16 limit of storage.Archetype's column index.
const MaxComponents = 4096

// Info describes one registered component.
type Info struct {
	ID   storage.CompID
	Type reflect.Type
	// NewColumn returns an empty column of the component type. It allocates,
	// which is fine because it only runs when an archetype is created.
	NewColumn func() storage.Column
}

// snapshot is an immutable view of the registry. Once published through
// Registry.snap it is never modified.
type snapshot struct {
	byType map[reflect.Type]storage.CompID
	infos  []Info // indexed by CompID
}

// Registry assigns dense IDs, starting at 0 in registration order and never
// reused, to component types.
//
// Reads use a copy-on-write snapshot behind an atomic pointer, so the steady
// state (an already registered type) costs one atomic load plus one map lookup
// and allocates nothing. Registering a new type takes a mutex and copies the
// map and slice: O(n) with n <= MaxComponents, paid once per type, mostly at
// package init.
type Registry struct {
	max  int
	mu   sync.Mutex // serializes writers
	snap atomic.Pointer[snapshot]
}

// New returns an empty registry holding at most max component types. Tests use
// a small max; package ecs uses MaxComponents.
func New(max int) *Registry {
	r := &Registry{max: max}
	r.snap.Store(&snapshot{byType: map[reflect.Type]storage.CompID{}})
	return r
}

// Register returns the ID of T, registering it on first use. It is idempotent.
//
// It panics with "ecs: component must be a non-pointer struct: <type>" when T is
// not a struct (pointers included), and with "ecs: component registry capacity
// exceeded: max <n>" when the registry is full. A registered type is returned
// before either check, so a full registry still resolves its existing types.
func Register[T any](r *Registry) storage.CompID {
	t := reflect.TypeFor[T]()
	if id, ok := r.Lookup(t); ok {
		return id
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("ecs: component must be a non-pointer struct: %v", t))
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.snap.Load()
	if id, ok := old.byType[t]; ok { // lost a race with another writer
		return id
	}
	if len(old.infos) >= r.max {
		panic(fmt.Sprintf("ecs: component registry capacity exceeded: max %d", r.max))
	}

	id := storage.CompID(len(old.infos))
	next := &snapshot{
		byType: make(map[reflect.Type]storage.CompID, len(old.byType)+1),
		infos:  make([]Info, len(old.infos), len(old.infos)+1),
	}
	maps.Copy(next.byType, old.byType)
	copy(next.infos, old.infos)
	next.byType[t] = id
	next.infos = append(next.infos, Info{
		ID:   id,
		Type: t,
		// The factory is built here because T is only known statically here.
		NewColumn: func() storage.Column { return storage.NewTypedColumn[T]() },
	})
	r.snap.Store(next)
	return id
}

// Lookup returns the ID of t and whether t is registered. It never registers
// and allocates nothing.
func (r *Registry) Lookup(t reflect.Type) (storage.CompID, bool) {
	id, ok := r.snap.Load().byType[t]
	return id, ok
}

// Info returns the entry for id. It panics with "ecs: component not registered:
// ComponentID(<id>)" for an unknown ID.
func (r *Registry) Info(id storage.CompID) Info {
	infos := r.snap.Load().infos
	if int(id) >= len(infos) {
		panic(fmt.Sprintf("ecs: component not registered: ComponentID(%d)", id))
	}
	return infos[id]
}

// Len returns the number of registered components.
func (r *Registry) Len() int { return len(r.snap.Load().infos) }
