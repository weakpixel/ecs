# Usage and performance guide

Practical rules for using the ECS in a real-time loop (a game, a simulation, a server tick). Signatures live in the API pages ([low-level](api-low-level.md), [generic](api-generic.md), [systems](api-systems.md)); this guide covers how to use them well.

Contents: [Mental model](#mental-model) · [Structuring a loop](#structuring-a-loop) · [Things to look out for](#things-to-look-out-for) · [Performance practices](#performance-practices) · [Rendering](#rendering) · [Measuring](#measuring) · [Checklist](#checklist)

## Mental model

- Entities with the same set of components live together in one **archetype**, stored column by column. Iterating a view walks contiguous arrays per archetype.
- The cheap operations are iteration, `GetComp`, `View.Get`, and replacing a component the entity already has (in place, no move).
- The expensive operations are **structural changes**: creating or deleting an entity, adding or removing a component. Adding or removing a component moves the whole row to another archetype. Deleting swap-removes the row.
- Every structural change bumps a world revision. That invalidates running iterations (they panic) and all pointers you obtained earlier.
- `ecs` has no `System` type and no scheduler. A system is a function (or a struct with an `Update` method) that you call in an order you choose. The optional [`system`](api-systems.md) package provides a stage-based `Schedule` and a `Commands` buffer for deferred structural changes.

## Structuring a loop

A `World` is not safe for concurrent use. Run every update and every read of the world on one goroutine, the one that owns the loop.

Build the world and views once, when the program starts:

```go
type Time struct{ Delta float64 }

type Game struct {
    world *ecs.World

    // views built once, reused every tick
    movement ecs.View2[Position, Velocity]
    sprites  ecs.View3[Position, Sprite, Layer]

    deferred []func() // structural changes, applied after the systems ran
}

func NewGame(tickRate int) *Game {
    w := ecs.NewWorld() // the generic API registers component types on first use
    ecs.SetResource(w, &Time{Delta: 1.0 / float64(tickRate)})

    return &Game{
        world:    w,
        movement: ecs.NewView2[Position, Velocity](w),
        sprites:  ecs.NewView3[Position, Sprite, Layer](w),
    }
}

func (g *Game) Update() {
    dt := ecs.GetResource[Time](g.world).Delta

    g.updateInput()
    g.updateMovement(dt)
    g.updateAI()
    g.applyDeferred() // spawns, despawns, add/remove components
}

func (g *Game) updateMovement(dt float64) {
    for _, c := range g.movement.Each() {
        pos, vel := c.Unpack()
        pos.X += vel.X * dt
        pos.Y += vel.Y * dt
    }
}

func (g *Game) Draw() {
    // read-only: no structural changes while drawing
    for _, c := range g.sprites.Each() {
        pos, sprite, _ := c.Unpack()
        // draw sprite at pos
    }
}

func main() {
    const tickRate = 60
    g := NewGame(tickRate)
    tick := time.NewTicker(time.Second / tickRate)
    defer tick.Stop()
    for range tick.C {
        g.Update()
        g.Draw()
    }
}
```

The example applies structural changes from a hand-written `deferred` list to show the idea. The optional [`system`](api-systems.md) package packages the same pattern: a `Schedule` runs systems in stages and applies queued `Commands` after each stage.

Guidelines:

- **Logic in `Update`, reads in `Draw`.** With a fixed timestep, `Update` can run zero, one or several times per drawn frame. Do not spawn, delete or move entities in `Draw`.
- **Fixed timestep.** Put the step (or a tick counter) in a resource instead of measuring wall-clock time per system. Simulation stays deterministic and `Draw` can interpolate if needed.
- **System order is your order.** Input, then simulation, then collision/AI, then the deferred structural changes, then animation/render preparation. Document it in one place (the `Update` body).
- **Cross-cutting state goes in resources** (time, input snapshot, camera, asset stores), not in components attached to a dummy entity.
- **Scenes/levels.** Prefer one `World` per scene and drop it on scene change over deleting thousands of entities one by one. Worlds share only the component registry, so this is safe. Build new views for the new world; a view is bound to the world it was built for.
- **Do not share the `World` with other goroutines**, such as loaders. Load assets in the background, then hand the result to `Update` through a channel and create the entities there.

## Things to look out for

### Structural changes during iteration panic

Creating or deleting entities and adding or removing components inside a `for ... range view.Each()` loop panics with `ecs: query invalidated by world mutation`. Writing through the component pointers, or replacing a component value the entity already has, is fine.

Record the intent and apply it after the loop:

```go
func (g *Game) updateLifetime() {
    var dead []ecs.Entity // reuse a field-backed slice (g.dead[:0]) to avoid allocating
    for e, c := range g.lifetimes.Each() {
        c.Comp1.Left--
        if c.Comp1.Left <= 0 {
            dead = append(dead, e)
        }
    }
    for _, e := range dead {
        g.world.DeleteEntity(e)
    }
}
```

For effects raised by many systems (spawn a bullet, remove `Stunned`), collect closures or small command structs in a slice and run them in one place after the systems (`applyDeferred` above). Reset the slice with `s = s[:0]` to keep its capacity.

### Pointers are only valid until the next structural change

`GetComp`, `View.Get`, `Each` results and `ComponentsN` point into archetype storage. After `NewEntity`, `DeleteEntity`, `SetComp` that adds a component, or `RemoveComp`, the row may have moved or been swap-removed, so the pointer may point at another entity's data. Never store them in structs, maps or closures. Store the `Entity` handle and look it up when needed.

### Stale handles panic

A handle to a deleted entity panics when used, even if its slot was reused (the generation counter catches that). Anything that holds an entity across frames (a target, a parent, an owner) must check `world.IsAlive(e)` first, and drop or replace the reference when it is dead. `IsAlive` never panics. The zero `ecs.Entity{}` is never alive, so it works as "no target" in a struct field.

### Do not iterate the same view inside itself

`Each` reuses one internal buffer per view (and per untyped `Query`). A nested iteration of the *same* view falls back to allocating a buffer. For pair checks (collision, proximity) use two separate views, or collect candidates into a reused slice first.

### Other panics that come from programmer errors

- The low-level API needs component types registered (`RegisterComp`) before use; the generic API registers on first use. The registry is process-global, max 4096.
- `GetComp` and `View.Get` on a missing component panic; use `GetCompOrNil` or check `View.Has` first for optional data. `GetResource` panics when the resource is not set; use `GetResourceOrNil`.
- Do not retain the untyped `[]any` slice or `ComponentsN` structs past the loop body.

The full list is in [Panics](api-low-level.md#panics).

## Performance practices

Measure first (see [Measuring](#measuring)); in a typical game, rendering costs far more than the ECS. These practices keep the ECS part at zero allocations per frame.

1. **Reuse views.** Views (`NewView1`…`NewView16`) are the fast path and iterate with 0 allocs/op. Prefer them to the untyped `world.Query(ids...)` in hot code.
2. **Avoid per-frame add/remove churn.** Every add/remove moves the row between archetypes. A component that toggles every frame (`OnGround`, `Highlighted`, `Moving`) should be a field (`bool`, state enum) of an existing component, not an add/remove tag. Reserve add/remove for changes that happen rarely (a state transition, equipping a weapon).
3. **Batch component changes.** Adding several components with `View.Set(e, v1, v2, v3)` (or the low-level `world.SetComp(e, v1, v2, v3)`) moves the entity once; three separate generic `SetComp` calls move it up to three times. `View.Remove` likewise removes several components in one move. Creating an entity with its full component set in one `NewEntityN`/`View.NewEntity` call is cheaper than creating and adding afterwards.
4. **Replace in place when possible.** `SetComp` of a component the entity already has, or writing through the pointer, does not move anything.
   Generic `SetComp[T]` and `View.Set` are allocation-free, in place and when they move the entity; the low-level `world.SetComp(e, values ...any)` boxes each value into `any`, which can allocate.
5. **Keep components small, plain data.** Position, velocity and health belong in tight structs. Avoid pointers, maps, slices and interfaces inside hot components: they cost cache locality and GC scanning. Large rarely-read data (names, dialogue, definitions) should live in an asset store or resource and be referenced by an ID.
6. **Split components by access pattern.** A system that only needs `Position` and `Velocity` should not drag a 200-byte `Stats` struct through the cache. Fine-grained components also make queries match precisely.
7. **Mind fragmentation.** Each distinct set of components is an archetype, and queries iterate their matching archetypes one after another. A few dozen archetypes are fine. Hundreds of tiny ones (many optional components, tag per state) make iteration slower and spawning more expensive. Use fixed bundles for object kinds and state fields instead of tag combinations.
8. **No allocations inside iteration.** No `fmt.Sprintf`, no new slices, no closures capturing loop variables, no boxing into `any` in the loop body. Reuse scratch slices stored on the system.
9. **Bulk spawn and despawn at tick boundaries**, not mid-system. Large waves (particles, bullets) are cheaper created in one go than interleaved with iteration. For many entities of the same shape, build a view once (`ecs.NewViewN[...](world)`) and call `NewEntity`: it resolves the archetype and column positions up front, so each spawn only claims an entity and appends the values (target 3-4x faster than `NewEntityN`, machine-specific, 0 allocs/op for both). A view is bound to one world; build it once and rebuild it for a new world. See [Views](api-generic.md#views). If spawn/despawn churn shows up in profiles, keep entities alive and toggle an `Active` field instead (pooling).
10. **Cache `Entity` handles, not lookups.** Resolve cross-entity references (target, parent) once and store the handle; `GetComp` is cheap but not free. For hot random access to a fixed component set, use a view's `Get`: it validates the entity and finds its archetype once for all components (target about 1.25x faster than the same `GetComp[T]` calls, machine-specific).
11. **Do not scan for singletons.** Use resources (`GetResource[T]`) for camera, input, time. Looking them up through a query each frame is wasteful.
12. **Filter archetypes, not entities.** To skip entities that have some component, build the view with `ecs.Without[T]()` instead of calling `HasComp` in the loop body. The filter is checked once per archetype; a generic `HasComp` per entity costs a type lookup each time (several times the loop body for a simple system). See [Filters](api-generic.md#filters).

## Rendering

For 2D games the cost is usually in draw calls and image state changes, not in iterating entities. The ECS does not draw; these rules are about how the drawing code reads the world.

- **Separate preparation from drawing.** A render-prepare system (in `Update`) walks the world, culls, and fills a reused list of draw items sorted by layer. `Draw` only walks that list. This keeps `Draw` free of world access and structural changes.
- **Cull before drawing.** Skip entities outside the camera bounds using a rectangle test in the prepare step.
- **Sort by layer, then by atlas/image**, so consecutive draws use the same source image and the graphics library can batch them. Use a `Layer`/`ZIndex` component for ordering, and sort a reused slice instead of allocating per frame (`slices.SortFunc`).
- **Use sprite atlases.** Draw sub-images of one big image; avoid many small images.
- **Draw many quads in one call.** For thousands of tiles or particles, one draw call per quad costs more CPU than the drawing itself. Append four vertices per quad (source coordinates in the atlas, the transformed corners) to a reused slice and draw a run of quads with the same image and blend mode at once.
- **Reuse draw options and sub-image values.** Fresh option structs and sub-images per draw allocate; cache sub-images per frame index and reset the reused options between draws.
- **Keep animation state as data.** Frame index, timer and playing flag live in a component; the sprite sheet lookup lives in a store referenced by ID.
- **Interpolate for smooth motion.** With a fixed 60 TPS and high-refresh displays, keep the previous position as a component field and blend between previous and current position when computing the draw position, instead of running more ticks.

## Measuring

- **Frame budget**: at 60 ticks per second you have about 16.6 ms per tick. Measure the actual tick and frame rate and show them in a debug overlay.
- **Profile** with `runtime/pprof` (CPU, alloc) or `net/http/pprof` while playing. Look for allocations per frame: the steady state of a well-structured game is zero allocs in `Update`. Use `GODEBUG=gctrace=1` to spot GC pauses.
- **Library benchmarks**: `make bench name=<label>` and `make benchcmp new=<label>` (needs `benchstat`). Results go to `build/bench/` and are machine-specific: compare only runs from the same machine, and do not run other work during a benchmark (noise of ±15–35% otherwise).
- **Check escape analysis** with `go build -gcflags=-m` before claiming an allocation win from a benchmark; synthetic call sites can be stack-allocated and hide real costs.
- **Benchmark your own systems**: build a world with a realistic number of entities and archetypes, `b.ReportAllocs()`, `b.ResetTimer()`, and iterate the view `b.N` times.

## Checklist

Do:

- Build the world and views once.
- Keep the world on the goroutine that runs the loop.
- Apply structural changes after the systems, from a deferred list.
- Store `Entity` handles, check `IsAlive`, and re-fetch pointers after structural changes.
- Use state fields for values that toggle often.
- Use views and reused scratch slices.
- Use `View.NewEntity` for entity kinds you create often.
- Prepare render lists in `Update`, sort by layer and image, cull, and use atlases.

Do not:

- Create, delete or add/remove components inside an `Each` loop.
- Keep component pointers, the untyped `[]any` slice or `ComponentsN` values beyond the loop body or a structural change.
- Build views per frame.
- Allocate inside iteration, or put maps, slices or pointers into hot components.
- Touch the world from other goroutines or from `Draw`.
- Add tag components that flip every frame.
