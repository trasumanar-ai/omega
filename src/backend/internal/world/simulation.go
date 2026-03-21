package world

import (
	"fmt"
	"math"
)

const maxMemory = 10

type Simulation struct {
	Config          Config
	tick            int
	countries       []country
	proposals       []Proposal
	nextPropID      int
	lastStats       TickStats
	currentTrades   []TradeRecord
	currentMessages []AgentMsg
}

type country struct {
	cfg        CountryConfig
	alive      bool
	stability  float64
	money      float64
	deposits   Resources
	stockpiles Resources
	assets     AssetConfig
	compute    float64 // efficiency 0-1, reduces energy loss

	memory []MemoryEntry
	inbox  []AgentMsg

	// last tick info
	energyProduced  float64
	energyNeeded    float64
	shortage        float64
	lastBuild       BuildKind
	lastBuildOk     bool
	lastBuildFail   string
	lastSource      string
	lastSummary     string
	lastError       string
	tradesCompleted int
	pendingActions  []AgentAction
}

func NewSimulation(cfg Config) (*Simulation, error) {
	if len(cfg.Countries) == 0 {
		cfg = DefaultConfig()
	}
	s := &Simulation{
		Config:    cfg,
		countries: make([]country, len(cfg.Countries)),
	}
	for i, cc := range cfg.Countries {
		s.countries[i] = country{
			cfg:       cc,
			alive:     true,
			stability: 100,
			money:     cc.StartingMoney,
			deposits:  cc.Deposits,
			assets:    cc.Assets,
		}
	}
	return s, nil
}

func (s *Simulation) Tick() int { return s.tick }

func (s *Simulation) Snapshot() WorldSnapshot {
	countries := make([]CountrySnapshot, len(s.countries))
	for i := range s.countries {
		countries[i] = s.countrySnapshot(i)
	}
	return WorldSnapshot{
		Tick:      s.tick,
		Countries: countries,
		Trades:    append([]TradeRecord(nil), s.currentTrades...),
		Proposals: append([]Proposal(nil), s.proposals...),
		Messages:  append([]AgentMsg(nil), s.currentMessages...),
	}
}

func (s *Simulation) LastStats() TickStats { return s.lastStats }

func (s *Simulation) BeginTick() {
	s.tick++
	s.resetTick()
	s.currentTrades = nil
	s.currentMessages = nil

	// 1. extract resources
	for i := range s.countries {
		s.extract(i)
	}

	// 2. generate energy & compute demand
	for i := range s.countries {
		s.generateEnergy(i)
	}
}

func (s *Simulation) ApplyCommand(cmd AgentCommand) {
	idx := s.findCountry(cmd.From)
	if idx < 0 || !s.countries[idx].alive {
		return
	}
	if cmd.ObservedTick != 0 && cmd.ObservedTick != s.tick {
		return
	}

	s.countries[idx].lastSource = cmd.Source
	s.countries[idx].lastSummary = cmd.Summary
	s.countries[idx].lastError = cmd.Error
	s.countries[idx].pendingActions = append(s.countries[idx].pendingActions, cmd.Action)

	from := s.countries[idx].cfg.ID
	a := cmd.Action
	switch a.Action {
	case ActionPropose:
		if a.To == "" || a.To == from {
			return
		}
		if !isResource(a.OfferResource) || !isResource(a.WantResource) {
			return
		}
		if a.OfferAmount <= 0 || a.WantAmount <= 0 {
			return
		}
		if s.getResource(idx, a.OfferResource) < a.OfferAmount {
			return
		}
		s.nextPropID++
		s.proposals = append(s.proposals, Proposal{
			ID:            s.nextPropID,
			From:          from,
			To:            a.To,
			OfferResource: a.OfferResource,
			OfferAmount:   a.OfferAmount,
			WantResource:  a.WantResource,
			WantAmount:    a.WantAmount,
			Tick:          s.tick,
		})

	case ActionAccept:
		trade := s.executeAccept(idx, a.ProposalID)
		if trade != nil {
			s.currentTrades = append(s.currentTrades, *trade)
		}

	case ActionBuild:
		s.executeBuild(idx, a.Build)

	case ActionBroadcast:
		if a.Message == "" {
			return
		}
		msg := AgentMsg{From: from, Message: a.Message}
		s.currentMessages = append(s.currentMessages, msg)
		for j := range s.countries {
			if j != idx && s.countries[j].alive {
				s.countries[j].inbox = append(s.countries[j].inbox, msg)
			}
		}

	case ActionSend:
		if a.To == "" || a.Message == "" {
			return
		}
		j := s.findCountry(a.To)
		if j >= 0 && j != idx && s.countries[j].alive {
			s.countries[j].inbox = append(s.countries[j].inbox, AgentMsg{From: from, Message: a.Message})
		}

	case ActionHold:
		return
	}
}

