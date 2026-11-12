package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

// --- N = 1 ---

func TestView1Constructor(t *testing.T) {
	w := ecs.NewWorld()
	ecs.NewView1[genComp1](w) // must not panic
}

func TestZeroView1Panics(t *testing.T) {
	var v ecs.View1[genComp1]
	w := ecs.NewWorld()
	e := w.NewEntity()

	mustPanic(t, "ecs: not bound to a world", func() { v.NewEntity(genComp1{}) })
	mustPanic(t, "ecs: not bound to a world", func() { v.Get(e) })
	mustPanic(t, "ecs: not bound to a world", func() { v.Has(e) })
	mustPanic(t, "ecs: not bound to a world", func() { v.Set(e, genComp1{}) })
	mustPanic(t, "ecs: not bound to a world", func() { v.Remove(e) })
	mustPanic(t, "ecs: not bound to a world", func() {
		for range v.Each() {
		}
	})
}

func TestView1NewEntity(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := v.NewEntity(genComp1{V: 7})

	if got := v.Get(e); got.V != 7 {
		t.Fatalf("Get after View1.NewEntity = %+v, want V=7", got)
	}
}

func TestView1GetPanicsOnMissingComponent(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := ecs.NewEntity1(w, genComp2{V: 1})
	mustPanic(t, "ecs: entity does not have component", func() { v.Get(e) })
}

func TestView1GetPanicsOnStaleEntity(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := v.NewEntity(genComp1{})
	w.DeleteEntity(e)
	mustPanic(t, "ecs: invalid or stale entity", func() { v.Get(e) })
}

func TestView1Has(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)

	withExtra := ecs.NewEntity2(w, genComp1{}, genComp2{})
	without := ecs.NewEntity1(w, genComp2{V: 1})

	if !v.Has(withExtra) {
		t.Fatalf("Has(entity with extra component) = false, want true")
	}
	if v.Has(without) {
		t.Fatalf("Has(entity lacking the component) = true, want false")
	}
}

func TestView1SetInPlace(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := v.NewEntity(genComp1{V: 1})

	before := ecs.GetComp[genComp1](w, e)
	v.Set(e, genComp1{V: 2})
	after := ecs.GetComp[genComp1](w, e)

	if before != after {
		t.Fatalf("in-place Set moved the component: before=%p after=%p", before, after)
	}
	if after.V != 2 {
		t.Fatalf("value after in-place Set = %+v, want V=2", after)
	}
}

func TestView1SetMoving(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := w.NewEntity()

	v.Set(e, genComp1{V: 5})
	if got := v.Get(e); got.V != 5 {
		t.Fatalf("Get after moving Set = %+v, want V=5", got)
	}
}

func TestView1Remove(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := v.NewEntity(genComp1{})

	v.Remove(e)
	if v.Has(e) {
		t.Fatalf("Has = true after Remove")
	}
}

func TestView1RemoveNoopWhenAbsent(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := ecs.NewEntity1(w, genComp2{V: 1})

	v.Remove(e) // genComp1 absent: no-op
	if got := ecs.GetComp[genComp2](w, e); got.V != 1 {
		t.Fatalf("genComp2 changed after no-op Remove: %+v", got)
	}
}

func TestView1EachBasic(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	a := v.NewEntity(genComp1{V: 1})
	b := v.NewEntity(genComp1{V: 2})

	seen := map[ecs.Entity]int{}
	for e, p := range v.Each() {
		seen[e] = p.V
	}
	if len(seen) != 2 || seen[a] != 1 || seen[b] != 2 {
		t.Fatalf("Each = %v, want {a:1, b:2}", seen)
	}
}

func TestView1EachSeesArchetypesCreatedAfterConstruction(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w) // no matching archetype exists yet
	e := v.NewEntity(genComp1{V: 9})

	count := 0
	for got := range v.Each() {
		if got != e {
			t.Fatalf("got entity %v, want %v", got, e)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("Each matched %d entities, want 1", count)
	}
}

func TestView1EachStructuralMutationDuringIterationPanics(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	v.NewEntity(genComp1{V: 1})

	mustPanic(t, "ecs: query invalidated by world mutation", func() {
		for range v.Each() {
			v.NewEntity(genComp1{V: 2}) // structural change; loop body doesn't break
		}
	})
}

func TestView1EachBreakRightAfterMutationDoesNotPanic(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	v.NewEntity(genComp1{V: 1})
	v.NewEntity(genComp1{V: 2})

	mustNotPanic(t, func() {
		for range v.Each() {
			v.NewEntity(genComp1{V: 3}) // structural change, but...
			break                       // ...break means no revision check
		}
	})
}

