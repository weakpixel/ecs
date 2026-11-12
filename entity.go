package ecs

import "github.com/weakpixel/ecs/internal/storage"

// EntityID is the slot index of an Entity. It is reused after the entity is
// deleted; Entity.Generation is what tells a stale handle from a live one.
type EntityID uint32

// Entity is a handle to one entity in a World. It is a small comparable
// value: copy it freely, store it in maps, slices and struct fields.
//
// A handle becomes stale once its entity is deleted, even if the slot is
// later reused for a new entity (the new entity gets a different
// Generation). The zero Entity{} is never a live handle, so it doubles as
// "no entity" in struct fields: World.IsAlive(Entity{}) is false, and other
// World methods panic with "ecs: invalid or stale entity".
type Entity struct {
	ID         EntityID
	Generation uint32
}

// toStorage converts e to storage.Entity. The two types share the same
// field layout; this is a value copy, not a representation change, and is
// free once inlined.
func (e Entity) toStorage() storage.Entity {
	return storage.Entity{ID: uint32(e.ID), Generation: e.Generation}
}

// entityFromStorage converts se to Entity, field by field.
func entityFromStorage(se storage.Entity) Entity {
	return Entity{ID: EntityID(se.ID), Generation: se.Generation}
}
