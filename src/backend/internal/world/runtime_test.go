package world_test

import (
	"testing"
	"time"

	"omega/backend/internal/agents"
	"omega/backend/internal/world"
)

func TestSimulationSnapshotIncludesTradesAndMessages(t *testing.T) {
	sim, err := world.NewSimulation(world.DefaultConfig())
	if err != nil {
		t.Fatalf("NewSimulation: %v", err)
	}

	sim.BeginTick()
	sim.ApplyCommand(world.AgentCommand{
		From:         "aurora",
		ObservedTick: sim.Tick(),
		Action: world.AgentAction{
			Action:  world.ActionBroadcast,
			Message: "need silicon, can trade copper",
		},
		Source: "test",
	})
	sim.ApplyCommand(world.AgentCommand{
		From:         "aurora",
		ObservedTick: sim.Tick(),
		Action: world.AgentAction{
			Action:        world.ActionPropose,
			To:            "silica",
			OfferResource: "copper",
			OfferAmount:   1,
			WantResource:  "silicon",
			WantAmount:    1,
		},
		Source: "test",
	})
	proposals := sim.Snapshot().Proposals
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	sim.ApplyCommand(world.AgentCommand{
		From:         "silica",
		ObservedTick: sim.Tick(),
		Action: world.AgentAction{
			Action:     world.ActionAccept,
			ProposalID: proposals[0].ID,
		},
		Source: "test",
	})

	snapshot := sim.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("expected 1 broadcast message, got %d", len(snapshot.Messages))
	}
	if len(snapshot.Trades) != 1 {
		t.Fatalf("expected 1 trade in snapshot, got %d", len(snapshot.Trades))
	}
	if got := snapshot.Trades[0]; got.From != "aurora" || got.To != "silica" {
		t.Fatalf("unexpected trade record: %+v", got)
	}

	stats := sim.FinalizeTick()
	if stats.TotalTrades != 1 {
		t.Fatalf("expected 1 trade in stats, got %d", stats.TotalTrades)
	}
}

func TestRuntimeHeuristicCommunicatesAndTrades(t *testing.T) {
	cfg := world.DefaultConfig()
	runtime, err := world.NewRuntime(cfg, agents.HeuristicProvider(cfg), 20*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer runtime.Close()

	runtime.RunSteps(1)
	first := runtime.Snapshot()
	if len(first.Messages) == 0 {
		t.Fatal("expected at least one broadcast in the first tick")
	}
	if len(first.Proposals) == 0 {
		t.Fatal("expected at least one proposal in the first tick")
	}

	stats := runtime.RunSteps(1)
	if stats.TotalTrades == 0 {
		t.Fatal("expected at least one trade by the second tick")
	}
	second := runtime.Snapshot()
	if len(second.Trades) == 0 {
		t.Fatal("expected trade records in the second tick snapshot")
	}
}