// TestView1EachNestedIsCorrectAndIsolated mirrors
// TestQueryNestedIterationNotAliased: a nested iteration of the same View1
// gets its own column buffer (the outer call's st.cols is still in use), so
// it can't corrupt the outer loop's in-progress pointer.
func TestView1EachNestedIsCorrectAndIsolated(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	a := v.NewEntity(genComp1{V: 1})
	b := v.NewEntity(genComp1{V: 2})

	outerSeen := map[ecs.Entity]int{}
	for oe, op := range v.Each() {
		wantV := op.V
		innerCount := 0
		for range v.Each() {
			innerCount++
		}
		if innerCount != 2 {
			t.Fatalf("inner Each matched %d, want 2", innerCount)
		}
		if op.V != wantV {
			t.Fatalf("outer pointer clobbered by nested Each: got %d, want %d", op.V, wantV)
		}
		outerSeen[oe] = op.V
	}
	if len(outerSeen) != 2 || outerSeen[a] != 1 || outerSeen[b] != 2 {
		t.Fatalf("outer Each = %v, want {a:1, b:2}", outerSeen)
	}
}

// --- N = 2 ---

func TestView2Constructor(t *testing.T) {
	w := ecs.NewWorld()
	ecs.NewView2[genComp1, genComp2](w)
}

func TestView2ConstructorPanicsOnDuplicateType(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() { ecs.NewView2[genComp1, genComp1](w) })
}

func TestView2NewEntityAndGet(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w)
	e := v.NewEntity(genComp1{V: 1}, genComp2{V: 2})

	c := v.Get(e)
	c1, c2 := c.Unpack()
	if c1.V != 1 || c2.V != 2 {
		t.Fatalf("Get = {%d %d}, want {1 2}", c1.V, c2.V)
	}
}

func TestView2SetInPlaceAndMoving(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w)
	e := v.NewEntity(genComp1{V: 1}, genComp2{V: 2})

	v.Set(e, genComp1{V: 10}, genComp2{V: 20}) // in place: e already has both
	c := v.Get(e)
	c1, c2 := c.Unpack()
	if c1.V != 10 || c2.V != 20 {
		t.Fatalf("Get after in-place Set = {%d %d}, want {10 20}", c1.V, c2.V)
	}

	e2 := w.NewEntity()
	v.Set(e2, genComp1{V: 100}, genComp2{V: 200}) // moving: e2 has neither
	c = v.Get(e2)
	c1, c2 = c.Unpack()
	if c1.V != 100 || c2.V != 200 {
		t.Fatalf("Get after moving Set = {%d %d}, want {100 200}", c1.V, c2.V)
	}
}

func TestView2Remove(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w)
	e := v.NewEntity(genComp1{}, genComp2{})

	v.Remove(e)
	if v.Has(e) {
		t.Fatalf("Has = true after Remove")
	}
	if ecs.HasComp[genComp1](w, e) || ecs.HasComp[genComp2](w, e) {
		t.Fatalf("components still present after Remove")
	}
}

func TestView2Each(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w)
	a := v.NewEntity(genComp1{V: 1}, genComp2{V: 10})
	b := v.NewEntity(genComp1{V: 2}, genComp2{V: 20})

	seen := map[ecs.Entity][2]int{}
	for e, c := range v.Each() {
		c1, c2 := c.Unpack()
		seen[e] = [2]int{c1.V, c2.V}
	}
	if seen[a] != [2]int{1, 10} || seen[b] != [2]int{2, 20} || len(seen) != 2 {
		t.Fatalf("Each = %v, want {a:{1 10}, b:{2 20}}", seen)
	}
}

// --- N = 3 ---

func TestView3Constructor(t *testing.T) {
	w := ecs.NewWorld()
	ecs.NewView3[genComp1, genComp2, genComp3](w)
}

func TestView3NewEntityGetHasSetRemoveEach(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	e := v.NewEntity(genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3})

	if !v.Has(e) {
		t.Fatalf("Has = false right after NewEntity")
	}
	c1, c2, c3 := v.Get(e).Unpack()
	if c1.V != 1 || c2.V != 2 || c3.V != 3 {
		t.Fatalf("Get = {%d %d %d}, want {1 2 3}", c1.V, c2.V, c3.V)
	}

	v.Set(e, genComp1{V: 11}, genComp2{V: 22}, genComp3{V: 33})
	c1, c2, c3 = v.Get(e).Unpack()
	if c1.V != 11 || c2.V != 22 || c3.V != 33 {
		t.Fatalf("Get after Set = {%d %d %d}, want {11 22 33}", c1.V, c2.V, c3.V)
	}

	count := 0
	for range v.Each() {
		count++
	}
	if count != 1 {
		t.Fatalf("Each matched %d, want 1", count)
	}

	v.Remove(e)
	if v.Has(e) {
		t.Fatalf("Has = true after Remove")
	}
}

