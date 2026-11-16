package ecs_test

import (
	"testing"

	"github.com/weakpixel/ecs"
)

// benchSink keeps the compiler from dropping the lookups under measurement.
var (
	benchSinkPtr  *genComp1
	benchSinkBool bool
)

// BenchmarkLookup compares one-component lookups by entity: the generic
// functions resolve the component type on every call (a registry lookup
// keyed by reflect.Type), a view resolved it at construction. The entity
// has the component, so every variant takes the same path through storage.
func BenchmarkLookup(b *testing.B) {
	w := ecs.NewWorld()
	v := ecs.NewView1[genComp1](w)
	e := v.NewEntity(genComp1{V: 1})

	b.Run("GetCompOrNil", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchSinkPtr = ecs.GetCompOrNil[genComp1](w, e)
		}
	})
	b.Run("HasComp", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchSinkBool = ecs.HasComp[genComp1](w, e)
		}
	})
	b.Run("View1.Get", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchSinkPtr = v.Get(e)
		}
	})
	b.Run("View1.GetOrNil", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchSinkPtr = v.GetOrNil(e)
		}
	})
	b.Run("View1.Has", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchSinkBool = v.Has(e)
		}
	})
}
