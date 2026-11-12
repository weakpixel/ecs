package system_test

import (
	"slices"
	"testing"

	"github.com/weakpixel/ecs"
	"github.com/weakpixel/ecs/system"
)

type (
	hitEvent  struct{ N int }
	missEvent struct{ N int }
)

// read collects what r yields.
func read[T any](r *system.EventReader[T]) []T { return slices.Collect(r.Read()) }

func TestEventsOf(t *testing.T) {
	w := ecs.NewWorld()
	e := system.EventsOf[hitEvent](w)
	if system.EventsOf[hitEvent](w) != e {
		t.Fatal("EventsOf returned a different queue for the same type")
	}
	if ecs.GetResourceOrNil[system.Events[hitEvent]](w) != e {
		t.Fatal("the queue is not the World's resource")
	}
	e.Send(hitEvent{1})
	e.Send(hitEvent{2})
	if e.Len() != 2 {
		t.Fatalf("Len = %d, want 2", e.Len())
	}
	if n := system.EventsOf[missEvent](w).Len(); n != 0 {
		t.Fatalf("another type's queue has %d events, want 0", n)
	}
	mustPanic(t, "system: nil world", func() { system.EventsOf[hitEvent](nil) })
	mustPanic(t, "system: nil world", func() { system.NewEventReader[hitEvent](nil) })
}

func TestEventReaderLifetime(t *testing.T) {
	w := ecs.NewWorld()
	before := system.NewEventReader[hitEvent](w) // runs before the sender
	after := system.NewEventReader[hitEvent](w)  // runs after it
	var got [][]hitEvent
	var gotAfter [][]hitEvent
	sendAt := 0 // tick that sends one event
	tick := 0

	var s system.Schedule
	s.AddStage("early", func(*system.Context) { got = append(got, read(before)) })
	s.AddStage("send", func(ctx *system.Context) {
		if tick == sendAt {
			system.EventsOf[hitEvent](ctx.World).Send(hitEvent{7})
		}
	})
	s.AddStage("late", func(*system.Context) { gotAfter = append(gotAfter, read(after)) })
	for range 3 {
		s.Run(w, 0)
		tick++
	}

	want := [][]hitEvent{nil, {{7}}, nil}
	if !eqTicks(got, want) {
		t.Errorf("reader before the sender got %v, want %v", got, want)
	}
	wantAfter := [][]hitEvent{{{7}}, nil, nil}
	if !eqTicks(gotAfter, wantAfter) {
		t.Errorf("reader after the sender got %v, want %v", gotAfter, wantAfter)
	}
	if n := system.EventsOf[hitEvent](w).Len(); n != 0 {
		t.Errorf("Len after two ticks = %d, want 0", n)
	}
}

func TestEventReaderDropsAfterTwoTicks(t *testing.T) {
	w := ecs.NewWorld()
	var s system.Schedule
	s.Add(func(*system.Context) {})
	r := system.NewEventReader[hitEvent](w)
	system.EventsOf[hitEvent](w).Send(hitEvent{1})
	s.Run(w, 0)
	s.Run(w, 0)
	system.EventsOf[hitEvent](w).Send(hitEvent{2})
	if got := read(r); !slices.Equal(got, []hitEvent{{2}}) {
		t.Fatalf("a reader that skipped two ticks got %v, want only the new event", got)
	}
}

func TestEventReaderOncePerReader(t *testing.T) {
	w := ecs.NewWorld()
	e := system.EventsOf[hitEvent](w)
	a, b := system.NewEventReader[hitEvent](w), system.NewEventReader[hitEvent](w)
	e.Send(hitEvent{1})
	e.Send(hitEvent{2})
	all := []hitEvent{{1}, {2}}
	if got := read(a); !slices.Equal(got, all) {
		t.Fatalf("reader a got %v, want %v", got, all)
	}
	if got := read(a); len(got) != 0 {
		t.Fatalf("reader a read again: %v, want nothing", got)
	}
	if got := read(b); !slices.Equal(got, all) {
		t.Fatalf("reader b got %v, want %v", got, all)
	}

	// Aging must not make read events new again.
	var s system.Schedule
	s.Add(func(*system.Context) {})
	s.Run(w, 0)
	e.Send(hitEvent{3})
	if got := read(a); !slices.Equal(got, []hitEvent{{3}}) {
		t.Fatalf("after a tick reader a got %v, want [{3}]", got)
	}
}

func TestEventReaderBreakKeepsRest(t *testing.T) {
	w := ecs.NewWorld()
	e := system.EventsOf[hitEvent](w)
	r := system.NewEventReader[hitEvent](w)
	for i := range 3 {
		e.Send(hitEvent{i})
	}
	for ev := range r.Read() {
		if ev.N != 0 {
			t.Fatalf("first event = %v, want {0}", ev)
		}
		break
	}
	if got := read(r); !slices.Equal(got, []hitEvent{{1}, {2}}) {
		t.Fatalf("after break got %v, want [{1} {2}]", got)
	}
}

func TestEventReaderSeesSendsDuringRead(t *testing.T) {
	w := ecs.NewWorld()
	e := system.EventsOf[hitEvent](w)
	r := system.NewEventReader[hitEvent](w)
	e.Send(hitEvent{0})
	var got []hitEvent
	for ev := range r.Read() {
		got = append(got, ev)
		if ev.N < 2 {
			e.Send(hitEvent{ev.N + 1})
		}
	}
	if !slices.Equal(got, []hitEvent{{0}, {1}, {2}}) {
		t.Fatalf("got %v, want [{0} {1} {2}]", got)
	}
}

func TestNewEventReaderSeesQueuedEvents(t *testing.T) {
	w := ecs.NewWorld()
	system.EventsOf[hitEvent](w).Send(hitEvent{1})
	if got := read(system.NewEventReader[hitEvent](w)); !slices.Equal(got, []hitEvent{{1}}) {
		t.Fatalf("a new reader got %v, want the queued [{1}]", got)
	}
}

func TestEventsAllocs(t *testing.T) {
	w := ecs.NewWorld()
	r := system.NewEventReader[hitEvent](w)
	sum := 0
	var s system.Schedule
	s.AddStage("send", func(ctx *system.Context) {
		e := system.EventsOf[hitEvent](ctx.World)
		for i := range 4 {
			e.Send(hitEvent{i})
		}
	})
	s.AddStage("read", func(*system.Context) {
		for ev := range r.Read() {
			sum += ev.N
		}
	})
	s.Run(w, 0) // warm up: resources and buffer capacity
	s.Run(w, 0)

	if n := testing.AllocsPerRun(100, func() { s.Run(w, 0) }); n != 0 {
		t.Fatalf("Run with events allocs = %v, want 0", n)
	}
	if sum == 0 {
		t.Fatal("the reader read nothing")
	}
}

// eqTicks compares per-tick event lists, treating nil and empty alike.
func eqTicks(a, b [][]hitEvent) bool {
	return slices.EqualFunc(a, b, func(x, y []hitEvent) bool { return slices.Equal(x, y) })
}
