package main

import (
	"fmt"
	"strings"
	"time"

	"omega/server/internal/econ"
)

const (
	maxLabHistory  = 180
	maxLabSessions = 12
)

type labPresetMeta struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type labScenarioSnapshot struct {
	ID           string                 `json:"id"`
	Label        string                 `json:"label"`
	PresetID     string                 `json:"presetId"`
	PresetLabel  string                 `json:"presetLabel"`
	Algorithm    econ.MarketAlgorithm   `json:"algorithm"`
	Summary      compareScenarioSummary `json:"summary"`
	StatsHistory []econ.TickStats       `json:"statsHistory"`
	World        econ.WorldSnapshot     `json:"world"`
	Market       econ.MarketSnapshot    `json:"market"`
	RecentEvents []string               `json:"recentEvents"`
}

type labSessionSnapshot struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
	Tick         int                    `json:"tick"`
	PresetIDs    []string               `json:"presetIds"`
	AlgorithmIDs []econ.MarketAlgorithm `json:"algorithmIds"`
	Scenarios    []labScenarioSnapshot  `json:"scenarios"`
}

type labResponse struct {
	Presets    []labPresetMeta        `json:"presets"`
	Algorithms []econ.MarketAlgorithm `json:"algorithms"`
	Sessions   []labSessionSnapshot   `json:"sessions"`
}

type labCreateRequest struct {
	Name       string                 `json:"name"`
	Presets    []string               `json:"presets"`
	Algorithms []econ.MarketAlgorithm `json:"algorithms"`
}

type labResetRequest struct {
	Presets    []string               `json:"presets"`
	Algorithms []econ.MarketAlgorithm `json:"algorithms"`
}

type labManager struct {
	baseConfig econ.Config
	provider   econ.DecisionProvider
	counter    int
	sessions   []*labSession
}

type labSession struct {
	id        string
	name      string
	createdAt string
	updatedAt string
	lab       *scenarioLab
}

type scenarioLab struct {
	baseConfig         econ.Config
	provider           econ.DecisionProvider
	tick               int
	selectedPresets    []string
	selectedAlgorithms []econ.MarketAlgorithm
	scenarios          []labScenarioState
}

type labScenarioState struct {
	id                   string
	label                string
	presetID             string
	presetLabel          string
	algorithm            econ.MarketAlgorithm
	engine               *econ.Simulation
	stats                econ.TickStats
	world                econ.WorldSnapshot
	market               econ.MarketSnapshot
	events               []string
	history              []econ.TickStats
	totalEnergyTrade     float64
	totalCargoTrade      float64
	totalEnergyDelivered float64
	peakShortage         float64
}

func newLabManager(base econ.Config, provider econ.DecisionProvider) (*labManager, error) {
	manager := &labManager{
		baseConfig: base,
		provider:   provider,
	}
	if _, err := manager.createSession("Baseline", nil, nil); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *labManager) createSession(name string, presets []string, algorithms []econ.MarketAlgorithm) (labSessionSnapshot, error) {
	if len(m.sessions) >= maxLabSessions {
		return labSessionSnapshot{}, fmt.Errorf("maximum lab session count reached (%d)", maxLabSessions)
	}

	lab := &scenarioLab{
		baseConfig: m.baseConfig,
		provider:   m.provider,
	}
	if err := lab.reset(presets, algorithms); err != nil {
		return labSessionSnapshot{}, err
	}

	m.counter++
	now := time.Now().UTC().Format(time.RFC3339)
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = fmt.Sprintf("Session %d", m.counter)
	}

	session := &labSession{
		id:        fmt.Sprintf("lab-%03d", m.counter),
		name:      trimmedName,
		createdAt: now,
		updatedAt: now,
		lab:       lab,
	}
	m.sessions = append([]*labSession{session}, m.sessions...)
	return session.snapshot(), nil
}

func (m *labManager) stepAll(count int) {
	for _, session := range m.sessions {
		session.lab.step(count)
		session.touch()
	}
}

