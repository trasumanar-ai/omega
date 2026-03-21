package main

import (
	"fmt"

	"omega/backend/internal/econ"
)

type experimentTemplate struct {
	id          string
	description string
	spec        experimentSpec
}

type experimentPreset struct {
	ID          string
	Label       string
	Description string
}

func builtInExperimentTemplates() []experimentTemplate {
	return []experimentTemplate{
		{
			id:          "default_algorithms",
			description: "Default basin with no_trade vs linear vs scarcity_spike.",
			spec: experimentSpec{
				Version:              1,
				Name:                 "Default Basin Algorithm Sweep",
				Question:             "Does enabling trade matching improve survival on the default resource layout?",
				Hypothesis:           "Linear or scarcity-spike matching should keep more countries alive and reduce shortage relative to no_trade.",
				ControlScenarioID:    "default_no_trade",
				IndependentVariables: []string{"marketAlgorithm"},
				DependentMetrics:     []string{"livingCountries", "peakShortage", "cumulativeEnergyDelivered", "cumulativeEnergyTraded", "finalTreasury"},
				Steps:                120,
				SampleEvery:          8,
				Scenarios: []experimentScenarioSpec{
					{ID: "default_no_trade", Label: "Default · No Trade", Role: "control", Preset: "default", Algorithm: econ.MarketAlgorithmNoTrade},
					{ID: "default_linear", Label: "Default · Linear", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmLinear},
					{ID: "default_spike", Label: "Default · Scarcity Spike", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmScarcitySpike},
				},
			},
		},
		{
			id:          "resource_linear",
			description: "Linear algorithm across all resource presets.",
			spec: experimentSpec{
				Version:              1,
				Name:                 "Linear Matching Resource Sweep",
				Question:             "Which resource topology best supports the linear trade algorithm?",
				Hypothesis:           "Renewable-heavy presets should reduce shortage and improve treasury relative to metal bottlenecks under linear matching.",
				ControlScenarioID:    "linear_default",
				IndependentVariables: []string{"resourcePreset"},
				DependentMetrics:     []string{"livingCountries", "peakShortage", "cumulativeEnergyDelivered", "finalTreasury"},
				Steps:                140,
				SampleEvery:          10,
				Scenarios: []experimentScenarioSpec{
					{ID: "linear_default", Label: "Linear · Default Basin", Role: "control", Preset: "default", Algorithm: econ.MarketAlgorithmLinear},
					{ID: "linear_fossil", Label: "Linear · Fossil Skew", Role: "variant", Preset: "fossil_skew", Algorithm: econ.MarketAlgorithmLinear},
					{ID: "linear_renewable", Label: "Linear · Renewable Skew", Role: "variant", Preset: "renewable_skew", Algorithm: econ.MarketAlgorithmLinear},
					{ID: "linear_metal", Label: "Linear · Metal Bottleneck", Role: "variant", Preset: "metal_bottleneck", Algorithm: econ.MarketAlgorithmLinear},
				},
			},
		},
		{
			id:          "route_capacity_linear",
			description: "Linear matching with route capacity as the experimental variable.",
			spec: experimentSpec{
				Version:              1,
				Name:                 "Linear Route Capacity Sweep",
				Question:             "How sensitive is the linear dispatch logic to route capacity?",
				Hypothesis:           "Higher route capacity should improve delivered energy and reduce shortage for the linear algorithm.",
				ControlScenarioID:    "linear_route_1x",
				IndependentVariables: []string{"routeCapacityMultiplier"},
				DependentMetrics:     []string{"livingCountries", "peakShortage", "cumulativeEnergyDelivered", "cumulativeEnergyTraded"},
				Steps:                120,
				SampleEvery:          8,
				Scenarios: []experimentScenarioSpec{
					{ID: "linear_route_075x", Label: "Linear · Route 0.75x", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmLinear, Adjustments: experimentConfigAdjusters{RouteCapacityMultiplier: 0.75}},
					{ID: "linear_route_1x", Label: "Linear · Route 1.00x", Role: "control", Preset: "default", Algorithm: econ.MarketAlgorithmLinear},
					{ID: "linear_route_135x", Label: "Linear · Route 1.35x", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmLinear, Adjustments: experimentConfigAdjusters{RouteCapacityMultiplier: 1.35}},
				},
			},
		},
		{
			id:          "demand_shock_algorithms",
			description: "Algorithm comparison under a global demand shock.",
			spec: experimentSpec{
				Version:              1,
				Name:                 "Demand Shock Algorithm Stress Test",
				Question:             "Which market algorithm handles a 20% demand shock best?",
				Hypothesis:           "Scarcity-spike matching should degrade more gracefully than linear or no-trade under demand pressure.",
				ControlScenarioID:    "shock_no_trade",
				IndependentVariables: []string{"marketAlgorithm"},
				DependentMetrics:     []string{"livingCountries", "peakShortage", "cumulativeEnergyDelivered", "finalTreasury"},
				Steps:                140,
				SampleEvery:          10,
				Scenarios: []experimentScenarioSpec{
					{ID: "shock_no_trade", Label: "Demand Shock · No Trade", Role: "control", Preset: "default", Algorithm: econ.MarketAlgorithmNoTrade, Adjustments: experimentConfigAdjusters{BaseDemandMultiplier: 1.20}},
					{ID: "shock_linear", Label: "Demand Shock · Linear", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmLinear, Adjustments: experimentConfigAdjusters{BaseDemandMultiplier: 1.20}},
					{ID: "shock_spike", Label: "Demand Shock · Scarcity Spike", Role: "variant", Preset: "default", Algorithm: econ.MarketAlgorithmScarcitySpike, Adjustments: experimentConfigAdjusters{BaseDemandMultiplier: 1.20}},
				},
			},
		},
	}
}

func loadExperimentTemplate(templateID string) (experimentSpec, error) {
	for _, template := range builtInExperimentTemplates() {
		if template.id == templateID {
			return template.spec, nil
		}
	}
	return experimentSpec{}, fmt.Errorf("unknown experiment template: %s", templateID)
}

func experimentPresets() []experimentPreset {
	return []experimentPreset{
		{ID: "default", Label: "Default Basin", Description: "Base world with the current resource layout."},
		{ID: "fossil_skew", Label: "Fossil Skew", Description: "Coal and oil are more concentrated; renewables are weaker."},
		{ID: "renewable_skew", Label: "Renewable Skew", Description: "Sun and wind are stronger; fossil extraction is weaker."},
		{ID: "metal_bottleneck", Label: "Metal Bottleneck", Description: "Copper and silicon are scarce, stressing build chains."},
	}
}

func applyExperimentPreset(base econ.Config, presetID string) econ.Config {
	cfg := base
	cfg.Countries = append([]econ.CountryConfig(nil), base.Countries...)
	cfg.Routes = append([]econ.RouteConfig(nil), base.Routes...)
	for i := range cfg.Countries {
		country := cfg.Countries[i]
		cfg.Countries[i] = country
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
	}
	return cfg
}
