package agents

import (
	"math"

	"omega/backend/internal/world"
)

func HeuristicProvider(cfg world.Config) world.DecisionProvider {
	return func(obs world.CountryObservation) world.DecisionResult {
		return HeuristicDecision(cfg, obs.You.ID, obs)
	}
}

func FallbackProvider(cfg world.Config, primary world.DecisionProvider) world.DecisionProvider {
	return func(obs world.CountryObservation) world.DecisionResult {
		if primary == nil {
			return HeuristicDecision(cfg, obs.You.ID, obs)
		}

		result := primary(obs)
		if len(result.Actions) > 0 {
			if result.Source == "" {
				result.Source = "primary"
			}
			return result
		}

		fallback := HeuristicDecision(cfg, obs.You.ID, obs)
		if result.Error != "" {
			fallback.Error = result.Error
		}
		return fallback
	}
}

func HeuristicDecision(cfg world.Config, countryID string, obs world.CountryObservation) world.DecisionResult {
	selfCfg, ok := findCountryConfig(cfg, countryID)
	if !ok {
		return world.DecisionResult{
			Actions: []world.AgentAction{{Action: world.ActionHold}},
			Source:  "heuristic",
			Summary: "missing country config",
		}
	}

	var actions []world.AgentAction

	for _, proposal := range obs.Proposals {
		if resourceAmount(obs.You, proposal.WantResource) >= proposal.WantAmount {
			actions = append(actions, world.AgentAction{Action: world.ActionAccept, ProposalID: proposal.ID})
		}
	}

	needs := countryNeeds(cfg, selfCfg, obs.You)
	offerResource, offerAmount, hasOffer := bestOffer(cfg, selfCfg, obs.You)
	if len(needs) > 0 {
		broadcast := "need " + needs[0]
		if hasOffer {
			broadcast += ", can trade " + offerResource
		}
		actions = append(actions, world.AgentAction{Action: world.ActionBroadcast, Message: broadcast})

		usedTargets := map[string]bool{}
		for _, need := range needs {
			targetID, ok := bestPartner(cfg, countryID, need)
			if !ok || usedTargets[targetID] {
				continue
			}
			usedTargets[targetID] = true
			msg := "need " + need
			if hasOffer {
				msg += ", can trade " + offerResource
			}
			actions = append(actions, world.AgentAction{Action: world.ActionSend, To: targetID, Message: msg})
			if hasOffer {
				wantAmt := math.Max(1, math.Min(offerAmount*0.9, targetAmount(obs.You, need)))
				actions = append(actions, world.AgentAction{
					Action:        world.ActionPropose,
					To:            targetID,
					OfferResource: offerResource,
					OfferAmount:   round2(offerAmount),
					WantResource:  need,
					WantAmount:    round2(wantAmt),
				})
			}
		}
	}

	if obs.You.Stockpiles.Copper >= cfg.EnergyBuildCopper && obs.You.Stockpiles.Silicon >= cfg.EnergyBuildSilicon {
		actions = append(actions, world.AgentAction{Action: world.ActionBuild, Build: bestEnergyBuild(selfCfg)})
	}

	if len(actions) == 0 {
		actions = append(actions, world.AgentAction{Action: world.ActionHold})
	}

	return world.DecisionResult{
		Actions: actions,
		Source:  "heuristic",
		Summary: "communicate, trade, build",
	}
}

func bestEnergyBuild(cfg world.CountryConfig) world.BuildKind {
	solarScore := cfg.SunPotential
	windScore := cfg.WindPotential
	coalScore := cfg.Deposits.Coal * 0.01
	oilScore := cfg.Deposits.Oil * 0.01

	best := world.BuildKindSolarFarm
	bestScore := solarScore
	if windScore > bestScore {
		best = world.BuildKindWindFarm
		bestScore = windScore
	}
	if coalScore > bestScore {
		best = world.BuildKindCoalPlant
		bestScore = coalScore
	}
	if oilScore > bestScore {
		best = world.BuildKindOilPlant
	}
	return best
}

