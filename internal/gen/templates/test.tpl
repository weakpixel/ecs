{{define "test"}}
// TestZZGenArity{{.N}} is a generated round-trip smoke test for arity {{.N}}:
// NewEntity{{.N}}, NewView{{.N}}/View{{.N}} and all its methods. It exists to
// catch per-arity template bugs across every N; exhaustive behavior and
// panic coverage lives in the hand-written tests (new-entity_test.go,
// view_test.go).
func TestZZGenArity{{.N}}(t *testing.T) {
	w := ecs.NewWorld()

	e := ecs.NewEntity{{.N}}[{{.GenTypes}}](w,
{{range .Fields}}		{{.Gen}}{V: 1},
{{end}}	)

{{range .Fields}}	if got := ecs.GetComp[{{.Gen}}](w, e); got.V != 1 {
		t.Fatalf("{{.Gen}} after NewEntity{{$.N}} = %+v, want V=1", got)
	}
{{end}}
	view := ecs.NewView{{.N}}[{{.GenTypes}}](w)
	if !view.Has(e) {
		t.Fatalf("view.Has = false after NewEntity{{.N}}")
	}

{{if eq .N 1}}
	if got := view.Get(e); got.V != 1 {
		t.Fatalf("view.Get = %+v, want V=1", got)
	}
	view.Set(e, genComp1{V: 2})
	if got := view.Get(e); got.V != 2 {
		t.Fatalf("view.Get after Set = %+v, want V=2", got)
	}
{{else}}
	if c := view.Get(e); {{range $i, $f := .Fields}}{{if $i}} || {{end}}c.{{$f.Comp}}.V != 1{{end}} {
		t.Fatalf("view.Get fields = %+v, want all V=1", c)
	}
	view.Set(e,
{{range .Fields}}		{{.Gen}}{V: 2},
{{end}}	)
	if c := view.Get(e); {{range $i, $f := .Fields}}{{if $i}} || {{end}}c.{{$f.Comp}}.V != 2{{end}} {
		t.Fatalf("view.Get after Set = %+v, want all V=2", c)
	}
{{end}}

	view.Remove(e)
	if view.Has(e) {
		t.Fatalf("view.Has = true after Remove")
	}

	view.NewEntity(
{{range .Fields}}		{{.Gen}}{V: 1},
{{end}}	)

	count := 0
	for range view.Each() {
		count++
	}
	if count != 1 {
		t.Fatalf("view.Each matched %d entities, want 1", count)
	}

	filtered := ecs.NewView{{.N}}[{{.GenTypes}}](w, ecs.Without[genExcluded]())
	excluded := view.NewEntity(
{{range .Fields}}		{{.Gen}}{V: 1},
{{end}}	)
	ecs.SetComp(w, excluded, genExcluded{})
	count = 0
	for e := range filtered.Each() {
		if e == excluded {
			t.Fatalf("filtered.Each yielded the excluded entity")
		}
		count++
	}
	if count != 1 {
		t.Fatalf("filtered.Each matched %d entities, want 1", count)
	}
	if filtered.Has(excluded) {
		t.Fatalf("filtered.Has = true for the excluded entity")
	}
}
{{end}}
