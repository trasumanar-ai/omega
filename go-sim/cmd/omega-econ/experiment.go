package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"omega/go-sim/internal/econ"
)

const defaultExperimentDir = "experiment-results"

type experimentSpec struct {
	Version              int                      `json:"version"`
	Name                 string                   `json:"name"`
	Question             string                   `json:"question"`
	Hypothesis           string                   `json:"hypothesis"`
	Notes                string                   `json:"notes"`
	ControlScenarioID    string                   `json:"controlScenarioId"`
	IndependentVariables []string                 `json:"independentVariables"`
	DependentMetrics     []string                 `json:"dependentMetrics"`
	Steps                int                      `json:"steps"`
	SampleEvery          int                      `json:"sampleEvery"`
	Scenarios            []experimentScenarioSpec `json:"scenarios"`
}

type experimentScenarioSpec struct {
	ID          string                    `json:"id"`
	Label       string                    `json:"label"`
	Role        string                    `json:"role"`
	Preset      string                    `json:"preset"`
	Algorithm   econ.MarketAlgorithm      `json:"algorithm"`
	Notes       string                    `json:"notes"`
	Adjustments experimentConfigAdjusters `json:"adjustments"`
}

type experimentConfigAdjusters struct {
	BaseDemandMultiplier        float64 `json:"baseDemandMultiplier,omitempty"`
	StartingTreasuryMultiplier  float64 `json:"startingTreasuryMultiplier,omitempty"`
	StartingReserveMultiplier   float64 `json:"startingReserveMultiplier,omitempty"`
	RouteCapacityMultiplier     float64 `json:"routeCapacityMultiplier,omitempty"`
	ExtractorMultiplier         float64 `json:"extractorMultiplier,omitempty"`
	FossilOutputMultiplier      float64 `json:"fossilOutputMultiplier,omitempty"`
	RenewableOutputMultiplier   float64 `json:"renewableOutputMultiplier,omitempty"`
	TradeShareMultiplier        float64 `json:"tradeShareMultiplier,omitempty"`
	BuildCostMultiplier         float64 `json:"buildCostMultiplier,omitempty"`
	EnergyPriceMultiplier       float64 `json:"energyPriceMultiplier,omitempty"`
	EmergencyPriceMultiplier    float64 `json:"emergencyPriceMultiplier,omitempty"`
	ComputeEfficiencyMultiplier float64 `json:"computeEfficiencyMultiplier,omitempty"`
	ComputeEfficiencyAdd        float64 `json:"computeEfficiencyAdd,omitempty"`
	InfrastructureMultiplier    float64 `json:"infrastructureMultiplier,omitempty"`
}

type experimentScenarioSummary struct {
	FinalTick                 int     `json:"finalTick"`
	LivingCountries           int     `json:"livingCountries"`
	FinalEnergyDelivered      float64 `json:"finalEnergyDelivered"`
	FinalShortage             float64 `json:"finalShortage"`
	FinalTreasury             float64 `json:"finalTreasury"`
	FinalComputeEfficiency    float64 `json:"finalComputeEfficiency"`
	CumulativeEnergyTraded    float64 `json:"cumulativeEnergyTraded"`
	CumulativeCargoTraded     float64 `json:"cumulativeCargoTraded"`
	CumulativeEnergyDelivered float64 `json:"cumulativeEnergyDelivered"`
	PeakShortage              float64 `json:"peakShortage"`
}

type experimentScenarioResult struct {
	Spec    experimentScenarioSpec    `json:"spec"`
	Config  econ.Config               `json:"config"`
	Final   econ.TickStats            `json:"final"`
	Summary experimentScenarioSummary `json:"summary"`
	World   econ.WorldSnapshot        `json:"world"`
	Market  econ.MarketSnapshot       `json:"market"`
	Trace   []tracePoint              `json:"trace"`
}

type experimentRanking struct {
	Rank          int     `json:"rank"`
	ScenarioID    string  `json:"scenarioId"`
	Label         string  `json:"label"`
	OverallScore  float64 `json:"overallScore"`
	Living        int     `json:"living"`
	PeakShortage  float64 `json:"peakShortage"`
	EnergyTrade   float64 `json:"energyTrade"`
	EnergyDeliver float64 `json:"energyDelivered"`
	Treasury      float64 `json:"treasury"`
}

