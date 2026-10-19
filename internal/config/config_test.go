package config

import (
	"strings"
	"testing"
	"time"
)

func TestContractConstants(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"initial header", InitialHeader, "Initial Grid:"},
		{"separator", Separator, strings.Repeat("-", 34)},
		{"step format", StepFmt, "    Step count: %d\n"},
		{"time format", TimeFmt, "    Execution time: %f\n"},
		{"not enough lines", ErrNotEnoughLines, "not enough lines in input"},
		{"line length", ErrLineLengthFmt, "invalid line length: %d, expected 9"},
		{"bad character", ErrCharFmt, "invalid character: %c"},
		{"grid length", ErrGridLengthFmt, "invalid grid length: %d, expected 81"},
		{"input error prefix", InputErrorPrefix, "Error reading input:"},
		{"grid unit row", GridUnitRow, "___ ___ ___"},
		{"label propagate", LabelPropagate, "Constraint propagation with MRV backtracking"},
		{"label cover", LabelCover, "Exact cover over the boolean incidence matrix"},
		{"label anneal", LabelAnneal, "Simulated annealing with restart portfolio"},
		{"label race", LabelRace, "Parallel portfolio race with mined ordering"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, tc.got, tc.want)
		}
	}
	if Size != 9 || Box != 3 || Cells != 81 || EmptyDigit != 0 {
		t.Errorf("geometry: got size %d box %d cells %d empty %d", Size, Box, Cells, EmptyDigit)
	}
	wantIDs := [4]string{"propagate", "cover", "anneal", "race"}
	if MethodIDs != wantIDs {
		t.Errorf("method ids: got %v want %v", MethodIDs, wantIDs)
	}
}

