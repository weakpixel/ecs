package storage

// CompID identifies a registered component type. The registry hands out dense
// IDs starting at 0, so an ID can index a slice.
type CompID uint16

// Entity is the handle stored in an archetype's row-to-entity table. It has
// the same layout as ecs.Entity; package ecs converts field by field.
type Entity struct {
	ID         uint32
	Generation uint32
}