type experimentComparison struct {
	ScenarioID             string  `json:"scenarioId"`
	Label                  string  `json:"label"`
	AgainstControlID       string  `json:"againstControlId"`
	DeltaLivingCountries   int     `json:"deltaLivingCountries"`
	DeltaPeakShortage      float64 `json:"deltaPeakShortage"`
	DeltaEnergyDelivered   float64 `json:"deltaEnergyDelivered"`
	DeltaEnergyTraded      float64 `json:"deltaEnergyTraded"`
	DeltaFinalTreasury     float64 `json:"deltaFinalTreasury"`
	DeltaOverallScore      float64 `json:"deltaOverallScore"`
	SupportsHypothesisHint bool    `json:"supportsHypothesisHint"`
}

type experimentResult struct {
	Version              int                        `json:"version"`
	Name                 string                     `json:"name"`
	Question             string                     `json:"question"`
	Hypothesis           string                     `json:"hypothesis"`
	Notes                string                     `json:"notes"`
	ControlScenarioID    string                     `json:"controlScenarioId"`
	IndependentVariables []string                   `json:"independentVariables"`
	DependentMetrics     []string                   `json:"dependentMetrics"`
	Steps                int                        `json:"steps"`
	SampleEvery          int                        `json:"sampleEvery"`
	StartedAt            string                     `json:"startedAt"`
	EndedAt              string                     `json:"endedAt"`
	DurationMS           int64                      `json:"durationMs"`
	DecisionMode         string                     `json:"decisionMode"`
	DecisionModel        string                     `json:"decisionModel"`
	OutputPath           string                     `json:"outputPath"`
	Rankings             []experimentRanking        `json:"rankings"`
	Comparisons          []experimentComparison     `json:"comparisons"`
	Scenarios            []experimentScenarioResult `json:"scenarios"`
}

func runExperimentCLI(args []string) error {
	if len(args) == 0 {
		return errors.New("experiment subcommand required: list, print, run, show")
	}

	switch args[0] {
	case "list":
		return listExperimentTemplates()
	case "print":
		return printExperimentTemplate(args[1:])
	case "run":
		return runExperimentCommand(args[1:])
	case "show":
		return showExperimentResult(args[1:])
	default:
		return fmt.Errorf("unknown experiment subcommand: %s", args[0])
	}
}

func listExperimentTemplates() error {
	fmt.Println("available templates:")
	for _, template := range builtInExperimentTemplates() {
		fmt.Printf("- %s: %s\n", template.id, template.description)
	}
	return nil
}

func printExperimentTemplate(args []string) error {
	fs := flag.NewFlagSet("omega-econ experiment print", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	templateID := fs.String("template", "default_algorithms", "template id to print")
	if err := fs.Parse(args); err != nil {
		return err
	}

	spec, err := loadExperimentTemplate(*templateID)
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(payload))
	return nil
}