func (s *Simulation) FinalizeTick() TickStats {
	stats := TickStats{Tick: s.tick, TotalTrades: len(s.currentTrades)}
	for i := range s.countries {
		s.consumeEnergy(i)
		if s.countries[i].alive {
			stats.LivingCountries++
			stats.TotalShortage += s.countries[i].shortage
			stats.TotalMoney += s.countries[i].money
		}
	}

	// 6. expire old proposals (keep current and previous tick)
	fresh := s.proposals[:0]
	for _, p := range s.proposals {
		if p.Tick >= s.tick-1 {
			fresh = append(fresh, p)
		}
	}
	s.proposals = fresh

	// 7. update memory
	for i := range s.countries {
		if !s.countries[i].alive {
			continue
		}
		entry := MemoryEntry{
			Tick:     s.tick,
			Actions:  append([]AgentAction(nil), s.countries[i].pendingActions...),
			Messages: append([]AgentMsg(nil), s.countries[i].inbox...),
		}
		if s.countries[i].lastBuildOk {
			entry.Results = append(entry.Results, fmt.Sprintf("built %s", s.countries[i].lastBuild))
		} else if s.countries[i].lastBuildFail != "" {
			entry.Results = append(entry.Results, "build failed: "+s.countries[i].lastBuildFail)
		}
		if s.countries[i].tradesCompleted > 0 {
			entry.Results = append(entry.Results, fmt.Sprintf("%d trades completed", s.countries[i].tradesCompleted))
		}
		if s.countries[i].shortage > 0 {
			entry.Results = append(entry.Results, fmt.Sprintf("energy shortage: %.1f", s.countries[i].shortage))
		}
		s.countries[i].memory = append(s.countries[i].memory, entry)
		if len(s.countries[i].memory) > maxMemory {
			s.countries[i].memory = s.countries[i].memory[len(s.countries[i].memory)-maxMemory:]
		}
		s.countries[i].pendingActions = nil
		s.countries[i].inbox = nil
	}

	s.lastStats = stats
	return stats
}

func (s *Simulation) Step(provider DecisionProvider) TickStats {
	s.BeginTick()

	for i := range s.countries {
		if !s.countries[i].alive {
			continue
		}
		obs := s.makeObservation(i)
		var result DecisionResult
		if provider != nil {
			result = provider(obs)
		}
		if result.Source == "" {
			result.Source = "world"
		}
		if len(result.Actions) == 0 {
			result.Actions = []AgentAction{{Action: ActionHold}}
		}
		for _, action := range result.Actions {
			s.ApplyCommand(AgentCommand{
				From:         s.countries[i].cfg.ID,
				ObservedTick: s.tick,
				Action:       action,
				Source:       result.Source,
				Summary:      result.Summary,
				Error:        result.Error,
			})
		}
	}

	return s.FinalizeTick()
}

// --- Core mechanics ---

func (s *Simulation) extract(i int) {
	c := &s.countries[i]
	if !c.alive {
		return
	}
	c.stockpiles.Coal += drain(&c.deposits.Coal, c.cfg.Extractors.Coal)
	c.stockpiles.Oil += drain(&c.deposits.Oil, c.cfg.Extractors.Oil)
	c.stockpiles.Copper += drain(&c.deposits.Copper, c.cfg.Extractors.Copper)
	c.stockpiles.Silicon += drain(&c.deposits.Silicon, c.cfg.Extractors.Silicon)
}