func TestView3GetPanicsOnMissingComponent(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView3[genComp1, genComp2, genComp3](w)
	e := ecs.NewEntity2(w, genComp1{}, genComp2{}) // missing genComp3
	mustPanic(t, "ecs: entity does not have component", func() { v.Get(e) })
}

// --- N = 16 ---

func newGenComp16View(w *ecs.World) ecs.View16[
	genComp1, genComp2, genComp3, genComp4, genComp5, genComp6, genComp7, genComp8,
	genComp9, genComp10, genComp11, genComp12, genComp13, genComp14, genComp15, genComp16,
] {
	return ecs.NewView16[
		genComp1, genComp2, genComp3, genComp4, genComp5, genComp6, genComp7, genComp8,
		genComp9, genComp10, genComp11, genComp12, genComp13, genComp14, genComp15, genComp16,
	](w)
}

func TestView16Constructor(t *testing.T) {
	w := ecs.NewWorld()
	newGenComp16View(w)
}

func TestView16NewEntityGetSetRemoveEach(t *testing.T) {
	w := ecs.NewWorld()
	v := newGenComp16View(w)

	e := v.NewEntity(
		genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3}, genComp4{V: 4},
		genComp5{V: 5}, genComp6{V: 6}, genComp7{V: 7}, genComp8{V: 8},
		genComp9{V: 9}, genComp10{V: 10}, genComp11{V: 11}, genComp12{V: 12},
		genComp13{V: 13}, genComp14{V: 14}, genComp15{V: 15}, genComp16{V: 16},
	)

	if !v.Has(e) {
		t.Fatalf("Has = false right after NewEntity")
	}
	c := v.Get(e)
	if c.Comp1.V != 1 || c.Comp16.V != 16 {
		t.Fatalf("Get boundary fields = {Comp1:%d Comp16:%d}, want {1 16}", c.Comp1.V, c.Comp16.V)
	}

	v.Set(e,
		genComp1{V: 101}, genComp2{V: 102}, genComp3{V: 103}, genComp4{V: 104},
		genComp5{V: 105}, genComp6{V: 106}, genComp7{V: 107}, genComp8{V: 108},
		genComp9{V: 109}, genComp10{V: 110}, genComp11{V: 111}, genComp12{V: 112},
		genComp13{V: 113}, genComp14{V: 114}, genComp15{V: 115}, genComp16{V: 116},
	)
	c = v.Get(e)
	if c.Comp1.V != 101 || c.Comp16.V != 116 {
		t.Fatalf("Get after Set boundary fields = {Comp1:%d Comp16:%d}, want {101 116}", c.Comp1.V, c.Comp16.V)
	}

	count := 0
	for range v.Each() {
		count++
	}
	if count != 1 {
		t.Fatalf("Each matched %d, want 1", count)
	}

	v.Remove(e)
	if v.Has(e) {
		t.Fatalf("Has = true after Remove")
	}
}

func TestView16GetPanicsOnMissingComponent(t *testing.T) {
	w := ecs.NewWorld()
	v := newGenComp16View(w)
	// Entity has only the first 15; genComp16 is missing.
	e := ecs.NewEntity15(w,
		genComp1{}, genComp2{}, genComp3{}, genComp4{},
		genComp5{}, genComp6{}, genComp7{}, genComp8{},
		genComp9{}, genComp10{}, genComp11{}, genComp12{},
		genComp13{}, genComp14{}, genComp15{},
	)
	mustPanic(t, "ecs: entity does not have component", func() { v.Get(e) })
}

// --- Filters ---

// eachEntities returns the entities v.Each yields, in order.
func eachEntities(v ecs.View2[genComp1, genComp2]) []ecs.Entity {
	var got []ecs.Entity
	for e := range v.Each() {
		got = append(got, e)
	}
	return got
}

