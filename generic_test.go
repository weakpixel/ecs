package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	gcPos struct{ X, Y int }
	gcVel struct{ X, Y int }
)

// Registered up front so every test's low-level w.NewEntity call (which,
// unlike the generic functions under test, doesn't auto-register) can use
// either type regardless of test order.
var (
	_ = ecs.RegisterComp[gcPos]()
	_ = ecs.RegisterComp[gcVel]()
)

func TestGenericGetCompWritesThrough(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{X: 1, Y: 2})

	p := ecs.GetComp[gcPos](w, e)
	p.X = 99

	if got := ecs.GetComp[gcPos](w, e); got.X != 99 {
		t.Fatalf("GetComp after write-through = %+v, want X=99", got)
	}
}

func TestGenericGetCompPanicsOnMissingComponent(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	mustPanic(t, "ecs: entity does not have component", func() { ecs.GetComp[gcVel](w, e) })
}

func TestGenericGetCompPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { ecs.GetComp[gcPos](w, e) })
}

func TestGenericGetCompOrNilNilWhenAbsent(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})

	if v := ecs.GetCompOrNil[gcVel](w, e); v != nil {
		t.Fatalf("GetCompOrNil(absent) = %v, want nil", v)
	}
	if v := ecs.GetCompOrNil[gcPos](w, e); v == nil {
		t.Fatalf("GetCompOrNil(present) = nil, want non-nil")
	}
}

func TestGenericGetCompOrNilPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { ecs.GetCompOrNil[gcPos](w, e) })
}

func TestGenericHasComp(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})

	if !ecs.HasComp[gcPos](w, e) {
		t.Fatalf("HasComp(present) = false")
	}
	if ecs.HasComp[gcVel](w, e) {
		t.Fatalf("HasComp(absent) = true")
	}
}

func TestGenericHasCompPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { ecs.HasComp[gcPos](w, e) })
}

func TestGenericSetCompInPlace(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{X: 1, Y: 2})

	ecs.SetComp(w, e, gcPos{X: 9, Y: 9})
	if got := ecs.GetComp[gcPos](w, e); got.X != 9 || got.Y != 9 {
		t.Fatalf("pos after in-place SetComp = %+v, want {9 9}", got)
	}
}

func TestGenericSetCompMoving(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{X: 1, Y: 2})

	ecs.SetComp(w, e, gcVel{X: 3, Y: 4})
	if got := ecs.GetComp[gcPos](w, e); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after adding vel: %+v", got)
	}
	if got := ecs.GetComp[gcVel](w, e); got.X != 3 || got.Y != 4 {
		t.Fatalf("vel = %+v, want {3 4}", got)
	}
}

func TestGenericSetCompPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { ecs.SetComp(w, e, gcVel{}) })
}

func TestGenericRemoveComp(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{X: 1, Y: 2}, gcVel{X: 3, Y: 4})

	ecs.RemoveComp[gcVel](w, e)
	if ecs.HasComp[gcVel](w, e) {
		t.Fatalf("vel still present after RemoveComp")
	}
	if got := ecs.GetComp[gcPos](w, e); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after removing vel: %+v", got)
	}
}

func TestGenericRemoveCompNoopWhenAbsent(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{X: 1, Y: 2})

	ecs.RemoveComp[gcVel](w, e) // absent: no-op
	if got := ecs.GetComp[gcPos](w, e); got.X != 1 || got.Y != 2 {
		t.Fatalf("pos changed after no-op RemoveComp: %+v", got)
	}
}

func TestGenericRemoveCompPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gcPos{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { ecs.RemoveComp[gcPos](w, e) })
}
