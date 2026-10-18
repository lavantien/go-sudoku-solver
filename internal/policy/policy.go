package policy

import (
	"sort"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func FixedOrder() []string {
	return []string{config.MethodIDs[0], config.MethodIDs[1], config.MethodIDs[2]}
}

func ResolveOrder(g sudoku.Grid) []string {
	return Order(Features(g))
}

func Order(features []float64) []string {
	fixed := FixedOrder()
	if len(TableWeights) != len(fixed) {
		return fixed
	}
	type score struct {
		id string
		v  float64
	}
	scores := make([]score, len(fixed))
	for i, id := range fixed {
		total := 0.0
		for j, w := range TableWeights[i] {
			if j < len(features) {
				total += w * features[j]
			}
		}
		scores[i] = score{id: id, v: total}
	}
	sort.SliceStable(scores, func(a, b int) bool { return scores[a].v < scores[b].v })
	out := make([]string, len(fixed))
	for i, s := range scores {
		out[i] = s.id
	}
	return out
}
