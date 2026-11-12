# ecs exported API checklist

Derived from `doc/api-low-level.md` and `doc/api-generic.md`. Compare it with the real API using `go doc -all .`.

Entries marked *(decided)* are choices the docs imply but don't state. If you change one, update the docs too.

All layers use the same verbs (see `doc/api-low-level.md#conventions`): `NewEntity`, `Get` (panics if missing), `GetOrNil` (single pointer results only), `Has`, `Set` (add or replace), `Remove` (ignores absent), `Each` (`iter.Seq2`).

## Hand-written (root *.go)

```go
// components
type ComponentID uint16                           // (decided) integer type; max 4096 registered
func RegisterComp[T any]() ComponentID             // non-pointer struct only; idempotent (also the ID lookup); init-safe; concurrency-safe

// entities
type EntityID uint32
type Entity struct {
    ID         EntityID
    Generation uint32                              // starts at 1: Entity{} is never alive
}

// world
type World struct{ /* unexported */ }
func NewWorld() *World
func (w *World) NewEntity(values ...any) Entity
func (w *World) DeleteEntity(e Entity)
func (w *World) IsAlive(e Entity) bool                       // never panics; false for Entity{}
func (w *World) GetComp(e Entity, id ComponentID) any        // *T as any
func (w *World) GetCompOrNil(e Entity, id ComponentID) any   // untyped nil when absent
func (w *World) HasComp(e Entity, id ComponentID) bool
func (w *World) SetComp(e Entity, values ...any)             // component chosen by dynamic type
func (w *World) RemoveComp(e Entity, ids ...ComponentID)

// filters (view-filter.go)
type Filter struct{ /* unexported; zero value panics */ }
func Without[T any]() Filter                               // excludes archetypes that have T

// queries
func (w *World) Query(ids ...ComponentID) Query             // empty = all archetypes; duplicate ids panic
type Query struct{ /* pointer to shared state; zero value panics */ }
func (q Query) Each() iter.Seq2[Entity, []any]

// generic single-type helpers (generic.go); register T on first use
func GetComp[T any](w *World, e Entity) *T
func GetCompOrNil[T any](w *World, e Entity) *T
func HasComp[T any](w *World, e Entity) bool
func SetComp[T any](w *World, e Entity, value T)           // 0 allocs, in place and when moving
func RemoveComp[T any](w *World, e Entity)

// resources (no registration)
func SetResource[T any](w *World, r *T)
func GetResource[T any](w *World) *T                        // panics "ecs: resource not set: <T>"
func GetResourceOrNil[T any](w *World) *T
func HasResource[T any](w *World) bool
func RemoveResource[T any](w *World)
```

## Generated (zz_gen_generic-api.go, N = 1..16)

```go
type ComponentsN[T1, ..., TN any] struct { Comp1 *T1; ...; CompN *TN }   // N >= 2 only (decided)
func (c ComponentsN[...]) Unpack() (*T1, ..., *TN)

func NewEntityN[T1, ..., TN any](w *World, v1 T1, ..., vN TN) Entity

func NewViewN[T1, ..., TN any](w *World, filters ...Filter) ViewN[T1, ..., TN]  // registers types; duplicates panic
type ViewN[T1, ..., TN any] struct{ /* pointer to shared state; zero value panics */ }
func (v ViewN[...]) NewEntity(v1 T1, ..., vN TN) Entity     // pre-resolved archetype
func (v ViewN[...]) Get(e Entity) ComponentsN[...]          // View1: *T1; panics if any missing
func (v ViewN[...]) Has(e Entity) bool                      // all N present
func (v ViewN[...]) Set(e Entity, v1 T1, ..., vN TN)        // add or replace; in place or one move
func (v ViewN[...]) Remove(e Entity)                        // removes present ones; one move
func (v ViewN[...]) Each() iter.Seq2[Entity, ComponentsN[...]] // View1: iter.Seq2[Entity, *T1]
```

## Panic messages (from the doc's table)

Each message is the stable text below, optionally followed by `: <detail>` (type name, ID or handle). Tests match the stable text with `strings.Contains`.

| Text | Trigger |
| --- | --- |
| `ecs: component must be a non-pointer struct` | `RegisterComp` or first generic use with a non-struct; pointer value to `NewEntity`/`SetComp` |
| `ecs: component registry capacity exceeded` | more than 4096 components |
| `ecs: component not registered` | low-level only: value of unregistered type, or unknown `ComponentID` |
| `ecs: duplicate component` | same component twice in `NewEntity`/`SetComp`/`NewViewN`/`World.Query` |
| `ecs: component both required and excluded` | `NewViewN` with `Without[T]` for one of its own types |
| `ecs: zero Filter` | `NewViewN` with an `ecs.Filter{}` not made by `Without` |
| `ecs: invalid or stale entity` | deleted, never created, out-of-range or zero `Entity` |
| `ecs: entity does not have component` | `GetComp`, `View.Get` on a missing component |
| `ecs: query invalidated by world mutation` | structural change during `Each` |
| `ecs: not bound to a world` | zero `Query` or zero `View` |
| `ecs: resource not set` | `GetResource` |
