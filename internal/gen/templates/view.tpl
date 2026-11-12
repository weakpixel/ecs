{{define "view"}}
// NewView{{.N}} returns a view over every entity with all {{.N}} of these
// component types. It registers and resolves the component IDs now; the
// archetype it spawns into (ViewN.NewEntity) and the archetype-match cache
// (ViewN.Each) are each resolved lazily, on first use. Filters such as
// [Without] narrow which entities Each and Has select; they are checked once
// per archetype, not per entity.
//
// Panics with "ecs: duplicate component: <type>" if two type parameters are
// the same type or two filters exclude the same type, with "ecs: component
// both required and excluded: <type>" if a filter excludes one of the type
// parameters, and with "ecs: zero Filter" for a Filter not made by Without.
func NewView{{.N}}[{{.TypeParamDecl}}](w *World, filters ...Filter) View{{.N}}[{{.TypeParams}}] {
	ids := []storage.CompID{
{{range .Fields}}		storage.CompID(RegisterComp[{{.Name}}]()),
{{end}}	}
	return View{{.N}}[{{.TypeParams}}]{state: newViewState(w, ids, filters)}
}

// View{{.N}} is a reusable view over entities with these {{.N}} fixed
// component types. It is a small value holding a pointer to shared state, so
// copies share it; store and reuse one instead of constructing it per call.
// The zero View{{.N}} panics with "ecs: not bound to a world" on every
// method.
type View{{.N}}[{{.TypeParamDecl}}] struct{ state *viewState }

// NewEntity creates an entity with exactly this view's components, using the
// archetype resolved on the view's first NewEntity call.
//
// Cost: 0 allocations in steady state.
func (v View{{.N}}[{{.TypeParams}}]) NewEntity({{.ArgList}}) Entity {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	arch := st.spawnArchetype()
	e, row := st.world.spawnIn(arch)
{{range $i, $f := .Fields}}	columnOf[{{$f.Name}}](arch, st.ids[{{$i}}]).Set(row, {{$f.Arg}})
{{end}}	st.world.revision++
	return e
}

{{if eq .N 1}}
// Get returns a pointer to e's live T1.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, and with
// "ecs: entity does not have component: <type>" if e doesn't currently have
// it.
//
// Cost: 0 allocations.
func (v View1[T1]) Get(e Entity) *T1 {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	rec := st.world.record(e)
	c1 := columnOf[T1](rec.Arch, st.ids[0])
	if c1 == nil {
		panicMissingComp(st.ids[0])
	}
	return c1.Ptr(int(rec.Row))
}
{{else}}
// Get returns pointers to e's live components, in type-parameter order.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale, and with
// "ecs: entity does not have component: <type>" on the first one e doesn't
// currently have.
//
// Cost: 0 allocations.
func (v View{{.N}}[{{.TypeParams}}]) Get(e Entity) Components{{.N}}[{{.TypeParams}}] {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	rec := st.world.record(e)
	arch := rec.Arch
	row := int(rec.Row)
	var c Components{{.N}}[{{.TypeParams}}]
{{range $i, $f := .Fields}}	col{{$f.Comp}} := columnOf[{{$f.Name}}](arch, st.ids[{{$i}}])
	if col{{$f.Comp}} == nil {
		panicMissingComp(st.ids[{{$i}}])
	}
	c.{{$f.Comp}} = col{{$f.Comp}}.Ptr(row)
{{end}}	return c
}
{{end}}

// Has reports whether e currently has all {{.N}} components and none that
// the view's filters exclude, which is whether Each would yield it.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
func (v View{{.N}}[{{.TypeParams}}]) Has(e Entity) bool {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	return st.has(e)
}

// Set adds or replaces all {{.N}} components. In place, with the revision
// unchanged, when e already has every one of them; otherwise e moves to a
// new archetype exactly once and the revision bumps once.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
//
// Cost: 0 allocations, in place and when moving.
func (v View{{.N}}[{{.TypeParams}}]) Set(e Entity, {{.ArgList}}) {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	rec, arch := st.setTarget(e)
	row := int(rec.Row)
{{range $i, $f := .Fields}}	columnOf[{{$f.Name}}](arch, st.ids[{{$i}}]).Set(row, {{$f.Arg}})
{{end}}}

