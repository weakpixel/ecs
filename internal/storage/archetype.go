package storage

import "slices"

// Archetype is a table of all entities that have exactly one set of
// components. Each component has a column; row i of every column and
// entities[i] describe the same entity, so all columns always have the same
// length as the entity list.
//
// An archetype never gains or loses components, which is what makes its dense
// colIdx lookup safe. Archetypes are created by an Index and live as long as it
// does.
type Archetype struct {
	comps    []CompID // sorted, duplicate-free
	cols     []Column // cols[i] stores comps[i]
	entities []Entity // row -> entity handle

	// colIdx maps a CompID to its index in comps/cols, or -1 when absent.
	// It is a dense slice (length = max ID + 1) instead of a map so the hot
	// "column for this component" lookup is a bounds check and a load. IDs
	// past the end are absent.
	colIdx []int16

	// Cached single-component transitions, allocated on first use. See Index.
	addEdge    map[CompID]*Archetype
	removeEdge map[CompID]*Archetype
}

// newArchetype builds an empty archetype for the set comps, creating one
// column per ID with newColumn. comps must be a valid set; it is copied.
func newArchetype(comps []CompID, newColumn func(CompID) Column) *Archetype {
	a := &Archetype{
		comps: slices.Clone(comps),
		cols:  make([]Column, len(comps)),
	}
	if len(comps) > 0 {
		a.colIdx = make([]int16, int(comps[len(comps)-1])+1)
		for i := range a.colIdx {
			a.colIdx[i] = -1
		}
	}
	for i, id := range comps {
		a.cols[i] = newColumn(id)
		a.colIdx[id] = int16(i)
	}
	return a
}

// Len returns the number of rows (entities).
func (a *Archetype) Len() int { return len(a.entities) }

// Comps returns the sorted component set. The slice is shared and must not be
// modified.
func (a *Archetype) Comps() []CompID { return a.comps }

// Has reports whether the archetype stores component id. It is O(1).
func (a *Archetype) Has(id CompID) bool { return a.ColumnIndex(id) >= 0 }

// ColumnIndex returns the index of id's column (the position of id in Comps),
// or -1 when the archetype doesn't store id. It is O(1).
func (a *Archetype) ColumnIndex(id CompID) int {
	if int(id) >= len(a.colIdx) {
		return -1
	}
	return int(a.colIdx[id])
}

// Column returns the i-th column, in Comps order. It panics if i is out of
// range.
func (a *Archetype) Column(i int) Column { return a.cols[i] }

// Entity returns the entity stored in row. It panics if row is out of range.
func (a *Archetype) Entity(row int) Entity { return a.entities[row] }

// Entities returns the row -> entity table. The slice is shared: it is valid
// until the next AppendRow, RemoveRow or MoveRow involving this archetype, and
// must not be modified.
func (a *Archetype) Entities() []Entity { return a.entities }

// AppendRow adds a row for e with zero values in every column and returns its
// index. The caller writes the real values next.
func (a *Archetype) AppendRow(e Entity) int {
	row := len(a.entities)
	a.entities = append(a.entities, e)
	for _, c := range a.cols {
		c.AppendZero()
	}
	return row
}

// RemoveRow swap-removes row: the last row moves into it. When another entity
// was moved, ok is true and moved is that entity, which now lives in row; the
// caller must update its record. ok is false when row was the last row. It
// panics if row is out of range.
func (a *Archetype) RemoveRow(row int) (moved Entity, ok bool) {
	last := len(a.entities) - 1
	for _, c := range a.cols {
		c.SwapRemove(row)
	}
	if row != last {
		moved = a.entities[last]
		a.entities[row] = moved
		ok = true
	}
	a.entities[last] = Entity{}
	a.entities = a.entities[:last]
	return moved, ok
}

// MoveRow moves row of src into a new row of dst and returns that row.
// Components stored by both archetypes keep their values, components only dst
// stores are zero. The row is then swap-removed from src: ok is true and moved
// is the entity that now occupies row in src, whose record the caller must fix
// (ok is false when row was the last row). The moved entity's own record
// (archetype and dstRow) is also the caller's to update.
//
// Preconditions, not checked: src != dst and row is in range. Pointers into
// both archetypes are invalidated. Cost is O(components) with no allocation
// once the destination columns have capacity.
func MoveRow(src *Archetype, row int, dst *Archetype) (dstRow int, moved Entity, ok bool) {
	dstRow = len(dst.entities)
	dst.entities = append(dst.entities, src.entities[row])

	// Both component lists are sorted, so a single merge pass pairs the
	// columns without any colIdx lookups.
	si := 0
	for di, id := range dst.comps {
		for si < len(src.comps) && src.comps[si] < id {
			si++
		}
		if si < len(src.comps) && src.comps[si] == id {
			src.cols[si].MoveRowTo(dst.cols[di], row)
		} else {
			dst.cols[di].AppendZero()
		}
	}
	moved, ok = src.RemoveRow(row)
	return dstRow, moved, ok
}
