package coverbadge

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

const (
	StartMarker = "<!-- coverage:START -->"
	EndMarker   = "<!-- coverage:END -->"
)

func Pct(profile string) (float64, error) {
	stmts := make(map[string]int)
	covered := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(profile))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		numStmt, err := strconv.Atoi(fields[len(fields)-2])
		if err != nil {
			return 0, fmt.Errorf("parse numStmt %q: %w", fields[len(fields)-2], err)
		}
		if numStmt < 0 {
			return 0, fmt.Errorf("numStmt %d out of range", numStmt)
		}
		count, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			return 0, fmt.Errorf("parse count %q: %w", fields[len(fields)-1], err)
		}
		if count < 0 {
			return 0, fmt.Errorf("count %d out of range", count)
		}
		stmts[fields[0]] = numStmt
		if count > 0 {
			covered[fields[0]] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	var total, coveredStmts int
	for key, n := range stmts {
		total += n
		if covered[key] {
			coveredStmts += n
		}
	}
	if total == 0 {
		return 0, fmt.Errorf("empty coverage profile")
	}
	return float64(coveredStmts) / float64(total) * 100, nil
}

func BadgeLine(pct float64) string {
	return fmt.Sprintf("coverage: %.1f%% of statements", pct)
}

func Render(readme string, pct float64) string {
	badge := BadgeLine(pct)
	start := strings.Index(readme, StartMarker)
	end := strings.Index(readme, EndMarker)
	if start >= 0 && end >= start {
		return readme[:start+len(StartMarker)] + "\n" + badge + "\n" + readme[end:]
	}
	section := StartMarker + "\n" + badge + "\n" + EndMarker + "\n"
	switch {
	case readme == "":
		return section
	case strings.HasSuffix(readme, "\n\n"):
		return readme + section
	case strings.HasSuffix(readme, "\n"):
		return readme + "\n" + section
	default:
		return readme + "\n\n" + section
	}
}
