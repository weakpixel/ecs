package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	wqPos struct{ X, Y int }
	wqVel struct{ X, Y int }
	wqTag struct{}
)

// mustNotPanic fails the test if f panics, reporting the recovered value.
func mustNotPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	f()
}

func TestQueryMatchesAndOutputOrder(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	velID := ecs.RegisterComp[wqVel]()
	ecs.RegisterComp[wqTag]()
	w := ecs.NewWorld()

	e1 := w.NewEntity(wqPos{X: 1, Y: 1}, wqVel{X: 10, Y: 10})
	e2 := w.NewEntity(wqPos{X: 2, Y: 2}, wqVel{X: 20, Y: 20}, wqTag{}) // extra component, still matches
	e3 := w.NewEntity(wqPos{X: 3, Y: 3})                               // missing Vel, must not match

	q := w.Query(velID, posID) // output order: vel, pos

	seen := map[ecs.Entity][2]int{}
	for e, comps := range q.Each() {
		vel := comps[0].(*wqVel)
		pos := comps[1].(*wqPos)
		seen[e] = [2]int{vel.X, pos.X}
	}

	if len(seen) != 2 {
		t.Fatalf("matched %d entities, want 2: %v", len(seen), seen)
	}
	if got := seen[e1]; got != [2]int{10, 1} {
		t.Fatalf("e1 = %v, want {10 1}", got)
	}
	if got := seen[e2]; got != [2]int{20, 2} {
		t.Fatalf("e2 = %v, want {20 2}", got)
	}
	if _, ok := seen[e3]; ok {
		t.Fatalf("e3 matched a query requiring Vel, but has no Vel")
	}
}

func TestQueryNoIDsMatchesEveryArchetype(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()

	empty := w.NewEntity()
	withPos := w.NewEntity(wqPos{X: 1})
	_ = posID

	q := w.Query()
	seen := map[ecs.Entity]bool{}
	for e, comps := range q.Each() {
		if len(comps) != 0 {
			t.Fatalf("comps = %v, want empty for a no-ID query", comps)
		}
		seen[e] = true
	}
	if !seen[empty] || !seen[withPos] {
		t.Fatalf("Query() with no ids didn't match every archetype: %v", seen)
	}
}

func TestQueryPanicsOnDuplicateID(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() { w.Query(posID, posID) })
}

func TestQueryPanicsOnUnknownID(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component not registered: ComponentID(", func() { w.Query(ecs.ComponentID(60010)) })
}

func TestZeroQueryPanics(t *testing.T) {
	var q ecs.Query
	mustPanic(t, "ecs: not bound to a world", func() {
		for range q.Each() {
		}
	})
}

func TestQuerySeesArchetypesCreatedAfterConstruction(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()

	q := w.Query(posID) // no matching archetype exists yet
	e := w.NewEntity(wqPos{X: 7})

	count := 0
	for got := range q.Each() {
		if got != e {
			t.Fatalf("got entity %v, want %v", got, e)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("matched %d entities, want 1", count)
	}
}

func TestQueryReuseAcrossSequentialEachCalls(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()
	w.NewEntity(wqPos{X: 1})

	q := w.Query(posID)

	count := 0
	for range q.Each() {
		count++
	}
	if count != 1 {
		t.Fatalf("first Each() matched %d, want 1", count)
	}

	w.NewEntity(wqPos{X: 2}) // new archetype instance, same shape: just another row

	count = 0
	for range q.Each() {
		count++
	}
	if count != 2 {
		t.Fatalf("second Each() matched %d, want 2", count)
	}
}

func TestQueryStructuralMutationDuringIterationPanics(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()
	w.NewEntity(wqPos{X: 1})

	q := w.Query(posID)
	mustPanic(t, "ecs: query invalidated by world mutation", func() {
		for range q.Each() {
			w.NewEntity(wqPos{X: 2}) // structural change; loop body doesn't break
		}
	})
}

func TestQueryBreakRightAfterMutationDoesNotPanic(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()
	w.NewEntity(wqPos{X: 1})
	w.NewEntity(wqPos{X: 2})

	q := w.Query(posID)
	mustNotPanic(t, func() {
		for range q.Each() {
			w.NewEntity(wqPos{X: 3}) // structural change, but...
			break                    // ...break means no revision check
		}
	})
}

func TestQueryInPlaceSetCompDuringEachAllowed(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	velID := ecs.RegisterComp[wqVel]()
	w := ecs.NewWorld()
	w.NewEntity(wqPos{X: 1}, wqVel{X: 1})
	w.NewEntity(wqPos{X: 2}, wqVel{X: 2})

	q := w.Query(posID, velID)
	mustNotPanic(t, func() {
		for e, comps := range q.Each() {
			pos := comps[0].(*wqPos)
			pos.X *= 10
			w.SetComp(e, wqVel{X: 99}) // already present: in place, no revision bump
			w.RemoveComp(e)            // no ids: no-op
		}
	})

	got := map[int]bool{}
	for _, comps := range collectAll(q) {
		got[comps[0].(*wqPos).X] = true
	}
	if !got[10] || !got[20] || len(got) != 2 {
		t.Fatalf("positions after in-place mutation during Each = %v, want {10, 20}", got)
	}
}

// collectAll drains q.Each() into a slice, for assertions after the loop.
// comps is reused between iterations, so each row is copied out (the
// documented way to retain results past the loop body).
func collectAll(q ecs.Query) [][]any {
	var out [][]any
	for _, comps := range q.Each() {
		out = append(out, append([]any(nil), comps...))
	}
	return out
}

func TestQueryNestedIterationNotAliased(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	w := ecs.NewWorld()
	a := w.NewEntity(wqPos{X: 1})
	b := w.NewEntity(wqPos{X: 2})
	_ = a
	_ = b

	q := w.Query(posID)

	outerSeen := map[ecs.Entity]int{}
	for oe, ocomps := range q.Each() {
		opos := ocomps[0].(*wqPos)
		wantX := opos.X

		innerCount := 0
		for _, icomps := range q.Each() {
			_ = icomps[0].(*wqPos).X // touch the inner buffer
			innerCount++
		}
		if innerCount != 2 {
			t.Fatalf("inner Each() matched %d, want 2", innerCount)
		}

		// The outer pointer/slice must not have been clobbered by the inner loop.
		if opos.X != wantX {
			t.Fatalf("outer pos for %v changed from %d to %d after nested Each", oe, wantX, opos.X)
		}
		outerSeen[oe] = opos.X
	}
	if len(outerSeen) != 2 {
		t.Fatalf("outer Each() matched %d, want 2: %v", len(outerSeen), outerSeen)
	}
}

func TestAllocsQueryEach(t *testing.T) {
	posID := ecs.RegisterComp[wqPos]()
	velID := ecs.RegisterComp[wqVel]()
	w := ecs.NewWorld()
	for range 8 {
		w.NewEntity(wqPos{}, wqVel{})
	}

	q := w.Query(posID, velID)
	for range q.Each() { // warm up the archetype-match cache
	}

	if n := testing.AllocsPerRun(100, func() {
		for e, comps := range q.Each() {
			pos := comps[0].(*wqPos)
			vel := comps[1].(*wqVel)
			pos.X += vel.X
			_ = e
		}
	}); n != 0 {
		t.Fatalf("Query.Each allocs = %v, want 0", n)
	}
}
