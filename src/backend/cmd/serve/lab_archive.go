package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"omega/backend/internal/econ"
)

type labArchiveSaveRequest struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

type labArchiveWinnerSummary struct {
	BestOverall    string `json:"bestOverall"`
	BestSurvival   string `json:"bestSurvival"`
	BestTrade      string `json:"bestTrade"`
	BestTreasury   string `json:"bestTreasury"`
	LowestShortage string `json:"lowestShortage"`
}

type labArchiveScenarioScore struct {
	ScenarioID      string  `json:"scenarioId"`
	Label           string  `json:"label"`
	LivingCountries int     `json:"livingCountries"`
	EnergyDelivered float64 `json:"energyDelivered"`
	EnergyTraded    float64 `json:"energyTraded"`
	PeakShortage    float64 `json:"peakShortage"`
	FinalTreasury   float64 `json:"finalTreasury"`
	OverallScore    float64 `json:"overallScore"`
}

type labArchiveSummary struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Notes         string                    `json:"notes"`
	SavedAt       string                    `json:"savedAt"`
	FilePath      string                    `json:"filePath"`
	SessionID     string                    `json:"sessionId"`
	SessionName   string                    `json:"sessionName"`
	Tick          int                       `json:"tick"`
	ScenarioCount int                       `json:"scenarioCount"`
	PresetIDs     []string                  `json:"presetIds"`
	AlgorithmIDs  []string                  `json:"algorithmIds"`
	Winners       labArchiveWinnerSummary   `json:"winners"`
	Rankings      []labArchiveScenarioScore `json:"rankings"`
}

type labArchiveRecord struct {
	Summary labArchiveSummary  `json:"summary"`
	Session labSessionSnapshot `json:"session"`
}

type labArchiveStore struct {
	dir string
}

func newLabArchiveStore(dir string) (*labArchiveStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &labArchiveStore{dir: dir}, nil
}

func (s *labArchiveStore) save(session labSessionSnapshot, name string, notes string) (labArchiveSummary, error) {
	now := time.Now().UTC()
	archiveID := fmt.Sprintf("lab-result-%s", now.Format("20060102T150405.000000000Z"))
	filePath := filepath.Join(s.dir, archiveID+".json")
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = fmt.Sprintf("%s @ tick %d", session.Name, session.Tick)
	}
	rankings := scoreLabScenarios(session.Scenarios)
	summary := labArchiveSummary{
		ID:            archiveID,
		Name:          trimmedName,
		Notes:         strings.TrimSpace(notes),
		SavedAt:       now.Format(time.RFC3339),
		FilePath:      filePath,
		SessionID:     session.ID,
		SessionName:   session.Name,
		Tick:          session.Tick,
		ScenarioCount: len(session.Scenarios),
		PresetIDs:     append([]string(nil), session.PresetIDs...),
		AlgorithmIDs:  stringifyAlgorithms(session.AlgorithmIDs),
		Winners:       deriveArchiveWinners(session.Scenarios),
		Rankings:      rankings,
	}
	record := labArchiveRecord{
		Summary: summary,
		Session: cloneLabSessionSnapshot(session),
	}
	if err := writeJSONFile(filePath, record); err != nil {
		return labArchiveSummary{}, err
	}
	return summary, nil
}

func (s *labArchiveStore) list() ([]labArchiveSummary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	records := make([]labArchiveSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := s.get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		records = append(records, record.Summary)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].SavedAt > records[j].SavedAt
	})
	return records, nil
}

