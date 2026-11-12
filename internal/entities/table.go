package entities

import "github.com/weakpixel/ecs/internal/storage"

// Record is the current location of a live entity: the archetype that stores
// it and the row within that archetype. Arch is nil for a free slot.
//
// A *Record returned by Table.Alloc or Table.Get is valid until the next call
// to Alloc, which may grow the backing slice; callers don't hold it across an
// allocation.
type Record struct {
	Arch *storage.Archetype
	Row  int32
}

// slot is one entry of a Table, indexed by storage.Entity.ID.
type slot struct {
	gen uint32 // 0 only before the slot's first Alloc; otherwise >= 1
	rec Record // rec.Arch == nil means the slot is free
}

// Table maps entity IDs to their current Record and detects stale handles
// through a per-slot generation counter. The zero Table is ready to use.
//
// Table is not safe for concurrent use; a World (which owns one) is
// single-goroutine.
type Table struct {
	slots []slot
	free  []uint32 // LIFO free list of slot indices (most recently freed first)
}

// Alloc reserves a slot and returns its handle together with a pointer to its
// Record, which the caller fills in (typically after appending a row to the
// destination archetype, since the row-to-entity table needs the handle
// first). The Record starts zeroed (Arch == nil); the caller must set it
// before the entity is considered alive.
//
// Alloc reuses the most recently freed slot (LIFO), which keeps the slot
// table compact under churn and gives it good cache locality. The first ever
// Alloc returns generation 1 (see Free for why generation 0 never appears).
func (t *Table) Alloc() (storage.Entity, *Record) {
	var idx uint32
	if n := len(t.free); n > 0 {
		idx = t.free[n-1]
		t.free = t.free[:n-1]
	} else {
		idx = uint32(len(t.slots))
		t.slots = append(t.slots, slot{gen: 1})
	}
	s := &t.slots[idx]
	return storage.Entity{ID: idx, Generation: s.gen}, &s.rec
}

// Get returns the Record for e and true when e is a live handle: in range,
// allocated, not freed and matching the slot's current generation. It never
// panics, so it is also how callers implement IsAlive without a stale-entity
// panic.
func (t *Table) Get(e storage.Entity) (*Record, bool) {
	if e.ID >= uint32(len(t.slots)) {
		return nil, false
	}
	s := &t.slots[e.ID]
	if s.rec.Arch == nil || s.gen != e.Generation {
		return nil, false
	}
	return &s.rec, true
}

// Free releases e's slot, invalidating every handle with its current
// generation, and makes the slot available for reuse by a later Alloc.
//
// The caller must have confirmed e is currently alive (for example via Get);
// Free doesn't validate e and panics on an out-of-range ID.
//
// The generation is incremented so stale handles are detected after reuse. On
// wrap from math.MaxUint32 it skips 0 and goes to 1, since generation 0 would
// make Entity{} (the zero Entity, ID 0, Generation 0) collide with a reused
// slot 0 — the zero Entity must never be alive.
func (t *Table) Free(e storage.Entity) {
	s := &t.slots[e.ID]
	s.gen++
	if s.gen == 0 {
		s.gen = 1
	}
	s.rec = Record{}
	t.free = append(t.free, e.ID)
}

// Len returns the number of currently live entities.
func (t *Table) Len() int { return len(t.slots) - len(t.free) }
