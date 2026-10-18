# qa baselines

Pinned fixtures, step semantics, constants, and golden rules the test suite relies on. Every claim here is checked by a test named alongside it, so drift fails the build first.

## fixture origins

- `Sample81` and `SampleSolution81` (`internal/puzzle/fixtures.go`): the 17-clue puzzle and its solution shipped in the legacy repo's README and read by the legacy `main.go`, captured from the main branch on 2026-10-19. The solution string was cross-checked against Norvig's published solution for this puzzle and matches. The sample solves by singles alone in exactly 64 propagate steps, the derivation is in [notes/features-sample.md](notes/features-sample.md).
- `Unsolvable81`: two 5s in row 1, the minimal immediate-conflict input for the unsolved path.
- `Anneal81`: derived on 2026-10-19 from the reference solution by deterministic cell removal, 40 clues, generator kept at `playground/make_fixture`, full origin and tuning evidence in [notes/anneal-tuning.md](notes/anneal-tuning.md).
- `SampleExtreme81` and `SampleExtremeSolution81`: evolved on 2026-10-19 by `tools/evolve` under `make evolve`, deterministic seed 20261019, winner at 21 clues and 4646 propagate steps, full run ledger in [notes/evolve-run.md](notes/evolve-run.md).

## step semantics

One step means a different unit of work per method, all four pinned by tests:

- propagate: every digit placement, constraint-forced singles and search guesses alike. The classic sample pins at 64, all singles, the extreme sample pins at 4646.
- cover: every row selected from the exact cover matrix, 729 rows by 324 columns, 81 cells x 9 digit choices against 4 x 81 constraint columns for cell, row-unit, column-unit, box-unit membership. Node visits cap at `SUDOKU_COVER_NODE_CAP`, default 1048576.
- anneal: every proposal evaluated, each delta computation counts whether the move is taken or rejected.
- race: the sum of the step counts of every racer, winner, losers, and unwound stragglers alike.

## the race contract

`Race` defaults its racers to propagate, cover, anneal in that order, and the policy layer may reorder them, see below. With `SUDOKU_WORKERS = 1` it runs them sequentially and returns the first solved result with the running step sum. Otherwise each racer runs in its own goroutine on a cancel context derived from the caller's. The first solved result to arrive cancels the context, racers still running unwind on that context and post their partial results into a buffered channel sized for the full field, and the reported steps are the sum over every received result. The winner is the first received result, replaced by the first solved one if a later arrival solved. No straggler's work is silently dropped and none of it keeps burning after the race is decided. This contract, including the unwind behavior on winner cancel, is pinned by the race tests in `internal/solver`.

## the policy table

`policy.ResolveOrder(grid)` scores the three racer methods with the committed `policy.TableWeights`, a 3 by 5 least-squares fit over the feature vector `[clues, PropagationFill, unitMin, unitMax, unitSum/27]` derived in [notes/features-sample.md](notes/features-sample.md), and orders the racers ascending by score, cheapest first. The weights regenerate deterministically with `make train`, which reruns `tools/train` through `go generate`. If the table's row count ever stops matching the racer count, `Order` falls back to `FixedOrder`, propagate, cover, anneal, rather than racing under a broken fit.

## anneal defaults

Restarts 4, iterations 150000 per restart, seed 1, TempStart 4000, TempEnd 100, all overridable through `SUDOKU_ANNEAL_*`. How each default was earned, and why the 17-clue classic sample is adversarial for local search, floors at cost 2 with the fixing neighbours being givens, is recorded in [notes/anneal-tuning.md](notes/anneal-tuning.md).

## golden normalization rules

The e2e report tests in `internal/app/run_test.go` compare against goldens under two deterministic-only normalizations, machine speed and scheduler interleaving must not change a golden:

- `Step count: [0-9]+` rewrites to `Step count: 0`.
- `Execution time: [0-9]+\.[0-9]+` rewrites to `Execution time: 0.000000`.

The full-report golden runs on `Anneal81`, where all four methods deterministically reach the identical solution render. The classic-sample exact-blocks test pins the header and every method's render to `SampleSolution81` for propagate, cover, and race, and deliberately skips the render assertion for the anneal block only, because anneal deterministically reports the 17-clue sample unsolved with its best-effort grid. The skipped assertion is a recorded contract, see the anneal tuning note for the analysis behind it.
