package main

import "omega/server/internal/econ"

type compareRequest struct {
	Count      int                    `json:"count"`
	Algorithms []econ.MarketAlgorithm `json:"algorithms"`
}

type compareScenario struct {
	Algorithm    econ.MarketAlgorithm   `json:"algorithm"`
	Label        string                 `json:"label"`
	Summary      compareScenarioSummary `json:"summary"`
	StatsHistory []econ.TickStats       `json:"statsHistory"`
	FinalWorld   econ.WorldSnapshot     `json:"finalWorld"`
	FinalMarket  econ.MarketSnapshot    `json:"finalMarket"`
	RecentEvents []string               `json:"recentEvents"`
}

type compareScenarioSummary struct {
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

type compareResponse struct {
	Count     int               `json:"count"`
	Scenarios []compareScenario `json:"scenarios"`
}

func runComparisons(base econ.Config, provider econ.DecisionProvider, count int, algorithms []econ.MarketAlgorithm) (compareResponse, error) {
	if count <= 0 {
		count = 120
	}
	if count > 800 {
		count = 800
	}

	if len(algorithms) == 0 {
		algorithms = econ.KnownMarketAlgorithms()
	}
	response := compareResponse{
		Count:     count,
		Scenarios: make([]compareScenario, 0, len(algorithms)),
	}

	for _, algorithm := range algorithms {
		cfg := base
		cfg.MarketAlgorithm = algorithm

		sim, err := econ.NewSimulation(cfg)
		if err != nil {
			return compareResponse{}, err
		}

		history := make([]econ.TickStats, 0, count)
		world := sim.Snapshot()
		market := sim.MarketSnapshot()
		events := make([]string, 0, 12)
		last := econ.TickStats{
			Tick:            0,
			LivingCountries: len(cfg.Countries),
		}
		totalEnergyTrade := 0.0
		totalCargoTrade := 0.0
		totalEnergyDelivered := 0.0
		peakShortage := 0.0

		for i := 0; i < count; i++ {
			prevWorld := world
			last = sim.Step(provider)
			world = sim.Snapshot()
			market = sim.MarketSnapshot()
			events = deriveEvents(prevWorld, world)
			history = append(history, last)
			totalEnergyTrade += last.TotalEnergyTraded
			totalCargoTrade += last.TotalCargoTraded
			totalEnergyDelivered += last.TotalEnergyDelivered
			if last.TotalShortage > peakShortage {
				peakShortage = last.TotalShortage
			}
		}

		response.Scenarios = append(response.Scenarios, compareScenario{
			Algorithm: algorithm,
			Label:     compareLabel(algorithm),
			Summary: compareScenarioSummary{
				FinalTick:                 last.Tick,
				LivingCountries:           last.LivingCountries,
				FinalEnergyDelivered:      last.TotalEnergyDelivered,
				FinalShortage:             last.TotalShortage,
				FinalTreasury:             last.TotalTreasury,
				FinalComputeEfficiency:    last.AvgComputeEfficiency,
				CumulativeEnergyTraded:    totalEnergyTrade,
				CumulativeCargoTraded:     totalCargoTrade,
				CumulativeEnergyDelivered: totalEnergyDelivered,
				PeakShortage:              peakShortage,
			},
			StatsHistory: history,
			FinalWorld:   world,
			FinalMarket:  market,
			RecentEvents: events,
		})
	}

	return response, nil
}

func compareLabel(algorithm econ.MarketAlgorithm) string {
	switch algorithm {
	case econ.MarketAlgorithmNoTrade:
		return "No Trade"
	case econ.MarketAlgorithmLinear:
		return "Linear Quotes"
	case econ.MarketAlgorithmScarcitySpike:
		return "Scarcity Spike"
	default:
		return string(algorithm)
	}
}
