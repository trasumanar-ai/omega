package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type experimentTracePoint struct {
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

type experimentScenarioSpecView struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Role      string `json:"role"`
	Preset    string `json:"preset"`
	Algorithm string `json:"algorithm"`
	Notes     string `json:"notes"`
}

type experimentScenarioSummaryView struct {
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

type experimentScenarioResultView struct {
	Spec    experimentScenarioSpecView    `json:"spec"`
	Summary experimentScenarioSummaryView `json:"summary"`
	Trace   []experimentTracePoint        `json:"trace"`
}

type experimentRankingView struct {
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

type experimentComparisonView struct {
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

type experimentResultView struct {
	Version              int                            `json:"version"`
	Name                 string                         `json:"name"`
	Question             string                         `json:"question"`
	Hypothesis           string                         `json:"hypothesis"`
	Notes                string                         `json:"notes"`
	ControlScenarioID    string                         `json:"controlScenarioId"`
	IndependentVariables []string                       `json:"independentVariables"`
	DependentMetrics     []string                       `json:"dependentMetrics"`
	Steps                int                            `json:"steps"`
	SampleEvery          int                            `json:"sampleEvery"`
	StartedAt            string                         `json:"startedAt"`
	EndedAt              string                         `json:"endedAt"`
	DurationMS           int64                          `json:"durationMs"`
	DecisionMode         string                         `json:"decisionMode"`
	DecisionModel        string                         `json:"decisionModel"`
	OutputPath           string                         `json:"outputPath"`
	Rankings             []experimentRankingView        `json:"rankings"`
	Comparisons          []experimentComparisonView     `json:"comparisons"`
	Scenarios            []experimentScenarioResultView `json:"scenarios"`
}

type experimentInfo struct {
	ID                   string                  `json:"id"`
	Name                 string                  `json:"name"`
	Question             string                  `json:"question"`
	Hypothesis           string                  `json:"hypothesis"`
	StartedAt            string                  `json:"startedAt"`
	EndedAt              string                  `json:"endedAt"`
	OutputPath           string                  `json:"outputPath"`
	DecisionMode         string                  `json:"decisionMode"`
	DecisionModel        string                  `json:"decisionModel"`
	ControlScenarioID    string                  `json:"controlScenarioId"`
	ScenarioCount        int                     `json:"scenarioCount"`
	IndependentVariables []string                `json:"independentVariables"`
	DependentMetrics     []string                `json:"dependentMetrics"`
	Rankings             []experimentRankingView `json:"rankings"`
}

type experimentStore struct {
	dir string
}

func newExperimentStore(dir string) (*experimentStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &experimentStore{dir: dir}, nil
}

func (s *experimentStore) list() ([]experimentInfo, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	items := make([]experimentInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := s.get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		items = append(items, record.info())
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].EndedAt > items[j].EndedAt
	})
	return items, nil
}

func (s *experimentStore) get(id string) (*experimentResultView, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var record experimentResultView
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *experimentResultView) info() experimentInfo {
	if r == nil {
		return experimentInfo{}
	}
	id := strings.TrimSuffix(filepath.Base(r.OutputPath), filepath.Ext(r.OutputPath))
	if id == "." || id == "" {
		id = strings.TrimSuffix(filepath.Base(r.OutputPath), ".json")
	}
	return experimentInfo{
		ID:                   id,
		Name:                 r.Name,
		Question:             r.Question,
		Hypothesis:           r.Hypothesis,
		StartedAt:            r.StartedAt,
		EndedAt:              r.EndedAt,
		OutputPath:           r.OutputPath,
		DecisionMode:         r.DecisionMode,
		DecisionModel:        r.DecisionModel,
		ControlScenarioID:    r.ControlScenarioID,
		ScenarioCount:        len(r.Scenarios),
		IndependentVariables: append([]string(nil), r.IndependentVariables...),
		DependentMetrics:     append([]string(nil), r.DependentMetrics...),
		Rankings:             append([]experimentRankingView(nil), r.Rankings...),
	}
}
