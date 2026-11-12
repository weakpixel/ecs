package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	waPos struct{ X, Y int }
	waVel struct{ X, Y int }
)

func TestAllocsGetComp(t *testing.T) {
	posID := ecs.RegisterComp[waPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(waPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		w.GetComp(e, posID)
	}); n != 0 {
		t.Fatalf("GetComp allocs = %v, want 0", n)
	}
}

func TestAllocsGetCompOrNil(t *testing.T) {
	posID := ecs.RegisterComp[waPos]()
	velID := ecs.RegisterComp[waVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(waPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		w.GetCompOrNil(e, posID)
		w.GetCompOrNil(e, velID)
	}); n != 0 {
		t.Fatalf("GetCompOrNil allocs = %v, want 0", n)
	}
}

func TestAllocsHasComp(t *testing.T) {
	posID := ecs.RegisterComp[waPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(waPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		w.HasComp(e, posID)
	}); n != 0 {
		t.Fatalf("HasComp allocs = %v, want 0", n)
	}
}

func TestAllocsIsAlive(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity()

	if n := testing.AllocsPerRun(100, func() {
		w.IsAlive(e)
	}); n != 0 {
		t.Fatalf("IsAlive allocs = %v, want 0", n)
	}
}

func TestAllocsSetCompInPlace(t *testing.T) {
	ecs.RegisterComp[waPos]()
	w := ecs.NewWorld()
	e := w.NewEntity(waPos{X: 1, Y: 2})
	v := any(waPos{X: 9, Y: 9}) // boxed once: a fresh conversion to any can allocate

	w.SetComp(e, v) // warm up
	if n := testing.AllocsPerRun(100, func() {
		w.SetComp(e, v)
	}); n != 0 {
		t.Fatalf("in-place SetComp allocs = %v, want 0", n)
	}
}

func TestAllocsSetCompRemoveCompMoving(t *testing.T) {
	ecs.RegisterComp[waPos]()
	velID := ecs.RegisterComp[waVel]()
	w := ecs.NewWorld()
	e := w.NewEntity(waPos{X: 1, Y: 2})
	v := any(waVel{X: 3, Y: 4})

	// Warm up the add/remove archetype edges before measuring.
	w.SetComp(e, v)
	w.RemoveComp(e, velID)
	w.SetComp(e, v)
	w.RemoveComp(e, velID)

	if n := testing.AllocsPerRun(100, func() {
		w.SetComp(e, v)
		w.RemoveComp(e, velID)
	}); n != 0 {
		t.Fatalf("moving SetComp+RemoveComp allocs = %v, want 0", n)
	}
}

func TestAllocsNewEntityDeleteEntityCycle(t *testing.T) {
	ecs.RegisterComp[waPos]()
	w := ecs.NewWorld()
	v := any(waPos{X: 1, Y: 2})

	for range 10 { // warm up: grow the entity table and archetype capacity
		e := w.NewEntity(v)
		w.DeleteEntity(e)
	}

	if n := testing.AllocsPerRun(100, func() {
		e := w.NewEntity(v)
		w.DeleteEntity(e)
	}); n != 0 {
		t.Fatalf("NewEntity+DeleteEntity cycle allocs = %v, want 0", n)
	}
}

type (
	gaPos struct{ X, Y int }
	gaVel struct{ X, Y int }
)

// Registered up front so w.NewEntity (which, unlike the generic functions
// under test, doesn't auto-register) can use either type regardless of
// test order.
var (
	_ = ecs.RegisterComp[gaPos]()
	_ = ecs.RegisterComp[gaVel]()
)

func TestAllocsGenericGetComp(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		ecs.GetComp[gaPos](w, e)
	}); n != 0 {
		t.Fatalf("GetComp[T] allocs = %v, want 0", n)
	}
}

func TestAllocsGenericGetCompOrNil(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		ecs.GetCompOrNil[gaPos](w, e)
		ecs.GetCompOrNil[gaVel](w, e)
	}); n != 0 {
		t.Fatalf("GetCompOrNil[T] allocs = %v, want 0", n)
	}
}

