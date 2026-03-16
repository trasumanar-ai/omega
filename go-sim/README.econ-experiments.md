# Econ Experiments CLI

`omega-econ` now supports CLI-first experiment runs with a simple scientific workflow:

1. define a question and hypothesis
2. choose independent variables
3. run a batch of scenarios
4. inspect rankings and control comparisons
5. save results to disk as JSON

## Commands

List built-in templates:

```bash
cd go-sim
go run ./cmd/omega-econ experiment list
```

Print a template spec as JSON:

```bash
cd go-sim
go run ./cmd/omega-econ experiment print -template default_algorithms
```

Run a built-in experiment:

```bash
cd go-sim
go run ./cmd/omega-econ experiment run -template default_algorithms
go run ./cmd/omega-econ experiment run -template route_capacity_linear
```

Run a custom spec file:

```bash
cd go-sim
go run ./cmd/omega-econ experiment run -spec ./experiments/my_experiment.json
```

Show a saved result:

```bash
cd go-sim
go run ./cmd/omega-econ experiment show -file ./experiment-results/experiment-20260316T221545.462975179Z.json
```

## Built-in templates

- `default_algorithms`
- `resource_linear`
- `route_capacity_linear`
- `demand_shock_algorithms`

## Result files

Each experiment run is written under:

```text
go-sim/experiment-results/
```

The saved JSON includes:

- experiment metadata
- hypothesis and question
- scenario configs
- sampled traces
- final rankings
- control comparisons
