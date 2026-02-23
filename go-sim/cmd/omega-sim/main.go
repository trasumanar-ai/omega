package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"omega/go-sim/internal/sim"
)

type tracePoint struct {
	Tick             int     `json:"tick"`
	AliveAgents      int     `json:"aliveAgents"`
	AvgEnergy        float64 `json:"avgEnergy"`
	AvgInventory     float64 `json:"avgInventory"`
	DeathsThisTick   int     `json:"deathsThisTick"`
	CollectedFruit   int     `json:"collectedFruit"`
	EatenFruit       int     `json:"eatenFruit"`
	TradedFruit      int     `json:"tradedFruit"`
	GroundFruitTotal int     `json:"groundFruitTotal"`
	DeficientAgents  int     `json:"deficientAgents"`
}

type runReport struct {
	StartedAt  string               `json:"startedAt"`
	EndedAt    string               `json:"endedAt"`
	DurationMS int64                `json:"durationMs"`
	Seed       uint32               `json:"seed"`
	Steps      int                  `json:"steps"`
	Config     sim.SimulationConfig `json:"config"`
	Final      sim.TickStats        `json:"final"`
	Trace      []tracePoint         `json:"trace"`
}

func main() {
	defaults := sim.DefaultConfig()

	steps := flag.Int("steps", 1000, "number of simulation steps")
	seedFlag := flag.Int64("seed", 0, "seed (0 = current time)")
	sampleEvery := flag.Int("sample-every", 10, "record trace every N steps")
	outPath := flag.String("out", "", "optional output file path for run report JSON")

	width := flag.Int("width", defaults.Width, "grid width")
	height := flag.Int("height", defaults.Height, "grid height")
	agents := flag.Int("agents", defaults.AgentCount, "initial agent count")
	slots := flag.Int("slots", defaults.MaxAgentSlots, "max agent slots")
	flag.Parse()

	if *steps <= 0 {
		fatalf("steps must be > 0")
	}
	if *sampleEvery <= 0 {
		fatalf("sample-every must be > 0")
	}

	seed := uint32(*seedFlag)
	if *seedFlag == 0 {
		seed = uint32(time.Now().UnixNano())
	}

	cfg := defaults
	cfg.Width = *width
	cfg.Height = *height
	cfg.AgentCount = *agents
	cfg.MaxAgentSlots = *slots

	engine, err := sim.NewSimulation(cfg, seed)
	if err != nil {
		fatalf("failed to create simulation: %v", err)
	}
	if err := engine.Validate(); err != nil {
		fatalf("invalid simulation: %v", err)
	}

	started := time.Now()
	trace := make([]tracePoint, 0, *steps/(*sampleEvery)+2)
	last := sim.TickStats{}
	for step := 1; step <= *steps; step++ {
		last = engine.Step(nil)
		if step == 1 || step%*sampleEvery == 0 || step == *steps {
			trace = append(trace, toTrace(last))
		}
	}
	ended := time.Now()

	report := runReport{
		StartedAt:  started.UTC().Format(time.RFC3339Nano),
		EndedAt:    ended.UTC().Format(time.RFC3339Nano),
		DurationMS: ended.Sub(started).Milliseconds(),
		Seed:       seed,
		Steps:      *steps,
		Config:     cfg,
		Final:      last,
		Trace:      trace,
	}

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatalf("failed to encode report: %v", err)
	}

	if *outPath != "" {
		if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
			fatalf("failed to write report file: %v", err)
		}
	}

	fmt.Println(string(payload))
}

func toTrace(stats sim.TickStats) tracePoint {
	return tracePoint{
		Tick:             stats.Tick,
		AliveAgents:      stats.AliveAgents,
		AvgEnergy:        stats.AvgEnergy,
		AvgInventory:     stats.AvgInventory,
		DeathsThisTick:   stats.DeathsThisTick,
		CollectedFruit:   stats.CollectedFruit,
		EatenFruit:       stats.EatenFruit,
		TradedFruit:      stats.TradedFruit,
		GroundFruitTotal: stats.GroundFruitTotal,
		DeficientAgents:  stats.DeficientAgents,
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
