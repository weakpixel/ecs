// Package ecs is an archetype-based Entity Component System.
//
// Components are plain non-pointer structs. The component registry is
// process-global and safe for concurrent use; a World is single-goroutine.
// Programmer errors panic with messages starting with "ecs: ".
package ecs
