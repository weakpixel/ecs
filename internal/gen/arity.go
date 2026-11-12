package main

import (
	"fmt"
	"strings"
)

// maxArity is the highest N for which NewEntityN, NewViewN/ViewN and
// ComponentsN exist, per doc/api-generic.md.
const maxArity = 16

// field is one type parameter's names across the templates: "T1" (the type
// parameter itself), "v1" (a NewEntityN/Set argument), "Comp1" (a
// ComponentsN field) and "genComp1" (a hand-written test type, used only by
// the generated smoke test).
type field struct {
	Name string // "T1".."T16"
	Arg  string // "v1".."v16"
	Comp string // "Comp1".."Comp16"
	Gen  string // "genComp1".."genComp16"
}

// arity is the per-N data handed to every template. Everything the
// templates range over is a slice built by a plain for loop, never a map,
// so rendering the same arities always produces byte-identical output.
type arity struct {
	N             int
	Fields        []field
	HasComponents bool // N >= 2: ComponentsN only exists for N >= 2
}

// buildArities returns the data for N = 1..maxArity, in order.
func buildArities() []arity {
	out := make([]arity, maxArity)
	for n := 1; n <= maxArity; n++ {
		a := arity{N: n, HasComponents: n >= 2}
		for i := 1; i <= n; i++ {
			a.Fields = append(a.Fields, field{
				Name: fmt.Sprintf("T%d", i),
				Arg:  fmt.Sprintf("v%d", i),
				Comp: fmt.Sprintf("Comp%d", i),
				Gen:  fmt.Sprintf("genComp%d", i),
			})
		}
		out[n-1] = a
	}
	return out
}

// The methods below are called from the templates (text/template only sees
// exported methods, even within this package). Each renders one
// comma-separated list derived from Fields; go/format.Source cleans up
// whitespace afterward, so these don't need to produce pretty output.

// TypeParams renders the bare type parameter list: "T1, T2, T3".
func (a arity) TypeParams() string {
	return joinField(a.Fields, func(f field) string { return f.Name })
}

// TypeParamDecl renders the type parameter declaration: "T1, T2, T3 any".
func (a arity) TypeParamDecl() string { return a.TypeParams() + " any" }

// ArgList renders the NewEntityN/Set argument list: "v1 T1, v2 T2".
func (a arity) ArgList() string {
	return joinField(a.Fields, func(f field) string { return f.Arg + " " + f.Name })
}

// UnpackReturn renders ComponentsN.Unpack's return types: "*T1, *T2".
func (a arity) UnpackReturn() string {
	return joinField(a.Fields, func(f field) string { return "*" + f.Name })
}

// UnpackBody renders ComponentsN.Unpack's return values: "c.Comp1, c.Comp2".
func (a arity) UnpackBody() string {
	return joinField(a.Fields, func(f field) string { return "c." + f.Comp })
}

// GenTypes renders the hand-written test types for this arity, used only by
// the generated smoke test: "genComp1, genComp2".
func (a arity) GenTypes() string {
	return joinField(a.Fields, func(f field) string { return f.Gen })
}

func joinField(fields []field, get func(field) string) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = get(f)
	}
	return strings.Join(parts, ", ")
}
