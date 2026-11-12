# ecs

`github.com/weakpixel/ecs` is an archetype-based Entity Component System for Go. Components are plain structs, entities are stable handles with a generation counter, and the `World` stores entities with the same components together, column by column, so views iterate contiguous arrays with zero allocations. An optional `system` package adds a stage-based `Schedule`, deferred `Commands` and event queues.

The module targets Go 1.27, uses the standard library only and has no dependencies. It builds for js/wasm.

| Package | Import path | What it is |
| --- | --- | --- |
| `ecs` | `github.com/weakpixel/ecs` | `World`, entities, components, views, queries, resources |
| `system` | `github.com/weakpixel/ecs/system` | Optional stage-based `Schedule`, deferred `Commands`, events |

## Installation

```bash
go get github.com/weakpixel/ecs
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/weakpixel/ecs"
)

type Position struct{ X, Y int }
type Velocity struct{ X, Y int }

func main() {
	world := ecs.NewWorld()

	entity := ecs.NewEntity2(world, Position{X: 1, Y: 2}, Velocity{X: 3, Y: 4})

	pos := ecs.GetComp[Position](world, entity)
	pos.X = 10

	movement := ecs.NewView2[Position, Velocity](world) // build once, reuse
	for _, c := range movement.Each() {
		pos, vel := c.Unpack()
		pos.X += vel.X
		pos.Y += vel.Y
	}
	fmt.Println(ecs.GetComp[Position](world, entity)) // &{13 6}

	world.DeleteEntity(entity)
	fmt.Println(world.IsAlive(entity)) // false
}
```

## Documentation

- [ECS guide](doc/README.md): concepts and which layer to use.
- [Generic API](doc/api-generic.md): typed entity creation, component access, views, filters, resources.
- [Low-level API](doc/api-low-level.md): `ComponentID`s, untyped queries, the panic table.
- [Systems API](doc/api-systems.md): `Schedule`, `Commands`, events.
- [Usage and performance guide](doc/performance-guide.md): structuring a loop, pitfalls, performance practices.
- [Design](doc/design.md): how storage, entities and iteration are built and why.

## Development

```bash
make all        # go generate, fmt, vet, test -race
make wasm-test  # the tests under GOOS=js GOARCH=wasm (needs Node)
make bench name=<label>    # benchmarks into build/bench/<label>.txt
make benchcmp new=<label>  # benchstat against build/bench/baseline.txt
```

## License

MIT, see [LICENSE](LICENSE).
