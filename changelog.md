# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-10-19

### Added

- Constraint propagation solver over bitmask candidates with MRV backtracking.
- Exact cover solver over the boolean incidence matrix, 729 rows by 324 columns, with a configurable node cap.
- Simulated annealing solver with a restart portfolio and integer-only metropolis accept path.
- Parallel portfolio race method with policy-mined racer ordering: a least-squares weight table over the grid feature vector scores the racers, regenerable with `make train`.
- htmx web ui (solve, generate, history, run detail) and JSON API at `/api/v1`, both persisting to sqlite with WAL and seeding the classic sample on an empty database.
- `sudoku gen` with clue target and deterministic seed, `sudoku serve`, `sudoku version`, `sudoku coverage`, `sudoku screenshot`.
- Extreme complexity fixture evolved by `tools/evolve`: 21 clues forcing 4646 propagate steps.
- Quality chain targets: `race`, `fuzz`, `mutate`, `memtest`, `e2e`, `verify`, `coverage`, `screenshots`, `diagrams`.

### Changed

- Replaced the legacy method set (backtracking, A-star, ant colony, minimax) with constraint propagation over bitmask candidates, exact cover over the boolean incidence matrix, simulated annealing with a restart portfolio, and a parallel portfolio race.
- Input errors now print to stderr and exit with code 1; the happy-path stdout format is unchanged.
