package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"omega/server/internal/econ"
)

type tracePoint struct {
	Tick                 int     `json:"tick"`
	LivingCountries      int     `json:"livingCountries"`
	TotalEnergyGenerated float64 `json:"totalEnergyGenerated"`
	TotalEnergyDelivered float64 `json:"totalEnergyDelivered"`
	TotalEnergyTraded    float64 `json:"totalEnergyTraded"`
	TotalCargoTraded     float64 `json:"totalCargoTraded"`
	TotalShortage        float64 `json:"totalShortage"`
	TotalTreasury        float64 `json:"totalTreasury"`
	AvgComputeEfficiency float64 `json:"avgComputeEfficiency"`
}

type runReport struct {
	StartedAt  string             `json:"startedAt"`
	EndedAt    string             `json:"endedAt"`
	DurationMS int64              `json:"durationMs"`
	Steps      int                `json:"steps"`
	Config     econ.Config        `json:"config"`
	Final      econ.TickStats     `json:"final"`
	World      econ.WorldSnapshot `json:"world"`
	Trace      []tracePoint       `json:"trace"`
}

type providerSetup struct {
	Provider econ.DecisionProvider
	Mode     string
	Model    string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "experiment" {
		if err := runExperimentCLI(os.Args[2:]); err != nil {
			fatalf("%v", err)
		}
		return
	}
	if err := runSingleSimulation(os.Args[1:]); err != nil {
		fatalf("%v", err)
	}
}

func runSingleSimulation(args []string) error {
	defaults := econ.DefaultConfig()

	fs := flag.NewFlagSet("omega-econ", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	steps := fs.Int("steps", 240, "number of simulation steps")
	sampleEvery := fs.Int("sample-every", 8, "record trace every N steps")
	outPath := fs.String("out", "", "optional output file path for run report JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *steps <= 0 {
		return fmt.Errorf("steps must be > 0")
	}
	if *sampleEvery <= 0 {
		return fmt.Errorf("sample-every must be > 0")
	}

	engine, err := econ.NewSimulation(defaults)
	if err != nil {
		return fmt.Errorf("failed to create econ simulation: %w", err)
	}
	provider := resolveProvider()
	if provider.Mode == "openrouter" {
		fmt.Fprintf(os.Stderr, "using OpenRouter model %s\n", provider.Model)
	} else {
		fmt.Fprintf(os.Stderr, "using heuristic decisions (OPENROUTER_API_KEY is not set)\n")
	}

	started := time.Now()
	trace := make([]tracePoint, 0, *steps/(*sampleEvery)+2)
	last := econ.TickStats{}
	for step := 1; step <= *steps; step++ {
		last = engine.Step(provider.Provider)
		if step == 1 || step%*sampleEvery == 0 || step == *steps {
			trace = append(trace, toTrace(last))
		}
	}
	ended := time.Now()

	report := runReport{
		StartedAt:  started.UTC().Format(time.RFC3339Nano),
		EndedAt:    ended.UTC().Format(time.RFC3339Nano),
		DurationMS: ended.Sub(started).Milliseconds(),
		Steps:      *steps,
		Config:     defaults,
		Final:      last,
		World:      engine.Snapshot(),
		Trace:      trace,
	}

	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode report: %w", err)
	}

	if *outPath != "" {
		if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
			return fmt.Errorf("failed to write report file: %w", err)
		}
	}

	fmt.Println(string(payload))
	return nil
}

func resolveProvider() providerSetup {
	setup := providerSetup{
		Mode: "heuristic",
	}
	if llmCfg, err := econ.LoadOpenRouterConfigFromEnv(); err == nil {
		setup.Provider = econ.NewOpenRouterProvider(llmCfg).ProviderFunc()
		setup.Mode = "openrouter"
		setup.Model = llmCfg.Model
	}
	return setup
}

func toTrace(stats econ.TickStats) tracePoint {
	return tracePoint{
		Tick:                 stats.Tick,
		LivingCountries:      stats.LivingCountries,
		TotalEnergyGenerated: stats.TotalEnergyGenerated,
		TotalEnergyDelivered: stats.TotalEnergyDelivered,
		TotalEnergyTraded:    stats.TotalEnergyTraded,
		TotalCargoTraded:     stats.TotalCargoTraded,
		TotalShortage:        stats.TotalShortage,
		TotalTreasury:        stats.TotalTreasury,
		AvgComputeEfficiency: stats.AvgComputeEfficiency,
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