// Remove removes whichever of this view's components e currently has, in one
// archetype move; a no-op, leaving the revision unchanged, if e has none of
// them.
//
// Panics with "ecs: invalid or stale entity: <e>" if e is stale.
func (v View{{.N}}[{{.TypeParams}}]) Remove(e Entity) {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	st.remove(e)
}

{{if eq .N 1}}
// Each yields every entity that currently has T1 (entities with additional
// components also match, unless a filter excludes one of them).
//
// Writing through the yielded pointer, or a Set on an entity that already
// has T1, is safe during iteration. A structural change (creating or
// deleting an entity, or an add/remove) panics with
// "ecs: query invalidated by world mutation" once the loop body that caused
// it returns true to continue; breaking out of the loop is always safe.
//
// Cost: 0 allocations per loop in the common case. A nested iteration of
// the same View1 (the same View1 used inside its own Each loop) allocates
// once: the per-archetype column buffer below is already in use by the
// outer call.
//
// Panics with "ecs: not bound to a world" if View1 is the zero value.
func (v View1[T1]) Each() iter.Seq2[Entity, *T1] {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	return func(yield func(Entity, *T1) bool) {
		st.scan()
		cols := st.cols
		if st.iterating {
			cols = make([]storage.Column, len(st.ids))
		} else {
			st.iterating = true
			defer func() { st.iterating = false }()
		}
		rev := st.world.revision
		for _, arch := range st.matched {
			n := arch.Len()
			if n == 0 {
				continue
			}
			cols[0] = arch.Column(arch.ColumnIndex(st.ids[0]))
			c1 := cols[0].(*storage.TypedColumn[T1])
			for row := range n {
				e := entityFromStorage(arch.Entity(row))
				if !yield(e, c1.Ptr(row)) {
					return
				}
				if st.world.revision != rev {
					panicQueryInvalidated()
				}
			}
		}
	}
}
{{else}}
// Each yields every entity that currently has all {{.N}} components
// (entities with additional components also match, unless a filter excludes
// one of them).
//
// Writing through the yielded pointers, or a Set on an entity that already
// has every component, is safe during iteration. A structural change
// (creating or deleting an entity, or an add/remove) panics with
// "ecs: query invalidated by world mutation" once the loop body that caused
// it returns true to continue; breaking out of the loop is always safe.
//
// Cost: 0 allocations per loop in the common case. A nested iteration of
// the same View{{.N}} (the same View{{.N}} used inside its own Each loop)
// allocates once: the per-archetype column buffer below is already in use
// by the outer call.
//
// Panics with "ecs: not bound to a world" if View{{.N}} is the zero value.
func (v View{{.N}}[{{.TypeParams}}]) Each() iter.Seq2[Entity, Components{{.N}}[{{.TypeParams}}]] {
	st := v.state
	if st == nil {
		panicNotBound()
	}
	return func(yield func(Entity, Components{{.N}}[{{.TypeParams}}]) bool) {
		st.scan()
		cols := st.cols
		if st.iterating {
			cols = make([]storage.Column, len(st.ids))
		} else {
			st.iterating = true
			defer func() { st.iterating = false }()
		}
		rev := st.world.revision
		for _, arch := range st.matched {
			n := arch.Len()
			if n == 0 {
				continue
			}
{{range $i, $f := .Fields}}			cols[{{$i}}] = arch.Column(arch.ColumnIndex(st.ids[{{$i}}]))
{{end}}{{range $i, $f := .Fields}}			col{{$f.Comp}} := cols[{{$i}}].(*storage.TypedColumn[{{$f.Name}}])
{{end}}			for row := range n {
				e := entityFromStorage(arch.Entity(row))
				comp := Components{{.N}}[{{.TypeParams}}]{
{{range .Fields}}					{{.Comp}}: col{{.Comp}}.Ptr(row),
{{end}}				}
				if !yield(e, comp) {
					return
				}
				if st.world.revision != rev {
					panicQueryInvalidated()
				}
			}
		}
	}
}
{{end}}
{{end}}
