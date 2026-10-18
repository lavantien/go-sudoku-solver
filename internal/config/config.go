package config

import "time"

const (
	Size       = 9
	Box        = 3
	Cells      = 81
	EmptyDigit = 0

	InitialHeader     = ""
	Separator         = ""
	StepFmt           = ""
	TimeFmt           = ""
	GridUnitRow       = ""
	ErrNotEnoughLines = ""
	ErrLineLengthFmt  = ""
	ErrCharFmt        = ""
	InputErrorPrefix  = ""
	LabelPropagate    = ""
	LabelCover        = ""
	LabelAnneal       = ""
	LabelRace         = ""
)

var MethodIDs = [4]string{"", "", "", ""}

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
	return Config{}
}

func Load() Config {
	return Defaults()
}

func (c Config) Validate() error {
	return nil
}
