package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/matmul"
	"github.com/lavantien/go-sudoku-solver/internal/policy"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

type labeled struct {
	features []float64
	winner   int
}

func main() {
	os.Exit(cli(config.Load()))
}

func cli(cfg config.Config) int {
	logger := logging.New(cfg.LogLevel)
	if err := run(logger, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "train:", err)
		return 1
	}
	return 0
}

func run(logger *slog.Logger, cfg config.Config) error {
	logging.Enter(logger, "train", "seeds", cfg.Train.Seeds)
	samples, err := corpus(logger, cfg)
	if err != nil {
		return err
	}
	weights, err := fit(logger, samples, cfg.MatmulCutoff, cfg.Train)
	if err != nil {
		return err
	}
	if err := emit(logger, weights, cfg); err != nil {
		return err
	}
	logging.Exit(logger, "train", "methods", len(weights))
	return nil
}

func clueTarget(seed uint64, t config.TrainConfig) int {
	return t.ClueBase + int(seed*uint64(t.ClueStride)%uint64(t.ClueSpan))
}

func racerMethods(count int) []solver.Method {
	racers := make([]solver.Method, count)
	copy(racers, solver.All()[:count])
	return racers
}

func corpus(logger *slog.Logger, cfg config.Config) ([]labeled, error) {
	logging.Enter(logger, "corpus", "seeds", cfg.Train.Seeds, "clues", cfg.Train.ClueBase, "to", cfg.Train.ClueBase+cfg.Train.ClueSpan-1)
	racers := racerMethods(cfg.Train.RacerCount)
	raceCfg := config.Defaults()
	out := make([]labeled, 0, cfg.Train.Seeds)
	totals := make([]time.Duration, cfg.Train.RacerCount)
	for seed := uint64(1); seed <= uint64(cfg.Train.Seeds); seed++ {
		target := clueTarget(seed, cfg.Train)
		gen, err := puzzle.Generate(target, seed, solver.CountSolutions)
		if err != nil {
			err = fmt.Errorf("seed %d: %w", seed, err)
			logging.Fail(logger, "corpus", err, "target", target)
			return nil, fmt.Errorf("corpus: %w", err)
		}
		winner, err := raceLabel(racers, gen.Grid, raceCfg, totals)
		if err != nil {
			err = fmt.Errorf("seed %d: %w", seed, err)
			logging.Fail(logger, "corpus", err, "clues", gen.Clues)
			return nil, fmt.Errorf("corpus: %w", err)
		}
		out = append(out, labeled{features: policy.Features(gen.Grid), winner: winner})
	}
	logging.Exit(logger, "corpus", "samples", len(out),
		"propagate_ms", totals[0].Milliseconds(),
		"cover_ms", totals[1].Milliseconds(),
		"anneal_ms", totals[2].Milliseconds())
	return out, nil
}

func raceLabel(racers []solver.Method, g sudoku.Grid, cfg config.Config, totals []time.Duration) (int, error) {
	winner := -1
	best := math.MaxInt
	for i, m := range racers {
		res := m.Fn(context.Background(), g, cfg)
		totals[i] += res.Dur
		if !res.Solved || res.Steps >= best {
			continue
		}
		winner = i
		best = res.Steps
	}
	if winner < 0 {
		return 0, errors.New("no racer solved the generated grid")
	}
	return winner, nil
}

