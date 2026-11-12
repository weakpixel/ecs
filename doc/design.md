# Design

How package `ecs` stores entities and components and why. The behavior it has to meet is the guide: [low-level API](api-low-level.md), [generic API](api-generic.md) and the cost model in the [performance guide](performance-guide.md#mental-model).

## Parts

`World` composes four parts and delegates to them:

| Part | Where | Owns |
| --- | --- | --- |
| Component registry | `internal/registry`, one process-global instance | `reflect.Type` ↔ `ComponentID`, a column factory per type |
| Entity table | `internal/entities` | Slots: generation, archetype, row; the free list |
| Archetype storage | `internal/storage` | Typed columns, archetypes, the append-only archetype index with add/remove edges |
| Resources | `World.resources` | `map[reflect.Type]any` of `*T` singletons |

Plus the world's `revision` counter and a set of reused scratch buffers. Views (`view.go`) and queries (`query.go`) hold a pointer to the world and their own cached state.

## Component registry

- **Process-global, not per World.** Callers, generated code among them, register components in package-level variables (`var CID_Health = ecs.RegisterComp[Health]()`), before any world exists, and IDs must agree between worlds so one `View` type works on all of them.
- **Copy-on-write.** Reads (the hot path: every `GetComp[T]` resolves `T`'s ID) are one atomic load of an immutable snapshot plus a map lookup, with no lock and no allocation. Registering a new type takes a mutex and copies the snapshot. This makes concurrent `RegisterComp` from package `init`s and parallel tests safe.
- **The column factory** is captured inside `RegisterComp[T]` (`func() Column { return &TypedColumn[T]{} }`). It is the only place the concrete `T` is known for values that later arrive as `any` or as a `ComponentID`.
- **IDs** are dense `uint16`s in registration order, capped at 4096, never reused, and not stable across processes (don't persist them). Re-registering returns the existing ID before any check, so a full registry still resolves known types.

## Entities

- A `[]slot` indexed by `EntityID`; a slot holds the generation and a `Record` (archetype pointer and row). A nil archetype means the slot is free.
- **Generation starts at 1** and increments on delete, so the zero `Entity{}` is never alive and works as "no entity". On wrap from `MaxUint32` it skips back to 1, never 0.
- **LIFO free list.** The most recently freed slot is reused first, which keeps the table compact under churn.
- `IsAlive` checks bounds, the archetype and the generation and never panics; every other use of a bad handle panics with `ecs: invalid or stale entity: {ID:3 Generation:2}`.

## Archetypes and columns

- **Columns are `[]T`** (`TypedColumn[T]`) behind a small `Column` interface. Components may hold pointers, strings and slices, and the GC must see them, so there is no `[]byte` + `unsafe` storage. Typed fast paths assert the concrete column.
- **Swap-remove.** Deleting or moving a row moves the last row into the hole and zeroes the vacated element (so the GC doesn't keep references alive). The caller fixes the record of the entity that was swapped.
- **Archetypes are never deleted.** The index is an append-only `[]*Archetype`; empty archetypes stay. This keeps archetype pointers stable for view caches and spawners, and lets queries scan incrementally.
- **Lookup by component set** goes through a map keyed by the sorted IDs encoded as a string, built in a stack buffer (`m[string(buf)]` does not allocate), so a hit is free. Single-component add and remove transitions are cached per archetype as edges (`WithAdded`, `WithRemoved`).

## Structural changes

- **Validate first, then mutate.** `NewEntity` and `SetComp` check the entity, every value (non-pointer struct, registered, no duplicates) before touching anything, and never modify the caller's slice.
- **At most one move.** `SetComp` with several new components computes the target set once and moves the row once; `View.Set` and `View.Remove` do the same for the view's set.
- **In place when possible.** `SetComp` of components the entity already has writes in place.
- **The revision** is a `uint64` bumped on every structural change and on nothing else. No-op calls (`SetComp` in place, `RemoveComp` of absent components) must not bump it, so a caller may always call `SetComp(e, all...)` followed by `RemoveComp(e, absent...)` to bring an entity to a known shape, and doing that in place is allowed during `Each`.
- `DeleteEntity` must be correct for any row order, so a loader can roll back by deleting the entities it created, in any order.

## Iteration

- **Incremental match cache.** A `Query` or view remembers how many archetypes it has scanned and on each `Each` checks only archetypes added since. Archetypes with zero rows are skipped. A view's filters (`Without`) are decided in the same check, so an excluded archetype never enters the cache and the row loop has no per-entity test.
- **Columns resolved per archetype**, then a tight loop over rows.
- **Reused buffers.** The `[]any` of an untyped query and the column buffer of a view are sized once and reused. A nested iteration of the *same* query or view allocates its own buffers, because the outer loop still uses the shared ones; that is the only allocation `Each` can make.
- **Revision check after `yield`.** The revision is recorded at the start; after `yield` returns true, a changed revision panics with `ecs: query invalidated by world mutation`. When `yield` returns false the loop returns without checking, so "find, mutate, break" is legal.
- **Views are small values** holding a pointer to shared state (`viewState`), so copies share the cache. The match cache is built lazily on the first `Each`, and the spawn archetype on the first `NewEntity`, so a view used only for `Get`/`Set` pays for neither.

## Typed API generation

`NewEntity1..16`, `NewView1..16`/`View1..16` and `Components2..16` are generated into `zz_gen_generic-api.go` by `internal/gen` (`//go:generate` in `generic.go`). The generated methods are thin: they resolve IDs and type-assert columns, and call hand-written helpers in `generic.go` and `view.go` for everything that doesn't depend on the concrete types.

## Resources

A `map[reflect.Type]any` holding `*T`, created on the first `SetResource` so `NewWorld` stays allocation-free. Resources need no registration and are not components.