func TestDefaults(t *testing.T) {
	c := Defaults()
	if c.Workers != 0 {
		t.Errorf("workers: got %d want 0", c.Workers)
	}
	if c.CoverNodeCap != 1<<20 {
		t.Errorf("cover node cap: got %d want %d", c.CoverNodeCap, 1<<20)
	}
	if c.MatmulCutoff != 64 {
		t.Errorf("matmul cutoff: got %d want 64", c.MatmulCutoff)
	}
	if c.Gen.Clues != 30 || c.Gen.Seed != 1 {
		t.Errorf("gen: got %+v want clues 30 seed 1", c.Gen)
	}
	if c.Server.Addr != "127.0.0.1:8080" {
		t.Errorf("server addr: got %q", c.Server.Addr)
	}
	if c.Server.ReadTimeout != 5*time.Second || c.Server.WriteTimeout != 10*time.Second || c.Server.IdleTimeout != 60*time.Second || c.Server.ShutdownTimeout != 5*time.Second {
		t.Errorf("server timeouts: got %+v", c.Server)
	}
	if c.DBPath != "database/sudoku.db" {
		t.Errorf("db path: got %q", c.DBPath)
	}
	if c.LogLevel != "" {
		t.Errorf("log level: got %q want empty", c.LogLevel)
	}
	if c.Anneal.Restarts <= 0 || c.Anneal.Iterations <= 0 || c.Anneal.Seed != 1 || c.Anneal.TempEnd <= 0 || c.Anneal.TempStart <= c.Anneal.TempEnd {
		t.Errorf("anneal defaults: got %+v", c.Anneal)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want func(Config) bool
	}{
		{"workers", map[string]string{"SUDOKU_WORKERS": "4"}, func(c Config) bool { return c.Workers == 4 }},
		{"anneal restarts", map[string]string{"SUDOKU_ANNEAL_RESTARTS": "9"}, func(c Config) bool { return c.Anneal.Restarts == 9 }},
		{"anneal iterations", map[string]string{"SUDOKU_ANNEAL_ITERATIONS": "1000"}, func(c Config) bool { return c.Anneal.Iterations == 1000 }},
		{"anneal seed", map[string]string{"SUDOKU_ANNEAL_SEED": "42"}, func(c Config) bool { return c.Anneal.Seed == 42 }},
		{"anneal temp start", map[string]string{"SUDOKU_ANNEAL_TEMP_START": "5000"}, func(c Config) bool { return c.Anneal.TempStart == 5000 }},
		{"anneal temp end", map[string]string{"SUDOKU_ANNEAL_TEMP_END": "25"}, func(c Config) bool { return c.Anneal.TempEnd == 25 }},
		{"cover node cap", map[string]string{"SUDOKU_COVER_NODE_CAP": "777"}, func(c Config) bool { return c.CoverNodeCap == 777 }},
		{"matmul cutoff", map[string]string{"SUDOKU_MATMUL_CUTOFF": "32"}, func(c Config) bool { return c.MatmulCutoff == 32 }},
		{"gen clues", map[string]string{"SUDOKU_GEN_CLUES": "40"}, func(c Config) bool { return c.Gen.Clues == 40 }},
		{"gen seed", map[string]string{"SUDOKU_GEN_SEED": "7"}, func(c Config) bool { return c.Gen.Seed == 7 }},
		{"server addr", map[string]string{"SUDOKU_SERVER_ADDR": "127.0.0.1:9999"}, func(c Config) bool { return c.Server.Addr == "127.0.0.1:9999" }},
		{"server read timeout", map[string]string{"SUDOKU_SERVER_READ_TIMEOUT": "3s"}, func(c Config) bool { return c.Server.ReadTimeout == 3*time.Second }},
		{"server write timeout", map[string]string{"SUDOKU_SERVER_WRITE_TIMEOUT": "4s"}, func(c Config) bool { return c.Server.WriteTimeout == 4*time.Second }},
		{"server idle timeout", map[string]string{"SUDOKU_SERVER_IDLE_TIMEOUT": "7s"}, func(c Config) bool { return c.Server.IdleTimeout == 7*time.Second }},
		{"server shutdown timeout", map[string]string{"SUDOKU_SERVER_SHUTDOWN_TIMEOUT": "2s"}, func(c Config) bool { return c.Server.ShutdownTimeout == 2*time.Second }},
		{"db path", map[string]string{"SUDOKU_DB_PATH": "t/a.db"}, func(c Config) bool { return c.DBPath == "t/a.db" }},
		{"evolve seed", map[string]string{"SUDOKU_EVOLVE_SEED": "77"}, func(c Config) bool { return c.Evolve.Seed == 77 }},
		{"evolve population", map[string]string{"SUDOKU_EVOLVE_POPULATION": "6"}, func(c Config) bool { return c.Evolve.Population == 6 }},
		{"evolve generations", map[string]string{"SUDOKU_EVOLVE_GENERATIONS": "3"}, func(c Config) bool { return c.Evolve.Generations == 3 }},
		{"train seeds", map[string]string{"SUDOKU_TRAIN_SEEDS": "4"}, func(c Config) bool { return c.Train.Seeds == 4 }},
		{"train table", map[string]string{"SUDOKU_TRAIN_TABLE": "tmp/table.go"}, func(c Config) bool { return c.Train.TablePath == "tmp/table.go" }},
		{"log level", map[string]string{"SUDOKU_LOG_LEVEL": "debug"}, func(c Config) bool { return c.LogLevel == "debug" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := Load(); !tc.want(got) {
				t.Errorf("Load with %v: got %+v", tc.env, got)
			}
		})
	}
}

