package registry

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/weakpixel/ecs/internal/storage"
)

type (
	posC  struct{ X, Y float32 }
	velC  struct{ DX, DY float32 }
	nameC struct{ Name string }
	c3    struct{ A int }
	c4    struct{ A int }
	c5    struct{ A int }
	c6    struct{ A int }
	c7    struct{ A int }
	iface interface{ M() }
)

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

func TestRegisterDenseAndIdempotent(t *testing.T) {
	r := New(MaxComponents)
	if got := Register[posC](r); got != 0 {
		t.Fatalf("first ID = %d, want 0", got)
	}
	if got := Register[velC](r); got != 1 {
		t.Fatalf("second ID = %d, want 1", got)
	}
	if got := Register[posC](r); got != 0 {
		t.Fatalf("re-register = %d, want 0", got)
	}
	if r.Len() != 2 {
		t.Fatalf("Len = %d, want 2", r.Len())
	}
}

func TestRegisterRejectsNonStruct(t *testing.T) {
	r := New(MaxComponents)
	tests := []struct {
		name string
		f    func()
	}{
		{"int", func() { Register[int](r) }},
		{"string", func() { Register[string](r) }},
		{"pointer", func() { Register[*posC](r) }},
		{"slice", func() { Register[[]posC](r) }},
		{"interface", func() { Register[iface](r) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustPanic(t, "ecs: component must be a non-pointer struct: ", tt.f)
		})
	}
	if r.Len() != 0 {
		t.Fatalf("rejected types were registered: Len = %d", r.Len())
	}
}

func TestRegisterCapacity(t *testing.T) {
	r := New(2)
	Register[posC](r)
	Register[velC](r)
	mustPanic(t, "ecs: component registry capacity exceeded: max 2", func() { Register[nameC](r) })
	// A pointer type panics with the struct message even when full (D6).
	mustPanic(t, "ecs: component must be a non-pointer struct", func() { Register[*posC](r) })
	// Existing types still resolve.
	if got := Register[velC](r); got != 1 {
		t.Fatalf("existing type after cap = %d, want 1", got)
	}
}

func TestInfo(t *testing.T) {
	r := New(MaxComponents)
	id := Register[nameC](r)
	info := r.Info(id)
	if info.ID != id || info.Type != reflect.TypeFor[nameC]() {
		t.Fatalf("Info = %+v", info)
	}
	col := info.NewColumn()
	col.AppendAny(nameC{"a"})
	if col.Len() != 1 {
		t.Fatalf("column Len = %d", col.Len())
	}
	if p := col.PtrAny(0).(*nameC); p.Name != "a" {
		t.Fatalf("column value = %+v", *p)
	}
	if info.NewColumn() == col {
		t.Fatal("NewColumn must return a fresh column")
	}
	mustPanic(t, "ecs: component not registered: ComponentID(7)", func() { r.Info(7) })
}

func TestLookup(t *testing.T) {
	r := New(MaxComponents)
	if _, ok := r.Lookup(reflect.TypeFor[posC]()); ok {
		t.Fatal("unregistered type reported present")
	}
	id := Register[posC](r)
	if got, ok := r.Lookup(reflect.TypeFor[posC]()); !ok || got != id {
		t.Fatalf("Lookup = %d, %v", got, ok)
	}
}

func TestConcurrentRegister(t *testing.T) {
	r := New(MaxComponents)
	regs := []func() storage.CompID{
		func() storage.CompID { return Register[posC](r) },
		func() storage.CompID { return Register[velC](r) },
		func() storage.CompID { return Register[nameC](r) },
		func() storage.CompID { return Register[c3](r) },
		func() storage.CompID { return Register[c4](r) },
		func() storage.CompID { return Register[c5](r) },
		func() storage.CompID { return Register[c6](r) },
		func() storage.CompID { return Register[c7](r) },
	}
	const workers = 64
	got := make([][]storage.CompID, workers)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			got[w] = make([]storage.CompID, len(regs))
			for _, i := range rand.Perm(len(regs)) {
				got[w][i] = regs[i]()
			}
		})
	}
	wg.Wait()

	seen := map[storage.CompID]bool{}
	for i := range regs {
		id := got[0][i]
		for w := range workers {
			if got[w][i] != id {
				t.Fatalf("type %d: worker %d got ID %d, worker 0 got %d", i, w, got[w][i], id)
			}
		}
		if id >= storage.CompID(len(regs)) || seen[id] {
			t.Fatalf("IDs not dense/unique: %d", id)
		}
		seen[id] = true
	}
	if r.Len() != len(regs) {
		t.Fatalf("Len = %d, want %d", r.Len(), len(regs))
	}
}

func TestAllocsRegisteredLookup(t *testing.T) {
	r := New(MaxComponents)
	Register[posC](r)
	typ := reflect.TypeFor[posC]()
	var sink storage.CompID
	if n := testing.AllocsPerRun(100, func() { sink = Register[posC](r) }); n != 0 {
		t.Errorf("Register of a registered type: %v allocs, want 0", n)
	}
	if n := testing.AllocsPerRun(100, func() { sink, _ = r.Lookup(typ) }); n != 0 {
		t.Errorf("Lookup: %v allocs, want 0", n)
	}
	_ = fmt.Sprint(sink)
}