func TestAllocsGenericHasComp(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		ecs.HasComp[gaPos](w, e)
	}); n != 0 {
		t.Fatalf("HasComp[T] allocs = %v, want 0", n)
	}
}

func TestAllocsGenericSetCompInPlace(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	ecs.SetComp(w, e, gaPos{X: 9, Y: 9}) // warm up
	if n := testing.AllocsPerRun(100, func() {
		ecs.SetComp(w, e, gaPos{X: 9, Y: 9})
	}); n != 0 {
		t.Fatalf("in-place SetComp[T] allocs = %v, want 0", n)
	}
}

func TestAllocsGenericSetCompRemoveCompMoving(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	// Warm up the add/remove archetype edges before measuring.
	ecs.SetComp(w, e, gaVel{X: 3, Y: 4})
	ecs.RemoveComp[gaVel](w, e)
	ecs.SetComp(w, e, gaVel{X: 3, Y: 4})
	ecs.RemoveComp[gaVel](w, e)

	if n := testing.AllocsPerRun(100, func() {
		ecs.SetComp(w, e, gaVel{X: 3, Y: 4})
		ecs.RemoveComp[gaVel](w, e)
	}); n != 0 {
		t.Fatalf("moving SetComp[T]+RemoveComp[T] allocs = %v, want 0", n)
	}
}

func TestAllocsGenericHasCompBonus(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	if n := testing.AllocsPerRun(100, func() {
		ecs.HasComp[gaVel](w, e)
	}); n != 0 {
		t.Fatalf("HasComp[T] (absent) allocs = %v, want 0", n)
	}
}

func TestAllocsGenericRemoveCompBonus(t *testing.T) {
	w := ecs.NewWorld()
	e := w.NewEntity(gaPos{X: 1, Y: 2})

	// Warm up: nothing to add/remove here since there's no archetype move
	// (genVel is never present), but keep the shape consistent with the
	// other alloc tests in this file.
	ecs.RemoveComp[gaVel](w, e) // absent: no-op path
	if n := testing.AllocsPerRun(100, func() {
		ecs.RemoveComp[gaVel](w, e)
	}); n != 0 {
		t.Fatalf("no-op RemoveComp[T] allocs = %v, want 0", n)
	}
}

func TestAllocsNewEntity3(t *testing.T) {
	w := ecs.NewWorld()
	for range 10 { // warm up: grow the entity table
		ecs.NewEntity3(w, genComp1{}, genComp2{}, genComp3{})
	}

	if n := testing.AllocsPerRun(100, func() {
		ecs.NewEntity3(w, genComp1{}, genComp2{}, genComp3{})
	}); n != 0 {
		t.Fatalf("NewEntity3 allocs = %v, want 0", n)
	}
}

func TestAllocsNewEntity7(t *testing.T) {
	w := ecs.NewWorld()
	for range 10 { // warm up: grow the entity table
		ecs.NewEntity7(w, genComp1{}, genComp2{}, genComp3{}, genComp4{}, genComp5{}, genComp6{}, genComp7{})
	}

	if n := testing.AllocsPerRun(100, func() {
		ecs.NewEntity7(w, genComp1{}, genComp2{}, genComp3{}, genComp4{}, genComp5{}, genComp6{}, genComp7{})
	}); n != 0 {
		t.Fatalf("NewEntity7 allocs = %v, want 0", n)
	}
}

func TestAllocsView3NewEntity(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	for range 10 { // warm up: grow the entity table and resolve the spawn archetype
		v.NewEntity(genComp1{}, genComp2{}, genComp3{})
	}

	if n := testing.AllocsPerRun(100, func() {
		v.NewEntity(genComp1{}, genComp2{}, genComp3{})
	}); n != 0 {
		t.Fatalf("View3.NewEntity allocs = %v, want 0", n)
	}
}

