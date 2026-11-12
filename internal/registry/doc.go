// Package registry maps component types to dense IDs and keeps, per component,
// the reflect.Type and a column factory captured at registration.
//
// The factory is created inside [Register], the only place where the concrete
// component type is known; later code that only holds an ID (archetype
// creation, values passed as any) builds columns through [Info.NewColumn].
//
// A Registry is safe for concurrent use. Reads are lock-free; see [Registry].
package registry
