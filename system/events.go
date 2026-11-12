package system

import (
	"iter"

	"github.com/weakpixel/ecs"
)

// Events is a World's queue of events of type T: systems [Events.Send]
// values and others read them with an [EventReader]. It is a World resource:
// get it with [EventsOf].
//
// An event can be read until the end of the tick after the one it was sent
// in; then it is dropped. [Schedule.Run] ages every queue of the World once,
// after its last stage. So a reader that runs after the sender sees the
// event in the same tick, and one that runs before it in the next tick.
// Events sent between Runs (setup code, a callback) count as part of the
// next tick.
//
// Events are plain values: an [ecs.Entity] in an event may no longer be
// alive when a reader sees it. Events are never saved.
type Events[T any] struct {
	// prev holds the events of the previous tick, cur those of this tick.
	// Aging swaps the two slices and empties the new cur, so the backing
	// arrays are reused and steady-state Send does not allocate.
	prev, cur []T
	// start is the sequence number of prev[0]. Sequence numbers count every
	// event ever sent, so a reader can tell which ones it has seen.
	start uint64
}

// eventQueue is an Events[T] of any T, as the aging in Schedule.Run sees it.
type eventQueue interface{ update() }

// eventQueues lists the World's Events queues so Schedule.Run can age them
// without knowing their types. It is an unexported World resource.
type eventQueues struct{ list []eventQueue }

// update ages every queue.
func (q *eventQueues) update() {
	for _, e := range q.list {
		e.update()
	}
}

// updateEvents ages the World's queues; a no-op if it has none.
func updateEvents(w *ecs.World) {
	if q := ecs.GetResourceOrNil[eventQueues](w); q != nil {
		q.update()
	}
}

// EventsOf returns the World's queue of T events, creating and registering
// it as a resource on first use.
//
// Panics with "system: nil world" if w is nil.
func EventsOf[T any](w *ecs.World) *Events[T] {
	if w == nil {
		panicf("nil world")
	}
	e := ecs.GetResourceOrNil[Events[T]](w)
	if e == nil {
		e = &Events[T]{}
		ecs.SetResource(w, e)
		q := ecs.GetResourceOrNil[eventQueues](w)
		if q == nil {
			q = &eventQueues{}
			ecs.SetResource(w, q)
		}
		q.list = append(q.list, e)
	}
	return e
}

// Send adds ev to the queue. Readers see it until the end of the next tick.
// Send does not allocate once the queue has grown to its usual size.
func (e *Events[T]) Send(ev T) { e.cur = append(e.cur, ev) }

// Len is the number of events in the queue: those of this tick and of the
// previous one.
func (e *Events[T]) Len() int { return len(e.prev) + len(e.cur) }

// end is the sequence number the next sent event gets.
func (e *Events[T]) end() uint64 { return e.start + uint64(len(e.prev)+len(e.cur)) }

// at returns the event with sequence number seq, which must be in
// [start, end).
func (e *Events[T]) at(seq uint64) T {
	i := int(seq - e.start)
	if i < len(e.prev) {
		return e.prev[i]
	}
	return e.cur[i-len(e.prev)]
}

// update drops the previous tick's events and starts a new tick.
func (e *Events[T]) update() {
	e.start += uint64(len(e.prev))
	clear(e.prev) // drop references held by the dropped events, keep capacity
	e.prev, e.cur = e.cur, e.prev[:0]
}

// EventReader reads a World's T events. Each reader sees each event at most
// once; several readers of the same type each see every event. A reader that
// does not read for more than one tick misses the dropped events silently.
// Keep one reader per system, created in its Init.
type EventReader[T any] struct {
	events *Events[T]
	// next is the sequence number of the first event not read yet.
	next uint64
}

// NewEventReader returns a reader of w's T events, creating the queue if
// needed. It starts with the oldest event still in the queue.
//
// Panics with "system: nil world" if w is nil.
func NewEventReader[T any](w *ecs.World) *EventReader[T] {
	e := EventsOf[T](w)
	return &EventReader[T]{events: e, next: e.start}
}

// Read yields the events this reader has not seen yet, oldest first. An
// event counts as read once it is yielded, so breaking out of the loop
// leaves the rest for the next Read. Events sent while the loop runs are
// yielded too. Read does not allocate.
func (r *EventReader[T]) Read() iter.Seq[T] {
	return func(yield func(T) bool) {
		e := r.events
		r.next = max(r.next, e.start)
		for r.next < e.end() {
			ev := e.at(r.next)
			r.next++
			if !yield(ev) {
				return
			}
		}
	}
}
