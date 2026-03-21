package econ

import (
	"fmt"
	"math"
	"sort"
)

type Simulation struct {
	Config       Config
	tick         int
	countries    []countryState
	routes       []routeState
	countryIndex map[string]int
	lastStats    TickStats
}

type countryState struct {
	cfg CountryConfig

	alive             bool
	stability         float64
	treasury          float64
	energyReserve     float64
	infrastructure    float64
	computeEfficiency float64
	deposits          DepositSnapshot
	stockpiles        ResourceStockpile
	assets            CountryAssetsSnapshot

	lastDemand            float64
	lastGeneratedEnergy   float64
	lastDeliveredEnergy   float64
	lastImportedEnergy    float64
	lastExportedEnergy    float64
	lastShortage          float64
	lastTreasuryDelta     float64
	lastTradeRevenue      float64
	lastTradeSpend        float64
	lastIndustrialRevenue float64
	lastEmergencySpend    float64
	lastBuildSpend        float64
	lastBuildFocus        BuildFocus
	lastBuildKind         BuildKind
	lastBuildSuccess      bool
	lastBuildFailure      string
	lastDecisionSource    string
	lastDecisionModel     string
	lastDecisionSummary   string
	lastDecisionError     string
	lastDecisionUsage     DecisionUsage
	lastMarketQuotes      []ResourceQuoteSnapshot
}

type routeState struct {
	cfg RouteConfig

	aIndex int
	bIndex int

	lastEnergyAB float64
	lastEnergyBA float64
	lastCargoAB  float64
	lastCargoBA  float64
	effectiveCap float64
}

type buildPlan struct {
	focus          BuildFocus
	kind           BuildKind
	treasuryCost   float64
	copperCost     float64
	siliconCost    float64
	coalPlantGain  float64
	oilPlantGain   float64
	solarGain      float64
	windGain       float64
	computeGain    float64
	infrastructure float64
	failureReason  string
}

type energyLedger struct {
	effectiveDemand float64
	grossEnergy     float64
	deliveredEnergy float64
	importedEnergy  float64
	exportedEnergy  float64
	shortage        float64
	energyOffer     float64
	energyNeed      float64
}

type resourceQuote struct {
	net        float64
	limitPrice float64
}

type countryQuotes struct {
	energy  resourceQuote
	coal    resourceQuote
	oil     resourceQuote
	copper  resourceQuote
	silicon resourceQuote
}

func NewSimulation(cfg Config) (*Simulation, error) {
	if len(cfg.Countries) == 0 {
		cfg = DefaultConfig()
	}
	if cfg.MarketAlgorithm == "" {
		cfg.MarketAlgorithm = MarketAlgorithmLinear
	}

	s := &Simulation{
		Config:       cfg,
		countries:    make([]countryState, len(cfg.Countries)),
		routes:       make([]routeState, len(cfg.Routes)),
		countryIndex: make(map[string]int, len(cfg.Countries)),
	}

	for i, country := range cfg.Countries {
		if _, exists := s.countryIndex[country.ID]; exists {
			return nil, fmt.Errorf("duplicate country id: %s", country.ID)
		}
		s.countryIndex[country.ID] = i
		s.countries[i] = countryState{
			cfg:               country,
			alive:             true,
			stability:         100,
			treasury:          country.StartingTreasury,
			energyReserve:     country.StartingReserve,
			infrastructure:    country.Infrastructure,
			computeEfficiency: country.ComputeEfficiency,
			deposits:          country.Deposits,
			assets: CountryAssetsSnapshot{
				CoalPlant: country.Assets.CoalPlant,
				OilPlant:  country.Assets.OilPlant,
				SolarFarm: country.Assets.SolarFarm,
				WindFarm:  country.Assets.WindFarm,
			},
			lastBuildFocus: BuildHold,
			lastBuildKind:  BuildKindNone,
		}
	}

	for i, route := range cfg.Routes {
		aIndex, ok := s.countryIndex[route.A]
		if !ok {
			return nil, fmt.Errorf("route references unknown country: %s", route.A)
		}
		bIndex, ok := s.countryIndex[route.B]
		if !ok {
			return nil, fmt.Errorf("route references unknown country: %s", route.B)
		}
		s.routes[i] = routeState{
			cfg:    route,
			aIndex: aIndex,
			bIndex: bIndex,
		}
	}

	return s, s.Validate()
}

func (s *Simulation) Tick() int {
	return s.tick
}

func (s *Simulation) Validate() error {
	if len(s.countries) == 0 {
		return fmt.Errorf("no countries configured")
	}
	if len(s.routes) == 0 {
		return fmt.Errorf("no routes configured")
	}
	for _, country := range s.countries {
		if country.cfg.BaseDemand <= 0 {
			return fmt.Errorf("country %s has non-positive base demand", country.cfg.ID)
		}
	}
	return nil
}

