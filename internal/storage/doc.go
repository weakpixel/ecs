// Package storage holds the archetype tables behind package ecs: typed
// component columns, archetypes (rows of entities that share one component
// set) and an append-only index of archetypes with cached add and remove
// edges.
//
// The package knows nothing about the component registry or the World. Callers
// inject a column factory into [NewIndex] and keep their own entity records,
// using the moved-entity results of row operations to fix them up.
//
// Nothing here is safe for concurrent use; an Index belongs to one World.
package storage
