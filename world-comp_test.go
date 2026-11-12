package ecs_test

import (
	"runtime"
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	wcPos struct{ X, Y int }
	wcVel struct{ X, Y int }
	wcBig struct {
		S string
		P *int
	}
)

func TestGetCompWritesThrough(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{X: 1, Y: 2})

	p := w.GetComp(e, posID).(*wcPos)
	p.X = 99

	if got := w.GetComp(e, posID).(*wcPos); got.X != 99 {
		t.Fatalf("GetComp after write-through = %+v, want X=99", got)
	}
}

func TestGetCompOrNilUntypedNilWhenAbsent(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{})

	if v := w.GetCompOrNil(e, velID); v != nil {
		t.Fatalf("GetCompOrNil(absent) = %#v, want nil", v)
	}
	if v := w.GetCompOrNil(e, posID); v == nil {
		t.Fatalf("GetCompOrNil(present) = nil, want non-nil")
	}
}

func TestHasComp(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{})

	if !w.HasComp(e, posID) {
		t.Fatalf("HasComp(present) = false")
	}
	if w.HasComp(e, velID) {
		t.Fatalf("HasComp(absent) = true")
	}
}

func TestGetCompPanicsOnMissingComponent(t *testing.T) {
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity()
	mustPanic(t, "ecs: entity does not have component", func() { w.GetComp(e, velID) })
}

func TestGetCompPanicsOnUnknownID(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity()
	mustPanic(t, "ecs: component not registered: ComponentID(", func() { w.GetComp(e, ecs.ComponentID(60000)) })
}

func TestHasCompPanicsOnUnknownID(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity()
	mustPanic(t, "ecs: component not registered: ComponentID(", func() { w.HasComp(e, ecs.ComponentID(60001)) })
}

func TestRemoveCompPanicsOnUnknownID(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity()
	mustPanic(t, "ecs: component not registered: ComponentID(", func() { w.RemoveComp(e, ecs.ComponentID(60002)) })
}

func TestGetCompPanicsOnStaleEntity(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { w.GetComp(e, posID) })
}

func TestSetCompPanicsOnDuplicateType(t *testing.T) {
	ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity()
	mustPanic(t, "ecs: duplicate component", func() { w.SetComp(e, wcVel{X: 1}, wcVel{X: 2}) })
}

func TestSetCompValidationFailureLeavesEntityUnchanged(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{X: 1, Y: 2})

	values := []any{wcVel{X: 1}, wcVel{X: 2}} // duplicate type
	snapshot := append([]any{}, values...)

	mustPanic(t, "ecs: duplicate component", func() { w.SetComp(e, values...) })

	for i, v := range values {
		if v != snapshot[i] {
			t.Fatalf("values[%d] changed from %#v to %#v", i, snapshot[i], v)
		}
	}
	if got := w.GetComp(e, posID).(*wcPos); got.X != 1 || got.Y != 2 {
		t.Fatalf("entity changed after failed SetComp: %+v", got)
	}
	if w.HasComp(e, velID) {
		t.Fatalf("entity gained a component after failed SetComp")
	}
}

func TestSetCompAndRemoveCompPreserveOtherComponents(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{X: 1, Y: 2}, wcVel{X: 3, Y: 4})

	w.RemoveComp(e, velID)
	if got := w.GetComp(e, posID).(*wcPos); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after removing vel: %+v", got)
	}
	if w.HasComp(e, velID) {
		t.Fatalf("vel still present after RemoveComp")
	}

	w.SetComp(e, wcVel{X: 5, Y: 6})
	if got := w.GetComp(e, posID).(*wcPos); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after re-adding vel: %+v", got)
	}
	if got := w.GetComp(e, velID).(*wcVel); got.X != 5 || got.Y != 6 {
		t.Fatalf("vel = %+v, want {5 6}", got)
	}
}

func TestRemoveCompNoopsAreIgnored(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{X: 1, Y: 2})

	w.RemoveComp(e)        // no ids
	w.RemoveComp(e, velID) // absent id
	if got := w.GetComp(e, posID).(*wcPos); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after no-op RemoveComp: %+v", got)
	}
}

func TestRemoveCompDuplicateIDsTolerated(t *testing.T) {
	posID := ecs.RegisterComp[wcPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(wcPos{X: 1, Y: 2})

	w.RemoveComp(e, posID, posID) // duplicate id, must not panic
	if w.HasComp(e, posID) {
		t.Fatalf("pos still present after RemoveComp with duplicate ids")
	}
}

func TestComponentsWithStringsAndPointersSurviveMovesAndGC(t *testing.T) {
	bigID := ecs.RegisterComp[wcBig]()
	velID := ecs.RegisterComp[wcVel]()
	w := ecs.NewWorld()

	n := 42
	e := w.NewEntity(wcBig{S: "hello world", P: &n})
	w.SetComp(e, wcVel{X: 1, Y: 2}) // move: add vel
	w.RemoveComp(e, velID)          // move: remove vel

	runtime.GC()

	got := w.GetComp(e, bigID).(*wcBig)
	if got.S != "hello world" || got.P == nil || *got.P != 42 {
		t.Fatalf("wcBig after moves+GC = %+v, want {hello world, *42}", got)
	}
}