func countryNeeds(cfg world.Config, selfCfg world.CountryConfig, self world.CountrySnapshot) []string {
	var needs []string
	if self.Shortage > 0 || self.Stockpiles.Coal+self.Stockpiles.Oil < selfCfg.BaseDemand*0.35 {
		if selfCfg.Extractors.Coal < selfCfg.Extractors.Oil {
			needs = append(needs, "oil")
		} else {
			needs = append(needs, "coal")
		}
	}
	if self.Stockpiles.Copper < cfg.EnergyBuildCopper {
		needs = append(needs, "copper")
	}
	if self.Stockpiles.Silicon < cfg.EnergyBuildSilicon {
		needs = append(needs, "silicon")
	}
	return uniqueStrings(needs)
}

func bestOffer(cfg world.Config, selfCfg world.CountryConfig, self world.CountrySnapshot) (string, float64, bool) {
	type candidate struct {
		name    string
		stock   float64
		rate    float64
		minimum float64
	}

	candidates := []candidate{
		{name: "coal", stock: self.Stockpiles.Coal, rate: selfCfg.Extractors.Coal, minimum: math.Max(2, selfCfg.BaseDemand*0.2)},
		{name: "oil", stock: self.Stockpiles.Oil, rate: selfCfg.Extractors.Oil, minimum: math.Max(2, selfCfg.BaseDemand*0.2)},
		{name: "copper", stock: self.Stockpiles.Copper, rate: selfCfg.Extractors.Copper, minimum: cfg.EnergyBuildCopper},
		{name: "silicon", stock: self.Stockpiles.Silicon, rate: selfCfg.Extractors.Silicon, minimum: cfg.EnergyBuildSilicon},
	}

	bestName := ""
	bestAmount := 0.0
	bestScore := 0.0
	for _, candidate := range candidates {
		excess := candidate.stock - candidate.minimum
		if excess <= 0 {
			continue
		}
		score := excess + candidate.rate
		if score > bestScore {
			bestScore = score
			bestName = candidate.name
			bestAmount = math.Min(excess*0.5, math.Max(1, candidate.rate))
		}
	}

	if bestName == "" || bestAmount <= 0 {
		return "", 0, false
	}
	return bestName, bestAmount, true
}

func bestPartner(cfg world.Config, selfID string, resource string) (string, bool) {
	bestID := ""
	bestScore := -1.0
	for _, country := range cfg.Countries {
		if country.ID == selfID {
			continue
		}
		score := resourceRate(country, resource) + resourceDeposit(country, resource)*0.01
		if score > bestScore {
			bestScore = score
			bestID = country.ID
		}
	}
	return bestID, bestID != ""
}

func targetAmount(self world.CountrySnapshot, resource string) float64 {
	switch resource {
	case "coal", "oil":
		return math.Max(1, self.EnergyNeeded*0.25)
	case "copper":
		return math.Max(1, 5-self.Stockpiles.Copper)
	case "silicon":
		return math.Max(1, 3-self.Stockpiles.Silicon)
	default:
		return 1
	}
}

func resourceAmount(self world.CountrySnapshot, resource string) float64 {
	switch resource {
	case "coal":
		return self.Stockpiles.Coal
	case "oil":
		return self.Stockpiles.Oil
	case "copper":
		return self.Stockpiles.Copper
	case "silicon":
		return self.Stockpiles.Silicon
	case "money":
		return self.Money
	default:
		return 0
	}
}

func resourceRate(country world.CountryConfig, resource string) float64 {
	switch resource {
	case "coal":
		return country.Extractors.Coal
	case "oil":
		return country.Extractors.Oil
	case "copper":
		return country.Extractors.Copper
	case "silicon":
		return country.Extractors.Silicon
	default:
		return 0
	}
}

func resourceDeposit(country world.CountryConfig, resource string) float64 {
	switch resource {
	case "coal":
		return country.Deposits.Coal
	case "oil":
		return country.Deposits.Oil
	case "copper":
		return country.Deposits.Copper
	case "silicon":
		return country.Deposits.Silicon
	default:
		return 0
	}
}

func findCountryConfig(cfg world.Config, countryID string) (world.CountryConfig, bool) {
	for _, country := range cfg.Countries {
		if country.ID == countryID {
			return country, true
		}
	}
	return world.CountryConfig{}, false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
