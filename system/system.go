// Package system runs game logic ("systems") against an [ecs.World].
//
// A system is a plain function. A [Schedule] groups systems into ordered
// stages and applies deferred structural changes ([Commands]) between
// stages, so systems can queue spawns and despawns while iterating a view.
//
// The package is deliberately separate from package ecs: the World knows
// nothing about systems, and nothing here knows about rendering or any
// game framework. A host loop (a game's update, a server tick, a test)
// owns the timing and calls [Schedule.Run].
package system

import (
	"time"

	"github.com/weakpixel/ecs"
)

// Context is what a [Schedule] hands to every system call. It carries the
// World and the per-tick data a system commonly needs, and is the place to
// add more of it later without changing the [System] signature.
//
// Rule for new fields: only data that is owned and refreshed by the Schedule
// each tick belongs here. State owned by the game (settings, score, input
// devices) lives in World resources.
//
// Lifetime: the Schedule reuses one Context for all of its calls, which is
// what keeps Run free of allocations. The pointer is valid only for the
// duration of the call it is passed to. Systems must not keep it, or its
// Commands or Time, after returning; keep ctx.World if the World is needed
// later.
type Context struct {
	// World is the World the schedule runs against.
	World *ecs.World
	// Time describes the current tick.
	Time Time
	// Commands is the World's deferred-change buffer, the same one
	// [CommandsOf] returns. The Schedule applies it after every stage.
	Commands *Commands
}

// Time is the timing of one [Schedule.Run].
type Time struct {
	// Delta is the duration of the current tick, as passed to Run.
	Delta time.Duration
	// Tick is the index of the current tick: 0 for the first Run of the
	// schedule, then 1, 2, and so on.
	Tick uint64
}

// System is one unit of game logic. It runs on the World's goroutine and may
// iterate views, read and write components and use resources. Structural
// changes made while a view is iterating panic in package ecs, so systems
// queue them with ctx.Commands instead. See [Context] for the lifetime of
// ctx.
type System func(ctx *Context)

// Runner is anything a [Schedule] can run. Use it for systems that keep state
// (prebuilt views, scratch buffers, configuration) in a struct:
//
//	type Movement struct{ view ecs.View2[Position, Velocity] }
//	func (m *Movement) Run(ctx *system.Context) { ... }
//
// A [System] is itself a Runner, so plain functions and structs mix freely.
type Runner interface {
	Run(ctx *Context)
}

// Initializer is an optional interface for a [Runner] that needs one-time
// setup against the World: setting resources, queueing initial spawns with
// ctx.Commands, or checking that a required resource exists. A [Schedule]
// calls Init once per runner, at the start of the first [Schedule.Run] that
// follows adding it, before any stage runs. ctx.Time describes that first
// tick. Plain [System] functions cannot implement it.
type Initializer interface {
	Init(ctx *Context)
}

// Run calls f(ctx). It makes every System a [Runner].
func (f System) Run(ctx *Context) { f(ctx) }

func panicf(msg string) {
	panic("system: " + msg)
}
