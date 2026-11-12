package ecs_test

import (
	"math/rand/v2"
	"testing"

	"github.com/weakpixel/ecs"
)

// Six distinct, identically shaped component types, enough to exercise a
// good spread of archetypes under random add/remove.
type (
	m1 struct{ V int }
	m2 struct{ V int }
	m3 struct{ V int }
	m4 struct{ V int }
	m5 struct{ V int }
	m6 struct{ V int }
)

const modelNumTypes = 6

func mkModelValue(k, v int) any {
	switch k {
	case 0:
		return m1{V: v}
	case 1:
		return m2{V: v}
	case 2:
		return m3{V: v}
	case 3:
		return m4{V: v}
	case 4:
		return m5{V: v}
	default:
		return m6{V: v}
	}
}

func getModelValue(w *ecs.World, e ecs.Entity, k int, ids []ecs.ComponentID) int {
	switch k {
	case 0:
		return w.GetComp(e, ids[0]).(*m1).V
	case 1:
		return w.GetComp(e, ids[1]).(*m2).V
	case 2:
		return w.GetComp(e, ids[2]).(*m3).V
	case 3:
		return w.GetComp(e, ids[3]).(*m4).V
	case 4:
		return w.GetComp(e, ids[4]).(*m5).V
	default:
		return w.GetComp(e, ids[5]).(*m6).V
	}
}

// TestWorldModel drives World through a long sequence of random
// NewEntity/DeleteEntity/SetComp/RemoveComp calls and checks every live
// entity's component values and every dead entity's staleness against a
// plain map model after every single operation.
func TestWorldModel(t *testing.T) {
	ids := []ecs.ComponentID{
		ecs.RegisterComp[m1](),
		ecs.RegisterComp[m2](),
		ecs.RegisterComp[m3](),
		ecs.RegisterComp[m4](),
		ecs.RegisterComp[m5](),
		ecs.RegisterComp[m6](),
	}

	rng := rand.New(rand.NewPCG(42, 7))
	w := ecs.NewWorld()

	type modelEntity struct {
		comps map[int]int // type index -> value
	}
	model := map[ecs.Entity]*modelEntity{}
	var live []ecs.Entity
	var dead []ecs.Entity

	// randSubset returns 1..modelNumTypes distinct type indices in random
	// order.
	randSubset := func() []int {
		n := rng.IntN(modelNumTypes) + 1
		seen := make(map[int]bool, n)
		out := make([]int, 0, n)
		for len(out) < n {
			k := rng.IntN(modelNumTypes)
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, k)
		}
		return out
	}

	checkConsistency := func() {
		t.Helper()
		for _, e := range live {
			if !w.IsAlive(e) {
				t.Fatalf("live entity %v reported not alive", e)
			}
			me := model[e]
			for k, want := range me.comps {
				if got := getModelValue(w, e, k, ids); got != want {
					t.Fatalf("entity %v type %d = %d, want %d", e, k, got, want)
				}
			}
		}
		for _, e := range dead {
			if w.IsAlive(e) {
				t.Fatalf("dead entity %v reported alive", e)
			}
		}
	}

	const numOps = 500
	for range numOps {
		switch {
		case len(live) == 0 || rng.IntN(4) == 0:
			ks := randSubset()
			values := make([]any, len(ks))
			me := &modelEntity{comps: make(map[int]int, len(ks))}
			for i, k := range ks {
				v := rng.IntN(1000)
				values[i] = mkModelValue(k, v)
				me.comps[k] = v
			}
			e := w.NewEntity(values...)
			live = append(live, e)
			model[e] = me

		case rng.IntN(3) == 0:
			i := rng.IntN(len(live))
			e := live[i]
			w.DeleteEntity(e)
			delete(model, e)
			live[i] = live[len(live)-1]
			live = live[:len(live)-1]
			dead = append(dead, e)

		case rng.IntN(2) == 0:
			i := rng.IntN(len(live))
			e := live[i]
			me := model[e]
			ks := randSubset()
			values := make([]any, len(ks))
			for j, k := range ks {
				v := rng.IntN(1000)
				values[j] = mkModelValue(k, v)
				me.comps[k] = v
			}
			w.SetComp(e, values...)

		default:
			i := rng.IntN(len(live))
			e := live[i]
			me := model[e]
			ks := randSubset()
			toRemove := make([]ecs.ComponentID, len(ks))
			for j, k := range ks {
				toRemove[j] = ids[k]
				delete(me.comps, k)
			}
			w.RemoveComp(e, toRemove...)
		}
		checkConsistency()
	}

	for _, e := range live {
		w.DeleteEntity(e)
		dead = append(dead, e)
	}
	live = nil
	checkConsistency()
}
