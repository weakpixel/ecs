---
name: ecs-benchmarks
description: Use to add or run ECS benchmarks/allocation tests, investigate performance, compare with benchstat, or update performance-guide results.
---

# Benchmarks and allocation guarantees

## What must hold

From `doc/performance-guide.md` and `doc/api-generic.md`:

| Operation | Guarantee |
| --- | --- |
| View iteration (`NewViewN(...).Each()` loop) | 0 allocs/op |
| `GetComp[T]`, `GetCompOrNil[T]`, `HasComp[T]` | 0 allocs/op |
| `SetComp[T]`, in place **and** when it moves the entity | 0 allocs/op |
| `NewEntityN`, `ViewN.NewEntity` (steady state, after capacity is reached) | 0 allocs/op |
| `ViewN.Get`/`Has`, `ViewN.Set` (in place and moving), `ViewN.Remove` | 0 allocs/op |
| `GetResource[T]`, `GetResourceOrNil[T]` | 0 allocs/op |
| `View7.NewEntity` vs `NewEntity7` | ≈3–4× faster (machine-specific; target quoted in the doc) |
| `ViewN.Get` vs the same `GetComp[T]` calls | ≈1.25× faster (target quoted in the doc) |

The low-level `World.NewEntity`/`World.SetComp(e, values ...any)` may allocate (boxing), and that's documented.

## Alloc tests (run under plain `go test`, unlike benchmarks)

The root package's are in `world-alloc_test.go`; `system` and `internal/*` have their own. They are the regression net, so a benchmark alone isn't enough.

```go
func TestTypedIterNoAllocs(t *testing.T) {
    if testing.CoverMode() != "" || raceEnabled {
        t.Skip("instrumentation allocates")
    }
    w := newBenchWorld(1000)
    v := ecs.NewView2[Position, Velocity](w)
    for range v.Each() {} // warm the archetype cache
    if n := testing.AllocsPerRun(100, func() {
        for _, c := range v.Each() { c.Comp1.X += c.Comp2.X }
    }); n != 0 {
        t.Fatalf("allocs = %v, want 0", n)
    }
}
```

Get `raceEnabled` from a pair of files with `//go:build race` and `//go:build !race`. Cover every row of the table above.

## Benchmark file layout (`benchmark_test.go`)

The module has no benchmarks yet. When adding them, put them in `benchmark_test.go` at the root and include at least:
- `BenchmarkNewEntity7` and `BenchmarkViewNewEntity` (a `View7`)
- `BenchmarkGetComp` (N separate `GetComp[T]` calls) and `BenchmarkViewGet` with the same N
- `BenchmarkView2Each` (typed) and `BenchmarkWorldQueryEach` (untyped `[]any`), with 1k and 10k entities across several archetypes
- `BenchmarkSetCompInPlace` and `BenchmarkSetCompMove` (add, then remove)
- `BenchmarkDeleteEntity`
- `BenchmarkFragmented` (the same query over ~100 small archetypes, which backs perf-guide practice 7)

Style:
- Build the world outside the timer, then call `b.ReportAllocs()` and `b.ResetTimer()`.
- Use `for b.Loop()` (Go 1.24+) instead of `for i := 0; i < b.N; i++`. It keeps the body from being optimized away and handles timer setup.
- For create benchmarks, recreate the world every K iterations with the timer stopped, or delete as you go, so memory doesn't grow without bound.
- Use realistic component sizes (16–64 bytes) and a mix of archetypes. A single archetype flatters iteration.

## Running and comparing

```bash
make bench name=baseline          # once, on a quiet machine → build/bench/baseline.txt
# ... change ...
make bench name=<label> && make benchcmp new=<label>
```

- `benchstat` must be installed (`go install golang.org/x/perf/cmd/benchstat@latest`). Ask before installing.
- Only compare runs from the same machine, with nothing else running. The guide puts noise at ±15–35% otherwise. Treat a change as real only when benchstat reports p < 0.05 and the delta is larger than the noise.
- Benchmark runs stay in `build/bench/` (ignored). Never overwrite `build/bench/baseline.txt` without asking.

## Before claiming an allocation or speed win

1. `go build -gcflags=-m . 2>&1 | grep -E 'escapes|moved to heap'` on the changed functions. Synthetic benchmark call sites can be stack-allocated and hide real costs.
2. Make sure the alloc test exists and passes.
3. Show the benchstat output in your report.
4. If a number quoted in `doc/performance-guide.md` changed meaningfully, update the doc and say it's machine-specific.

## Profiling a slow path

```bash
go test . -run '^$' -bench 'View2Each' -cpuprofile build/cpu.out -memprofile build/mem.out
go tool pprof -top -nodecount 25 build/cpu.out
```

Write profiles to `build/` or the scratchpad, never the repo root, and don't commit them.
