{{define "newEntity"}}
// NewEntity{{.N}} creates an entity holding exactly these {{.N}} components,
// inferring the component types from the argument types. Registers each
// type on first use.
//
// Panics with "ecs: duplicate component: <type>" if two type parameters are
// the same type.
//
// Cost: 0 allocations in steady state.
func NewEntity{{.N}}[{{.TypeParamDecl}}](w *World, {{.ArgList}}) Entity {
	ids := [{{.N}}]storage.CompID{
{{range .Fields}}		storage.CompID(RegisterComp[{{.Name}}]()),
{{end}}	}
	arch := w.newEntityArchetype(ids[:])
	e, row := w.spawnIn(arch)
{{range $i, $f := .Fields}}	columnOf[{{$f.Name}}](arch, ids[{{$i}}]).Set(row, {{$f.Arg}})
{{end}}	w.revision++
	return e
}
{{end}}
