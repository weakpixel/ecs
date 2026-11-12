package system_test

import (
	"fmt"
	"time"

	"github.com/weakpixel/ecs"
	"github.com/weakpixel/ecs/system"
)

// A movement system plus a cleanup stage. The host loop (here a for loop,
// in a game an Update method) owns the timing.
func ExampleSchedule() {
	w := ecs.NewWorld()
	ecs.NewEntity2(w, Position{X: 0}, Velocity{X: 2})
	ecs.NewEntity2(w, Position{X: 9}, Velocity{X: 2})
	view := ecs.NewView2[Position, Velocity](w)

	var s system.Schedule
	s.AddStage("move", func(ctx *system.Context) {
		for _, c := range view.Each() {
			p, v := c.Unpack()
			p.X += v.X * int(ctx.Time.Delta/time.Second)
		}
	})
	s.AddStage("cleanup", func(ctx *system.Context) {
		cmds := ctx.Commands
		for e, c := range view.Each() {
			if p, _ := c.Unpack(); p.X > 10 {
				cmds.Despawn(e) // deferred: safe while iterating
			}
		}
	})

	for range 3 {
		s.Run(w, time.Second)
	}
	n := 0
	for range view.Each() {
		n++
	}
	fmt.Println(n)
	// Output: 1
}
