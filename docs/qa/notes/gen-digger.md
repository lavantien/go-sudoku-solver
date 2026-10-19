# gen digger notes

Fixture origin: the clue-stall table quoted in the release notes was measured on 2026-10-19 at commit d5a6250. The binary was rebuilt from that commit with CGO_ENABLED=1 through `go build -o dist/sudoku.exe ./cmd/sudoku`. Each run executed `dist/sudoku.exe gen --clues 17 --seed N` for N from 1 through 8 on Windows.

Counting method: `gen` prints only the 9-line puzzle grid, one digit per clue and one `.` per blank (`internal/cli/gen.go`), so the clue count is the number of digit characters in stdout.

Target 17 results, one run per seed: 22, 24, 24, 23, 24, 25, 24, and 25 clues, 5 to 8 above the target. The digger is seeded, so a rerun at the same commit reproduces every grid byte for byte.

Default target check: the clue flag defaults to 30 (`internal/config/config.go`), and `dist/sudoku.exe gen --seed N` without the clue flag landed on exactly 30 clues for all of seeds 1 through 8 in the same session.
