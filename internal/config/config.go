package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	Size       = 9
	Box        = 3
	Cells      = 81
	EmptyDigit = 0

	InitialHeader     = "Initial Grid:"
	Separator         = "----------------------------------"
	StepFmt           = "    Step count: %d\n"
	TimeFmt           = "    Execution time: %f\n"
	GridUnitRow       = "___ ___ ___"
	ErrNotEnoughLines = "not enough lines in input"
	ErrLineLengthFmt  = "invalid line length: %d, expected 9"
	ErrCharFmt        = "invalid character: %c"
	ErrGridLengthFmt  = "invalid grid length: %d, expected 81"
	InputErrorPrefix  = "Error reading input:"
	LabelPropagate    = "Constraint propagation with MRV backtracking"
	LabelCover        = "Exact cover over the boolean incidence matrix"
	LabelAnneal       = "Simulated annealing with restart portfolio"
	LabelRace         = "Parallel portfolio race with mined ordering"
)

var MethodIDs = [4]string{"propagate", "cover", "anneal", "race"}

var methodLabels = [4]string{LabelPropagate, LabelCover, LabelAnneal, LabelRace}

func MethodLabel(id string) string {
	for i, mid := range MethodIDs {
		if mid == id {
			return methodLabels[i]
		}
	}
	return ""
}

type AnnealConfig struct {
	Restarts   int
	Iterations int
	Seed       uint64
	TempStart  int64
	TempEnd    int64
}

type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type GenConfig struct {
	Clues int
	Seed  uint64
}

type Config struct {
	Workers      int
	CoverNodeCap int
	MatmulCutoff int
	Anneal       AnnealConfig
	Gen          GenConfig
	Server       ServerConfig
	DBPath       string
	LogLevel     string
}

func Defaults() Config {
	return Config{
		Workers:      0,
		CoverNodeCap: 1 << 20,
		MatmulCutoff: 64,
		Anneal: AnnealConfig{
			Restarts:   4,
			Iterations: 150000,
			Seed:       1,
			TempStart:  4000,
			TempEnd:    100,
		},
		Gen: GenConfig{Clues: 30, Seed: 1},
		Server: ServerConfig{
			Addr:            "127.0.0.1:8080",
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		DBPath:   "database/sudoku.db",
		LogLevel: "",
	}
}

func Load() Config {
	c := Defaults()
	if v, ok := envInt("SUDOKU_WORKERS"); ok {
		c.Workers = v
	}
	if v, ok := envInt("SUDOKU_ANNEAL_RESTARTS"); ok {
		c.Anneal.Restarts = v
	}
	if v, ok := envInt("SUDOKU_ANNEAL_ITERATIONS"); ok {
		c.Anneal.Iterations = v
	}
	if v, ok := envInt("SUDOKU_ANNEAL_TEMP_START"); ok {
		c.Anneal.TempStart = int64(v)
	}
	if v, ok := envInt("SUDOKU_ANNEAL_TEMP_END"); ok {
		c.Anneal.TempEnd = int64(v)
	}
	if v, ok := envUint("SUDOKU_ANNEAL_SEED"); ok {
		c.Anneal.Seed = v
	}
	if v, ok := envInt("SUDOKU_COVER_NODE_CAP"); ok {
		c.CoverNodeCap = v
	}
	if v, ok := envInt("SUDOKU_MATMUL_CUTOFF"); ok {
		c.MatmulCutoff = v
	}
	if v, ok := envInt("SUDOKU_GEN_CLUES"); ok {
		c.Gen.Clues = v
	}
	if v, ok := envUint("SUDOKU_GEN_SEED"); ok {
		c.Gen.Seed = v
	}
	if v, ok := envString("SUDOKU_SERVER_ADDR"); ok {
		c.Server.Addr = v
	}
	if v, ok := envDuration("SUDOKU_SERVER_READ_TIMEOUT"); ok {
		c.Server.ReadTimeout = v
	}
	if v, ok := envDuration("SUDOKU_SERVER_WRITE_TIMEOUT"); ok {
		c.Server.WriteTimeout = v
	}
	if v, ok := envDuration("SUDOKU_SERVER_IDLE_TIMEOUT"); ok {
		c.Server.IdleTimeout = v
	}
	if v, ok := envDuration("SUDOKU_SERVER_SHUTDOWN_TIMEOUT"); ok {
		c.Server.ShutdownTimeout = v
	}
	if v, ok := envString("SUDOKU_DB_PATH"); ok {
		c.DBPath = v
	}
	if v, ok := envString("SUDOKU_LOG_LEVEL"); ok {
		c.LogLevel = v
	}
	return c
}

func (c Config) Validate() error {
	if c.Workers < 0 {
		return fmt.Errorf("workers must be >= 0")
	}
	if c.CoverNodeCap <= 0 {
		return fmt.Errorf("cover node cap must be > 0")
	}
	if c.MatmulCutoff <= 0 {
		return fmt.Errorf("matmul cutoff must be > 0")
	}
	if c.Anneal.Restarts <= 0 {
		return fmt.Errorf("anneal restarts must be > 0")
	}
	if c.Anneal.Iterations <= 0 {
		return fmt.Errorf("anneal iterations must be > 0")
	}
	if c.Anneal.TempEnd <= 0 {
		return fmt.Errorf("anneal end temperature must be > 0")
	}
	if c.Anneal.TempStart <= c.Anneal.TempEnd {
		return fmt.Errorf("anneal start temperature must exceed end temperature")
	}
	if c.Gen.Clues < 17 || c.Gen.Clues > Cells {
		return fmt.Errorf("generator clues must be between 17 and %d", Cells)
	}
	if c.Server.Addr == "" {
		return fmt.Errorf("server addr must not be empty")
	}
	if c.Server.ReadTimeout <= 0 || c.Server.WriteTimeout <= 0 || c.Server.IdleTimeout <= 0 || c.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server timeouts must be > 0")
	}
	if c.DBPath == "" {
		return fmt.Errorf("db path must not be empty")
	}
	switch c.LogLevel {
	case "", "discard", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log level must be one of discard, debug, info, warn, error")
	}
	return nil
}

func envInt(name string) (int, bool) {
	v := os.Getenv(name)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func envUint(name string) (uint64, bool) {
	v := os.Getenv(name)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func envDuration(name string) (time.Duration, bool) {
	v := os.Getenv(name)
	if v == "" {
		return 0, false
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, false
	}
	return d, true
}

func envString(name string) (string, bool) {
	v := os.Getenv(name)
	if v == "" {
		return "", false
	}
	return v, true
}