func fit(logger *slog.Logger, samples []labeled, cutoff int, t config.TrainConfig) ([][]float64, error) {
	logging.Enter(logger, "fit", "samples", len(samples), "cutoff", cutoff)
	width := len(samples[0].features)
	a := matmul.Dense{N: width, Data: make([]float64, width*width)}
	xty := make([]float64, width*t.RacerCount)
	for _, s := range samples {
		for i, fi := range s.features {
			for j, fj := range s.features {
				a.Data[i*width+j] += fi * fj
			}
			for m := range t.RacerCount {
				loss := 0.0
				if m != s.winner {
					loss = 1.0
				}
				xty[i*t.RacerCount+m] += fi * loss
			}
		}
	}
	ridge := matmul.Dense{N: width, Data: make([]float64, width*width)}
	for i := range width {
		ridge.Data[i*width+i] = t.RidgeLambda
	}
	a = matmul.Add(a, ridge)
	inv, err := invert(a, t.PivotFloor)
	if err != nil {
		logging.Fail(logger, "fit", err, "ridge", t.RidgeLambda)
		return nil, fmt.Errorf("fit: %w", err)
	}
	check := matmul.MulStrassen(a, inv, cutoff)
	for i := range width {
		for j := range width {
			want := 0.0
			if i == j {
				want = 1.0
			}
			if math.Abs(check.Data[i*width+j]-want) > t.InverseTol {
				err := fmt.Errorf("inverse check failed at %d,%d: %.6g", i, j, check.Data[i*width+j])
				logging.Fail(logger, "fit", err)
				return nil, fmt.Errorf("fit: %w", err)
			}
		}
	}
	weights := make([][]float64, t.RacerCount)
	for m := range t.RacerCount {
		weights[m] = make([]float64, width)
		for i := range width {
			for j := range width {
				weights[m][i] += inv.Data[i*width+j] * xty[j*t.RacerCount+m]
			}
		}
	}
	logging.Exit(logger, "fit", "ridge", t.RidgeLambda, "features", width)
	return weights, nil
}

func invert(a matmul.Dense, pivotFloor float64) (matmul.Dense, error) {
	n := a.N
	cols := 2 * n
	m := make([]float64, n*cols)
	for i := range n {
		copy(m[i*cols:i*cols+n], a.Data[i*n:(i+1)*n])
		m[i*cols+n+i] = 1
	}
	for col := range n {
		pivot := col
		for r := col + 1; r < n; r++ {
			if math.Abs(m[r*cols+col]) > math.Abs(m[pivot*cols+col]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot*cols+col]) < pivotFloor {
			return matmul.Dense{}, fmt.Errorf("singular normal matrix at pivot %d", col)
		}
		if pivot != col {
			for j := range cols {
				m[col*cols+j], m[pivot*cols+j] = m[pivot*cols+j], m[col*cols+j]
			}
		}
		d := m[col*cols+col]
		for j := range cols {
			m[col*cols+j] /= d
		}
		for r := range n {
			if r == col {
				continue
			}
			f := m[r*cols+col]
			if f == 0 {
				continue
			}
			for j := range cols {
				m[r*cols+j] -= f * m[col*cols+j]
			}
		}
	}
	out := matmul.Dense{N: n, Data: make([]float64, n*n)}
	for i := range n {
		copy(out.Data[i*n:(i+1)*n], m[i*cols+n:(i+1)*cols])
	}
	return out, nil
}

func emit(logger *slog.Logger, weights [][]float64, cfg config.Config) error {
	logging.Enter(logger, "emit")
	path, err := tablePath(cfg)
	if err != nil {
		logging.Fail(logger, "emit", err)
		return fmt.Errorf("emit: %w", err)
	}
	if err := os.WriteFile(path, render(weights), cfg.Train.TablePerm); err != nil {
		logging.Fail(logger, "emit", err, "path", path)
		return fmt.Errorf("emit: %w", err)
	}
	logging.Exit(logger, "emit", "path", path)
	return nil
}

func tablePath(cfg config.Config) (string, error) {
	if cfg.Train.TablePath != "" {
		return filepath.Clean(cfg.Train.TablePath), nil
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("trainer source path unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "internal", "policy", "table_gen.go")), nil
}

func render(weights [][]float64) []byte {
	var b strings.Builder
	b.WriteString("package policy\n\n//go:generate go run ../../tools/train\nvar TableWeights = [][]float64{\n")
	for _, row := range weights {
		b.WriteString("\t{")
		for i, w := range row {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.FormatFloat(w, 'g', -1, 64))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}
