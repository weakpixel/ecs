package system

import (
	"time"

	"github.com/weakpixel/ecs"
)

// defaultStage is the stage [Schedule.Add] creates when none exists yet.
const defaultStage = "default"

type stage struct {
	name    string
	systems []Runner
}

// Schedule runs systems in a fixed order, grouped into named stages. The
// zero value is an empty schedule ready to use. A Schedule is not safe for
// concurrent use.
type Schedule struct {
	stages []stage
	// world is the World of the first Run. A schedule initializes its
	// runners against one World only, so later Runs must use the same one.
	world *ecs.World
	// pending holds runners that implement Initializer and have not been
	// initialized yet. Detected once in AddRunner, so Run needs no type
	// assertions in steady state.
	pending []Initializer
	// ctx is the one Context passed to every call. Reusing it, instead of
	// building one per call, keeps Run at 0 allocs.
	ctx Context
	// ticks is the number of completed Runs; it becomes ctx.Time.Tick.
	ticks uint64
}

// AddStage appends a stage named name holding sys, and makes it the stage
// that later [Schedule.Add] calls extend. Stages run in the order they were
// added.
//
// Panics with "system: duplicate stage: <name>" if a stage of that name
// exists and with "system: nil system" if any system is nil.
func (s *Schedule) AddStage(name string, sys ...System) {
	for _, st := range s.stages {
		if st.name == name {
			panicf("duplicate stage: " + name)
		}
	}
	s.stages = append(s.stages, stage{name: name})
	s.Add(sys...)
}

// Add appends sys to the most recently added stage, creating a stage named
// "default" if the schedule has none. Systems run in the order added.
//
// Panics with "system: nil system" if any system is nil.
func (s *Schedule) Add(sys ...System) {
	for _, fn := range sys {
		if fn == nil {
			panicf("nil system")
		}
		s.AddRunner(fn)
	}
}

// AddRunner is like [Schedule.Add] for stateful systems implemented as a
// [Runner]. The Runner is called as-is, so it keeps its own state between
// ticks. If it implements [Initializer], its Init runs at the start of the
// next [Schedule.Run].
//
// Panics with "system: nil system" if any runner is nil.
func (s *Schedule) AddRunner(runners ...Runner) {
	for _, r := range runners {
		if r == nil {
			panicf("nil system")
		}
	}
	if len(s.stages) == 0 {
		s.stages = append(s.stages, stage{name: defaultStage})
	}
	last := &s.stages[len(s.stages)-1]
	last.systems = append(last.systems, runners...)
	for _, r := range runners {
		if init, ok := r.(Initializer); ok {
			s.pending = append(s.pending, init)
		}
	}
}

// Run executes one tick: every stage in order, with delta recorded as
// ctx.Time.Delta and the tick index as ctx.Time.Tick. After each stage it
// applies the World's [Commands], so changes queued by one stage are visible
// to the next. Before the first stage it calls Init on runners added since
// the last Run (see [Initializer]), in the order they were added, and
// applies the commands they queued. After the last stage it ages the World's
// [Events] queues, dropping the events of the previous tick. Run makes no
// allocations in steady state.
//
// Panics with "system: nil world" if w is nil, and with
// "system: schedule used with a different world" if w is not the World of
// the schedule's first Run.
func (s *Schedule) Run(w *ecs.World, delta time.Duration) {
	cmds := CommandsOf(w)
	switch s.world {
	case nil:
		s.world = w
	case w:
	default:
		panicf("schedule used with a different world")
	}
	s.ctx = Context{World: w, Time: Time{Delta: delta, Tick: s.ticks}, Commands: cmds}
	s.ticks++

	s.initPending()
	for i := range s.stages {
		for _, r := range s.stages[i].systems {
			r.Run(&s.ctx)
		}
		cmds.Apply(w)
	}
	updateEvents(w)
}

// initPending runs Init on runners added since the last Run, then applies
// the commands they queued so the first stage sees the result.
func (s *Schedule) initPending() {
	if len(s.pending) == 0 {
		return
	}
	// Index loop: an Init may add runners, which are then initialized too.
	for i := 0; i < len(s.pending); i++ {
		s.pending[i].Init(&s.ctx)
	}
	clear(s.pending) // drop references, keep capacity
	s.pending = s.pending[:0]
	s.ctx.Commands.Apply(s.ctx.World)
}