func (s *Simulation) Snapshot() WorldSnapshot {
	countries := make([]CountrySnapshot, len(s.countries))
	for i := range s.countries {
		countries[i] = s.countrySnapshot(i)
	}

	routes := make([]RouteSnapshot, len(s.routes))
	for i, route := range s.routes {
		routes[i] = RouteSnapshot{
			A:            route.cfg.A,
			B:            route.cfg.B,
			Capacity:     route.cfg.Capacity,
			LastEnergyAB: route.lastEnergyAB,
			LastEnergyBA: route.lastEnergyBA,
			LastCargoAB:  route.lastCargoAB,
			LastCargoBA:  route.lastCargoBA,
			EffectiveCap: route.effectiveCap,
		}
	}

	return WorldSnapshot{
		Tick:      s.tick,
		Countries: countries,
		Routes:    routes,
	}
}

func (s *Simulation) Step(provider DecisionProvider) TickStats {
	s.tick++
	s.resetRouteFlows()
	s.resetTickAccounting()

	plans := make([]buildPlan, len(s.countries))
	ledgers := make([]energyLedger, len(s.countries))

	for i := range s.countries {
		if !s.countries[i].alive {
			continue
		}
		observation := s.makeObservation(i)
		result := DecisionResult{
			Decision: s.defaultDecision(i),
			Source:   "heuristic",
			Model:    "builtin/default",
		}
		if provider != nil {
			providerResult := provider(observation)
			if isValidBuildFocus(providerResult.Decision.BuildFocus) {
				result = providerResult
			} else {
				if providerResult.Source == "" {
					providerResult.Source = "heuristic_fallback"
				}
				if providerResult.Model == "" {
					providerResult.Model = "builtin/default"
				}
				providerResult.Decision = s.defaultDecision(i)
				if providerResult.Error == "" {
					providerResult.Error = "invalid_build_focus"
				}
				result = providerResult
			}
		}
		if !isValidBuildFocus(result.Decision.BuildFocus) {
			result.Decision.BuildFocus = BuildHold
		}
		s.recordDecision(i, result)
		plans[i] = s.planBuild(i, result.Decision.BuildFocus)
	}

	for i := range s.countries {
		s.extractResources(i)
	}
	for i := range s.countries {
		ledgers[i] = s.runLocalEnergy(i)
	}

	quotes := make([]countryQuotes, len(s.countries))
	for i := range s.countries {
		quotes[i] = s.makeCountryQuotes(i, ledgers[i], plans[i])
		s.countries[i].lastMarketQuotes = quotes[i].snapshot()
	}

	var totalEnergyTrade float64
	var totalCargoTrade float64
	for i := range s.routes {
		route := &s.routes[i]
		effectiveCap := s.routeCapacity(route)
		route.effectiveCap = effectiveCap

		energyCap := effectiveCap * s.Config.TradeEnergyShare
		cargoCap := effectiveCap * s.Config.TradeCargoShare

		totalEnergyTrade += s.tradeEnergy(route, route.aIndex, route.bIndex, energyCap, ledgers, quotes)
		totalEnergyTrade += s.tradeEnergy(route, route.bIndex, route.aIndex, energyCap, ledgers, quotes)

		totalCargoTrade += s.tradeCargo(route, route.aIndex, route.bIndex, cargoCap, quotes)
		totalCargoTrade += s.tradeCargo(route, route.bIndex, route.aIndex, cargoCap, quotes)
	}

	stats := TickStats{
		Tick:              s.tick,
		TotalEnergyTraded: totalEnergyTrade,
		TotalCargoTraded:  totalCargoTrade,
	}

	for i := range s.countries {
		aliveBefore := s.countries[i].alive
		s.finalizeCountry(i, &ledgers[i], plans[i])
		if !aliveBefore && !s.countries[i].alive {
			continue
		}
		if s.countries[i].alive {
			stats.LivingCountries++
		}
		stats.TotalEnergyGenerated += ledgers[i].grossEnergy
		stats.TotalEnergyDelivered += ledgers[i].deliveredEnergy + ledgers[i].importedEnergy
		stats.TotalShortage += s.countries[i].lastShortage
		stats.TotalTreasury += s.countries[i].treasury
		stats.AvgComputeEfficiency += s.countries[i].computeEfficiency
		stats.CoalRemaining += s.countries[i].deposits.Coal
		stats.OilRemaining += s.countries[i].deposits.Oil
		stats.CopperRemaining += s.countries[i].deposits.Copper
		stats.SiliconRemaining += s.countries[i].deposits.Silicon
		if s.countries[i].lastBuildSuccess {
			switch s.countries[i].lastBuildFocus {
			case BuildEnergy:
				stats.EnergyBuilds++
			case BuildCompute:
				stats.ComputeBuilds++
			case BuildInfrastructure:
				stats.InfraBuilds++
			}
		}
	}

	if stats.LivingCountries > 0 {
		stats.AvgComputeEfficiency /= float64(stats.LivingCountries)
	}
	s.lastStats = stats
	return stats
}

func (s *Simulation) makeObservation(countryIndex int) CountryObservation {
	neighbors := make([]NeighborView, 0, 4)
	for _, route := range s.routes {
		var neighbor int
		switch countryIndex {
		case route.aIndex:
			neighbor = route.bIndex
		case route.bIndex:
			neighbor = route.aIndex
		default:
			continue
		}
		n := s.countries[neighbor]
		neighbors = append(neighbors, NeighborView{
			ID:                n.cfg.ID,
			Name:              n.cfg.Name,
			Stability:         n.stability,
			EnergyReserve:     n.energyReserve,
			ComputeEfficiency: n.computeEfficiency,
			Infrastructure:    n.infrastructure,
		})
	}

	return CountryObservation{
		Tick:        s.tick,
		Country:     s.countrySnapshot(countryIndex),
		Neighbors:   neighbors,
		WorldTotals: s.lastStats,
	}
}

