package system_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/weakpixel/ecs"
	"github.com/weakpixel/ecs/system"
)

type (
	Position struct{ X, Y int }
	Velocity struct{ X, Y int }
)

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Fatalf("panic %v does not contain %q", r, want)
		}
	}()
	f()
}

func TestScheduleOrder(t *testing.T) {
	var got []string
	rec := func(name string) system.System {
		return func(*system.Context) { got = append(got, name) }
	}

	var s system.Schedule
	s.Add(rec("a"), rec("b")) // implicit default stage
	s.AddStage("late", rec("c"))
	s.Add(rec("d"))
	s.Run(ecs.NewWorld(), 0)

	if want := []string{"a", "b", "c", "d"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

type counter struct{ runs int }

func (c *counter) Run(*system.Context) { c.runs++ }

func TestScheduleRunner(t *testing.T) {
	c := &counter{}
	var got []string
	var s system.Schedule
	s.Add(func(*system.Context) { got = append(got, "fn") })
	s.AddRunner(c)
	s.AddStage("next")
	s.AddRunner(c)

	w := ecs.NewWorld()
	s.Run(w, 0)
	s.Run(w, 0)
	if c.runs != 4 || len(got) != 2 {
		t.Fatalf("runs = %d, fn calls = %d, want 4 and 2", c.runs, len(got))
	}
}

type initRunner struct {
	name string
	log  *[]string
	init func(*system.Context)
}

func (r *initRunner) Init(ctx *system.Context) {
	*r.log = append(*r.log, "init "+r.name)
	if r.init != nil {
		r.init(ctx)
	}
}
func (r *initRunner) Run(*system.Context) { *r.log = append(*r.log, "run "+r.name) }

func TestScheduleInit(t *testing.T) {
	var log []string
	a := &initRunner{name: "a", log: &log}
	b := &initRunner{name: "b", log: &log}

	var s system.Schedule
	s.AddRunner(a)
	s.Add(func(*system.Context) { log = append(log, "fn") })
	s.AddRunner(b)

	w := ecs.NewWorld()
	s.Run(w, 0)
	s.Run(w, 0)

	want := []string{
		"init a", "init b", // before any stage, in the order added
		"run a", "fn", "run b",
		"run a", "fn", "run b", // no second init
	}
	if !slices.Equal(log, want) {
		t.Fatalf("log = %v, want %v", log, want)
	}

	// A runner added after the first Run is initialized on the next Run.
	log = nil
	c := &initRunner{name: "c", log: &log}
	s.AddRunner(c)
	s.Run(w, 0)
	if want := []string{"init c", "run a", "fn", "run b", "run c"}; !slices.Equal(log, want) {
		t.Fatalf("late log = %v, want %v", log, want)
	}
}

func TestScheduleInitCommandsApplyBeforeFirstStage(t *testing.T) {
	w := ecs.NewWorld()
	pos := ecs.NewView1[Position](w)
	var log []string
	var seen int

	var s system.Schedule
	s.AddRunner(&initRunner{name: "a", log: &log, init: func(ctx *system.Context) {
		ctx.Commands.Spawn(Position{1, 1})
	}})
	s.Add(func(*system.Context) {
		for range pos.Each() {
			seen++
		}
	})
	s.Run(w, 0)
	if seen != 1 {
		t.Fatalf("first stage saw %d entities, want 1", seen)
	}
}

func TestScheduleSingleWorld(t *testing.T) {
	var s system.Schedule
	s.Add(func(*system.Context) {})
	w := ecs.NewWorld()
	s.Run(w, 0)
	s.Run(w, 0)
	mustPanic(t, "system: schedule used with a different world", func() { s.Run(ecs.NewWorld(), 0) })
}

func TestContext(t *testing.T) {
	w := ecs.NewWorld()
	var times []system.Time
	var s system.Schedule
	s.AddRunner(&initRunner{name: "i", log: new([]string), init: func(ctx *system.Context) {
		times = append(times, ctx.Time) // Init sees the first tick
	}})
	s.AddStage("a", func(ctx *system.Context) {
		if ctx.World != w {
			t.Error("ctx.World is not the run World")
		}
		if ctx.Commands != system.CommandsOf(w) {
			t.Error("ctx.Commands is not the CommandsOf buffer")
		}
		times = append(times, ctx.Time)
	})
	s.AddStage("b", func(ctx *system.Context) { times = append(times, ctx.Time) })

	s.Run(w, 16*time.Millisecond)
	s.Run(w, 20*time.Millisecond)

	a, b := system.Time{Delta: 16 * time.Millisecond}, system.Time{Delta: 20 * time.Millisecond, Tick: 1}
	if want := []system.Time{a, a, a, b, b}; !slices.Equal(times, want) {
		t.Fatalf("times = %v, want %v", times, want)
	}
}

func TestScheduleCommandsFlushBetweenStages(t *testing.T) {
	w := ecs.NewWorld()
	pos := ecs.NewView1[Position](w)

	count := func() (n int) {
		for range pos.Each() {
			n++
		}
		return n
	}
	var seen []int

	var s system.Schedule
	s.AddStage("spawn", func(ctx *system.Context) {
		c := ctx.Commands
		c.Spawn(Position{1, 1})
		c.Spawn(Position{2, 2})
		seen = append(seen, count()) // not applied yet
	})
	s.AddStage("observe", func(*system.Context) { seen = append(seen, count()) })
	s.Run(w, 0)

	if want := []int{0, 2}; !slices.Equal(seen, want) {
		t.Fatalf("counts = %v, want %v", seen, want)
	}
}

func TestDespawnWhileIterating(t *testing.T) {
	w := ecs.NewWorld()
	for i := range 4 {
		ecs.NewEntity1(w, Position{X: i})
	}
	pos := ecs.NewView1[Position](w)

	var s system.Schedule
	s.Add(func(ctx *system.Context) {
		c := ctx.Commands
		for e, p := range pos.Each() {
			if p.X%2 == 0 {
				c.Despawn(e)
				c.Despawn(e) // twice is harmless
			}
		}
	})
	s.Run(w, 0)

	var left []int
	for _, p := range pos.Each() {
		left = append(left, p.X)
	}
	slices.Sort(left)
	if want := []int{1, 3}; !slices.Equal(left, want) {
		t.Fatalf("left = %v, want %v", left, want)
	}
}

func TestCommandsDoQueuedDuringApply(t *testing.T) {
	w := ecs.NewWorld()
	c := system.CommandsOf(w)
	var got []int
	c.Do(func(*ecs.World) {
		got = append(got, 1)
		c.Do(func(*ecs.World) { got = append(got, 2) })
	})
	c.Apply(w)
	if !slices.Equal(got, []int{1, 2}) || c.Len() != 0 {
		t.Fatalf("got %v, len %d", got, c.Len())
	}
}

func TestCommandsOfIsStable(t *testing.T) {
	w := ecs.NewWorld()
	if system.CommandsOf(w) != system.CommandsOf(w) {
		t.Fatal("CommandsOf returned different buffers for one world")
	}
}

func TestPanics(t *testing.T) {
	tests := []struct {
		name string
		want string
		f    func()
	}{
		{"nil world run", "system: nil world", func() { new(system.Schedule).Run(nil, 0) }},
		{"nil world commands", "system: nil world", func() { system.CommandsOf(nil) }},
		{"nil system", "system: nil system", func() { new(system.Schedule).Add(nil) }},
		{"nil runner", "system: nil system", func() { new(system.Schedule).AddRunner(nil) }},
		{"nil system in stage", "system: nil system", func() { new(system.Schedule).AddStage("x", nil) }},
		{"duplicate stage", "system: duplicate stage: x", func() {
			var s system.Schedule
			s.AddStage("x")
			s.AddStage("x")
		}},
		{"nil command", "system: nil command", func() { system.CommandsOf(ecs.NewWorld()).Do(nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { mustPanic(t, tt.want, tt.f) })
	}
}

func TestRunAllocs(t *testing.T) {
	w := ecs.NewWorld()
	for i := range 8 {
		ecs.NewEntity2(w, Position{X: i}, Velocity{X: 1})
	}
	view := ecs.NewView2[Position, Velocity](w)
	var dead []ecs.Entity
	noop := func(*ecs.World) {} // Do callbacks take the World

	var s system.Schedule
	s.AddStage("move", func(*system.Context) {
		for _, c := range view.Each() {
			p, v := c.Unpack()
			p.X += v.X
		}
	})
	s.AddStage("cleanup", func(ctx *system.Context) {
		c := ctx.Commands
		for _, e := range dead {
			c.Despawn(e)
		}
		c.Do(noop)
	})
	s.AddRunner(&initRunner{log: new([]string)}) // Init must not cost anything after the first Run
	s.Run(w, 0)                                  // warm up: init, resource, caches, queue capacity

	if n := testing.AllocsPerRun(100, func() { s.Run(w, 0) }); n != 0 {
		t.Fatalf("Run allocs = %v, want 0", n)
	}
}
