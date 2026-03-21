package world

import (
	"context"
	"time"
)

const DefaultDecisionWindow = 25 * time.Millisecond

type Runtime struct {
	config         Config
	sim            *Simulation
	stats          TickStats
	provider       DecisionProvider
	decisionWindow time.Duration
	commands       chan AgentCommand
	agents         map[string]*runtimeAgent
	cancel         context.CancelFunc
}

type runtimeAgent struct {
	id           string
	config       Config
	provider     DecisionProvider
	observations chan CountryObservation
	commands     chan<- AgentCommand
}

func NewRuntime(cfg Config, provider DecisionProvider, decisionWindow time.Duration) (*Runtime, error) {
	if decisionWindow <= 0 {
		decisionWindow = DefaultDecisionWindow
	}

	sim, err := NewSimulation(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	runtime := &Runtime{
		config:         cfg,
		sim:            sim,
		stats:          TickStats{Tick: 0, LivingCountries: len(cfg.Countries)},
		provider:       provider,
		decisionWindow: decisionWindow,
		commands:       make(chan AgentCommand, len(cfg.Countries)*16),
		agents:         make(map[string]*runtimeAgent, len(cfg.Countries)),
		cancel:         cancel,
	}

	for _, country := range cfg.Countries {
		agent := &runtimeAgent{
			id:           country.ID,
			config:       cfg,
			provider:     provider,
			observations: make(chan CountryObservation, 1),
			commands:     runtime.commands,
		}
		runtime.agents[country.ID] = agent
		go agent.run(ctx)
	}

	return runtime, nil
}

func (r *Runtime) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runtime) Config() Config {
	return r.config
}

func (r *Runtime) Snapshot() WorldSnapshot {
	return r.sim.Snapshot()
}

func (r *Runtime) Stats() TickStats {
	return r.stats
}

func (r *Runtime) RunSteps(count int) TickStats {
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		r.runStep()
	}
	return r.stats
}

func (r *Runtime) runStep() {
	r.sim.BeginTick()
	for i := range r.config.Countries {
		observation := r.sim.makeObservation(i)
		if agent := r.agents[r.config.Countries[i].ID]; agent != nil {
			agent.publish(observation)
		}
	}

	deadline := time.Now().Add(r.decisionWindow)
	for {
		processed := r.drainCommands()
		if time.Now().After(deadline) {
			break
		}
		if !processed {
			time.Sleep(time.Millisecond)
		}
	}

	r.drainCommands()
	r.stats = r.sim.FinalizeTick()
}

func (r *Runtime) drainCommands() bool {
	processed := false
	for {
		select {
		case cmd := <-r.commands:
			r.sim.ApplyCommand(cmd)
			processed = true
		default:
			return processed
		}
	}
}

func (a *runtimeAgent) publish(observation CountryObservation) {
	select {
	case a.observations <- observation:
	default:
		select {
		case <-a.observations:
		default:
		}
		a.observations <- observation
	}
}

func (a *runtimeAgent) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case observation := <-a.observations:
			var result DecisionResult
			if a.provider != nil {
				result = a.provider(observation)
			}
			if result.Source == "" {
				result.Source = "runtime"
			}
			if len(result.Actions) == 0 {
				result.Actions = []AgentAction{{Action: ActionHold}}
			}
			for _, action := range result.Actions {
				select {
				case <-ctx.Done():
					return
				case a.commands <- AgentCommand{
					From:         a.id,
					ObservedTick: observation.Tick,
					Action:       action,
					Source:       result.Source,
					Summary:      result.Summary,
					Error:        result.Error,
				}:
				}
			}
		}
	}
}
