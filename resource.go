package ecs

import "reflect"

// SetResource stores v as the World's singleton resource of type T,
// replacing any value set earlier. Resources are keyed by type, are not
// components, need no registration and are not attached to any entity.
func SetResource[T any](w *World, v *T) {
	if w.resources == nil {
		w.resources = make(map[reflect.Type]any)
	}
	w.resources[reflect.TypeFor[T]()] = v
}

// GetResourceOrNil returns the World's T resource, or nil if none is set.
func GetResourceOrNil[T any](w *World) *T {
	v, ok := w.resources[reflect.TypeFor[T]()]
	if !ok {
		return nil
	}
	return v.(*T)
}

// GetResource returns the World's T resource.
//
// Panics with "ecs: resource not set: <T>" if none is set.
func GetResource[T any](w *World) *T {
	v := GetResourceOrNil[T](w)
	if v == nil {
		panicResourceNotSet(reflect.TypeFor[T]())
	}
	return v
}

// HasResource reports whether a T resource is currently set.
func HasResource[T any](w *World) bool {
	_, ok := w.resources[reflect.TypeFor[T]()]
	return ok
}

// RemoveResource removes the World's T resource; a no-op if none is set.
func RemoveResource[T any](w *World) {
	delete(w.resources, reflect.TypeFor[T]())
}
