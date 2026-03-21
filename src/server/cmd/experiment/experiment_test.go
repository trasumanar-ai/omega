package main

import "testing"

func TestBuiltInExperimentTemplatesValidate(t *testing.T) {
	for _, template := range builtInExperimentTemplates() {
		if err := validateExperimentSpec(template.spec); err != nil {
			t.Fatalf("template %s failed validation: %v", template.id, err)
		}
	}
}

func TestRunExperimentProducesRankings(t *testing.T) {
	spec, err := loadExperimentTemplate("default_algorithms")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	spec.Steps = 8
	spec.SampleEvery = 2

	result, err := runExperiment(spec, providerSetup{Mode: "heuristic"})
	if err != nil {
		t.Fatalf("run experiment: %v", err)
	}
	if len(result.Scenarios) != 3 {
		t.Fatalf("expected 3 scenarios, got %d", len(result.Scenarios))
	}
	if len(result.Rankings) != 3 {
		t.Fatalf("expected 3 rankings, got %d", len(result.Rankings))
	}
	if len(result.Comparisons) != 2 {
		t.Fatalf("expected 2 comparisons, got %d", len(result.Comparisons))
	}
}