func TestAllocsView3Get(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	e := v.NewEntity(genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3})

	if n := testing.AllocsPerRun(100, func() {
		v.Get(e)
	}); n != 0 {
		t.Fatalf("View3.Get allocs = %v, want 0", n)
	}
}

func TestAllocsView3Has(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	e := v.NewEntity(genComp1{}, genComp2{}, genComp3{})

	if n := testing.AllocsPerRun(100, func() {
		v.Has(e)
	}); n != 0 {
		t.Fatalf("View3.Has allocs = %v, want 0", n)
	}
}

func TestAllocsView3SetInPlace(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	e := v.NewEntity(genComp1{}, genComp2{}, genComp3{})

	v.Set(e, genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3}) // warm up
	if n := testing.AllocsPerRun(100, func() {
		v.Set(e, genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3})
	}); n != 0 {
		t.Fatalf("in-place View3.Set allocs = %v, want 0", n)
	}
}

func TestAllocsView3SetMoving(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)

	// Warm up the add/remove archetype edges before measuring.
	for range 2 {
		e := w.NewEntity()
		v.Set(e, genComp1{}, genComp2{}, genComp3{})
		v.Remove(e)
	}

	if n := testing.AllocsPerRun(100, func() {
		e := w.NewEntity()
		v.Set(e, genComp1{}, genComp2{}, genComp3{})
		v.Remove(e)
	}); n != 0 {
		t.Fatalf("moving View3.Set+Remove allocs = %v, want 0", n)
	}
}

func TestAllocsView3Remove(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)

	v.Set(ecs.NewEntity1(w, genComp4{}), genComp1{}, genComp2{}, genComp3{}) // warm up the remove edge
	v.Remove(ecs.NewEntity1(w, genComp4{}))

	if n := testing.AllocsPerRun(100, func() {
		e := ecs.NewEntity1(w, genComp4{})
		v.Set(e, genComp1{}, genComp2{}, genComp3{})
		v.Remove(e)
	}); n != 0 {
		t.Fatalf("View3.Remove allocs = %v, want 0", n)
	}
}

func TestAllocsView3EachNonNested(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	for range 8 {
		v.NewEntity(genComp1{}, genComp2{}, genComp3{})
	}
	for range v.Each() { // warm up the archetype-match cache
	}

	if n := testing.AllocsPerRun(100, func() {
		for e, c := range v.Each() {
			c1, _, _ := c.Unpack()
			c1.V++
			_ = e
		}
	}); n != 0 {
		t.Fatalf("View3.Each (non-nested) allocs = %v, want 0", n)
	}
}

func TestAllocsView3EachNestedAllocatesOnce(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	v.NewEntity(genComp1{}, genComp2{}, genComp3{})
	v.NewEntity(genComp1{}, genComp2{}, genComp3{})
	for range v.Each() { // warm up the archetype-match cache
	}

	// Documented exception (same as Query.Each, verified to cost the same 2
	// allocations there too): a nested iteration of the same view can't
	// reuse the outer call's column buffer, so it allocates its own, and the
	// inner Each() closure itself also escapes to the heap once it's no
	// longer the single top-level range operand. This pins the count so a
	// regression (it growing with entity count, becoming a leak) is caught.
	if n := testing.AllocsPerRun(100, func() {
		for range v.Each() {
			for range v.Each() {
			}
		}
	}); n != 2 {
		t.Fatalf("nested View3.Each allocs = %v, want 2", n)
	}
}

func TestAllocsView3EachFiltered(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w, ecs.Without[genComp4]())
	for range 8 {
		v.NewEntity(genComp1{}, genComp2{}, genComp3{})
		w.NewEntity(genComp1{}, genComp2{}, genComp3{}, genComp4{})
	}
	for range v.Each() { // warm up the archetype-match cache
	}

	if n := testing.AllocsPerRun(100, func() {
		for e, c := range v.Each() {
			c1, _, _ := c.Unpack()
			c1.V++
			_ = e
		}
	}); n != 0 {
		t.Fatalf("filtered View3.Each allocs = %v, want 0", n)
	}
}
