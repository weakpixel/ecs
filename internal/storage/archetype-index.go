package storage

import "fmt"

// stackKeyIDs is how many IDs fit in the stack buffer used to build map keys.
const stackKeyIDs = 32

// Index owns every archetype of one World and finds them by component set.
//
// The index is append-only: archetypes are never removed, reordered or
// replaced, and empty ones stay. Archetype pointers therefore stay valid and
// the position in Archetypes is stable, so a query cache can remember how many
// archetypes it has scanned and look only at the new ones next time.
type Index struct {
	newColumn  func(CompID) Column
	archetypes []*Archetype
	byKey      map[string]*Archetype
	scratch    []CompID // reused by WithAdded/WithRemoved on the miss path
}

// NewIndex returns an index whose archetypes create their columns with
// newColumn, called once per component of each new archetype. The archetype
// for the empty set is created immediately and is available as Empty.
func NewIndex(newColumn func(CompID) Column) *Index {
	ix := &Index{newColumn: newColumn, byKey: make(map[string]*Archetype)}
	ix.Get(nil)
	return ix
}

// Empty returns the archetype with no components. It is always
// Archetypes()[0].
func (ix *Index) Empty() *Archetype { return ix.archetypes[0] }

// Archetypes returns every archetype in creation order. The slice only grows
// and existing elements never change, so callers may remember its length and
// scan [length:] later. It is shared and must not be modified.
func (ix *Index) Archetypes() []*Archetype { return ix.archetypes }

// Get returns the archetype for the component set ids, creating it when it
// doesn't exist yet. ids must be strictly ascending (sorted, no duplicates);
// Get panics with "ecs: internal error: ..." otherwise, which public API
// validation is expected to make unreachable. ids isn't retained.
//
// A hit doesn't allocate and costs O(len(ids)).
func (ix *Index) Get(ids []CompID) *Archetype {
	// Build the key in a stack buffer. m[string(b)] with a []byte conversion
	// directly in the index expression doesn't allocate, so a hit is free.
	// Larger sets fall back to the heap.
	var stack [2 * stackKeyIDs]byte
	buf := stack[:0]
	if len(ids) > stackKeyIDs {
		buf = make([]byte, 0, 2*len(ids))
	}
	for _, id := range ids {
		buf = append(buf, byte(id), byte(id>>8))
	}
	if a, ok := ix.byKey[string(buf)]; ok {
		return a
	}
	// Only a miss needs the sortedness check: an unsorted set can't have
	// been stored, so it can never hit.
	if !IsSet(ids) {
		panic(fmt.Sprintf("ecs: internal error: component set %v is not sorted and duplicate-free", ids))
	}
	a := newArchetype(ids, ix.newColumn)
	ix.archetypes = append(ix.archetypes, a)
	ix.byKey[string(buf)] = a
	return a
}

// WithAdded returns the archetype for a's components plus id, or a itself when
// it already has id. The result is cached on a, so repeated calls are a map
// lookup without allocation.
func (ix *Index) WithAdded(a *Archetype, id CompID) *Archetype {
	if a.Has(id) {
		return a
	}
	if next, ok := a.addEdge[id]; ok {
		return next
	}
	one := [1]CompID{id}
	ix.scratch = Union(ix.scratch, a.comps, one[:])
	next := ix.Get(ix.scratch)
	if a.addEdge == nil {
		a.addEdge = make(map[CompID]*Archetype)
	}
	a.addEdge[id] = next
	return next
}

// WithRemoved returns the archetype for a's components minus id, or a itself
// when it doesn't have id. The result is cached on a like WithAdded.
func (ix *Index) WithRemoved(a *Archetype, id CompID) *Archetype {
	if !a.Has(id) {
		return a
	}
	if next, ok := a.removeEdge[id]; ok {
		return next
	}
	one := [1]CompID{id}
	ix.scratch = Diff(ix.scratch, a.comps, one[:])
	next := ix.Get(ix.scratch)
	if a.removeEdge == nil {
		a.removeEdge = make(map[CompID]*Archetype)
	}
	a.removeEdge[id] = next
	return next
}
