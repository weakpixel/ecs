---
name: ecs-core
description: Use for ECS core APIs and internals, including registry, storage, entities, world operations, queries, resources, panics, pointer validity, or allocations.
---

# ECS core implementation notes

The API contract is in [the low-level guide](../../../doc/api-low-level.md), [the generic guide](../../../doc/api-generic.md) and [the performance guide](../../../doc/performance-guide.md). Design rationale and the current implementation are in [the design page](../../../doc/design.md); the exported API checklist is in [reference/api.md](reference/api.md). This skill records caller contracts and focused regression checks that are easy to miss.

## Caller contracts

Callers, including code generators and loaders built on the low-level API, depend on these. Preserve them when changing core behavior:

1. `ComponentID` is an integer type. Registration is process-global, safe for concurrent calls, and works during package initialization (`var idHealth = ecs.RegisterComp[Health]()`).
2. `GetCompOrNil(e, id)` returns an untyped `nil` when absent, not a typed nil pointer.
3. `World.NewEntity(values ...any)` accepts component values (`T`), not pointers (`*T`).
4. `World.Query()` with no IDs matches every archetype.
5. Queries and views are small value types sharing state through a pointer. Copies share caches; zero values panic with `ecs: not bound to a world`.
6. No-op structural operations leave the revision unchanged, so `SetComp(e, all...)` followed by `RemoveComp(e, absent...)` is legal during `Each` when nothing moves.
7. `DeleteEntity` must correctly remove any row, so a loader can roll back the entities it created in any order.

Everything a caller needs must be exported API: code outside this module cannot reach `internal/`.

## Focused regression checks

- Cover every row of the low-level panic table and stale handles after slot reuse.
- Check swap-remove repairs the moved entity, multi-component `SetComp` moves once, and validation failures do not partially mutate.
- Check in-place writes and no-op removals during `Each`, structural-mutation panics, and early break after mutation.
- Check queries see later archetypes and nested iteration does not alias component slices.
- Check pointer-bearing components survive moves and GC, and concurrent registration returns one ID per type.
- Use `package ecs` tests for invariants and `package ecs_test`/`example_test.go` for public examples. Run `go test -race ./...` for core changes.
