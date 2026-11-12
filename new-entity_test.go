package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

func TestNewEntity1(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity1(w, genComp1{V: 1})

	if got := ecs.GetComp[genComp1](w, e); got.V != 1 {
		t.Fatalf("genComp1 = %+v, want V=1", got)
	}
}

func TestNewEntity2(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity2(w, genComp1{V: 1}, genComp2{V: 2})

	if got := ecs.GetComp[genComp1](w, e); got.V != 1 {
		t.Fatalf("genComp1 = %+v, want V=1", got)
	}
	if got := ecs.GetComp[genComp2](w, e); got.V != 2 {
		t.Fatalf("genComp2 = %+v, want V=2", got)
	}
}

func TestNewEntity2PanicsOnDuplicateType(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() { ecs.NewEntity2(w, genComp1{V: 1}, genComp1{V: 2}) })
}

func TestNewEntity3(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity3(w, genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3})

	if got := ecs.GetComp[genComp1](w, e); got.V != 1 {
		t.Fatalf("genComp1 = %+v, want V=1", got)
	}
	if got := ecs.GetComp[genComp2](w, e); got.V != 2 {
		t.Fatalf("genComp2 = %+v, want V=2", got)
	}
	if got := ecs.GetComp[genComp3](w, e); got.V != 3 {
		t.Fatalf("genComp3 = %+v, want V=3", got)
	}
}

func TestNewEntity3PanicsOnDuplicateType(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() { ecs.NewEntity3(w, genComp1{V: 1}, genComp2{V: 2}, genComp1{V: 3}) })
}

func TestNewEntity16(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity16(w,
		genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3}, genComp4{V: 4},
		genComp5{V: 5}, genComp6{V: 6}, genComp7{V: 7}, genComp8{V: 8},
		genComp9{V: 9}, genComp10{V: 10}, genComp11{V: 11}, genComp12{V: 12},
		genComp13{V: 13}, genComp14{V: 14}, genComp15{V: 15}, genComp16{V: 16},
	)

	if got := ecs.GetComp[genComp1](w, e); got.V != 1 {
		t.Fatalf("genComp1 = %+v, want V=1", got)
	}
	if got := ecs.GetComp[genComp16](w, e); got.V != 16 {
		t.Fatalf("genComp16 = %+v, want V=16", got)
	}
}

func TestNewEntity16PanicsOnDuplicateType(t *testing.T) {
	w := ecs.NewWorld()
	mustPanic(t, "ecs: duplicate component", func() {
		ecs.NewEntity16(w,
			genComp1{V: 1}, genComp2{V: 2}, genComp3{V: 3}, genComp4{V: 4},
			genComp5{V: 5}, genComp6{V: 6}, genComp7{V: 7}, genComp8{V: 8},
			genComp9{V: 9}, genComp10{V: 10}, genComp11{V: 11}, genComp12{V: 12},
			genComp13{V: 13}, genComp14{V: 14}, genComp15{V: 15}, genComp1{V: 99},
		)
	})
}