func runExperimentCommand(args []string) error {
	fs := flag.NewFlagSet("omega-econ experiment run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	templateID := fs.String("template", "", "built-in template id")
	specPath := fs.String("spec", "", "path to experiment spec JSON")
	outPath := fs.String("out", "", "optional output file path")
	outDir := fs.String("out-dir", defaultExperimentDir, "directory for generated result files")
	stepsOverride := fs.Int("steps", 0, "optional override for spec steps")
	sampleEveryOverride := fs.Int("sample-every", 0, "optional override for spec sampleEvery")
	printJSON := fs.Bool("json", false, "print full JSON result after saving")
	if err := fs.Parse(args); err != nil {
		return err
	}

	spec, err := loadExperimentSpec(*templateID, *specPath)
	if err != nil {
		return err
	}
	if *stepsOverride > 0 {
		spec.Steps = *stepsOverride
	}
	if *sampleEveryOverride > 0 {
		spec.SampleEvery = *sampleEveryOverride
	}
	if err := validateExperimentSpec(spec); err != nil {
		return err
	}

	provider := resolveProvider()
	result, err := runExperiment(spec, provider)
	if err != nil {
		return err
	}
	if *outPath == "" {
		*outPath = filepath.Join(*outDir, fmt.Sprintf("experiment-%s.json", time.Now().UTC().Format("20060102T150405.000000000Z")))
	}
	result.OutputPath = *outPath
	if err := writeExperimentResult(*outPath, result); err != nil {
		return err
	}

	printExperimentSummary(result)
	fmt.Printf("\nsaved result: %s\n", result.OutputPath)
	if *printJSON {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
	}
	return nil
}

func showExperimentResult(args []string) error {
	fs := flag.NewFlagSet("omega-econ experiment show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	filePath := fs.String("file", "", "path to saved experiment result JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *filePath == "" {
		return errors.New("show requires -file")
	}

	result, err := readExperimentResult(*filePath)
	if err != nil {
		return err
	}
	printExperimentSummary(result)
	fmt.Printf("\nloaded from: %s\n", *filePath)
	return nil
}

func loadExperimentSpec(templateID string, specPath string) (experimentSpec, error) {
	switch {
	case templateID != "" && specPath != "":
		return experimentSpec{}, errors.New("use either -template or -spec, not both")
	case templateID != "":
		return loadExperimentTemplate(templateID)
	case specPath != "":
		return readExperimentSpec(specPath)
	default:
		return experimentSpec{}, errors.New("run requires -template or -spec")
	}
}

func readExperimentSpec(path string) (experimentSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return experimentSpec{}, err
	}
	var spec experimentSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return experimentSpec{}, err
	}
	return spec, nil
}

func validateExperimentSpec(spec experimentSpec) error {
	if spec.Name == "" {
		return errors.New("experiment name is required")
	}
	if spec.Question == "" {
		return errors.New("experiment question is required")
	}
	if spec.Hypothesis == "" {
		return errors.New("experiment hypothesis is required")
	}
	if spec.Steps <= 0 {
		return errors.New("steps must be > 0")
	}
	if spec.SampleEvery <= 0 {
		return errors.New("sampleEvery must be > 0")
	}
	if len(spec.Scenarios) == 0 {
		return errors.New("at least one scenario is required")
	}

	knownAlgorithms := make(map[econ.MarketAlgorithm]struct{}, len(econ.KnownMarketAlgorithms()))
	for _, algorithm := range econ.KnownMarketAlgorithms() {
		knownAlgorithms[algorithm] = struct{}{}
	}
	knownPresets := make(map[string]struct{}, len(experimentPresets()))
	for _, preset := range experimentPresets() {
		knownPresets[preset.ID] = struct{}{}
	}

	seenIDs := make(map[string]struct{}, len(spec.Scenarios))
	hasControl := spec.ControlScenarioID == ""
	for _, scenario := range spec.Scenarios {
		if scenario.ID == "" {
			return errors.New("scenario id is required")
		}
		if scenario.Label == "" {
			return fmt.Errorf("scenario %s label is required", scenario.ID)
		}
		if _, ok := seenIDs[scenario.ID]; ok {
			return fmt.Errorf("duplicate scenario id: %s", scenario.ID)
		}
		seenIDs[scenario.ID] = struct{}{}
		if _, ok := knownPresets[scenario.Preset]; !ok {
			return fmt.Errorf("scenario %s uses unknown preset %s", scenario.ID, scenario.Preset)
		}
		if _, ok := knownAlgorithms[scenario.Algorithm]; !ok {
			return fmt.Errorf("scenario %s uses unknown algorithm %s", scenario.ID, scenario.Algorithm)
		}
		if scenario.ID == spec.ControlScenarioID {
			hasControl = true
		}
	}
	if !hasControl {
		return fmt.Errorf("control scenario %s not found", spec.ControlScenarioID)
	}
	return nil
}