func TestLoadMalformedEnvKeepsDefaults(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want func(Config) bool
	}{
		{"workers", map[string]string{"SUDOKU_WORKERS": "two"}, func(c Config) bool { return c.Workers == 0 }},
		{"anneal restarts", map[string]string{"SUDOKU_ANNEAL_RESTARTS": "x"}, func(c Config) bool { return c.Anneal.Restarts == 4 }},
		{"anneal seed", map[string]string{"SUDOKU_ANNEAL_SEED": "-1"}, func(c Config) bool { return c.Anneal.Seed == 1 }},
		{"cover node cap", map[string]string{"SUDOKU_COVER_NODE_CAP": "big"}, func(c Config) bool { return c.CoverNodeCap == 1<<20 }},
		{"gen seed", map[string]string{"SUDOKU_GEN_SEED": "0x1f"}, func(c Config) bool { return c.Gen.Seed == 1 }},
		{"server read timeout", map[string]string{"SUDOKU_SERVER_READ_TIMEOUT": "soon"}, func(c Config) bool { return c.Server.ReadTimeout == 5*time.Second }},
		{"server write timeout", map[string]string{"SUDOKU_SERVER_WRITE_TIMEOUT": "10"}, func(c Config) bool { return c.Server.WriteTimeout == 10*time.Second }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := Load(); !tc.want(got) {
				t.Errorf("Load with %v: got %+v", tc.env, got)
			}
		})
	}
}

func TestMethodLabel(t *testing.T) {
	for i, id := range MethodIDs[:3] {
		if got := MethodLabel(id); got == "" {
			t.Errorf("method label for %q is empty", id)
		} else if got != methodLabels[i] {
			t.Errorf("method label for %q: got %q want %q", id, got, methodLabels[i])
		}
	}
	if got := MethodLabel("race"); got != LabelRace {
		t.Errorf("race label: got %q want %q", got, LabelRace)
	}
	if got := MethodLabel("bogus"); got != "" {
		t.Errorf("unknown id label: got %q want empty", got)
	}
}

func TestLoadWithoutEnvGivesDefaults(t *testing.T) {
	if got := Load(); got != Defaults() {
		t.Errorf("Load without env: got %+v want %+v", got, Defaults())
	}
}

func TestValidate(t *testing.T) {
	mut := func(f func(*Config)) Config {
		c := Defaults()
		f(&c)
		return c
	}
	bad := []struct {
		name string
		c    Config
	}{
		{"negative workers", mut(func(c *Config) { c.Workers = -1 })},
		{"zero node cap", mut(func(c *Config) { c.CoverNodeCap = 0 })},
		{"zero cutoff", mut(func(c *Config) { c.MatmulCutoff = 0 })},
		{"zero restarts", mut(func(c *Config) { c.Anneal.Restarts = 0 })},
		{"zero iterations", mut(func(c *Config) { c.Anneal.Iterations = 0 })},
		{"zero temp end", mut(func(c *Config) { c.Anneal.TempEnd = 0 })},
		{"temp end above start", mut(func(c *Config) { c.Anneal.TempEnd = c.Anneal.TempStart + 1 })},
		{"clues too low", mut(func(c *Config) { c.Gen.Clues = 16 })},
		{"clues too high", mut(func(c *Config) { c.Gen.Clues = 82 })},
		{"evolve population too small", mut(func(c *Config) { c.Evolve.Population = 2 })},
		{"evolve negative generations", mut(func(c *Config) { c.Evolve.Generations = -1 })},
		{"train seeds zero", mut(func(c *Config) { c.Train.Seeds = 0 })},
		{"empty addr", mut(func(c *Config) { c.Server.Addr = "" })},
		{"zero read timeout", mut(func(c *Config) { c.Server.ReadTimeout = 0 })},
		{"zero shutdown timeout", mut(func(c *Config) { c.Server.ShutdownTimeout = 0 })},
		{"empty db path", mut(func(c *Config) { c.DBPath = "" })},
		{"bad log level", mut(func(c *Config) { c.LogLevel = "loud" })},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.c.Validate(); err == nil {
				t.Errorf("%s: expected error for %+v", tc.name, tc.c)
			}
		})
	}
	good := []Config{
		Defaults(),
		mut(func(c *Config) { c.Workers = 8; c.LogLevel = "debug" }),
		mut(func(c *Config) { c.LogLevel = "discard" }),
	}
	for i, c := range good {
		if err := c.Validate(); err != nil {
			t.Errorf("good case %d: unexpected error %v", i, err)
		}
	}
}
