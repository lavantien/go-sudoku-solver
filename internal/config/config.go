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

	Version = "v1.0"
	BinName = "sudoku"

	LabelFmt = "\n%s:\n"

	CmdSolve   = "solve"
	CmdGen     = "gen"
	CmdVersion = "version"

	UsageSolve   = "solve a puzzle read from stdin"
	UsageGen     = "generate a puzzle with a unique solution"
	UsageVersion = "print the version"

	UsageHeaderFmt = "usage: %s <command> [args]\ncommands:\n"
	UsageCmdFmt    = "  %-7s %s\n"
	RunErrFmt      = "%s %s: %v\n"
	VersionFmt     = "%s %s\n"

	FlagClues      = "clues"
	FlagSeed       = "seed"
	FlagCluesUsage = "target clue count between 17 and 81"
	FlagSeedUsage  = "generator seed"

	EmptyChar = '.'

	ErrSilentExit = "silent exit"
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

type EvolveConfig struct {
	Seed        uint64
	Population  int
	Generations int
}

type TrainConfig struct {
	Seeds     int
	TablePath string
}

type Config struct {
	Workers      int
	CoverNodeCap int
	MatmulCutoff int
	Anneal       AnnealConfig
	Gen          GenConfig
	Evolve       EvolveConfig
	Train        TrainConfig
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
		Gen:    GenConfig{Clues: 30, Seed: 1},
		Evolve: EvolveConfig{Seed: 20261019, Population: 40, Generations: 400},
		Train:  TrainConfig{Seeds: 200, TablePath: ""},
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
	if v, ok := envUint("SUDOKU_EVOLVE_SEED"); ok {
		c.Evolve.Seed = v
	}
	if v, ok := envInt("SUDOKU_EVOLVE_POPULATION"); ok {
		c.Evolve.Population = v
	}
	if v, ok := envInt("SUDOKU_EVOLVE_GENERATIONS"); ok {
		c.Evolve.Generations = v
	}
	if v, ok := envInt("SUDOKU_TRAIN_SEEDS"); ok {
		c.Train.Seeds = v
	}
	if v, ok := envString("SUDOKU_TRAIN_TABLE"); ok {
		c.Train.TablePath = v
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
	if c.Evolve.Population < 3 {
		return fmt.Errorf("evolve population must be >= 3")
	}
	if c.Evolve.Generations < 0 {
		return fmt.Errorf("evolve generations must be >= 0")
	}
	if c.Train.Seeds < 1 {
		return fmt.Errorf("train seeds must be >= 1")
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

const (
	CmdCoverage      = "coverage"
	UsageCoverage    = "refresh the readme coverage badge: -profile -readme"
	FlagProfile      = "profile"
	FlagReadme       = "readme"
	FlagProfileUsage = "Go cover profile path"
	FlagReadmeUsage  = "readme path to update"
	DefaultProfile   = "coverage.out"
	DefaultReadme    = "readme.md"
)

const (
	CmdServe      = "serve"
	UsageServe    = "serve the htmx web ui and the json api"
	FlagAddr      = "addr"
	FlagAddrUsage = "listen address for the web server"
	FlagDB        = "db"
	FlagDBUsage   = "sqlite database path"

	ServeAddrLineFmt = "%s serve listening on %s\n"
	ServeAddrPrefix  = BinName + " serve listening on "

	SourceSeed = "seed"
	SourceWeb  = "web"
	SourceAPI  = "api"

	HomePath          = "/"
	GeneratePagePath  = "/generate"
	HistoryPagePath   = "/history"
	RunPageRoute      = "/runs/{id}"
	RunPagePathFmt    = "/runs/%d"
	StaticMountPath   = "/static/"
	FragmentSolvePath = "/fragment/solve"
	FragmentGenPath   = "/fragment/generate"

	APIMountPath    = "/api/"
	APISolvePath    = "/api/v1/solve"
	APIRunsPath     = "/api/v1/runs"
	APIRunRoute     = "/api/v1/runs/{id}"
	APIRunsPrefix   = "/api/v1/runs/"
	APIRunPathFmt   = "/api/v1/runs/%d"
	APIGeneratePath = "/api/v1/generate"
	APIPuzzlesPath  = "/api/v1/puzzles"
	APIHealthPath   = "/api/v1/health"
	APIVersionPath  = "/api/v1/version"

	FormGrid81 = "grid81"

	QueryLimit  = "limit"
	QueryOffset = "offset"
	QueryPage   = "page"

	JSONContentType = "application/json"
	ErrorKey        = "error"
	HealthStatus    = "ok"
	NameKey         = "name"
	VersionKey      = "version"
	DurationMSKey   = "duration_ms"

	HistoryPageSize = 20
	APIPageLimit    = 20
	APIPageLimitMax = 100
)

const (
	CmdScreenshot      = "screenshot"
	UsageScreenshot    = "capture the web ui and terminal journeys into " + DefaultPicturesDir
	FlagOut            = "out"
	FlagOutUsage       = "output directory for the captured pictures"
	DefaultPicturesDir = "project-pictures"

	BrowserEnv = "SUDOKU_BROWSER"

	ShotWidth  = 1280
	ShotHeight = 800
	ShotBudget = 5 * time.Minute

	PictureHome       = "01-home.png"
	PictureSolve      = "02-solve-results.png"
	PictureGenerator  = "03-generator.png"
	PictureHistory    = "04-history.png"
	PictureRunDetail  = "05-run-detail.png"
	PictureAPIJSON    = "06-api-json.png"
	PictureCLISession = "07-cli-session.png"
	PictureCurlCard   = "08-curl-session.png"

	CliPrompt     = "$ " + BinName + " " + CmdSolve
	CurlPromptFmt = "$ curl -s -X POST http://%s%s -H %q -d '{\"%s\": %q}'"
)

var PictureNames = [8]string{
	PictureHome,
	PictureSolve,
	PictureGenerator,
	PictureHistory,
	PictureRunDetail,
	PictureAPIJSON,
	PictureCLISession,
	PictureCurlCard,
}