func runExperiment(spec experimentSpec, provider providerSetup) (experimentResult, error) {
	if err := validateExperimentSpec(spec); err != nil {
		return experimentResult{}, err
	}

	started := time.Now()
	results := make([]experimentScenarioResult, 0, len(spec.Scenarios))

	for _, scenario := range spec.Scenarios {
		cfg, err := buildScenarioConfig(scenario)
		if err != nil {
			return experimentResult{}, err
		}
		engine, err := econ.NewSimulation(cfg)
		if err != nil {
			return experimentResult{}, err
		}

		trace := make([]tracePoint, 0, spec.Steps/spec.SampleEvery+2)
		last := econ.TickStats{}
		cumulativeEnergyDelivered := 0.0
		cumulativeEnergyTraded := 0.0
		cumulativeCargoTraded := 0.0
		peakShortage := 0.0

		for step := 1; step <= spec.Steps; step++ {
			last = engine.Step(provider.Provider)
			cumulativeEnergyDelivered += last.TotalEnergyDelivered
			cumulativeEnergyTraded += last.TotalEnergyTraded
			cumulativeCargoTraded += last.TotalCargoTraded
			if last.TotalShortage > peakShortage {
				peakShortage = last.TotalShortage
			}
			if step == 1 || step%spec.SampleEvery == 0 || step == spec.Steps {
				trace = append(trace, toTrace(last))
			}
		}

		results = append(results, experimentScenarioResult{
			Spec:   scenario,
			Config: cfg,
			Final:  last,
			Summary: experimentScenarioSummary{
				FinalTick:                 last.Tick,
				LivingCountries:           last.LivingCountries,
				FinalEnergyDelivered:      last.TotalEnergyDelivered,
				FinalShortage:             last.TotalShortage,
				FinalTreasury:             last.TotalTreasury,
				FinalComputeEfficiency:    last.AvgComputeEfficiency,
				CumulativeEnergyTraded:    cumulativeEnergyTraded,
				CumulativeCargoTraded:     cumulativeCargoTraded,
				CumulativeEnergyDelivered: cumulativeEnergyDelivered,
				PeakShortage:              peakShortage,
			},
			World:  engine.Snapshot(),
			Market: engine.MarketSnapshot(),
			Trace:  trace,
		})
	}

	rankings := rankExperimentResults(results)
	comparisons := compareAgainstControl(spec.ControlScenarioID, results)
	ended := time.Now()

	return experimentResult{
		Version:              max(spec.Version, 1),
		Name:                 spec.Name,
		Question:             spec.Question,
		Hypothesis:           spec.Hypothesis,
		Notes:                spec.Notes,
		ControlScenarioID:    spec.ControlScenarioID,
		IndependentVariables: append([]string(nil), spec.IndependentVariables...),
		DependentMetrics:     append([]string(nil), spec.DependentMetrics...),
		Steps:                spec.Steps,
		SampleEvery:          spec.SampleEvery,
		StartedAt:            started.UTC().Format(time.RFC3339Nano),
		EndedAt:              ended.UTC().Format(time.RFC3339Nano),
		DurationMS:           ended.Sub(started).Milliseconds(),
		DecisionMode:         provider.Mode,
		DecisionModel:        provider.Model,
		Rankings:             rankings,
		Comparisons:          comparisons,
		Scenarios:            results,
	}, nil
}

func buildScenarioConfig(scenario experimentScenarioSpec) (econ.Config, error) {
	cfg := econ.DefaultConfig()
	cfg = applyExperimentPreset(cfg, scenario.Preset)
	cfg.MarketAlgorithm = scenario.Algorithm
	applyScenarioAdjustments(&cfg, scenario.Adjustments)
	return cfg, nil
}

