package system

import "github.com/weakpixel/ecs"

type commandKind uint8

const (
	cmdSpawn commandKind = iota
	cmdDespawn
	cmdDo
)

type command struct {
	kind   commandKind
	entity ecs.Entity
	values []any
	fn     func(*ecs.World)
}

// Commands buffers structural World changes so they can be applied after
// the systems that requested them have finished iterating. A Commands value
// is a World resource: get it with [CommandsOf]. [Schedule.Run] applies it
// after every stage.
type Commands struct {
	queue []command
}

// CommandsOf returns the World's Commands buffer, creating and registering
// it as a resource on first use.
//
// Panics with "system: nil world" if w is nil.
func CommandsOf(w *ecs.World) *Commands {
	if w == nil {
		panicf("nil world")
	}
	c := ecs.GetResourceOrNil[Commands](w)
	if c == nil {
		c = &Commands{}
		ecs.SetResource(w, c)
	}
	return c
}

// Spawn queues w.NewEntity(values...). The values are not copied, so the
// caller must not modify the slice's contents before the commands are
// applied. The variadic slice itself allocates.
func (c *Commands) Spawn(values ...any) {
	c.queue = append(c.queue, command{kind: cmdSpawn, values: values})
}

// Despawn queues w.DeleteEntity(e). If e is no longer alive when the command
// is applied (for example it was despawned twice in one stage) the command
// is skipped rather than panicking.
func (c *Commands) Despawn(e ecs.Entity) {
	c.queue = append(c.queue, command{kind: cmdDespawn, entity: e})
}

// Do queues fn to run with the World when the commands are applied. Use it
// for changes that have no dedicated command, such as SetComp. A closure that
// captures variables allocates; a function without captures does not.
func (c *Commands) Do(fn func(w *ecs.World)) {
	if fn == nil {
		panicf("nil command")
	}
	c.queue = append(c.queue, command{kind: cmdDo, fn: fn})
}

// Len reports the number of queued commands.
func (c *Commands) Len() int { return len(c.queue) }

// Apply runs the queued commands in the order they were queued, then empties
// the buffer. Commands queued by a command (via Do) while Apply runs are
// applied in the same call. The backing slice is reused, so steady-state
// Apply does not allocate.
//
// Apply must not be called while a view is iterating.
func (c *Commands) Apply(w *ecs.World) {
	// Index loop, not range: Do callbacks may append to c.queue.
	for i := 0; i < len(c.queue); i++ {
		cmd := c.queue[i]
		switch cmd.kind {
		case cmdSpawn:
			w.NewEntity(cmd.values...)
		case cmdDespawn:
			if w.IsAlive(cmd.entity) {
				w.DeleteEntity(cmd.entity)
			}
		case cmdDo:
			cmd.fn(w)
		}
	}
	// Zero the entries so applied values and closures can be collected,
	// then keep the capacity for the next tick.
	clear(c.queue)
	c.queue = c.queue[:0]
}
