package ecs_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/weakpixel/ecs"
)

type (
	regPos  struct{ X, Y int }
	regVel  struct{ DX, DY int }
	regOnce struct{ A int }
)

// Registered during package initialization, before any test runs.
var initID = ecs.RegisterComp[regOnce]()

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", want)
		}
		var msg string
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, want) {
			t.Fatalf("panic %q does not contain %q", msg, want)
		}
	}()
	f()
}

func TestRegisterCompIdempotent(t *testing.T) {
	a := ecs.RegisterComp[regPos]()
	b := ecs.RegisterComp[regVel]()
	if a == b {
		t.Fatalf("distinct types share ID %d", a)
	}
	if again := ecs.RegisterComp[regPos](); again != a {
		t.Fatalf("second call = %d, want %d", again, a)
	}
}

func TestRegisterCompPackageInit(t *testing.T) {
	if got := ecs.RegisterComp[regOnce](); got != initID {
		t.Fatalf("ID changed after init: %d vs %d", got, initID)
	}
}

func TestRegisterCompPanics(t *testing.T) {
	mustPanic(t, "ecs: component must be a non-pointer struct: int", func() { ecs.RegisterComp[int]() })
	mustPanic(t, "ecs: component must be a non-pointer struct", func() { ecs.RegisterComp[*regPos]() })
}

type conc struct{ A int }

func TestRegisterCompConcurrent(t *testing.T) {
	ids := make([]ecs.ComponentID, 32)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Go(func() { ; ids[i] = ecs.RegisterComp[conc]() })
	}
	wg.Wait()
	for _, id := range ids {
		if id != ids[0] {
			t.Fatalf("IDs differ: %v", ids)
		}
	}
}
