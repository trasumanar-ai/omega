package econ

import "testing"

func TestSimulationProducesEnergyAndDepletesDeposits(t *testing.T) {
	sim, err := NewSimulation(DefaultConfig())
	if err != nil {
		t.Fatalf("NewSimulation() error = %v", err)
	}

	before := sim.Snapshot()
	stats := sim.Step(nil)
	after := sim.Snapshot()

	if stats.TotalEnergyGenerated <= 0 {
		t.Fatalf("expected positive energy generation, got %v", stats.TotalEnergyGenerated)
	}
	if stats.TotalEnergyDelivered <= 0 {
		t.Fatalf("expected positive delivered energy, got %v", stats.TotalEnergyDelivered)
	}
	if before.Countries[0].Deposits.Coal <= after.Countries[0].Deposits.Coal {
		t.Fatalf("expected coal deposit to deplete, before=%v after=%v", before.Countries[0].Deposits.Coal, after.Countries[0].Deposits.Coal)
	}
}

func TestSimulationStaysAliveInEarlyWindow(t *testing.T) {
	sim, err := NewSimulation(DefaultConfig())
	if err != nil {
		t.Fatalf("NewSimulation() error = %v", err)
	}

	var last TickStats
	for i := 0; i < 60; i++ {
		last = sim.Step(nil)
	}
	if last.LivingCountries != 4 {
		t.Fatalf("expected all countries alive at tick 60, got %d", last.LivingCountries)
	}
}

func TestSimulationBuildsComputeOverTime(t *testing.T) {
	sim, err := NewSimulation(DefaultConfig())
	if err != nil {
		t.Fatalf("NewSimulation() error = %v", err)
	}

	initial := sim.Snapshot()
	initialAvg := 0.0
	for _, country := range initial.Countries {
		initialAvg += country.ComputeEfficiency
	}
	initialAvg /= float64(len(initial.Countries))

	var last TickStats
	for i := 0; i < 60; i++ {
		last = sim.Step(nil)
	}
	if last.AvgComputeEfficiency <= initialAvg {
		t.Fatalf("expected compute efficiency to improve, initial=%v final=%v", initialAvg, last.AvgComputeEfficiency)
	}
}

func TestSimulationPublishesPerResourceQuotes(t *testing.T) {
	sim, err := NewSimulation(DefaultConfig())
	if err != nil {
		t.Fatalf("NewSimulation() error = %v", err)
	}

	sim.Step(nil)
	world := sim.Snapshot()
	market := sim.MarketSnapshot()

	if len(world.Countries) == 0 {
		t.Fatalf("expected countries in snapshot")
	}
	if got := len(world.Countries[0].MarketQuotes); got != 5 {
		t.Fatalf("expected 5 country market quotes, got %d", got)
	}
	if len(market.Countries) == 0 || len(market.Countries[0].Quotes) != 5 {
		t.Fatalf("expected market snapshot quotes for each country")
	}
	for _, quote := range market.Countries[0].Quotes {
		if quote.LimitPrice <= 0 {
			t.Fatalf("expected positive limit price for %s, got %v", quote.Resource, quote.LimitPrice)
		}
	}
}

func TestNoTradeAlgorithmProducesZeroTrade(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MarketAlgorithm = MarketAlgorithmNoTrade

	sim, err := NewSimulation(cfg)
	if err != nil {
		t.Fatalf("NewSimulation() error = %v", err)
	}

	var last TickStats
	for i := 0; i < 20; i++ {
		last = sim.Step(nil)
	}

	if last.TotalEnergyTraded != 0 {
		t.Fatalf("expected no energy trade, got %v", last.TotalEnergyTraded)
	}
	if last.TotalCargoTraded != 0 {
		t.Fatalf("expected no cargo trade, got %v", last.TotalCargoTraded)
	}
}

func TestScarcitySpikeBidsHigherThanLinear(t *testing.T) {
	linearCfg := DefaultConfig()
	linearCfg.MarketAlgorithm = MarketAlgorithmLinear
	linear, err := NewSimulation(linearCfg)
	if err != nil {
		t.Fatalf("linear NewSimulation() error = %v", err)
	}

	spikeCfg := DefaultConfig()
	spikeCfg.MarketAlgorithm = MarketAlgorithmScarcitySpike
	spike, err := NewSimulation(spikeCfg)
	if err != nil {
		t.Fatalf("scarcity NewSimulation() error = %v", err)
	}

	linear.Step(nil)
	spike.Step(nil)

	linearQuote := quoteForResource(linear.Snapshot().Countries[0].MarketQuotes, "energy")
	spikeQuote := quoteForResource(spike.Snapshot().Countries[0].MarketQuotes, "energy")
	if linearQuote.Net <= 0 || spikeQuote.Net <= 0 {
		t.Fatalf("expected positive energy buy quotes, got linear=%+v spike=%+v", linearQuote, spikeQuote)
	}
	if spikeQuote.LimitPrice <= linearQuote.LimitPrice {
		t.Fatalf("expected scarcity spike bid > linear bid, linear=%v spike=%v", linearQuote.LimitPrice, spikeQuote.LimitPrice)
	}
}

func quoteForResource(quotes []ResourceQuoteSnapshot, resource string) ResourceQuoteSnapshot {
	for _, quote := range quotes {
		if quote.Resource == resource {
			return quote
		}
	}
	return ResourceQuoteSnapshot{}
}