func (m *labManager) resetAll() error {
	for _, session := range m.sessions {
		session.lab.baseConfig = m.baseConfig
		session.lab.provider = m.provider
		if err := session.lab.reset(session.lab.selectedPresets, session.lab.selectedAlgorithms); err != nil {
			return err
		}
		session.touch()
	}
	return nil
}

func (m *labManager) rebase(base econ.Config) error {
	m.baseConfig = base
	return m.resetAll()
}

func (m *labManager) stepSession(id string, count int) error {
	session, err := m.findSession(id)
	if err != nil {
		return err
	}
	session.lab.step(count)
	session.touch()
	return nil
}

func (m *labManager) resetSession(id string, presets []string, algorithms []econ.MarketAlgorithm) error {
	session, err := m.findSession(id)
	if err != nil {
		return err
	}
	session.lab.baseConfig = m.baseConfig
	session.lab.provider = m.provider
	if err := session.lab.reset(presets, algorithms); err != nil {
		return err
	}
	session.touch()
	return nil
}

func (m *labManager) deleteSession(id string) error {
	index := -1
	for i, session := range m.sessions {
		if session.id == id {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("lab session not found: %s", id)
	}

	m.sessions = append(m.sessions[:index], m.sessions[index+1:]...)
	if len(m.sessions) == 0 {
		_, err := m.createSession("Baseline", nil, nil)
		return err
	}
	return nil
}

func (m *labManager) snapshot() labResponse {
	response := labResponse{
		Presets:    knownPresets(),
		Algorithms: econ.KnownMarketAlgorithms(),
		Sessions:   make([]labSessionSnapshot, len(m.sessions)),
	}
	for i, session := range m.sessions {
		response.Sessions[i] = session.snapshot()
	}
	return response
}

func (m *labManager) findSession(id string) (*labSession, error) {
	for _, session := range m.sessions {
		if session.id == id {
			return session, nil
		}
	}
	return nil, fmt.Errorf("lab session not found: %s", id)
}

func (s *labSession) touch() {
	s.updatedAt = time.Now().UTC().Format(time.RFC3339)
}

func (s *labSession) snapshot() labSessionSnapshot {
	return labSessionSnapshot{
		ID:           s.id,
		Name:         s.name,
		CreatedAt:    s.createdAt,
		UpdatedAt:    s.updatedAt,
		Tick:         s.lab.tick,
		PresetIDs:    append([]string(nil), s.lab.selectedPresets...),
		AlgorithmIDs: append([]econ.MarketAlgorithm(nil), s.lab.selectedAlgorithms...),
		Scenarios:    s.lab.scenarioSnapshots(),
	}
}

func (l *scenarioLab) reset(presets []string, algorithms []econ.MarketAlgorithm) error {
	normalizedPresets, err := normalizePresetIDs(presets)
	if err != nil {
		return err
	}
	normalizedAlgorithms, err := normalizeAlgorithms(algorithms)
	if err != nil {
		return err
	}

	scenarios := make([]labScenarioState, 0, len(normalizedPresets)*len(normalizedAlgorithms))
	for _, presetID := range normalizedPresets {
		presetMeta, ok := lookupPreset(presetID)
		if !ok {
			return fmt.Errorf("unknown preset: %s", presetID)
		}
		for _, algorithm := range normalizedAlgorithms {
			cfg := applyDistributionPreset(l.baseConfig, presetID)
			cfg.MarketAlgorithm = algorithm
			engine, err := econ.NewSimulation(cfg)
			if err != nil {
				return err
			}
			world := engine.Snapshot()
			scenarios = append(scenarios, labScenarioState{
				id:          fmt.Sprintf("%s__%s", presetID, algorithm),
				label:       fmt.Sprintf("%s · %s", presetMeta.Label, compareLabel(algorithm)),
				presetID:    presetMeta.ID,
				presetLabel: presetMeta.Label,
				algorithm:   algorithm,
				engine:      engine,
				stats: econ.TickStats{
					Tick:            0,
					LivingCountries: len(cfg.Countries),
				},
				world:  world,
				market: engine.MarketSnapshot(),
				events: []string{
					fmt.Sprintf("%s initialized.", presetMeta.Label),
					fmt.Sprintf("Market algorithm: %s.", algorithm),
				},
				history: make([]econ.TickStats, 0, maxLabHistory),
			})
		}
	}

	l.tick = 0
	l.selectedPresets = append([]string(nil), normalizedPresets...)
	l.selectedAlgorithms = append([]econ.MarketAlgorithm(nil), normalizedAlgorithms...)
	l.scenarios = scenarios
	return nil
}

func (l *scenarioLab) step(count int) {
	if count <= 0 {
		count = 1
	}
	if count > 500 {
		count = 500
	}
	for i := 0; i < count; i++ {
		l.tick++
		for idx := range l.scenarios {
			scenario := &l.scenarios[idx]
			prevWorld := scenario.world
			scenario.stats = scenario.engine.Step(l.provider)
			scenario.world = scenario.engine.Snapshot()
			scenario.market = scenario.engine.MarketSnapshot()
			scenario.events = deriveEvents(prevWorld, scenario.world)
			scenario.totalEnergyTrade += scenario.stats.TotalEnergyTraded
			scenario.totalCargoTrade += scenario.stats.TotalCargoTraded
			scenario.totalEnergyDelivered += scenario.stats.TotalEnergyDelivered
			if scenario.stats.TotalShortage > scenario.peakShortage {
				scenario.peakShortage = scenario.stats.TotalShortage
			}
			scenario.history = append(scenario.history, scenario.stats)
			if len(scenario.history) > maxLabHistory {
				scenario.history = append([]econ.TickStats(nil), scenario.history[len(scenario.history)-maxLabHistory:]...)
			}
		}
	}
}

func (l *scenarioLab) scenarioSnapshots() []labScenarioSnapshot {
	response := make([]labScenarioSnapshot, len(l.scenarios))
	for i, scenario := range l.scenarios {
		response[i] = labScenarioSnapshot{
			ID:           scenario.id,
			Label:        scenario.label,
			PresetID:     scenario.presetID,
			PresetLabel:  scenario.presetLabel,
			Algorithm:    scenario.algorithm,
			Summary:      scenario.summary(),
			StatsHistory: append([]econ.TickStats(nil), scenario.history...),
			World:        cloneWorld(scenario.world),
			Market:       cloneMarket(scenario.market),
			RecentEvents: append([]string(nil), scenario.events...),
		}
	}
	return response
}

func (s *labScenarioState) summary() compareScenarioSummary {
	return compareScenarioSummary{
		FinalTick:                 s.stats.Tick,
		LivingCountries:           s.stats.LivingCountries,
		FinalEnergyDelivered:      s.stats.TotalEnergyDelivered,
		FinalShortage:             s.stats.TotalShortage,
		FinalTreasury:             s.stats.TotalTreasury,
		FinalComputeEfficiency:    s.stats.AvgComputeEfficiency,
		CumulativeEnergyTraded:    s.totalEnergyTrade,
		CumulativeCargoTraded:     s.totalCargoTrade,
		CumulativeEnergyDelivered: s.totalEnergyDelivered,
		PeakShortage:              s.peakShortage,
	}
}

func knownPresets() []labPresetMeta {
	return []labPresetMeta{
		{ID: "default", Label: "Default Basin", Description: "Base world with the current resource layout."},
		{ID: "fossil_skew", Label: "Fossil Skew", Description: "Coal and oil are more concentrated; renewables are weaker."},
		{ID: "renewable_skew", Label: "Renewable Skew", Description: "Sun and wind are stronger; fossil extraction is weaker."},
		{ID: "metal_bottleneck", Label: "Metal Bottleneck", Description: "Copper and silicon are scarce, stressing build chains."},
	}
}

func allPresetIDs() []string {
	presets := knownPresets()
	ids := make([]string, len(presets))
	for i, preset := range presets {
		ids[i] = preset.ID
	}
	return ids
}

func lookupPreset(id string) (labPresetMeta, bool) {
	for _, preset := range knownPresets() {
		if preset.ID == id {
			return preset, true
		}
	}
	return labPresetMeta{}, false
}

func normalizePresetIDs(presets []string) ([]string, error) {
	if len(presets) == 0 {
		return allPresetIDs(), nil
	}

	seen := make(map[string]struct{}, len(presets))
	normalized := make([]string, 0, len(presets))
	for _, presetID := range presets {
		if _, ok := lookupPreset(presetID); !ok {
			return nil, fmt.Errorf("unknown preset: %s", presetID)
		}
		if _, exists := seen[presetID]; exists {
			continue
		}
		seen[presetID] = struct{}{}
		normalized = append(normalized, presetID)
	}
	return normalized, nil
}

func normalizeAlgorithms(algorithms []econ.MarketAlgorithm) ([]econ.MarketAlgorithm, error) {
	if len(algorithms) == 0 {
		return econ.KnownMarketAlgorithms(), nil
	}

	known := make(map[econ.MarketAlgorithm]struct{}, len(econ.KnownMarketAlgorithms()))
	for _, algorithm := range econ.KnownMarketAlgorithms() {
		known[algorithm] = struct{}{}
	}

	seen := make(map[econ.MarketAlgorithm]struct{}, len(algorithms))
	normalized := make([]econ.MarketAlgorithm, 0, len(algorithms))
	for _, algorithm := range algorithms {
		if _, ok := known[algorithm]; !ok {
			return nil, fmt.Errorf("unknown market algorithm: %s", algorithm)
		}
		if _, exists := seen[algorithm]; exists {
			continue
		}
		seen[algorithm] = struct{}{}
		normalized = append(normalized, algorithm)
	}
	return normalized, nil
}

func applyDistributionPreset(base econ.Config, presetID string) econ.Config {
	cfg := base
	cfg.Countries = append([]econ.CountryConfig(nil), base.Countries...)
	cfg.Routes = append([]econ.RouteConfig(nil), base.Routes...)

	for i := range cfg.Countries {
		cfg.Countries[i] = cloneCountryConfig(cfg.Countries[i])
	}

	switch presetID {
	case "fossil_skew":
		for i := range cfg.Countries {
			c := &cfg.Countries[i]
			switch c.ID {
			case "coalreach":
				c.Deposits.Coal *= 1.55
				c.Extractors.Coal *= 1.35
				c.SunPotential *= 0.9
			case "petroport":
				c.Deposits.Oil *= 1.55
				c.Extractors.Oil *= 1.3
				c.WindPotential *= 0.9
			default:
				c.Deposits.Copper *= 0.92
				c.Deposits.Silicon *= 0.88
				c.SunPotential *= 0.86
				c.WindPotential *= 0.88
			}
		}
	case "renewable_skew":
		for i := range cfg.Countries {
			c := &cfg.Countries[i]
			c.Deposits.Coal *= 0.82
			c.Deposits.Oil *= 0.8
			c.Extractors.Coal *= 0.82
			c.Extractors.Oil *= 0.82
			switch c.ID {
			case "silica":
				c.SunPotential *= 1.3
				c.Assets.SolarFarm *= 1.25
			case "aurora":
				c.WindPotential *= 1.34
				c.Assets.WindFarm *= 1.22
			default:
				c.SunPotential *= 1.08
				c.WindPotential *= 1.05
			}
		}
	case "metal_bottleneck":
		for i := range cfg.Countries {
			c := &cfg.Countries[i]
			c.Deposits.Copper *= 0.58
			c.Deposits.Silicon *= 0.55
			c.Extractors.Copper *= 0.62
			c.Extractors.Silicon *= 0.6
			if c.ID == "silica" {
				c.Deposits.Silicon *= 1.2
			}
			if c.ID == "aurora" {
				c.Deposits.Copper *= 1.16
			}
		}
	default:
	}

	return cfg
}

func cloneCountryConfig(country econ.CountryConfig) econ.CountryConfig {
	return country
}