func (s *Simulation) generateEnergy(i int) {
	c := &s.countries[i]
	if !c.alive {
		return
	}

	loss := s.Config.EnergyLossRate * (1 - c.compute*0.5)
	loss = clamp(loss, 0, 0.4)

	// renewables first
	energy := c.assets.SolarFarm*s.Config.SolarUnitOutput*c.cfg.SunPotential +
		c.assets.WindFarm*s.Config.WindUnitOutput*c.cfg.WindPotential

	// then burn fuel as needed
	needed := c.cfg.BaseDemand / (1 - loss)
	fuelNeeded := math.Max(0, needed-energy)

	coalBurn := math.Min(c.stockpiles.Coal, fuelNeeded/math.Max(0.1, s.Config.CoalEnergyPerUnit))
	c.stockpiles.Coal -= coalBurn
	energy += coalBurn * s.Config.CoalEnergyPerUnit
	fuelNeeded = math.Max(0, needed-energy)

	oilBurn := math.Min(c.stockpiles.Oil, fuelNeeded/math.Max(0.1, s.Config.OilEnergyPerUnit))
	c.stockpiles.Oil -= oilBurn
	energy += oilBurn * s.Config.OilEnergyPerUnit

	c.energyProduced = energy * (1 - loss)
	c.energyNeeded = c.cfg.BaseDemand
}

func (s *Simulation) consumeEnergy(i int) {
	c := &s.countries[i]
	if !c.alive {
		return
	}

	if c.energyProduced >= c.energyNeeded {
		c.shortage = 0
		c.stability = math.Min(100, c.stability+s.Config.StabilityRecovery)
	} else {
		c.shortage = c.energyNeeded - c.energyProduced
		c.stability = math.Max(0, c.stability-c.shortage*s.Config.StabilityPenalty)
	}

	if c.stability <= 0 {
		c.alive = false
	}
}

func (s *Simulation) executeAccept(acceptorIdx int, proposalID int) *TradeRecord {
	// find proposal
	var prop *Proposal
	propIdx := -1
	for j, p := range s.proposals {
		if p.ID == proposalID && p.To == s.countries[acceptorIdx].cfg.ID {
			prop = &s.proposals[j]
			propIdx = j
			break
		}
	}
	if prop == nil {
		return nil
	}

	fromIdx := s.findCountry(prop.From)
	if fromIdx < 0 || !s.countries[fromIdx].alive {
		return nil
	}

	// check both sides have enough
	if s.getResource(fromIdx, prop.OfferResource) < prop.OfferAmount {
		return nil
	}
	if s.getResource(acceptorIdx, prop.WantResource) < prop.WantAmount {
		return nil
	}

	// execute trade
	s.addResource(fromIdx, prop.OfferResource, -prop.OfferAmount)
	s.addResource(acceptorIdx, prop.OfferResource, prop.OfferAmount)
	s.addResource(acceptorIdx, prop.WantResource, -prop.WantAmount)
	s.addResource(fromIdx, prop.WantResource, prop.WantAmount)

	s.countries[fromIdx].tradesCompleted++
	s.countries[acceptorIdx].tradesCompleted++

	// remove proposal
	s.proposals = append(s.proposals[:propIdx], s.proposals[propIdx+1:]...)

	return &TradeRecord{
		Tick:          s.tick,
		From:          prop.From,
		To:            prop.To,
		OfferResource: prop.OfferResource,
		OfferAmount:   prop.OfferAmount,
		WantResource:  prop.WantResource,
		WantAmount:    prop.WantAmount,
	}
}

func (s *Simulation) executeBuild(i int, kind BuildKind) {
	c := &s.countries[i]
	if !c.alive {
		return
	}
	c.lastBuild = kind

	switch kind {
	case BuildKindCoalPlant, BuildKindOilPlant, BuildKindSolarFarm, BuildKindWindFarm:
		if c.stockpiles.Copper < s.Config.EnergyBuildCopper {
			c.lastBuildFail = "need_copper"
			return
		}
		if c.stockpiles.Silicon < s.Config.EnergyBuildSilicon {
			c.lastBuildFail = "need_silicon"
			return
		}
		c.stockpiles.Copper -= s.Config.EnergyBuildCopper
		c.stockpiles.Silicon -= s.Config.EnergyBuildSilicon
		switch kind {
		case BuildKindCoalPlant:
			c.assets.CoalPlant += s.Config.PlantBuildGain
		case BuildKindOilPlant:
			c.assets.OilPlant += s.Config.PlantBuildGain
		case BuildKindSolarFarm:
			c.assets.SolarFarm += s.Config.SolarBuildGain
		case BuildKindWindFarm:
			c.assets.WindFarm += s.Config.WindBuildGain
		}
		c.lastBuildOk = true

	case BuildKindComputeHub:
		if c.stockpiles.Copper < s.Config.ComputeBuildCopper {
			c.lastBuildFail = "need_copper"
			return
		}
		if c.stockpiles.Silicon < s.Config.ComputeBuildSilicon {
			c.lastBuildFail = "need_silicon"
			return
		}
		c.stockpiles.Copper -= s.Config.ComputeBuildCopper
		c.stockpiles.Silicon -= s.Config.ComputeBuildSilicon
		c.compute = math.Min(s.Config.MaxCompute, c.compute+s.Config.ComputeGain)
		c.lastBuildOk = true
	}
}

