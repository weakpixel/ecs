package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	wePos struct{ X, Y int }
	weVel struct{ X, Y int }
)

type weUnregistered struct{ N int }

func TestEntityZeroNeverAlive(t *testing.T) {
	w := ecs.NewWorld()
	if w.IsAlive(ecs.Entity{}) {
		t.Fatalf("IsAlive(Entity{}) = true, want false")
	}
}

func TestDeleteEntityStaleHandlePanics(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity()
	w.DeleteEntity(e)
	if w.IsAlive(e) {
		t.Fatalf("IsAlive after delete = true, want false")
	}
	mustPanic(t, "ecs: invalid or stale entity", func() { w.DeleteEntity(e) })
}

func TestNewEntityPanicsOnPointerValue(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component must be a non-pointer struct", func() {
		w.NewEntity(&wePos{})
	})
}

func TestNewEntityPanicsOnNilValue(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component must be a non-pointer struct: <nil>", func() {
		w.NewEntity(nil)
	})
}

func TestNewEntityPanicsOnNonStructValue(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component must be a non-pointer struct: int", func() {
		w.NewEntity(42)
	})
}

func TestNewEntityPanicsOnUnregisteredType(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component not registered", func() {
		w.NewEntity(weUnregistered{})
	})
}

func TestNewEntityPanicsOnDuplicateType(t *testing.T) {
	ecs.RegisterComp[wePos]()
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() {
		w.NewEntity(wePos{X: 1}, wePos{X: 2})
	})
}

func TestNewEntityMultipleComponents(t *testing.T) {
	ecs.RegisterComp[wePos]()
	ecs.RegisterComp[weVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(wePos{X: 1, Y: 2}, weVel{X: 3, Y: 4})
	if !w.IsAlive(e) {
		t.Fatalf("new entity not alive")
	}
}