func (s *Simulation) defaultDecision(countryIndex int) CountryDecision {
	country := s.countries[countryIndex]
	if !country.alive {
		return CountryDecision{BuildFocus: BuildHold}
	}

	reserveTarget := s.reserveCapacity(countryIndex) * 0.55
	demand := s.countryDemand(countryIndex)
	coverage := (country.lastDeliveredEnergy + country.lastImportedEnergy + country.energyReserve) / maxFloat(1, demand)

	switch {
	case country.lastShortage > 1.5:
		return CountryDecision{BuildFocus: BuildEnergy}
	case country.energyReserve < reserveTarget*0.35:
		return CountryDecision{BuildFocus: BuildEnergy}
	case coverage < 1.15:
		return CountryDecision{BuildFocus: BuildEnergy}
	case country.infrastructureSaturated(s.gridCapacity(countryIndex)):
		return CountryDecision{BuildFocus: BuildInfrastructure}
	case country.computeEfficiency < 0.52 && country.stockpiles.Silicon+country.deposits.Silicon*0.01 > 6:
		return CountryDecision{BuildFocus: BuildCompute}
	case country.treasury > 170 && country.computeEfficiency < 0.82:
		return CountryDecision{BuildFocus: BuildCompute}
	case country.treasury > 180:
		return CountryDecision{BuildFocus: BuildInfrastructure}
	default:
		return CountryDecision{BuildFocus: BuildHold}
	}
}

func (s *Simulation) extractResources(countryIndex int) {
	country := &s.countries[countryIndex]
	if !country.alive {
		return
	}
	country.stockpiles.Coal += s.extract(&country.deposits.Coal, country.cfg.Extractors.Coal)
	country.stockpiles.Oil += s.extract(&country.deposits.Oil, country.cfg.Extractors.Oil)
	country.stockpiles.Copper += s.extract(&country.deposits.Copper, country.cfg.Extractors.Copper)
	country.stockpiles.Silicon += s.extract(&country.deposits.Silicon, country.cfg.Extractors.Silicon)
}

func (s *Simulation) extract(deposit *float64, rate float64) float64 {
	if *deposit <= 0 || rate <= 0 {
		return 0
	}
	out := minFloat(*deposit, rate)
	*deposit -= out
	return out
}

func (s *Simulation) runLocalEnergy(countryIndex int) energyLedger {
	country := &s.countries[countryIndex]
	if !country.alive {
		return energyLedger{}
	}

	gridCap := s.gridCapacity(countryIndex)
	reserveCap := s.reserveCapacity(countryIndex)
	effectiveDemand := s.countryDemand(countryIndex)
	reserveGap := maxFloat(0, reserveCap*0.55-country.energyReserve)
	dispatchTarget := minFloat(gridCap+s.connectedRouteEnergy(countryIndex)*0.45, effectiveDemand+reserveGap+5)

	localLossRate := s.Config.LocalLossRate / (1 + country.computeEfficiency*s.Config.ComputeLossDividend)
	localLossRate = clamp(localLossRate, 0, 0.45)

	renewable := country.assets.SolarFarm*s.Config.SolarUnitOutput*country.cfg.SunPotential +
		country.assets.WindFarm*s.Config.WindUnitOutput*country.cfg.WindPotential

	requiredGross := dispatchTarget / maxFloat(0.25, 1-localLossRate)
	grossEnergy := renewable
	remainingGross := maxFloat(0, requiredGross-renewable)

	coalBurn := minFloat(country.stockpiles.Coal, minFloat(country.assets.CoalPlant, remainingGross/maxFloat(0.1, s.Config.CoalEnergyPerUnit)))
	country.stockpiles.Coal -= coalBurn
	grossEnergy += coalBurn * s.Config.CoalEnergyPerUnit
	remainingGross = maxFloat(0, requiredGross-grossEnergy)

	oilBurn := minFloat(country.stockpiles.Oil, minFloat(country.assets.OilPlant, remainingGross/maxFloat(0.1, s.Config.OilEnergyPerUnit)))
	country.stockpiles.Oil -= oilBurn
	grossEnergy += oilBurn * s.Config.OilEnergyPerUnit

	deliverable := grossEnergy * (1 - localLossRate)
	delivered := minFloat(deliverable, gridCap)
	desiredReserve := reserveCap * 0.55
	energyNeed := maxFloat(0, effectiveDemand+maxFloat(0, desiredReserve-country.energyReserve)-delivered)
	energyOffer := maxFloat(0, delivered-effectiveDemand-maxFloat(0, desiredReserve-country.energyReserve))

	return energyLedger{
		effectiveDemand: effectiveDemand,
		grossEnergy:     grossEnergy,
		deliveredEnergy: delivered,
		energyOffer:     energyOffer,
		energyNeed:      energyNeed,
	}
}

