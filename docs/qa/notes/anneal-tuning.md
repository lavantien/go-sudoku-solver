# anneal tuning notes

Fixture origin: `Anneal81` in `internal/puzzle/fixtures.go` was derived on 2026-10-19 from the reference solution by removing cells in the deterministic order `(k*37+11) mod 81` while `solver.CountSolutions(grid, 2) == 1` held, stopping at 40 clues. The generator lives at `playground/make_fixture`. The derivation is re-pinned by `TestAnnealFixtureDerivesFromSolution`, which checks every clue against the reference solution and the clue count.

Rules carried from the corpus (dsa ch 41, simulated annealing and the metropolis rule, cooling schedules and restarts, stopping: portfolios and restart budgets):

- the rational metropolis rule: accept an uphill move with delta exactly when `below(T + delta) < T`, no floating point anywhere in the accept path.
- the geometric integral temperature ladder with a floor: `T = T0 >> (clock / span)`, floored at `TempEnd`.
- the restart portfolio: restart r seeds PCG with `(seed, r)`, banks only improve.

Adaptations for sudoku, each verified by sweep:

- temperatures are milli-units and the climb is scaled to match: the accept test is `IntN(T + delta*1000) < T` with `T` in milli-units. With raw units the coldest rung `T = 1` still accepts a delta 1 climb half the time and nothing ever freezes. Milli-units give a real cold tail (`T = 40` accepts a delta 1 climb at 3.8 percent).
- the ladder span is `iterations / halvings(T0, TempEnd)` so the halvings stretch across the whole budget. Without the division the ladder would finish in the first 4 percent of it.
- proposals are conflict-biased: pick a cell involved in a column or box duplicate, swap it with a random other free cell of its row. Uniform proposals floor higher on every schedule tried.
- one in five proposals is a row 3-cycle over three free cells, which preserves the row permutation invariant and unlocks double-conflict patterns that pairwise swaps cannot.
- the chapter reheat (reset to the bank, double T0, restart the clock after a stuck streak) was implemented and removed: with a reject-streak trigger the sudoku plateaus never fire it, and with a stagnation or non-improving trigger it measurably degrades every sweep because mid-run resets destroy the slow cooling. The evidence is in the git history of this note's commit window.

Defaults: restarts 4, iterations 150000 per restart, seed 1, TempStart 4000, TempEnd 100 (4.0 down to 0.1). The 40-clue fixture solves in about 0.06 s under these defaults, deterministically.

The 17-clue legacy sample is adversarial for local search: across eight seeds, five schedules, and budgets to 600000 iterations per restart, the walk floors at cost 2, one duplicate pair in each of two columns, and the fixing neighbours are givens. The anneal method therefore reports that sample unsolved with its deterministic best-effort grid, givens preserved, every row still a permutation. The deterministic solvers pin the sample's exact solution. The e2e golden normalizes the anneal grid block for the sample and records why.
