package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var targetLine = regexp.MustCompile(`^[a-zA-Z0-9_-]+:.*##`)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	path := "Makefile"
	if len(args) > 0 {
		path = args[0]
	}
	out, err := render(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(out)
	return 0
}

func render(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Usage: make <target>\n\nTargets:\n")
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !targetLine.MatchString(line) {
			continue
		}
		fmt.Fprintf(&sb, "  %-15s %s\n", name(line), desc(line))
	}
	return sb.String(), nil
}

func name(line string) string {
	return line[:strings.IndexByte(line, ':')]
}

func desc(line string) string {
	return line[strings.LastIndex(line, "##")+2:]
}