// --- Resource helpers ---

func isResource(r string) bool {
	switch r {
	case "coal", "oil", "copper", "silicon", "money":
		return true
	}
	return false
}

func (s *Simulation) getResource(i int, r string) float64 {
	c := &s.countries[i]
	switch r {
	case "coal":
		return c.stockpiles.Coal
	case "oil":
		return c.stockpiles.Oil
	case "copper":
		return c.stockpiles.Copper
	case "silicon":
		return c.stockpiles.Silicon
	case "money":
		return c.money
	}
	return 0
}

func (s *Simulation) addResource(i int, r string, amount float64) {
	c := &s.countries[i]
	switch r {
	case "coal":
		c.stockpiles.Coal += amount
	case "oil":
		c.stockpiles.Oil += amount
	case "copper":
		c.stockpiles.Copper += amount
	case "silicon":
		c.stockpiles.Silicon += amount
	case "money":
		c.money += amount
	}
}

func (s *Simulation) findCountry(id string) int {
	for i, c := range s.countries {
		if c.cfg.ID == id {
			return i
		}
	}
	return -1
}

// --- Observation ---

func (s *Simulation) makeObservation(i int) CountryObservation {
	others := make([]PublicView, 0, len(s.countries)-1)
	for j, c := range s.countries {
		if j == i {
			continue
		}
		others = append(others, PublicView{
			ID:        c.cfg.ID,
			Name:      c.cfg.Name,
			Alive:     c.alive,
			Stability: c.stability,
		})
	}

	// pending proposals addressed to this agent
	myID := s.countries[i].cfg.ID
	var pending []Proposal
	for _, p := range s.proposals {
		if p.To == myID {
			pending = append(pending, p)
		}
	}

	return CountryObservation{
		Tick:      s.tick,
		You:       s.countrySnapshot(i),
		Others:    others,
		Proposals: pending,
		Inbox:     s.countries[i].inbox,
		Memory:    s.countries[i].memory,
	}
}

func (s *Simulation) countrySnapshot(i int) CountrySnapshot {
	c := s.countries[i]
	return CountrySnapshot{
		ID:                  c.cfg.ID,
		Name:                c.cfg.Name,
		Alive:               c.alive,
		Stability:           c.stability,
		Money:               c.money,
		Deposits:            c.deposits,
		Stockpiles:          c.stockpiles,
		Assets:              c.assets,
		EnergyProduced:      c.energyProduced,
		EnergyNeeded:        c.energyNeeded,
		Shortage:            c.shortage,
		LastBuild:           c.lastBuild,
		LastBuildOk:         c.lastBuildOk,
		LastDecisionSource:  c.lastSource,
		LastDecisionSummary: c.lastSummary,
		LastDecisionError:   c.lastError,
		TradesCompleted:     c.tradesCompleted,
	}
}

func (s *Simulation) resetTick() {
	for i := range s.countries {
		s.countries[i].energyProduced = 0
		s.countries[i].energyNeeded = 0
		s.countries[i].shortage = 0
		s.countries[i].lastBuild = BuildKindNone
		s.countries[i].lastBuildOk = false
		s.countries[i].lastBuildFail = ""
		s.countries[i].tradesCompleted = 0
		s.countries[i].pendingActions = nil
	}
}

// --- Helpers ---

func drain(deposit *float64, rate float64) float64 {
	if *deposit <= 0 || rate <= 0 {
		return 0
	}
	out := math.Min(*deposit, rate)
	*deposit -= out
	return out
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
