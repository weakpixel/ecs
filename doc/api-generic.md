# Generic API

The generic API is a typed layer over the [low-level API](api-low-level.md): no `ComponentID`s, no `any`, no type assertions. Components are identified by their Go type, and it uses the same [verbs](api-low-level.md#conventions) as the other layers.

```go
import "github.com/weakpixel/ecs"

type Position struct{ X, Y int }
type Velocity struct{ X, Y int }

world := ecs.NewWorld()
```

Contents: [Registration](#registration) · [Entity creation](#entity-creation) · [Component access](#component-access) · [Views](#views) · [Resources](#resources)

## Registration

Every generic function registers its component types on first use, so the generic API needs no `RegisterComp` calls. A type must still be a non-pointer struct; otherwise the first use panics with `ecs: component must be a non-pointer struct`.

Call `RegisterComp[T]()` (see [Components](api-low-level.md#components)) when you need a type's `ComponentID` for the low-level API, or when the low-level API is the first to see a type.

## Entity creation

`NewEntity1` … `NewEntity16` create an entity with 1 to 16 components, inferring the types from the arguments.

```go
e := ecs.NewEntity2(world, Position{X: 1, Y: 2}, Velocity{X: 3, Y: 4})
```

For more than 16 components, or a dynamic set, use `world.NewEntity(values ...any)`. Duplicate component types panic. To create many entities of the same shape, use [`View.NewEntity`](#views), which resolves the archetype once. Deleting and liveness checks are `world.DeleteEntity(e)` and `world.IsAlive(e)`, see [Entities](api-low-level.md#entities).

## Component access

| Function | Description |
| --- | --- |
| `GetComp[T](world, e) *T` | Pointer to the live component. Panics if `e` is stale or lacks `T`. |
| `GetCompOrNil[T](world, e) *T` | Same, but `nil` when `e` lacks `T`. |
| `HasComp[T](world, e) bool` | Whether `e` has `T`. |
| `SetComp[T](world, e, value T)` | Adds `T` or replaces it. |
| `RemoveComp[T](world, e)` | Removes `T`; no-op if absent. |

```go
pos := ecs.GetComp[Position](world, e)
pos.X = 10 // writes to the world's storage

if vel := ecs.GetCompOrNil[Velocity](world, e); vel != nil {
    vel.X = 99
}

ecs.SetComp(world, e, Velocity{X: 1, Y: 1})
ecs.RemoveComp[Velocity](world, e)
```

All five panic if `e` is stale. To set or remove several components in one archetype move, use a [View](#views).

## Views

A view is a fixed set of component types: `NewView1[T1](world)` … `NewView16[T1…T16](world)` return a `View1` … `View16`. Every method is about entities that have **all** of the view's components. Construct a view once per world and reuse it. It resolves the component IDs at construction, and the archetype it creates entities in on first use.

```go
movement := ecs.NewView2[Position, Velocity](world)

e := movement.NewEntity(Position{X: 1}, Velocity{X: 2})

pos, vel := movement.Get(e).Unpack()
pos.X += vel.X

movement.Set(e, Position{}, Velocity{X: 5}) // add or replace both, at most one move

for e, c := range movement.Each() {
    c.Comp1.X += c.Comp2.X
    _ = e
}

movement.Remove(e) // removes Position and Velocity, at most one move
```

| Method | Description |
| --- | --- |
| `NewEntity(v1, v2, ...) Entity` | Creates an entity with exactly the view's components. The fastest way to create many entities of one shape. |
| `Get(e)` | Pointers in type-parameter order as a `ComponentsN` (`View1` returns `*T1`). Panics if `e` is stale or lacks any component. |
| `Has(e) bool` | `true` only when `e` has all components and none that a [filter](#filters) excludes, so exactly when `Each` would yield `e`. Panics if `e` is stale. |
| `Set(e, v1, v2, ...)` | Adds or replaces all components. In place when `e` already has all of them, otherwise one archetype move. |
| `Remove(e)` | Removes the view's components that `e` has, in one archetype move. |
| `Each() iter.Seq2[Entity, ComponentsN]` | Yields every entity that has all components (`View1` yields `*T1`). Entities with additional components also match, unless a [filter](#filters) excludes one of them. |

There is no `GetOrNil` on views, because a `ComponentsN` result can't be nil. Use `Has` first:

```go
if movement.Has(target) {
    pos, _ := movement.Get(target).Unpack()
    pos.X = 0
}
```

### Filters

The constructors take optional filters after the world: `NewViewN[T1…TN](world, filters ...ecs.Filter)`. `ecs.Without[T]()` drops every entity that has a `T`:

```go
moving := ecs.NewView2[Position, Velocity](world, ecs.Without[Body]())

for _, c := range moving.Each() { // entities with a Body are not yielded
    c.Comp1.X += c.Comp2.X
}
```

- `Each` and `Has` follow the filters. `Get`, `Set`, `Remove` and `NewEntity` work on the view's components and ignore them.
- A filter is checked once per archetype, when the view's match cache takes the archetype in. It costs nothing per entity, unlike a `HasComp` check in the loop body. An entity that gains an excluded component moves to an archetype the view does not match, so the next `Each` skips it.
- `Without[T]` registers `T`, like the other generic functions.

### ComponentsN

`ComponentsN` (N ≥ 2) holds pointer fields `Comp1`, `Comp2`, … in type-parameter order. `Unpack()` returns them as separate values, which lets you name them:

```go
for _, c := range movement.Each() {
    pos, vel := c.Unpack()
    pos.X += vel.X
}
```

### Rules

- **Store and reuse views.** A view is a small value holding a pointer to shared state, so copies share it. The archetype match cache for `Each` is built lazily on the first `Each`, so a view used only for `NewEntity`, `Get` or `Set` costs no query bookkeeping. A view is bound to the world passed to the constructor. For another world, call the constructor again.
- **Mutation during iteration.** Writing through the pointers, or a `Set` on an entity that already has all components (in place), is allowed. Structural changes (creating or deleting entities, adding or removing components, including a `Set` that adds or a `Remove`) make the iteration panic with `ecs: query invalidated by world mutation`. Collect entities first and apply changes after the loop.
- **Pointers** returned by `Get` and `Each` point into archetype storage and are invalidated by structural changes. Do not keep `ComponentsN` values after a structural change or past the loop body.
- The constructor panics with `ecs: duplicate component` if a type is listed twice (as type parameters, or in two `Without` filters), with `ecs: component both required and excluded` if a `Without` names one of the view's types, and with `ecs: zero Filter` for an `ecs.Filter{}` not made by `Without`. The zero `View` panics with `ecs: not bound to a world`.
- `NewEntity`, `Get`, `Set` (in place and when moving) and `Each` (with or without filters) are 0 allocs/op.

```go
type Game struct {
    world    *ecs.World
    movement ecs.View2[Position, Velocity] // built once
}

func (g *Game) Update() {
    for _, c := range g.movement.Each() {
        pos, vel := c.Unpack()
        pos.X += vel.X
    }
}
```

## Resources

Resources are world-level singletons keyed by type. They are not components and need no registration. The functions follow the same [convention](api-low-level.md#conventions) as component access.

| Function | Description |
| --- | --- |
| `SetResource[T](world, *T)` | Stores the pointer, replacing any earlier `T`. |
| `GetResource[T](world) *T` | Returns the resource. Panics with `ecs: resource not set: <T>` if it is not set. |
| `GetResourceOrNil[T](world) *T` | Same, but `nil` when not set. |
| `HasResource[T](world) bool` | Whether a `T` is set. |
| `RemoveResource[T](world)` | Removes the resource; no-op if absent. |

```go
type Time struct{ Delta float64 }

ecs.SetResource(world, &Time{Delta: 1.0 / 60})
dt := ecs.GetResource[Time](world).Delta

if cam := ecs.GetResourceOrNil[Camera](world); cam != nil {
    cam.X = 0
}
ecs.RemoveResource[Time](world)
```

## Generated code

`NewEntityN`, `NewViewN`, `ViewN` and `ComponentsN` live in `zz_gen_generic-api.go`, generated by `internal/gen`. Do not edit it by hand.