func TestViewWithoutEach(t *testing.T) {
	tests := []struct {
		name string
		// run builds entities in w and returns the ones Each must yield.
		run func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity
	}{
		{"excluded archetype is skipped", func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity {
			keep := w.NewEntity(genComp1{}, genComp2{})
			w.NewEntity(genComp1{}, genComp2{}, genComp3{})
			return []ecs.Entity{keep}
		}},
		{"archetype created after the first Each is filtered too", func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity {
			keep := w.NewEntity(genComp1{}, genComp2{})
			eachEntities(v) // fill the match cache
			w.NewEntity(genComp1{}, genComp2{}, genComp3{}, genComp5{})
			keep2 := w.NewEntity(genComp1{}, genComp2{}, genComp5{})
			return []ecs.Entity{keep, keep2}
		}},
		{"entity that gains the excluded component disappears", func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity {
			keep := w.NewEntity(genComp1{}, genComp2{})
			gone := w.NewEntity(genComp1{}, genComp2{})
			eachEntities(v)
			ecs.SetComp(w, gone, genComp3{})
			return []ecs.Entity{keep}
		}},
		{"entity that loses the excluded component reappears", func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity {
			back := w.NewEntity(genComp1{}, genComp2{}, genComp3{})
			eachEntities(v)
			ecs.RemoveComp[genComp3](w, back)
			return []ecs.Entity{back}
		}},
		{"second filter excludes its own type", func(w *ecs.World, v ecs.View2[genComp1, genComp2]) []ecs.Entity {
			keep := w.NewEntity(genComp1{}, genComp2{}, genComp5{})
			w.NewEntity(genComp1{}, genComp2{}, genComp4{})
			w.NewEntity(genComp1{}, genComp2{}, genComp3{}, genComp4{})
			return []ecs.Entity{keep}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := ecs.NewWorld()
			v := ecs.NewView2[genComp1, genComp2](w, ecs.Without[genComp3](), ecs.Without[genComp4]())
			want := tc.run(w, v)
			got := eachEntities(v)
			if len(got) != len(want) {
				t.Fatalf("Each = %v, want %v", got, want)
			}
			for _, e := range want {
				found := false
				for _, g := range got {
					found = found || g == e
				}
				if !found {
					t.Fatalf("Each = %v, missing %v", got, e)
				}
				if !v.Has(e) {
					t.Fatalf("Has(%v) = false for a yielded entity", e)
				}
			}
		})
	}
}

func TestViewWithoutHasFollowsFilterGetSetRemoveIgnoreIt(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w, ecs.Without[genComp3]())
	e := w.NewEntity(genComp1{V: 1}, genComp2{V: 2}, genComp3{})

	if v.Has(e) {
		t.Fatalf("Has = true for an entity with an excluded component")
	}
	if c := v.Get(e); c.Comp1.V != 1 || c.Comp2.V != 2 {
		t.Fatalf("Get = %+v, want V=1, V=2", c)
	}
	v.Set(e, genComp1{V: 3}, genComp2{V: 4})
	if c := v.Get(e); c.Comp1.V != 3 || c.Comp2.V != 4 {
		t.Fatalf("Get after Set = %+v, want V=3, V=4", c)
	}
	v.Remove(e)
	if ecs.HasComp[genComp1](w, e) || ecs.HasComp[genComp2](w, e) || !ecs.HasComp[genComp3](w, e) {
		t.Fatalf("Remove did not remove exactly the view's components")
	}

	n := v.NewEntity(genComp1{}, genComp2{})
	if !v.Has(n) {
		t.Fatalf("Has = false for an entity made by NewEntity")
	}
}

func TestViewWithoutMutationDuringIterationPanics(t *testing.T) {
	w := ecs.NewWorld()
	v := ecs.NewView2[genComp1, genComp2](w, ecs.Without[genComp3]())
	w.NewEntity(genComp1{}, genComp2{})
	w.NewEntity(genComp1{}, genComp2{})

	mustPanic(t, "ecs: query invalidated by world mutation", func() {
		for e := range v.Each() {
			ecs.SetComp(w, e, genComp3{}) // structural change; loop body doesn't break
		}
	})
}

func TestViewFilterPanics(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: component both required and excluded: ecs_test.genComp2", func() {
		ecs.NewView2[genComp1, genComp2](w, ecs.Without[genComp2]())
	})
	mustPanic(t, "ecs: duplicate component: ecs_test.genComp3", func() {
		ecs.NewView2[genComp1, genComp2](w, ecs.Without[genComp3](), ecs.Without[genComp3]())
	})
	mustPanic(t, "ecs: zero Filter", func() {
		ecs.NewView1[genComp1](w, ecs.Filter{})
	})
}