func (s *Simulation) finalizeCountry(countryIndex int, ledger *energyLedger, plan buildPlan) {
	country := &s.countries[countryIndex]
	if !country.alive {
		return
	}

	country.lastDemand = ledger.effectiveDemand

	available := ledger.deliveredEnergy + ledger.importedEnergy
	if available < ledger.effectiveDemand {
		fromReserve := minFloat(country.energyReserve, ledger.effectiveDemand-available)
		country.energyReserve -= fromReserve
		available += fromReserve
	}

	if available < ledger.effectiveDemand && country.treasury > 0 {
		emergency := minFloat(
			s.Config.EmergencyEnergyPerTick,
			minFloat(ledger.effectiveDemand-available, country.treasury/maxFloat(0.1, s.Config.EmergencyEnergyPrice)),
		)
		country.treasury -= emergency * s.Config.EmergencyEnergyPrice
		country.lastEmergencySpend += emergency * s.Config.EmergencyEnergyPrice
		country.lastTreasuryDelta -= emergency * s.Config.EmergencyEnergyPrice
		available += emergency
	}

	shortage := maxFloat(0, ledger.effectiveDemand-available)
	if shortage <= 0 {
		reserveCap := s.reserveCapacity(countryIndex)
		buffer := maxFloat(0, available-ledger.effectiveDemand)
		store := minFloat(maxFloat(0, reserveCap-country.energyReserve), buffer)
		country.energyReserve += store
		buffer -= store
		if buffer > 0 {
			revenue := buffer * s.Config.EnergySalePrice * 0.3
			country.treasury += revenue
			country.lastTradeRevenue += revenue
			country.lastTreasuryDelta += revenue
		}
	}

	productiveEnergy := maxFloat(0, available-ledger.effectiveDemand*0.75)
	if shortage <= 0.25 {
		revenue := productiveEnergy * s.Config.IndustrialOutputScale * s.Config.IndustrialOutputPrice * (1 + country.computeEfficiency*0.22)
		country.treasury += revenue
		country.lastIndustrialRevenue += revenue
		country.lastTreasuryDelta += revenue
		country.stability = minFloat(100, country.stability+s.Config.StabilityRecovery)
	} else {
		country.stability = maxFloat(0, country.stability-shortage*s.Config.StabilityPenalty)
	}

	country.lastBuildFocus = plan.focus
	country.lastBuildKind = plan.kind
	country.lastBuildSuccess = false
	country.lastBuildFailure = plan.failureReason
	if plan.focus != BuildHold && plan.kind != BuildKindNone {
		switch {
		case country.treasury < plan.treasuryCost:
			country.lastBuildFailure = "insufficient_treasury"
		case country.stockpiles.Copper < plan.copperCost:
			country.lastBuildFailure = "insufficient_copper"
		case country.stockpiles.Silicon < plan.siliconCost:
			country.lastBuildFailure = "insufficient_silicon"
		default:
			country.treasury -= plan.treasuryCost
			country.lastBuildSpend += plan.treasuryCost
			country.lastTreasuryDelta -= plan.treasuryCost
			country.stockpiles.Copper -= plan.copperCost
			country.stockpiles.Silicon -= plan.siliconCost
			country.assets.CoalPlant += plan.coalPlantGain
			country.assets.OilPlant += plan.oilPlantGain
			country.assets.SolarFarm += plan.solarGain
			country.assets.WindFarm += plan.windGain
			country.infrastructure += plan.infrastructure
			country.computeEfficiency = minFloat(s.Config.MaxComputeEfficiency, country.computeEfficiency+plan.computeGain)
			country.lastBuildSuccess = true
			country.lastBuildFailure = ""
		}
	}

	country.lastGeneratedEnergy = ledger.grossEnergy
	country.lastDeliveredEnergy = ledger.deliveredEnergy
	country.lastImportedEnergy = ledger.importedEnergy
	country.lastExportedEnergy = ledger.exportedEnergy
	country.lastShortage = shortage
	ledger.shortage = shortage
	if country.stability <= 0 {
		country.alive = false
		country.energyReserve = 0
	}
}

func (s *Simulation) tradeEnergy(route *routeState, fromIndex, toIndex int, routeEnergyCap float64, ledgers []energyLedger, quotes []countryQuotes) float64 {
	from := &s.countries[fromIndex]
	to := &s.countries[toIndex]
	if !from.alive || !to.alive {
		return 0
	}

	sellerQuote := &quotes[fromIndex].energy
	buyerQuote := &quotes[toIndex].energy
	offer := maxFloat(0, -sellerQuote.net)
	need := maxFloat(0, buyerQuote.net)
	if offer <= 0 || need <= 0 || routeEnergyCap <= 0 || buyerQuote.limitPrice < sellerQuote.limitPrice {
		return 0
	}

	lossRate := s.Config.TradeLossRate / (1 + ((from.computeEfficiency+to.computeEfficiency)/2)*s.Config.ComputeLossDividend)
	lossRate = clamp(lossRate, 0, 0.35)
	efficiency := maxFloat(0.4, 1-lossRate)
	maxSendForNeed := need / efficiency
	unitPrice := (sellerQuote.limitPrice + buyerQuote.limitPrice) / 2
	maxAffordable := to.treasury / maxFloat(0.1, unitPrice)
	sent := minFloat(offer, minFloat(routeEnergyCap, minFloat(maxSendForNeed, maxAffordable)))
	if sent <= 0 {
		return 0
	}
	received := sent * efficiency
	cost := sent * unitPrice

	from.treasury += cost
	from.lastTradeRevenue += cost
	from.lastTreasuryDelta += cost
	to.treasury -= cost
	to.lastTradeSpend += cost
	to.lastTreasuryDelta -= cost
	sellerQuote.net += sent
	buyerQuote.net = maxFloat(0, buyerQuote.net-received)
	ledgers[fromIndex].energyOffer -= sent
	ledgers[fromIndex].exportedEnergy += sent
	ledgers[toIndex].energyNeed = maxFloat(0, ledgers[toIndex].energyNeed-received)
	ledgers[toIndex].importedEnergy += received

	if route.aIndex == fromIndex && route.bIndex == toIndex {
		route.lastEnergyAB += received
	} else if route.bIndex == fromIndex && route.aIndex == toIndex {
		route.lastEnergyBA += received
	}
	return received
}

