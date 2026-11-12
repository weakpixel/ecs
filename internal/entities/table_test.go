package entities

import (
	"math"
	"testing"

	"github.com/weakpixel/ecs/internal/storage"
)

// dummyArch is a non-nil *storage.Archetype used only as a Record.Arch
// sentinel; its fields are never read by Table.
var dummyArch = &storage.Archetype{}

func TestAllocFirstHandle(t *testing.T) {
	var tbl Table
	e, rec := tbl.Alloc()
	if e.ID != 0 || e.Generation != 1 {
		t.Fatalf("first handle = %+v, want {0 1}", e)
	}
	rec.Arch = dummyArch
	if _, ok := tbl.Get(e); !ok {
		t.Fatalf("Get(%+v) = false after Alloc+Arch set", e)
	}
}

func TestGetFalseCases(t *testing.T) {
	var tbl Table

	if _, ok := tbl.Get(storage.Entity{}); ok {
		t.Fatalf("Get(zero Entity) = true, want false")
	}
	if _, ok := tbl.Get(storage.Entity{ID: 5}); ok {
		t.Fatalf("Get(out-of-range) = true, want false")
	}

	e, rec := tbl.Alloc()
	// Allocated but Arch never set: not alive yet.
	if _, ok := tbl.Get(e); ok {
		t.Fatalf("Get(allocated, no Arch) = true, want false")
	}
	rec.Arch = dummyArch
	if _, ok := tbl.Get(e); !ok {
		t.Fatalf("Get(live) = false, want true")
	}

	tbl.Free(e)
	if _, ok := tbl.Get(e); ok {
		t.Fatalf("Get(freed) = true, want false")
	}

	stale := e
	stale.Generation++
	if _, ok := tbl.Get(stale); ok {
		t.Fatalf("Get(wrong generation) = true, want false")
	}
}

func TestFreeLIFOReuseBumpsGeneration(t *testing.T) {
	var tbl Table
	a, recA := tbl.Alloc()
	recA.Arch = dummyArch
	b, recB := tbl.Alloc()
	recB.Arch = dummyArch

	tbl.Free(a)
	tbl.Free(b)

	// LIFO: b's slot comes back first.
	reused, rec := tbl.Alloc()
	rec.Arch = dummyArch
	if reused.ID != b.ID {
		t.Fatalf("reused ID = %d, want %d (LIFO)", reused.ID, b.ID)
	}
	if reused.Generation != b.Generation+1 {
		t.Fatalf("reused generation = %d, want %d", reused.Generation, b.Generation+1)
	}
	if _, ok := tbl.Get(b); ok {
		t.Fatalf("old handle b still alive after reuse")
	}
}

func TestFreeWrapSkipsZero(t *testing.T) {
	var tbl Table
	e, rec := tbl.Alloc()
	rec.Arch = dummyArch
	tbl.slots[e.ID].gen = math.MaxUint32

	tbl.Free(storage.Entity{ID: e.ID, Generation: math.MaxUint32})

	reused, rec2 := tbl.Alloc()
	rec2.Arch = dummyArch
	if reused.Generation != 1 {
		t.Fatalf("generation after wrap = %d, want 1", reused.Generation)
	}
}

func TestLen(t *testing.T) {
	var tbl Table
	if tbl.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", tbl.Len())
	}
	a, recA := tbl.Alloc()
	recA.Arch = dummyArch
	_, recB := tbl.Alloc()
	recB.Arch = dummyArch
	if tbl.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", tbl.Len())
	}
	tbl.Free(a)
	if tbl.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", tbl.Len())
	}
}

func TestAllocGetNoAllocs(t *testing.T) {
	var tbl Table
	// Warm up: grow past any initial capacity and establish a steady free
	// list before measuring.
	handles := make([]storage.Entity, 0, 64)
	for range 64 {
		e, rec := tbl.Alloc()
		rec.Arch = dummyArch
		handles = append(handles, e)
	}
	for _, e := range handles {
		tbl.Free(e)
	}

	if n := testing.AllocsPerRun(100, func() {
		e, rec := tbl.Alloc()
		rec.Arch = dummyArch
		tbl.Free(e)
	}); n != 0 {
		t.Fatalf("Alloc+Free allocs = %v, want 0", n)
	}

	e, rec := tbl.Alloc()
	rec.Arch = dummyArch
	if n := testing.AllocsPerRun(100, func() {
		tbl.Get(e)
	}); n != 0 {
		t.Fatalf("Get allocs = %v, want 0", n)
	}
}
