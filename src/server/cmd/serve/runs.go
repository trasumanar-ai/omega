package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"omega/server/internal/econ"
)

type runSummary struct {
	TickCount            int     `json:"tickCount"`
	CurrentTick          int     `json:"currentTick"`
	LivingCountries      int     `json:"livingCountries"`
	FinalEnergyDelivered float64 `json:"finalEnergyDelivered"`
	FinalEnergyTraded    float64 `json:"finalEnergyTraded"`
	FinalCargoTraded     float64 `json:"finalCargoTraded"`
	FinalShortage        float64 `json:"finalShortage"`
	FinalTreasury        float64 `json:"finalTreasury"`
}

type tickRecord struct {
	Tick       int                 `json:"tick"`
	RecordedAt string              `json:"recordedAt"`
	Stats      econ.TickStats      `json:"stats"`
	World      econ.WorldSnapshot  `json:"world"`
	Market     econ.MarketSnapshot `json:"market"`
	Events     []string            `json:"events"`
}

type runInfo struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	StartedAt   string     `json:"startedAt"`
	UpdatedAt   string     `json:"updatedAt"`
	CompletedAt string     `json:"completedAt,omitempty"`
	FilePath    string     `json:"filePath"`
	LLMMode     string     `json:"llmMode"`
	LLMModel    string     `json:"llmModel"`
	RecordCount int        `json:"recordCount"`
	Summary     runSummary `json:"summary"`
}

type runRecord struct {
	ID          string       `json:"id"`
	Status      string       `json:"status"`
	StartedAt   string       `json:"startedAt"`
	UpdatedAt   string       `json:"updatedAt"`
	CompletedAt string       `json:"completedAt,omitempty"`
	FilePath    string       `json:"filePath"`
	LLMMode     string       `json:"llmMode"`
	LLMModel    string       `json:"llmModel"`
	Config      econ.Config  `json:"config"`
	Records     []tickRecord `json:"records"`
	Summary     runSummary   `json:"summary"`
}

type runRecorder struct {
	dir     string
	current *runRecord
}

func newRunRecorder(dir string) (*runRecorder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &runRecorder{dir: dir}, nil
}

func (r *runRecorder) start(cfg econ.Config, llmMode string, llmModel string, initial tickRecord) error {
	runID := fmt.Sprintf("econ-%s", time.Now().UTC().Format("20060102T150405.000000000Z"))
	filePath := filepath.Join(r.dir, runID+".json")
	now := initial.RecordedAt
	r.current = &runRecord{
		ID:        runID,
		Status:    "running",
		StartedAt: now,
		UpdatedAt: now,
		FilePath:  filePath,
		LLMMode:   llmMode,
		LLMModel:  llmModel,
		Config:    cfg,
		Records:   []tickRecord{initial},
	}
	r.current.refreshSummary()
	return r.saveCurrent()
}

func (r *runRecorder) append(record tickRecord) error {
	if r.current == nil {
		return fmt.Errorf("no active run")
	}
	r.current.Records = append(r.current.Records, record)
	r.current.UpdatedAt = record.RecordedAt
	r.current.refreshSummary()
	return r.saveCurrent()
}

func (r *runRecorder) finalize(status string) error {
	if r.current == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.current.Status = status
	r.current.CompletedAt = now
	r.current.UpdatedAt = now
	r.current.refreshSummary()
	return r.saveCurrent()
}

func (r *runRecorder) currentInfo() runInfo {
	if r.current == nil {
		return runInfo{}
	}
	return r.current.info()
}

func (r *runRecorder) currentRecord() *runRecord {
	if r.current == nil {
		return nil
	}
	copy := cloneRunRecord(r.current)
	return &copy
}

func (r *runRecorder) list() ([]runInfo, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	runs := make([]runInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := r.loadByPath(filepath.Join(r.dir, entry.Name()))
		if err != nil {
			continue
		}
		runs = append(runs, record.info())
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].UpdatedAt > runs[j].UpdatedAt
	})
	return runs, nil
}

func (r *runRecorder) get(id string) (*runRecord, error) {
	if r.current != nil && r.current.ID == id {
		copy := cloneRunRecord(r.current)
		return &copy, nil
	}
	return r.loadByPath(filepath.Join(r.dir, id+".json"))
}

func (r *runRecorder) loadByPath(path string) (*runRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record runRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *runRecorder) saveCurrent() error {
	if r.current == nil {
		return nil
	}
	return writeJSONFile(r.current.FilePath, r.current)
}

func (r *runRecord) refreshSummary() {
	if len(r.Records) == 0 {
		r.Summary = runSummary{}
		return
	}
	last := r.Records[len(r.Records)-1]
	r.Summary = runSummary{
		TickCount:            len(r.Records),
		CurrentTick:          last.Tick,
		LivingCountries:      last.Stats.LivingCountries,
		FinalEnergyDelivered: last.Stats.TotalEnergyDelivered,
		FinalEnergyTraded:    last.Stats.TotalEnergyTraded,
		FinalCargoTraded:     last.Stats.TotalCargoTraded,
		FinalShortage:        last.Stats.TotalShortage,
		FinalTreasury:        last.Stats.TotalTreasury,
	}
}

func (r *runRecord) info() runInfo {
	return runInfo{
		ID:          r.ID,
		Status:      r.Status,
		StartedAt:   r.StartedAt,
		UpdatedAt:   r.UpdatedAt,
		CompletedAt: r.CompletedAt,
		FilePath:    r.FilePath,
		LLMMode:     r.LLMMode,
		LLMModel:    r.LLMModel,
		RecordCount: len(r.Records),
		Summary:     r.Summary,
	}
}