func (s *Simulation) tradeCargo(route *routeState, fromIndex, toIndex int, routeCargoCap float64, quotes []countryQuotes) float64 {
	from := &s.countries[fromIndex]
	to := &s.countries[toIndex]
	if !from.alive || !to.alive || routeCargoCap <= 0 {
		return 0
	}

	var total float64
	used := 0.0
	for _, resource := range sortResourcesForTrade(quotes[fromIndex], quotes[toIndex]) {
		used += s.tradeSingleCargo(from, to, &total, &used, routeCargoCap, resource, &quotes[fromIndex], &quotes[toIndex])
	}

	if route.aIndex == fromIndex && route.bIndex == toIndex {
		route.lastCargoAB += total
	} else if route.bIndex == fromIndex && route.aIndex == toIndex {
		route.lastCargoBA += total
	}
	return total
}

func (s *Simulation) tradeSingleCargo(
	from *countryState,
	to *countryState,
	total *float64,
	used *float64,
	routeCargoCap float64,
	resource string,
	fromQuotes *countryQuotes,
	toQuotes *countryQuotes,
) float64 {
	remainingCap := routeCargoCap - *used
	if remainingCap <= 0 {
		return 0
	}

	sellerQuote := fromQuotes.quote(resource)
	buyerQuote := toQuotes.quote(resource)
	offer := maxFloat(0, -sellerQuote.net)
	need := maxFloat(0, buyerQuote.net)
	if offer <= 0 || need <= 0 || buyerQuote.limitPrice < sellerQuote.limitPrice {
		return 0
	}

	unitPrice := (sellerQuote.limitPrice + buyerQuote.limitPrice) / 2
	affordable := to.treasury / maxFloat(0.1, unitPrice)
	amount := minFloat(offer, minFloat(need, minFloat(remainingCap, affordable)))
	if amount <= 0 {
		return 0
	}

	s.addResource(&from.stockpiles, resource, -amount)
	s.addResource(&to.stockpiles, resource, amount)

	cost := amount * unitPrice
	from.treasury += cost
	from.lastTradeRevenue += cost
	from.lastTreasuryDelta += cost
	to.treasury -= cost
	to.lastTradeSpend += cost
	to.lastTreasuryDelta -= cost
	fromQuotes.adjust(resource, amount)
	toQuotes.adjust(resource, -amount)
	*total += amount
	return amount
}

func (s *Simulation) planBuild(countryIndex int, focus BuildFocus) buildPlan {
	if focus == "" || focus == BuildHold {
		return buildPlan{focus: BuildHold, kind: BuildKindNone}
	}
	switch focus {
	case BuildEnergy:
		kind := s.chooseEnergyBuild(countryIndex)
		plan := buildPlan{
			focus:        BuildEnergy,
			kind:         kind,
			treasuryCost: s.Config.EnergyBuildTreasury,
			copperCost:   s.Config.EnergyBuildCopper,
			siliconCost:  s.Config.EnergyBuildSilicon,
		}
		switch kind {
		case BuildKindCoalPlant:
			plan.coalPlantGain = s.Config.CoalPlantBuildGain
		case BuildKindOilPlant:
			plan.oilPlantGain = s.Config.OilPlantBuildGain
		case BuildKindSolarFarm:
			plan.solarGain = s.Config.SolarBuildGain
		case BuildKindWindFarm:
			plan.windGain = s.Config.WindBuildGain
		default:
			plan.failureReason = "no_energy_path"
		}
		return plan
	case BuildCompute:
		return buildPlan{
			focus:        BuildCompute,
			kind:         BuildKindComputeHub,
			treasuryCost: s.Config.ComputeBuildTreasury,
			copperCost:   s.Config.ComputeBuildCopper,
			siliconCost:  s.Config.ComputeBuildSilicon,
			computeGain:  s.Config.ComputeEfficiencyGain,
		}
	case BuildInfrastructure:
		return buildPlan{
			focus:          BuildInfrastructure,
			kind:           BuildKindGridUpgrade,
			treasuryCost:   s.Config.InfrastructureTreasury,
			copperCost:     s.Config.InfrastructureCopper,
			infrastructure: s.Config.InfrastructureGain,
		}
	default:
		return buildPlan{
			focus:         BuildHold,
			kind:          BuildKindNone,
			failureReason: "unknown_focus",
		}
	}
}

