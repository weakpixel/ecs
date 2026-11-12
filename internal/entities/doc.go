// Package entities tracks the live location of every entity in a World: a
// slot table keyed by entity ID, holding a generation counter (to detect
// stale handles) and a Record (which archetype and row currently store the
// entity).
//
// The package works entirely in terms of storage.Entity; package ecs converts
// to and from its own Entity type field by field. It imports nothing beyond
// the standard library and ecs/internal/storage.
package entities
