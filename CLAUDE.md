# ecs

An archetype-based Entity Component System for Go: module `github.com/weakpixel/ecs`, Go 1.27, MIT. Package `ecs` at the repository root (`World`, entities, components, views, queries, resources), the optional package `system` (`Schedule`, `Commands`, events), and the arity generator `internal/gen`.

## The docs are the spec

`doc/` holds the guide ([doc/README.md](doc/README.md) is the entry point; `api-generic.md`, `api-low-level.md`, `api-systems.md`, `performance-guide.md`) and [doc/design.md](doc/design.md) (how it is built and why). Implement against the guide. Every topic has one home; link to it instead of repeating it.

- If the code and a doc disagree, don't silently pick one. Fix the code, or if the doc is wrong or underspecified, update the doc in the same change and say so.
- Behavior the docs leave open: decide, implement, test, and add it to the doc.
- Panic messages in the panic tables are API. Tests assert them with `strings.Contains`.

## Layout

| Path | What | Skill |
| --- | --- | --- |
| `*.go` (root) | Package `ecs`: `World`, entities, components, views, queries, resources, panics | `ecs-core` |
| `internal/registry`, `internal/entities`, `internal/storage` | Component registry, entity table, archetype storage | `ecs-core` |
| `generic.go`, `view.go`, `internal/gen`, generated `zz_gen_generic-api.go` | Arity-N generic API (`NewEntityN`, `NewViewN`/`ViewN`, `ComponentsN`) | `ecs-arity-gen` |
| `system/` | `Schedule`, `Commands`, events | [api-systems.md](doc/api-systems.md) |
| `deps_test.go` | Enforces stdlib only and no `require` | |

## Commands

```bash
make all                      # go generate, fmt, vet, test -race
make wasm-test                # go vet and go test under GOOS=js GOARCH=wasm (Node)
go test ./...                 # quick
go test ./internal/gen/       # generated files are up to date
make bench name=<label>       # benchmarks -> build/bench/<label>.txt
make benchcmp new=<label>     # benchstat vs build/bench/baseline.txt
```

While iterating, run only the tests of the packages you changed. Run `make all` and `make wasm-test` once before opening a PR; CI runs both and fails on a generate or gofmt diff.

## Hard rules

- **Standard library only.** `go.mod` has no `require`, and no package imports anything outside the standard library and this module. `deps_test.go` enforces it.
- **The repository is self-contained.** Code, comments, docs, skills, commits and PRs describe this module on its own terms. Examples use a neutral host loop (a plain `for` loop, a server tick, a test), not a particular framework or program that uses the module.
- **Panics, not errors**, for programmer errors. Every message starts with `ecs: ` (`system: ` in package `system`) and the stable texts are in the panic tables of the guide.
- **The `World` is single-goroutine.** The component registry is process-global and must be safe for concurrent `RegisterComp` (package `init`s, parallel tests).
- **Hot paths are 0 allocs/op:** `View.Each`, `GetComp[T]`, `SetComp[T]` (in place and when moving), `NewEntityN`, `View.NewEntity`, `View.Get`, `View.Set` (in place and when moving), steady-state `Schedule.Run`. Enforce this with `testing.AllocsPerRun` tests, not by eye.
- **Never hand-edit a generated file** (`zz_gen_generic-api.go`, `zz_gen_generic-api_test.go`). Change `internal/gen` and run `go generate ./...`.
- **Build outputs go into `build/`** (`go build -o build/<name>`, benchmark runs, profiles). `build/` is ignored. Never commit binaries or `*.test` files. Before every commit run `git status --short` and `git diff --cached --stat`, and stage files by name, not with `git add -A` or `git add .`.
- **Versions stay 0.** No tag or release without the maintainer's word, and no backward compatibility shims before a release.
- **No attribution and no names:** no Co-Authored-By, Claude-Session or "Generated with" lines in commits, PRs, docs or code; never mention a person's name in any document, commit or PR; no hidden HTML markers or comments anywhere.

## Code quality

- **Readability first.** Clear names, small functions, straightforward control flow. An optimization that makes code harder to follow must earn its place with a benchmark (see `ecs-benchmarks`).
- **Model the domain; no master struct.** Each concept has its own type and file or package (registry, entity table, archetypes and columns, resources). `World` composes them and delegates. New unexported packages go under `internal/`, so the public API stays in `ecs`.
- **Document the public API.** Every exported identifier has a doc comment that states what it does, its contract (preconditions, ownership, pointer validity, cost) and which panics it can raise.
- **Explain performance tricks inline** (unsafe pointer arithmetic, swap-remove, cached indices, avoiding escapes, preallocation, bounds-check hints), so a reader does not "simplify" them away.
- **Keep generated code thin.** The logic lives in hand-written, tested unexported helpers that the generated code calls.

## Conventions

- File names are kebab-case: `world-comp.go`, `view-filter.go`, `world-comp_test.go`.
- Tests use the standard `testing` package only (no testify), table-driven with `t.Run`, with a `mustPanic(t, want, f)` helper that matches a substring.
- Use modern Go: `iter.Seq2`, range-over-func, `for i := range n`, `reflect.TypeFor[T]()`, `slices`/`maps`.

## Workflow

Non-trivial work goes through `/plan` (writes one plan file, no code) and then `/implement` (carries it out step by step and keeps the plan file up to date). Small, obvious requests don't need a plan.

| Skill | Use when |
| --- | --- |
| `plan` | User runs `/plan` |
| `implement` | User runs `/implement` |
| `ecs-core` | Storage, entities, archetypes, `World`, `SetComp`/`RemoveComp`, queries, resources, panics |
| `ecs-arity-gen` | `internal/gen`, `generic.go`, `zz_gen_generic-api.go` |
| `ecs-benchmarks` | Benchmarks, alloc tests, performance claims |
| `doc-writer` | Any documentation change |
