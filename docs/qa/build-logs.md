# build logs

Tails of the latest full `make verify` and `make race` runs on the development machine.

```text
$ make verify
ok  	github.com/lavantien/go-sudoku-solver/tools/mutator	31.895s
ok  	github.com/lavantien/go-sudoku-solver/tools/train	(cached)
?   	github.com/lavantien/go-sudoku-solver/web	[no test files]

CGO_ENABLED=1 GOMEMLIMIT=512MiB go test  -count=1 -timeout 15m ./internal/solver ./internal/e2e
ok  	github.com/lavantien/go-sudoku-solver/internal/solver	16.432s
ok  	github.com/lavantien/go-sudoku-solver/internal/e2e	18.600s

golangci-lint run
0 issues.

sloc ok

fuzz: elapsed: 21s, execs: 1184 (342/sec), new interesting: 2 (total: 8)
PASS
ok  	github.com/lavantien/go-sudoku-solver/internal/solver	21.874s

progress 650/650
slots=8 mutants=650 killed=570 invalid=80 survived=0

--- PASS: TestUISolveJourney (4.33s)
PASS
ok  	github.com/lavantien/go-sudoku-solver/internal/e2e	39.660s

$ make race
ok  	github.com/lavantien/go-sudoku-solver/internal/solver	32.358s
ok  	github.com/lavantien/go-sudoku-solver/internal/store	2.872s
ok  	github.com/lavantien/go-sudoku-solver/internal/server	102.529s
ok  	github.com/lavantien/go-sudoku-solver/internal/cli	35.527s
ok  	github.com/lavantien/go-sudoku-solver/internal/e2e	26.833s
```