func applyScenarioAdjustments(cfg *econ.Config, adjust experimentConfigAdjusters) {
	if cfg == nil {
		return
	}
	baseDemandMultiplier := defaultMultiplier(adjust.BaseDemandMultiplier)
	startingTreasuryMultiplier := defaultMultiplier(adjust.StartingTreasuryMultiplier)
	startingReserveMultiplier := defaultMultiplier(adjust.StartingReserveMultiplier)
	routeCapacityMultiplier := defaultMultiplier(adjust.RouteCapacityMultiplier)
	extractorMultiplier := defaultMultiplier(adjust.ExtractorMultiplier)
	fossilOutputMultiplier := defaultMultiplier(adjust.FossilOutputMultiplier)
	renewableOutputMultiplier := defaultMultiplier(adjust.RenewableOutputMultiplier)
	tradeShareMultiplier := defaultMultiplier(adjust.TradeShareMultiplier)
	buildCostMultiplier := defaultMultiplier(adjust.BuildCostMultiplier)
	energyPriceMultiplier := defaultMultiplier(adjust.EnergyPriceMultiplier)
	emergencyPriceMultiplier := defaultMultiplier(adjust.EmergencyPriceMultiplier)
	computeEfficiencyMultiplier := defaultMultiplier(adjust.ComputeEfficiencyMultiplier)
	infrastructureMultiplier := defaultMultiplier(adjust.InfrastructureMultiplier)

	for i := range cfg.Countries {
		country := &cfg.Countries[i]
		country.BaseDemand *= baseDemandMultiplier
		country.StartingTreasury *= startingTreasuryMultiplier
		country.StartingReserve *= startingReserveMultiplier
		country.Infrastructure *= infrastructureMultiplier
		country.ComputeEfficiency = clamp(country.ComputeEfficiency*computeEfficiencyMultiplier+adjust.ComputeEfficiencyAdd, 0, cfg.MaxComputeEfficiency)
		country.Extractors.Coal *= extractorMultiplier
		country.Extractors.Oil *= extractorMultiplier
		country.Extractors.Copper *= extractorMultiplier
		country.Extractors.Silicon *= extractorMultiplier
	}

	for i := range cfg.Routes {
		cfg.Routes[i].Capacity *= routeCapacityMultiplier
	}

	cfg.CoalEnergyPerUnit *= fossilOutputMultiplier
	cfg.OilEnergyPerUnit *= fossilOutputMultiplier
	cfg.SolarUnitOutput *= renewableOutputMultiplier
	cfg.WindUnitOutput *= renewableOutputMultiplier
	cfg.TradeEnergyShare *= tradeShareMultiplier
	cfg.TradeCargoShare *= tradeShareMultiplier
	cfg.EnergySalePrice *= energyPriceMultiplier
	cfg.EmergencyEnergyPrice *= emergencyPriceMultiplier
	cfg.EnergyBuildTreasury *= buildCostMultiplier
	cfg.ComputeBuildTreasury *= buildCostMultiplier
	cfg.InfrastructureTreasury *= buildCostMultiplier
}

func rankExperimentResults(results []experimentScenarioResult) []experimentRanking {
	rankings := make([]experimentRanking, len(results))
	for i, scenario := range results {
		score := experimentOverallScore(scenario)
		rankings[i] = experimentRanking{
			ScenarioID:    scenario.Spec.ID,
			Label:         scenario.Spec.Label,
			OverallScore:  score,
			Living:        scenario.Summary.LivingCountries,
			PeakShortage:  scenario.Summary.PeakShortage,
			EnergyTrade:   scenario.Summary.CumulativeEnergyTraded,
			EnergyDeliver: scenario.Summary.CumulativeEnergyDelivered,
			Treasury:      scenario.Summary.FinalTreasury,
		}
	}
	sort.Slice(rankings, func(i, j int) bool {
		if rankings[i].OverallScore == rankings[j].OverallScore {
			return rankings[i].ScenarioID < rankings[j].ScenarioID
		}
		return rankings[i].OverallScore > rankings[j].OverallScore
	})
	for i := range rankings {
		rankings[i].Rank = i + 1
	}
	return rankings
}

