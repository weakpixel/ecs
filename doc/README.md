# ECS guide

`github.com/weakpixel/ecs` is an archetype-based Entity Component System: a `World` holding entities and their components. The [quick start](../README.md#quick-start) shows a complete program.

## Concepts

- **Components** are plain non-pointer structs. The generic API registers them on first use; the low-level API needs `RegisterComp` first.
- **Entities** are handles: a slot `ID` plus a `Generation`. Handles to deleted entities are stale and panic when used. The zero `Entity{}` is never alive, so it works as "no entity".
- **Archetypes.** Entities with the same set of components are stored together, column by column. Iteration walks contiguous arrays; adding or removing a component moves the entity to another archetype.
- **Pointers into storage.** `GetComp`, views and queries hand back pointers to live world data. Writing through them changes the world. They are valid until the next structural change (creating or deleting an entity, adding or removing a component).
- **Iteration and mutation.** Replacing a component value during iteration is allowed. Structural changes during iteration panic; queue them with [`Commands`](api-systems.md#commands) or apply them after the loop.
- **Views and queries cache their matching archetypes.** Build them once and reuse them every tick.
- **Resources** are per-type singletons stored on the `World` (time, camera, input), not components.
- **Systems** are not part of `World`. A system is a function you call; [`system`](api-systems.md) adds an optional stage-based `Schedule`.
- **The `World` is single-goroutine.** Only the component registry is process-global and safe for concurrent registration.

## One set of verbs

Every layer uses the same verbs with the same meaning: `NewEntity`, `Get` (panics if missing), `GetOrNil`, `Has`, `Set` (add or replace), `Remove`, `Each`. The full table is in [Conventions](api-low-level.md#conventions).

## Which layer to use

| Layer | Use it for | Page |
| --- | --- | --- |
| Generic API | Almost everything: typed, no registration, 0-alloc views | [api-generic.md](api-generic.md) |
| Low-level API | Component types known only at run time (`ComponentID`, `any`), tools, serialization | [api-low-level.md](api-low-level.md) |
| Systems | Ordering systems in stages and deferring structural changes | [api-systems.md](api-systems.md) |

The layers operate on the same entities and mix freely: a `View2[Position, Velocity]`, `GetComp[Health]` and a low-level `World.Query` can all touch one entity.

## Then

- Write a game loop: [Usage and performance guide](performance-guide.md).
- How it is built and why: [Design](design.md).
