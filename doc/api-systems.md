# Systems API (package `system`)

Systems are **not part of `World`**. Package `ecs` has no `System` type and no scheduler; package `system` (`github.com/weakpixel/ecs/system`) is an optional layer on top of it. It imports `ecs` and the standard library only, and knows nothing about rendering or any game framework. The host loop (a game's update, a server tick, a test) owns the timing and calls `Schedule.Run(world, delta)`.

```go
import "github.com/weakpixel/ecs/system"
```

## Types

| Identifier | Meaning |
| --- | --- |
| `type Context struct{ World *ecs.World; Time Time; Commands *Commands }` | Passed to every system call. Carries the World and the per-tick data systems commonly need. |
| `type System func(ctx *Context)` | One unit of logic. Runs on the World's goroutine. |
| `type Runner interface{ Run(ctx *Context) }` | A system with state, as a struct. Every `System` is a `Runner`. |
| `type Initializer interface{ Init(ctx *Context) }` | Optional on a `Runner`: one-time setup, run by the first `Run` after the runner is added. |
| `type Schedule` | Ordered, named stages of systems. The zero value is ready to use. |
| `type Commands` | Buffer of deferred structural changes. A World resource. |
| `type Time struct{ Delta time.Duration; Tick uint64 }` | Timing of one `Run`: `Delta` as passed to `Run`, `Tick` counting from 0. |

## Context

The schedule owns one `Context`, refreshes it at the start of every `Run` and passes the same pointer to every call. That is why `Run` makes no allocations.

- **Lifetime:** the pointer is valid only for the call it is passed to. Never keep it, or `ctx.Time` or `ctx.Commands`, after returning. Keep `ctx.World` if you need the World later.
- **Rule for new fields:** only data that the schedule owns and refreshes each tick belongs in `Context`. Game-owned state (settings, score, input devices) stays in World resources.
- `ctx.Commands` is the same buffer `CommandsOf(ctx.World)` returns. `CommandsOf` stays for code that has only a `*ecs.World`, such as `Commands.Do` callbacks.

## Schedule

```go
var s system.Schedule
s.AddStage("input", readInput)
s.AddStage("simulate", movement, ai)
s.Add(collisions)            // extends the last stage ("simulate")
s.Run(world, delta)          // once per tick
```

- `AddStage(name, sys...)` appends a stage and makes it the target of later `Add` calls.
- `Add(sys...)` appends to the most recently added stage. If there is none, it creates a stage named `default`.
- `AddRunner(r...)` is `Add` for `Runner` values: a struct system keeps prebuilt views and scratch data between ticks.
- `Run(w, delta)` runs one tick: it sets `ctx.Time.Delta` and `ctx.Time.Tick`, then runs the stages in order and applies `Commands` after each stage, so changes queued in one stage are visible to the next. Before the first stage it runs pending `Init`s (below).
- Systems and stages run in the order added. There is no automatic dependency ordering and no parallel execution.

### Struct systems

```go
type Movement struct{ view ecs.View2[Position, Velocity] }

func NewMovement(w *ecs.World) *Movement {
    return &Movement{view: ecs.NewView2[Position, Velocity](w)}
}

func (m *Movement) Run(ctx *system.Context) {
    dt := ctx.Time.Delta.Seconds()
    for _, c := range m.view.Each() {
        p, v := c.Unpack()
        p.X += v.X * dt // X is float64 here
    }
}

s.AddStage("simulate")
s.AddRunner(NewMovement(world))
```

### Init

A `Runner` that also implements `Initializer` gets `Init(ctx)` once, at the start of the first `Run` after it was added, before any stage runs. Runners are initialized in the order added, and `Commands` queued by `Init` are applied before the first stage. A runner added after the schedule has already run is initialized at the start of the next `Run`. The check for `Initializer` happens in `AddRunner`, so steady-state `Run` pays nothing. Function systems have no `Init`; set them up before `Add`.

A schedule initializes its runners against one World. `Run` binds the schedule to the first World it sees and panics if it is later called with another one.

## Commands

Structural changes inside `View.Each` panic (see [api-low-level.md](api-low-level.md)). A system queues them instead (`CommandsOf(w)` returns the same buffer, creating the World resource on first use):

```go
cmds := ctx.Commands
for e, c := range view.Each() {
    if dead(c) { cmds.Despawn(e) }
}
```

| Method | Effect when applied |
| --- | --- |
| `Spawn(values ...any)` | `w.NewEntity(values...)`. The slice is not copied. |
| `Despawn(e)` | `w.DeleteEntity(e)`, skipped if `e` is no longer alive (queueing it twice is harmless). |
| `Do(fn)` | Calls `fn(w)`. Use it for `SetComp`, `RemoveComp` and other changes. |
| `Apply(w)` | Runs the queue in order and empties it. Commands queued by a `Do` callback run in the same call. |
| `Len()` | Number of queued commands. |

`Schedule.Run` calls `Apply` for you. Call it yourself only if you use `Commands` without a `Schedule`, and never while a view is iterating.

## Events

An event queue lets one system tell others that something happened without knowing who listens. `EventsOf[T](w)` returns the World's queue of `T` events, creating the resource on first use. A sender calls `Send`. A reader keeps an `EventReader[T]`, created once (in `Init`), and ranges over `Read()`:

```go
type Hit struct {
    Target ecs.Entity
    Damage int
}

// In the combat system:
system.EventsOf[Hit](ctx.World).Send(Hit{Target: e, Damage: 1})

// In a sound system, created in Init: s.hits = system.NewEventReader[Hit](ctx.World)
for h := range s.hits.Read() {
    if ctx.World.IsAlive(h.Target) {
        playHitSound(h)
    }
}
```

| Function or method | Effect |
| --- | --- |
| `EventsOf[T](w)` | The World's `*Events[T]`, created on first use. |
| `Events.Send(ev)` | Adds `ev` to the queue. |
| `Events.Len()` | Number of events in the queue (this tick and the previous one). |
| `NewEventReader[T](w)` | A reader that starts with the oldest event still in the queue. |
| `EventReader.Read()` | Yields the events this reader has not seen yet, oldest first. |

- **Lifetime.** An event can be read until the end of the tick after the one it was sent in. `Schedule.Run` ages every queue once, after its last stage. A reader that runs after the sender sees the event in the same tick; one that runs before it sees it in the next tick. Events sent between `Run` calls count as part of the next tick.
- **Once per reader.** Each reader sees each event at most once; every reader of a type sees every event. An event counts as read once it is yielded, so breaking out of the loop leaves the rest for the next `Read`. Events sent while the loop runs are yielded too.
- **Missed events.** A reader that does not read for more than one tick misses the dropped events, without a panic. This happens, for example, to a system in a stage the host skips while paused.
- **Stale entities.** Events are plain values. An entity in an event may be deleted by the time a reader sees it; check `IsAlive`.
- Events are never saved; they are not part of the World's entities or resources snapshot.

Use a callback instead when one owner sets up a single hook, for example an `OnDone` function on a timer component.

## Panics

| Condition | Message |
| --- | --- |
| nil `*ecs.World` passed to `Run(w, delta)`, `CommandsOf`, `EventsOf` or `NewEventReader` | `system: nil world` |
| nil system passed to `Add`, `AddRunner` or `AddStage` | `system: nil system` |
| `AddStage` with a name already used | `system: duplicate stage: <name>` |
| `Run` with a World other than the one of the first `Run` | `system: schedule used with a different world` |
| nil function passed to `Commands.Do` | `system: nil command` |

## Performance

In steady state `Schedule.Run` makes 0 allocations (the command queue reuses its capacity), as asserted by `TestRunAllocs`. `Commands.Spawn` allocates its variadic slice, and a `Do` closure that captures variables allocates; `Despawn` and a capture-free `Do` do not. Event queues reuse their buffers: once warm, `Send`, `Read` in a range loop and the aging in `Run` make 0 allocations (`TestEventsAllocs`).

## Rendering

Not part of the schedule. Keep drawing outside `Run`, read-only, as described in [performance-guide.md](performance-guide.md).