func (s *labArchiveStore) get(id string) (*labArchiveRecord, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var record labArchiveRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func deriveArchiveWinners(scenarios []labScenarioSnapshot) labArchiveWinnerSummary {
	if len(scenarios) == 0 {
		return labArchiveWinnerSummary{}
	}

	bestOverall := scoreLabScenarios(scenarios)
	bestSurvival := scenarios[0]
	bestTrade := scenarios[0]
	bestTreasury := scenarios[0]
	lowestShortage := scenarios[0]

	for _, scenario := range scenarios[1:] {
		if compareSurvival(scenario, bestSurvival) > 0 {
			bestSurvival = scenario
		}
		if scenario.Summary.CumulativeEnergyTraded > bestTrade.Summary.CumulativeEnergyTraded {
			bestTrade = scenario
		}
		if scenario.Summary.FinalTreasury > bestTreasury.Summary.FinalTreasury {
			bestTreasury = scenario
		}
		if compareShortage(scenario, lowestShortage) > 0 {
			lowestShortage = scenario
		}
	}

	return labArchiveWinnerSummary{
		BestOverall:    bestOverall[0].Label,
		BestSurvival:   bestSurvival.Label,
		BestTrade:      bestTrade.Label,
		BestTreasury:   bestTreasury.Label,
		LowestShortage: lowestShortage.Label,
	}
}

func scoreLabScenarios(scenarios []labScenarioSnapshot) []labArchiveScenarioScore {
	scores := make([]labArchiveScenarioScore, len(scenarios))
	for i, scenario := range scenarios {
		score := float64(scenario.Summary.LivingCountries)*100000 +
			scenario.Summary.CumulativeEnergyDelivered*140 +
			scenario.Summary.CumulativeEnergyTraded*18 +
			scenario.Summary.FinalTreasury*4 -
			scenario.Summary.PeakShortage*150
		scores[i] = labArchiveScenarioScore{
			ScenarioID:      scenario.ID,
			Label:           scenario.Label,
			LivingCountries: scenario.Summary.LivingCountries,
			EnergyDelivered: scenario.Summary.CumulativeEnergyDelivered,
			EnergyTraded:    scenario.Summary.CumulativeEnergyTraded,
			PeakShortage:    scenario.Summary.PeakShortage,
			FinalTreasury:   scenario.Summary.FinalTreasury,
			OverallScore:    score,
		}
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].OverallScore == scores[j].OverallScore {
			return scores[i].ScenarioID < scores[j].ScenarioID
		}
		return scores[i].OverallScore > scores[j].OverallScore
	})
	return scores
}

func compareSurvival(a labScenarioSnapshot, b labScenarioSnapshot) int {
	if a.Summary.LivingCountries != b.Summary.LivingCountries {
		return a.Summary.LivingCountries - b.Summary.LivingCountries
	}
	if a.Summary.CumulativeEnergyDelivered > b.Summary.CumulativeEnergyDelivered {
		return 1
	}
	if a.Summary.CumulativeEnergyDelivered < b.Summary.CumulativeEnergyDelivered {
		return -1
	}
	return 0
}

func compareShortage(a labScenarioSnapshot, b labScenarioSnapshot) int {
	if a.Summary.PeakShortage < b.Summary.PeakShortage {
		return 1
	}
	if a.Summary.PeakShortage > b.Summary.PeakShortage {
		return -1
	}
	if a.Summary.CumulativeEnergyDelivered > b.Summary.CumulativeEnergyDelivered {
		return 1
	}
	if a.Summary.CumulativeEnergyDelivered < b.Summary.CumulativeEnergyDelivered {
		return -1
	}
	return 0
}

func stringifyAlgorithms(algorithms []econ.MarketAlgorithm) []string {
	values := make([]string, len(algorithms))
	for i, algorithm := range algorithms {
		values[i] = string(algorithm)
	}
	return values
}

func cloneLabSessionSnapshot(session labSessionSnapshot) labSessionSnapshot {
	clone := session
	clone.PresetIDs = append([]string(nil), session.PresetIDs...)
	clone.AlgorithmIDs = append([]econ.MarketAlgorithm(nil), session.AlgorithmIDs...)
	clone.Scenarios = make([]labScenarioSnapshot, len(session.Scenarios))
	for i, scenario := range session.Scenarios {
		clone.Scenarios[i] = labScenarioSnapshot{
			ID:           scenario.ID,
			Label:        scenario.Label,
			PresetID:     scenario.PresetID,
			PresetLabel:  scenario.PresetLabel,
			Algorithm:    scenario.Algorithm,
			Summary:      scenario.Summary,
			StatsHistory: append([]econ.TickStats(nil), scenario.StatsHistory...),
			World:        cloneWorld(scenario.World),
			Market:       cloneMarket(scenario.Market),
			RecentEvents: append([]string(nil), scenario.RecentEvents...),
		}
	}
	return clone
}