func (s *Simulation) chooseEnergyBuild(countryIndex int) BuildKind {
	country := s.countries[countryIndex]
	coalScore := country.deposits.Coal*0.008 + country.stockpiles.Coal*0.04
	oilScore := country.deposits.Oil*0.008 + country.stockpiles.Oil*0.04
	solarScore := country.cfg.SunPotential * (1 + maxFloat(0, 4-country.assets.SolarFarm*0.2))
	windScore := country.cfg.WindPotential * (1 + maxFloat(0, 4-country.assets.WindFarm*0.2))

	bestKind := BuildKindCoalPlant
	bestScore := coalScore
	if oilScore > bestScore {
		bestKind = BuildKindOilPlant
		bestScore = oilScore
	}
	if solarScore > bestScore {
		bestKind = BuildKindSolarFarm
		bestScore = solarScore
	}
	if windScore > bestScore {
		bestKind = BuildKindWindFarm
	}
	return bestKind
}

func (s *Simulation) countryDemand(countryIndex int) float64 {
	country := s.countries[countryIndex]
	base := country.cfg.BaseDemand +
		s.Config.ComputeUpkeepBase +
		country.computeEfficiency*s.Config.ComputeUpkeepScale +
		country.infrastructure*s.Config.InfraUpkeepScale
	return base / (1 + country.computeEfficiency*s.Config.ComputeEfficiencyDividend)
}

func (s *Simulation) reserveCapacity(countryIndex int) float64 {
	country := s.countries[countryIndex]
	return s.Config.EnergyReserveBase + country.infrastructure*s.Config.EnergyReservePerInfra
}

func (s *Simulation) gridCapacity(countryIndex int) float64 {
	country := s.countries[countryIndex]
	return s.Config.GridCapacityBase + country.infrastructure*s.Config.GridCapacityPerInfra
}

func (s *Simulation) connectedRouteEnergy(countryIndex int) float64 {
	total := 0.0
	for _, route := range s.routes {
		if route.aIndex == countryIndex || route.bIndex == countryIndex {
			total += route.cfg.Capacity
		}
	}
	return total
}

func (s *Simulation) routeCapacity(route *routeState) float64 {
	left := s.countries[route.aIndex]
	right := s.countries[route.bIndex]
	scaleLeft := 0.75 + left.infrastructure*s.Config.RouteInfraScale
	scaleRight := 0.75 + right.infrastructure*s.Config.RouteInfraScale
	return route.cfg.Capacity * minFloat(scaleLeft, scaleRight)
}

func (s *Simulation) countrySnapshot(countryIndex int) CountrySnapshot {
	country := s.countries[countryIndex]
	quotes := make([]ResourceQuoteSnapshot, 0, len(country.lastMarketQuotes))
	quotes = append(quotes, country.lastMarketQuotes...)
	return CountrySnapshot{
		ID:                    country.cfg.ID,
		Name:                  country.cfg.Name,
		X:                     country.cfg.X,
		Y:                     country.cfg.Y,
		Alive:                 country.alive,
		Stability:             country.stability,
		Treasury:              country.treasury,
		EnergyReserve:         country.energyReserve,
		ReserveCapacity:       s.reserveCapacity(countryIndex),
		GridCapacity:          s.gridCapacity(countryIndex),
		Infrastructure:        country.infrastructure,
		ComputeEfficiency:     country.computeEfficiency,
		BaseDemand:            country.cfg.BaseDemand,
		Deposits:              country.deposits,
		Stockpiles:            country.stockpiles,
		Assets:                country.assets,
		LastDemand:            country.lastDemand,
		LastGeneratedEnergy:   country.lastGeneratedEnergy,
		LastDeliveredEnergy:   country.lastDeliveredEnergy,
		LastImportedEnergy:    country.lastImportedEnergy,
		LastExportedEnergy:    country.lastExportedEnergy,
		LastShortage:          country.lastShortage,
		LastTreasuryDelta:     country.lastTreasuryDelta,
		LastTradeRevenue:      country.lastTradeRevenue,
		LastTradeSpend:        country.lastTradeSpend,
		LastIndustrialRevenue: country.lastIndustrialRevenue,
		LastEmergencySpend:    country.lastEmergencySpend,
		LastBuildSpend:        country.lastBuildSpend,
		LastBuildFocus:        country.lastBuildFocus,
		LastBuildKind:         country.lastBuildKind,
		LastBuildSuccess:      country.lastBuildSuccess,
		LastBuildFailure:      country.lastBuildFailure,
		LastDecisionSource:    country.lastDecisionSource,
		LastDecisionModel:     country.lastDecisionModel,
		LastDecisionSummary:   country.lastDecisionSummary,
		LastDecisionError:     country.lastDecisionError,
		LastDecisionUsage:     country.lastDecisionUsage,
		MarketQuotes:          quotes,
	}
}

func (s *Simulation) recordDecision(countryIndex int, result DecisionResult) {
	country := &s.countries[countryIndex]
	country.lastDecisionSource = result.Source
	country.lastDecisionModel = result.Model
	country.lastDecisionSummary = result.Summary
	country.lastDecisionError = result.Error
	country.lastDecisionUsage = result.Usage
}