func compareAgainstControl(controlID string, results []experimentScenarioResult) []experimentComparison {
	if controlID == "" {
		return nil
	}
	controlIndex := -1
	for i, result := range results {
		if result.Spec.ID == controlID {
			controlIndex = i
			break
		}
	}
	if controlIndex == -1 {
		return nil
	}
	control := results[controlIndex]
	comparisons := make([]experimentComparison, 0, len(results)-1)
	controlScore := experimentOverallScore(control)
	for _, scenario := range results {
		if scenario.Spec.ID == controlID {
			continue
		}
		score := experimentOverallScore(scenario)
		comparisons = append(comparisons, experimentComparison{
			ScenarioID:             scenario.Spec.ID,
			Label:                  scenario.Spec.Label,
			AgainstControlID:       controlID,
			DeltaLivingCountries:   scenario.Summary.LivingCountries - control.Summary.LivingCountries,
			DeltaPeakShortage:      scenario.Summary.PeakShortage - control.Summary.PeakShortage,
			DeltaEnergyDelivered:   scenario.Summary.CumulativeEnergyDelivered - control.Summary.CumulativeEnergyDelivered,
			DeltaEnergyTraded:      scenario.Summary.CumulativeEnergyTraded - control.Summary.CumulativeEnergyTraded,
			DeltaFinalTreasury:     scenario.Summary.FinalTreasury - control.Summary.FinalTreasury,
			DeltaOverallScore:      score - controlScore,
			SupportsHypothesisHint: score > controlScore,
		})
	}
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].DeltaOverallScore > comparisons[j].DeltaOverallScore
	})
	return comparisons
}

func experimentOverallScore(scenario experimentScenarioResult) float64 {
	return float64(scenario.Summary.LivingCountries)*100000 +
		scenario.Summary.CumulativeEnergyDelivered*140 +
		scenario.Summary.CumulativeEnergyTraded*18 +
		scenario.Summary.FinalTreasury*4 -
		scenario.Summary.PeakShortage*150
}

func writeExperimentResult(path string, result experimentResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func readExperimentResult(path string) (experimentResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return experimentResult{}, err
	}
	var result experimentResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return experimentResult{}, err
	}
	return result, nil
}

func printExperimentSummary(result experimentResult) {
	fmt.Printf("Experiment: %s\n", result.Name)
	fmt.Printf("Question:   %s\n", result.Question)
	fmt.Printf("Hypothesis: %s\n", result.Hypothesis)
	if result.Notes != "" {
		fmt.Printf("Notes:      %s\n", result.Notes)
	}
	fmt.Printf("Mode:       %s", result.DecisionMode)
	if result.DecisionModel != "" {
		fmt.Printf(" (%s)", result.DecisionModel)
	}
	fmt.Printf("\nSteps:      %d\n", result.Steps)

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Rank\tScenario\tAlive\tPeakShort\tDelivered\tTrade\tTreasury\tScore")
	for _, ranking := range result.Rankings {
		fmt.Fprintf(w, "%d\t%s\t%d\t%.2f\t%.1f\t%.1f\t%.1f\t%.0f\n",
			ranking.Rank,
			ranking.Label,
			ranking.Living,
			ranking.PeakShortage,
			ranking.EnergyDeliver,
			ranking.EnergyTrade,
			ranking.Treasury,
			ranking.OverallScore,
		)
	}
	_ = w.Flush()

	if len(result.Comparisons) > 0 {
		fmt.Printf("\nControl comparisons vs %s:\n", result.ControlScenarioID)
		cw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(cw, "Scenario\tAliveDelta\tPeakShortDelta\tDeliveredDelta\tTradeDelta\tTreasuryDelta\tScoreDelta")
		for _, comparison := range result.Comparisons {
			fmt.Fprintf(cw, "%s\t%+d\t%+.2f\t%+.1f\t%+.1f\t%+.1f\t%+.0f\n",
				comparison.Label,
				comparison.DeltaLivingCountries,
				comparison.DeltaPeakShortage,
				comparison.DeltaEnergyDelivered,
				comparison.DeltaEnergyTraded,
				comparison.DeltaFinalTreasury,
				comparison.DeltaOverallScore,
			)
		}
		_ = cw.Flush()
		best := result.Comparisons[0]
		verdict := "not supported"
		if best.SupportsHypothesisHint {
			verdict = "supported"
		}
		fmt.Printf("\nHypothesis verdict: %s (best variant: %s)\n", verdict, best.Label)
	}
}

func defaultMultiplier(value float64) float64 {
	if value == 0 {
		return 1
	}
	return value
}

func clamp(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
