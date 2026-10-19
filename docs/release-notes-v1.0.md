# release notes v1.0

## what shipped

- a solver portfolio of four methods behind one registry: constraint propagation with MRV backtracking, exact cover over the boolean incidence matrix, simulated annealing with a restart portfolio, and a parallel portfolio race with mined ordering.
- the race runs each racer on its own goroutine under a cancel context, the first solved result cancels the field, stragglers unwind and their partial work still counts toward the reported steps.
- the ordering is mined: a least-squares fit over the feature vector [clues, propagation fill, min and max unit clue load, mean unit clue load] scores the racers per grid, and `make train` regenerates the committed weights deterministically.
- a web ui and a JSON API over one server: htmx pages for solving, generating, history, and run detail at the root routes, versioned endpoints under `/api/v1`, both persisting runs and generated puzzles into sqlite with WAL, seeded with the classic sample on an empty database.
- the cli contract preserved: bare `sudoku` or `sudoku solve` still reads 9 lines from stdin and prints the method report on stdout, joined by `gen`, `serve`, `version`, `coverage`, and `screenshot` subcommands.
- a quality chain wired into make: fmt, vet, lint, typecheck, a 1000-line sloc ceiling, race, fuzz over every input surface, mutation testing where surviving mutants fail the build, a memory-capped e2e pass, and `make verify` chaining the whole sequence.
- an extreme complexity fixture evolved for step-count stress: 21 clues forcing 4646 propagate steps, 72.6 times the classic sample's 64, with the run ledger and benchmarks in [qa/notes/evolve-run.md](qa/notes/evolve-run.md).

## scope notes

- anneal is best-effort on adversarial inputs. The 17-clue classic sample floors at cost 2 across every schedule and budget tried, one duplicate pair in each of two columns with the fixing neighbours being givens, so the anneal method deterministically reports that sample unsolved with a best-effort grid. The deterministic methods pin the exact solution, and the e2e goldens record the anneal exception. Evidence in [qa/notes/anneal-tuning.md](qa/notes/anneal-tuning.md).
- the policy table optimizes step-efficiency, not wall-clock. The weights fit observed step counts, so the ordering prefers the racer expected to take fewer steps, which correlates with but does not guarantee the fastest finish on any single grid.
- 17-clue generation is out of reach for the greedy digger. One random uniqueness-checked removal pass stalls well above target: at clue target 17, seeds 1 through 8 landed on 22, 24, 33, 32, 33, 34, 33, and 34 clues. The default target of 30 is where the digger lands reliably.
- exact cover node visits cap at 1048576 by default (`SUDOKU_COVER_NODE_CAP`), so a pathological grid returns unsolved at the cap rather than running unbounded.
- windows and linux binaries ship for amd64 and arm64. darwin builds from source with `go build`: the pinned zig cc carries only libSystem for darwin and no stubs for the system frameworks the go runtime links.