func (s *Simulation) resetRouteFlows() {
	for i := range s.routes {
		s.routes[i].lastEnergyAB = 0
		s.routes[i].lastEnergyBA = 0
		s.routes[i].lastCargoAB = 0
		s.routes[i].lastCargoBA = 0
		s.routes[i].effectiveCap = 0
	}
}

func (s *Simulation) resetTickAccounting() {
	for i := range s.countries {
		s.countries[i].lastDemand = 0
		s.countries[i].lastTreasuryDelta = 0
		s.countries[i].lastTradeRevenue = 0
		s.countries[i].lastTradeSpend = 0
		s.countries[i].lastIndustrialRevenue = 0
		s.countries[i].lastEmergencySpend = 0
		s.countries[i].lastBuildSpend = 0
		s.countries[i].lastMarketQuotes = nil
	}
}

func (s *Simulation) MarketSnapshot() MarketSnapshot {
	market := MarketSnapshot{
		EnergyPrice:            s.Config.EnergySalePrice,
		EmergencyEnergyPrice:   s.Config.EmergencyEnergyPrice,
		CoalPrice:              s.Config.CoalPrice,
		OilPrice:               s.Config.OilPrice,
		CopperPrice:            s.Config.CopperPrice,
		SiliconPrice:           s.Config.SiliconPrice,
		EnergyBuildTreasury:    s.Config.EnergyBuildTreasury,
		ComputeBuildTreasury:   s.Config.ComputeBuildTreasury,
		InfrastructureTreasury: s.Config.InfrastructureTreasury,
		Countries:              make([]MarketCountrySnapshot, len(s.countries)),
	}

	for i, country := range s.countries {
		demand := country.lastDemand
		if demand <= 0 {
			demand = s.countryDemand(i)
		}

		market.Countries[i] = MarketCountrySnapshot{
			ID:                country.cfg.ID,
			Name:              country.cfg.Name,
			Treasury:          country.treasury,
			TreasuryDelta:     country.lastTreasuryDelta,
			EnergyReserve:     country.energyReserve,
			Demand:            demand,
			GeneratedEnergy:   country.lastGeneratedEnergy,
			DeliveredEnergy:   country.lastDeliveredEnergy,
			ImportedEnergy:    country.lastImportedEnergy,
			ExportedEnergy:    country.lastExportedEnergy,
			Shortage:          country.lastShortage,
			TradeRevenue:      country.lastTradeRevenue,
			TradeSpend:        country.lastTradeSpend,
			IndustrialRevenue: country.lastIndustrialRevenue,
			EmergencySpend:    country.lastEmergencySpend,
			BuildSpend:        country.lastBuildSpend,
			LastBuildFocus:    country.lastBuildFocus,
			LastBuildKind:     country.lastBuildKind,
			Quotes:            append(make([]ResourceQuoteSnapshot, 0, len(country.lastMarketQuotes)), country.lastMarketQuotes...),
		}

		market.TotalDemand += demand
		market.TotalGenerated += country.lastGeneratedEnergy
		market.TotalDelivered += country.lastDeliveredEnergy
		market.TotalImported += country.lastImportedEnergy
		market.TotalExported += country.lastExportedEnergy
		market.TotalShortage += country.lastShortage
		market.TotalTreasury += country.treasury
	}

	return market
}

func (s *Simulation) makeCountryQuotes(countryIndex int, ledger energyLedger, plan buildPlan) countryQuotes {
	if s.Config.MarketAlgorithm == MarketAlgorithmNoTrade {
		return s.noTradeQuotes()
	}

	country := s.countries[countryIndex]
	reserveRatio := country.energyReserve / maxFloat(1, s.reserveCapacity(countryIndex))

	energyPressure := maxFloat(0, ledger.energyNeed) + maxFloat(0, 0.55-reserveRatio)*3.5
	energyBuyPrice := s.priceForNeed(s.Config.EnergySalePrice, energyPressure, 2.2)
	energySellPrice := s.priceForOffer(s.Config.EnergySalePrice, ledger.energyOffer+reserveRatio, 1.2)

	return countryQuotes{
		energy: resourceQuote{
			net:        ledger.energyNeed - ledger.energyOffer,
			limitPrice: s.limitPrice(ledger.energyNeed, ledger.energyOffer, energyBuyPrice, energySellPrice, s.Config.EnergySalePrice),
		},
		coal:    s.quoteForStockpile(country.stockpiles.Coal, s.targetCoal(plan, country), s.Config.CoalPrice),
		oil:     s.quoteForStockpile(country.stockpiles.Oil, s.targetOil(plan, country), s.Config.OilPrice),
		copper:  s.quoteForStockpile(country.stockpiles.Copper, s.targetCopper(plan, country), s.Config.CopperPrice),
		silicon: s.quoteForStockpile(country.stockpiles.Silicon, s.targetSilicon(plan, country), s.Config.SiliconPrice),
	}
}

func (s *Simulation) quoteForStockpile(current float64, target float64, basePrice float64) resourceQuote {
	if target <= 0 {
		target = 1
	}
	shortage := maxFloat(0, target-current)
	surplus := maxFloat(0, current-target)
	buyPrice := s.priceForNeed(basePrice, shortage/maxFloat(1, target)*3.2, 1.8)
	sellPrice := s.priceForOffer(basePrice, surplus/maxFloat(1, target)*2.4, 1.1)
	return resourceQuote{
		net:        shortage - surplus,
		limitPrice: s.limitPrice(shortage, surplus, buyPrice, sellPrice, basePrice),
	}
}

