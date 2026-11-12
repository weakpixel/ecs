package ecs

import (
	"strings"
	"testing"

	"github.com/weakpixel/ecs/internal/storage"
)

type colComp struct{ N int }

func TestNewColumn(t *testing.T) {
	id := RegisterComp[colComp]()
	col := newColumn(storage.CompID(id))
	col.AppendAny(colComp{N: 3})
	if p := col.PtrAny(0).(*colComp); p.N != 3 {
		t.Fatalf("got %+v", *p)
	}
	defer func() {
		r, _ := recover().(string)
		if !strings.Contains(r, "ecs: component not registered: ComponentID(") {
			t.Fatalf("panic = %q", r)
		}
	}()
	newColumn(60000)
}
