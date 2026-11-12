# Low-level API

The low-level API identifies components by `ComponentID` and passes values as `any`. The [generic API](api-generic.md) is built on it. Use it when component types are only known at run time, for example in tools, serialization or code generators.

```go
import "github.com/weakpixel/ecs"
```

Contents: [Conventions](#conventions) · [Components](#components) · [World](#world) · [Entities](#entities) · [Component access](#component-access) · [Queries](#queries) · [Resources](#resources) · [Panics](#panics)

## Conventions

Both layers (low-level and [generic](api-generic.md)) use the same verbs with the same meaning:

| Verb | Meaning |
| --- | --- |
| `NewEntity` | Creates an entity. `DeleteEntity` exists only on `World`. |
| `Get` | Returns live data. **Panics** if it is missing. |
| `GetOrNil` | Same, but returns `nil` when missing. Only exists where the result is a single pointer. |
| `Has` | Reports whether the data is present. For multi-component results (views), check `Has` before `Get`. |
| `Set` | Adds or replaces. In place when everything is already present, otherwise one archetype move. |
| `Remove` | Removes. Anything that is not present is ignored. |
| `Each` | Yields every match, as an `iter.Seq2` to use with `range`. |

Function and method names abbreviate *component* as `Comp` (`GetComp`, `RegisterComp`). Type names spell it out (`ComponentID`, `ComponentsN`).

## Components

Components are plain non-pointer structs. The component registry is process-global (not per `World`) and safe for concurrent registration.

```go
type Position struct{ X, Y int }

posID := ecs.RegisterComp[Position]() // ComponentID, stable for the process
```

| Function | Description |
| --- | --- |
| `RegisterComp[T]() ComponentID` | Registers `T` if needed and returns its ID. Idempotent, so it is also the way to look up an ID. Panics if `T` is not a struct or the registry is full (4096 components). |

The [generic API](api-generic.md) registers types on first use. The low-level API cannot know a type statically: before you pass a value to `NewEntity`/`SetComp` or use an ID, the type must have been registered, either by `RegisterComp` or by any earlier generic call with that type. IDs are dense, start at 0, are assigned in registration order and never reused. They are not stable across processes; don't persist them. Registering an already registered type returns its ID before any check, so a full registry still resolves existing types.

## World

```go
world := ecs.NewWorld()
```

A `World` owns all entities, archetypes and resources. Worlds are independent; they share only the component registry. A `World` is not safe for concurrent use.

## Entities

```go
type EntityID uint32

type Entity struct {
    ID         EntityID // slot index, reused after deletion
    Generation uint32   // starts at 1, incremented each time the slot is reused
}
```

`Entity` is a small comparable value; copy it freely. A handle becomes stale once its entity is deleted, even if the slot is reused for a new entity.

The zero `Entity{}` is never a live handle, so use it as "no entity" in struct fields: `IsAlive(ecs.Entity{})` is `false`, and other operations panic with `ecs: invalid or stale entity`.

`Generation` starts at 1 and is incremented every time a slot is freed, which is what makes the zero `Entity{}` (`Generation: 0`) always stale. A freed slot is reused by the next `NewEntity`, most recently freed first (LIFO), which keeps the entity table compact under churn. On the rare wrap from `math.MaxUint32` the generation skips back to 1 instead of 0, so a wrapped slot never collides with `Entity{}`.

| Method | Description |
| --- | --- |
| `NewEntity(values ...any) Entity` | Creates an entity holding the given component values (stored inline in the archetype). Each value's dynamic type selects the component. Panics on duplicate or unregistered types. |
| `DeleteEntity(e Entity)` | Removes the entity and invalidates all handles to it. Panics if `e` is stale. |
| `IsAlive(e Entity) bool` | Reports whether `e` is a current live handle. Never panics. |

```go
e := world.NewEntity(Position{X: 1, Y: 2}, Velocity{X: 3, Y: 4})
world.DeleteEntity(e)
fmt.Println(world.IsAlive(e)) // false
```

## Component access

| Method | Description |
| --- | --- |
| `GetComp(e, id) any` | Pointer (`*T`, as `any`) to the live component. Panics if `e` is stale or lacks the component. |
| `GetCompOrNil(e, id) any` | Same, but returns an untyped `nil` when the component is absent. Still panics on a stale entity. |
| `HasComp(e, id) bool` | Reports whether `e` has the component. Panics if `e` is stale. |
| `SetComp(e, values ...any)` | Adds or replaces components, identified by each value's dynamic type. Never removes any. An empty call is a no-op. |
| `RemoveComp(e, ids ...ComponentID)` | Removes components; IDs the entity does not have are ignored. |

```go
posID := ecs.RegisterComp[Position]()
velID := ecs.RegisterComp[Velocity]()

pos := world.GetComp(e, posID).(*Position)
pos.X = 99 // writes to the world's storage

if v := world.GetCompOrNil(e, velID); v != nil {
    v.(*Velocity).X = 1
}

world.SetComp(e, Velocity{X: 10, Y: 20}, Position{X: 4, Y: 5})
world.RemoveComp(e, velID)
```

Notes:

- Returned pointers point into archetype storage. They are invalidated by any structural change to the world (the entity moves, rows are swap-removed). Re-fetch them after `SetComp` adds a component, `RemoveComp`, `NewEntity` or `DeleteEntity`.
- `SetComp` replaces values in place when the entity already has every given component. Otherwise the entity moves to a new archetype exactly once, however many components are added.
- `SetComp` validates everything (stale entity, pointer or unregistered values, duplicate types) before modifying the entity. The `values` slice is not modified. The entity check runs first, even for an empty call: `SetComp(e)` on a stale `e` still panics with `ecs: invalid or stale entity`, it just does nothing otherwise.
- `RemoveComp` tolerates the same ID more than once in `ids` (unlike the duplicate-type check in `NewEntity`/`SetComp`); repeats are just removed once.
- Values passed as `any` are boxed, which can allocate. For hot code use [`SetComp[T]`](api-generic.md#component-access) or [`View.Set`](api-generic.md#views), which don't.

## Queries

`world.Query(ids ...ComponentID)` returns a `Query` over every entity whose archetype contains all of `ids`. `World.Query()` with no IDs matches every archetype. Duplicate IDs panic.

`Query.Each()` is an `iter.Seq2[Entity, []any]`; the slice holds one pointer per requested component, in the order `ids` was given.

```go
q := world.Query(posID, velID)

for e, comps := range q.Each() {
    pos := comps[0].(*Position)
    vel := comps[1].(*Velocity)
    pos.X += vel.X
    _ = e
}
```

Rules:

- **Do not retain the `[]any` slice** (or keep it past the loop body); it is reused between iterations. Copy the pointers out if needed.
- **Reuse queries.** A `Query` caches its matching archetypes and keeps working after the world changes, so create it once and iterate it every frame.
- **Mutation during iteration.** Replacing an existing component value (`SetComp` on a component the entity already has, or writing through a pointer) is allowed. Structural changes (`NewEntity`, `DeleteEntity`, adding or removing components) make the iteration panic with `ecs: query invalidated by world mutation`. Collect entities first and apply changes after the loop.
- Breaking out of the loop early is fine.
- The zero `Query` has no world and panics with `ecs: not bound to a world` when iterated.

## Resources

Resources are singleton values stored on the `World` and keyed by type. They are not components and are not attached to entities. The resource functions are generic, need no registration, and follow the same `Get`/`GetOrNil`/`Has`/`Set`/`Remove` convention as components. See [Resources in the generic API](api-generic.md#resources).

## Panics

The API panics on programmer errors instead of returning errors. Each message starts with a stable text (below), optionally followed by `: <detail>` naming the type, ID, handle or resource, for example `ecs: entity does not have component: main.Weapon`. Match the stable text with `strings.Contains`.

| Message | Cause |
| --- | --- |
| `ecs: component must be a non-pointer struct: <type>` | `RegisterComp` or first generic use with a non-struct `T`; a pointer value passed to `NewEntity` or `SetComp` (detail `<nil>` for a `nil` value). |
| `ecs: component registry capacity exceeded: max 4096` | More than 4096 registered components. |
| `ecs: component not registered: <type> or ComponentID(<id>)` | Low-level API only: `<type>` for a value of an unregistered type (`NewEntity`, `SetComp`), `ComponentID(<id>)` for an unknown ID (`GetComp`, `GetCompOrNil`, `HasComp`, `RemoveComp`, `World.Query`). |
| `ecs: duplicate component` | The same component twice in one `NewEntity`, `SetComp`, `NewViewN` (type parameters, or two `Without` filters) or `World.Query` call; detail is the type. |
| `ecs: component both required and excluded` | `NewViewN` with a `Without[T]` filter for one of its own type parameters; detail is the type. |
| `ecs: zero Filter` | `NewViewN` with an `ecs.Filter{}` that was not made by `Without`. |
| `ecs: invalid or stale entity: <entity>` | Using a deleted, never-created or zero `Entity`; detail is `%+v` of the handle, e.g. `{ID:3 Generation:2}`. |
| `ecs: entity does not have component` | `GetComp` or `View.Get` for a missing component. |
| `ecs: query invalidated by world mutation` | Structural change while iterating with `Each`. |
| `ecs: not bound to a world` | Using a zero `Query` or zero `View`. |
| `ecs: resource not set: <type>` | `GetResource[T]` for a resource that was never set or was removed; detail is `T`. |