func cloneRunRecord(run *runRecord) runRecord {
	if run == nil {
		return runRecord{}
	}
	clone := *run
	clone.Config.Countries = append([]econ.CountryConfig(nil), run.Config.Countries...)
	clone.Config.Routes = append([]econ.RouteConfig(nil), run.Config.Routes...)
	clone.Records = make([]tickRecord, len(run.Records))
	for i, record := range run.Records {
		clone.Records[i] = tickRecord{
			Tick:       record.Tick,
			RecordedAt: record.RecordedAt,
			Stats:      record.Stats,
			World:      cloneWorld(record.World),
			Market:     cloneMarket(record.Market),
			Events:     append([]string(nil), record.Events...),
		}
	}
	return clone
}

func cloneWorld(world econ.WorldSnapshot) econ.WorldSnapshot {
	countries := make([]econ.CountrySnapshot, len(world.Countries))
	for i, country := range world.Countries {
		countries[i] = country
		countries[i].MarketQuotes = append([]econ.ResourceQuoteSnapshot(nil), country.MarketQuotes...)
	}
	return econ.WorldSnapshot{
		Tick:      world.Tick,
		Countries: countries,
		Routes:    append([]econ.RouteSnapshot(nil), world.Routes...),
	}
}

func cloneMarket(market econ.MarketSnapshot) econ.MarketSnapshot {
	countries := make([]econ.MarketCountrySnapshot, len(market.Countries))
	for i, country := range market.Countries {
		countries[i] = country
		countries[i].Quotes = append([]econ.ResourceQuoteSnapshot(nil), country.Quotes...)
	}
	return econ.MarketSnapshot{
		EnergyPrice:            market.EnergyPrice,
		EmergencyEnergyPrice:   market.EmergencyEnergyPrice,
		CoalPrice:              market.CoalPrice,
		OilPrice:               market.OilPrice,
		CopperPrice:            market.CopperPrice,
		SiliconPrice:           market.SiliconPrice,
		EnergyBuildTreasury:    market.EnergyBuildTreasury,
		ComputeBuildTreasury:   market.ComputeBuildTreasury,
		InfrastructureTreasury: market.InfrastructureTreasury,
		TotalDemand:            market.TotalDemand,
		TotalGenerated:         market.TotalGenerated,
		TotalDelivered:         market.TotalDelivered,
		TotalImported:          market.TotalImported,
		TotalExported:          market.TotalExported,
		TotalShortage:          market.TotalShortage,
		TotalTreasury:          market.TotalTreasury,
		Countries:              countries,
	}
}

func writeJSONFile(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func makeInitialRecord(stats econ.TickStats, world econ.WorldSnapshot, market econ.MarketSnapshot, llmMode string) tickRecord {
	return tickRecord{
		Tick:       stats.Tick,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Stats:      stats,
		World:      world,
		Market:     market,
		Events: []string{
			fmt.Sprintf("Run started at tick %d.", world.Tick),
			fmt.Sprintf("Decision mode: %s.", llmMode),
		},
	}
}

func deriveEvents(prev econ.WorldSnapshot, current econ.WorldSnapshot) []string {
	events := make([]string, 0, 12)
	prevCountries := make(map[string]econ.CountrySnapshot, len(prev.Countries))
	for _, country := range prev.Countries {
		prevCountries[country.ID] = country
	}

	for _, country := range current.Countries {
		if before, ok := prevCountries[country.ID]; ok && before.Alive && !country.Alive {
			events = append(events, fmt.Sprintf("%s collapsed after stability hit zero.", country.Name))
		}
		if country.LastBuildSuccess {
			events = append(events, fmt.Sprintf("%s built %s.", country.Name, humanizeBuildKind(string(country.LastBuildKind))))
		}
		if country.LastImportedEnergy > 1 {
			events = append(events, fmt.Sprintf("%s imported %.1f energy.", country.Name, country.LastImportedEnergy))
		}
		if country.LastExportedEnergy > 1 {
			events = append(events, fmt.Sprintf("%s exported %.1f energy.", country.Name, country.LastExportedEnergy))
		}
		if country.LastShortage > 0.5 {
			events = append(events, fmt.Sprintf("%s had a %.1f energy shortage.", country.Name, country.LastShortage))
		}
		if country.LastEmergencySpend > 0.5 {
			events = append(events, fmt.Sprintf("%s spent %.1f on emergency energy.", country.Name, country.LastEmergencySpend))
		}
		if country.LastBuildSpend > 0.5 {
			events = append(events, fmt.Sprintf("%s invested %.1f treasury into %s.", country.Name, country.LastBuildSpend, string(country.LastBuildFocus)))
		}
	}

	for _, route := range current.Routes {
		energyFlow := route.LastEnergyAB + route.LastEnergyBA
		cargoFlow := route.LastCargoAB + route.LastCargoBA
		if energyFlow > 2.5 {
			events = append(events, fmt.Sprintf("Route %s-%s moved %.1f energy.", route.A, route.B, energyFlow))
		}
		if cargoFlow > 1.5 {
			events = append(events, fmt.Sprintf("Route %s-%s moved %.1f cargo.", route.A, route.B, cargoFlow))
		}
	}

	if len(events) == 0 {
		events = append(events, fmt.Sprintf("Tick %d stayed stable with no major shortages or builds.", current.Tick))
	}
	if len(events) > 14 {
		events = events[:14]
	}
	return events
}

func humanizeBuildKind(kind string) string {
	if kind == "" || kind == "none" {
		return "hold"
	}
	return strings.ReplaceAll(kind, "_", " ")
}