func (s *Simulation) targetCoal(plan buildPlan, country countryState) float64 {
	target := 6 + country.assets.CoalPlant*2.4
	if plan.kind == BuildKindCoalPlant {
		target += 4
	}
	return target
}

func (s *Simulation) targetOil(plan buildPlan, country countryState) float64 {
	target := 5 + country.assets.OilPlant*2.2
	if plan.kind == BuildKindOilPlant {
		target += 4
	}
	return target
}

func (s *Simulation) targetCopper(plan buildPlan, country countryState) float64 {
	target := 4 + plan.copperCost + country.infrastructure*0.6
	if plan.focus == BuildInfrastructure {
		target += 2
	}
	return target
}

func (s *Simulation) targetSilicon(plan buildPlan, country countryState) float64 {
	target := 2 + plan.siliconCost + country.computeEfficiency*4
	if plan.focus == BuildCompute {
		target += 2.5
	}
	return target
}

func (s *Simulation) limitPrice(buyNeed float64, sellOffer float64, buyPrice float64, sellPrice float64, neutral float64) float64 {
	switch {
	case buyNeed > 0.05:
		return buyPrice
	case sellOffer > 0.05:
		return sellPrice
	default:
		return neutral
	}
}

func (s *Simulation) noTradeQuotes() countryQuotes {
	neutral := resourceQuote{net: 0, limitPrice: 0}
	return countryQuotes{
		energy:  neutral,
		coal:    neutral,
		oil:     neutral,
		copper:  neutral,
		silicon: neutral,
	}
}

func (s *Simulation) priceForNeed(basePrice float64, pressure float64, scale float64) float64 {
	pressure = maxFloat(0, pressure)
	switch s.Config.MarketAlgorithm {
	case MarketAlgorithmScarcitySpike:
		return basePrice * (1.02 + (math.Exp(minFloat(2.4, pressure*0.32))-1)*scale*0.32)
	case MarketAlgorithmLinear:
		fallthrough
	default:
		return basePrice * (1.03 + pressure*0.18*scale)
	}
}

func (s *Simulation) priceForOffer(basePrice float64, surplus float64, scale float64) float64 {
	surplus = maxFloat(0, surplus)
	switch s.Config.MarketAlgorithm {
	case MarketAlgorithmScarcitySpike:
		return basePrice * maxFloat(0.5, 0.94-(math.Exp(minFloat(2.2, surplus*0.18))-1)*0.12*scale)
	case MarketAlgorithmLinear:
		fallthrough
	default:
		return basePrice * maxFloat(0.58, 0.92-surplus*0.04*scale)
	}
}

func (q countryQuotes) snapshot() []ResourceQuoteSnapshot {
	return []ResourceQuoteSnapshot{
		{Resource: "energy", Net: q.energy.net, LimitPrice: q.energy.limitPrice},
		{Resource: "coal", Net: q.coal.net, LimitPrice: q.coal.limitPrice},
		{Resource: "oil", Net: q.oil.net, LimitPrice: q.oil.limitPrice},
		{Resource: "copper", Net: q.copper.net, LimitPrice: q.copper.limitPrice},
		{Resource: "silicon", Net: q.silicon.net, LimitPrice: q.silicon.limitPrice},
	}
}

func (q *countryQuotes) quote(resource string) *resourceQuote {
	switch resource {
	case "energy":
		return &q.energy
	case "coal":
		return &q.coal
	case "oil":
		return &q.oil
	case "copper":
		return &q.copper
	case "silicon":
		return &q.silicon
	default:
		return &q.energy
	}
}

func (q *countryQuotes) adjust(resource string, delta float64) {
	quote := q.quote(resource)
	quote.net += delta
}

func sortResourcesForTrade(from countryQuotes, to countryQuotes) []string {
	resources := []string{"copper", "silicon", "coal", "oil"}
	sort.SliceStable(resources, func(i, j int) bool {
		left := resourceSpread(from, to, resources[i])
		right := resourceSpread(from, to, resources[j])
		return left > right
	})
	return resources
}

func resourceSpread(from countryQuotes, to countryQuotes, resource string) float64 {
	seller := from.quote(resource)
	buyer := to.quote(resource)
	if seller.net >= -0.05 || buyer.net <= 0.05 {
		return -1
	}
	return buyer.limitPrice - seller.limitPrice
}

func (s *Simulation) addResource(stockpiles *ResourceStockpile, resource string, amount float64) {
	switch resource {
	case "coal":
		stockpiles.Coal += amount
	case "oil":
		stockpiles.Oil += amount
	case "copper":
		stockpiles.Copper += amount
	case "silicon":
		stockpiles.Silicon += amount
	}
}

func (c countryState) infrastructureSaturated(gridCap float64) bool {
	if gridCap <= 0 {
		return true
	}
	return c.lastGeneratedEnergy > 0 && c.lastDeliveredEnergy/gridCap > 0.88
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func isValidBuildFocus(focus BuildFocus) bool {
	switch focus {
	case BuildHold, BuildEnergy, BuildCompute, BuildInfrastructure:
		return true
	default:
		return false
	}
}
