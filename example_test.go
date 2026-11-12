package ecs_test

import (
	"fmt"

	"github.com/weakpixel/ecs"
)

type (
	Position struct{ X, Y int }
	Velocity struct{ X, Y int }
)

func ExampleRegisterComp() {
	id := ecs.RegisterComp[Position]()
	fmt.Println(id == ecs.RegisterComp[Position]()) // idempotent
	// Output: true
}

func ExampleWorld_NewEntity() {
	ecs.RegisterComp[Position]()
	ecs.RegisterComp[Velocity]()

	world := ecs.NewWorld()
	e := world.NewEntity(Position{X: 1, Y: 2}, Velocity{X: 3, Y: 4})
	world.DeleteEntity(e)
	fmt.Println(world.IsAlive(e))
	// Output: false
}

func ExampleWorld_GetComp() {
	world := ecs.NewWorld()
	posID := ecs.RegisterComp[Position]()
	velID := ecs.RegisterComp[Velocity]()

	e := world.NewEntity(Position{X: 1, Y: 2}, Velocity{X: 3, Y: 4})

	pos := world.GetComp(e, posID).(*Position)
	pos.X = 99 // writes to the world's storage

	if v := world.GetCompOrNil(e, velID); v != nil {
		v.(*Velocity).X = 1
	}

	world.SetComp(e, Velocity{X: 10, Y: 20}, Position{X: 4, Y: 5})
	world.RemoveComp(e, velID)

	fmt.Println(world.GetComp(e, posID).(*Position))
	fmt.Println(world.HasComp(e, velID))
	// Output:
	// &{4 5}
	// false
}
