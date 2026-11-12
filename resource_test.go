package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	rsTime   struct{ Delta float64 }
	rsCamera struct{ X, Y int }
)

func TestResourceSetGetRoundTrip(t *testing.T) {
	w := ecs.NewWorld()
	ecs.SetResource(w, &rsTime{Delta: 0.5})

	got := ecs.GetResource[rsTime](w)
	if got.Delta != 0.5 {
		t.Fatalf("GetResource = %+v, want Delta=0.5", got)
	}
}

func TestResourceGetPanicsWhenUnset(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: resource not set: ecs_test.rsTime", func() { ecs.GetResource[rsTime](w) })
}

func TestResourceGetOrNilAndHasWhenUnset(t *testing.T) {
	w := ecs.NewWorld()
	if v := ecs.GetResourceOrNil[rsTime](w); v != nil {
		t.Fatalf("GetResourceOrNil(unset) = %v, want nil", v)
	}
	if ecs.HasResource[rsTime](w) {
		t.Fatalf("HasResource(unset) = true")
	}
}

func TestResourceHasAndGetOrNilWhenSet(t *testing.T) {
	w := ecs.NewWorld()
	ecs.SetResource(w, &rsTime{Delta: 1})

	if !ecs.HasResource[rsTime](w) {
		t.Fatalf("HasResource(set) = false")
	}
	if v := ecs.GetResourceOrNil[rsTime](w); v == nil {
		t.Fatalf("GetResourceOrNil(set) = nil")
	}
}

func TestResourceRemove(t *testing.T) {
	w := ecs.NewWorld()
	ecs.SetResource(w, &rsTime{Delta: 1})
	ecs.RemoveResource[rsTime](w)

	if ecs.HasResource[rsTime](w) {
		t.Fatalf("HasResource after Remove = true")
	}
	mustPanic(t, "ecs: resource not set", func() { ecs.GetResource[rsTime](w) })
}

func TestResourceRemoveAbsentIsNoop(t *testing.T) {
	w := ecs.NewWorld()
	mustNotPanic(t, func() { ecs.RemoveResource[rsTime](w) })
}

func TestResourceSetReplacesEarlierValue(t *testing.T) {
	w := ecs.NewWorld()
	ecs.SetResource(w, &rsTime{Delta: 1})
	ecs.SetResource(w, &rsTime{Delta: 2})

	if got := ecs.GetResource[rsTime](w); got.Delta != 2 {
		t.Fatalf("GetResource after replace = %+v, want Delta=2", got)
	}
}

func TestResourceMultipleTypesCoexist(t *testing.T) {
	w := ecs.NewWorld()
	ecs.SetResource(w, &rsTime{Delta: 1})
	ecs.SetResource(w, &rsCamera{X: 1, Y: 2})

	if got := ecs.GetResource[rsTime](w); got.Delta != 1 {
		t.Fatalf("rsTime = %+v, want Delta=1", got)
	}
	if got := ecs.GetResource[rsCamera](w); got.X != 1 || got.Y != 2 {
		t.Fatalf("rsCamera = %+v, want {1 2}", got)
	}
}
