package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"omega/go-sim/internal/sim"
)

type runEndReason string

const (
	runEndManualReset      runEndReason = "manual_reset"
	runEndConfigChanged    runEndReason = "config_changed"
	runEndReplayRequest    runEndReason = "replay_request"
	runEndServerShutdown   runEndReason = "server_shutdown"
	runEndRecoveredRestart runEndReason = "recovered_after_restart"
	runHistoryLimit                     = 400
	runTraceLimit                       = 2400
	runSampleEvery                      = 5
)

type runTracePoint struct {
	Tick             int     `json:"tick"`
	AliveAgents      int     `json:"aliveAgents"`
	AvgEnergy        float64 `json:"avgEnergy"`
	AvgInventory     float64 `json:"avgInventory"`
	DeathsThisTick   int     `json:"deathsThisTick"`
	CollectedFruit   int     `json:"collectedFruit"`
	EatenFruit       int     `json:"eatenFruit"`
	TradedFruit      int     `json:"tradedFruit"`
	GroundFruitTotal int     `json:"groundFruitTotal"`
	DeficientAgents  int     `json:"deficientAgents"`
}

type runSummary struct {
	ID            string       `json:"id"`
	SchemaVersion int          `json:"schemaVersion"`
	ServerVersion string       `json:"serverVersion"`
	APIVersion    string       `json:"apiVersion"`
	Seed          uint32       `json:"seed"`
	StartedAt     string       `json:"startedAt"`
	EndedAt       string       `json:"endedAt"`
	Reason        runEndReason `json:"reason"`
	StepCount     int          `json:"stepCount"`
	MaxTick       int          `json:"maxTick"`
	FinalTick     int          `json:"finalTick"`
	FinalAlive    int          `json:"finalAlive"`
	FinalEnergy   float64      `json:"finalEnergy"`
}

type runRecord struct {
	ID            string               `json:"id"`
	SchemaVersion int                  `json:"schemaVersion"`
	ServerVersion string               `json:"serverVersion"`
	APIVersion    string               `json:"apiVersion"`
	Seed          uint32               `json:"seed"`
	StartedAt     string               `json:"startedAt"`
	EndedAt       string               `json:"endedAt"`
	Reason        runEndReason         `json:"reason"`
	Config        sim.SimulationConfig `json:"config"`
	StepCount     int                  `json:"stepCount"`
	MaxTick       int                  `json:"maxTick"`
	Final         sim.TickStats        `json:"final"`
	Trace         []runTracePoint      `json:"trace"`
}

type activeRunRecord struct {
	ID            string               `json:"id"`
	SchemaVersion int                  `json:"schemaVersion"`
	ServerVersion string               `json:"serverVersion"`
	APIVersion    string               `json:"apiVersion"`
	Seed          uint32               `json:"seed"`
	StartedAt     string               `json:"startedAt"`
	UpdatedAt     string               `json:"updatedAt"`
	Config        sim.SimulationConfig `json:"config"`
	StepCount     int                  `json:"stepCount"`
	MaxTick       int                  `json:"maxTick"`
	LastStats     sim.TickStats        `json:"lastStats"`
	Trace         []runTracePoint      `json:"trace"`
}

type runLogger struct {
	dir          string
	active       *activeRunRecord
	runIndex     []runSummary
	historyLimit int
	traceLimit   int
	sampleEvery  int
}

func newRunLogger(dir string) (*runLogger, error) {
	if dir == "" {
		dir = "./runs"
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve runs dir: %w", err)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("create runs dir: %w", err)
	}

	rl := &runLogger{
		dir:          absDir,
		historyLimit: runHistoryLimit,
		traceLimit:   runTraceLimit,
		sampleEvery:  runSampleEvery,
	}
	if err := rl.loadIndex(); err != nil {
		return nil, err
	}
	if err := rl.recoverActiveRun(); err != nil {
		return nil, err
	}
	return rl, nil
}

func (r *runLogger) directory() string {
	return r.dir
}

func (r *runLogger) start(cfg sim.SimulationConfig, seed uint32, initialStats sim.TickStats) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	r.active = &activeRunRecord{
		ID:            createRunID(seed),
		SchemaVersion: runSchemaVersion,
		ServerVersion: serverVersion,
		APIVersion:    apiVersion,
		Seed:          seed,
		StartedAt:     now,
		UpdatedAt:     now,
		Config:        cfg,
		StepCount:     0,
		MaxTick:       initialStats.Tick,
		LastStats:     initialStats,
		Trace:         []runTracePoint{toTrace(initialStats)},
	}
	return r.saveActive()
}

func (r *runLogger) recordStep(stats sim.TickStats) error {
	if r.active == nil {
		return nil
	}
	clock := time.Now().UTC().Format(time.RFC3339Nano)
	r.active.StepCount++
	if stats.Tick > r.active.MaxTick {
		r.active.MaxTick = stats.Tick
	}
	r.active.LastStats = stats
	r.active.UpdatedAt = clock

	shouldRecord :=
		r.active.StepCount <= 120 ||
			r.active.StepCount%r.sampleEvery == 0 ||
			stats.DeathsThisTick > 0 ||
			stats.TradedFruit > 0

	if shouldRecord {
		r.active.Trace = append(r.active.Trace, toTrace(stats))
		if len(r.active.Trace) > r.traceLimit {
			r.active.Trace = r.active.Trace[len(r.active.Trace)-r.traceLimit:]
		}
	}

	return r.saveActive()
}

func (r *runLogger) finalize(reason runEndReason) error {
	if r.active == nil {
		return nil
	}
	active := r.active
	r.active = nil

	if active.StepCount <= 0 {
		_ = os.Remove(r.activePath())
		return nil
	}

	record := runRecord{
		ID:            active.ID,
		SchemaVersion: active.SchemaVersion,
		ServerVersion: active.ServerVersion,
		APIVersion:    active.APIVersion,
		Seed:          active.Seed,
		StartedAt:     active.StartedAt,
		EndedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		Reason:        reason,
		Config:        active.Config,
		StepCount:     active.StepCount,
		MaxTick:       active.MaxTick,
		Final:         active.LastStats,
		Trace:         append([]runTracePoint(nil), active.Trace...),
	}

	if err := r.saveRunRecord(record); err != nil {
		return err
	}
	r.runIndex = append(r.runIndex, summarizeRun(record))
	if len(r.runIndex) > r.historyLimit {
		r.runIndex = r.runIndex[len(r.runIndex)-r.historyLimit:]
	}
	if err := r.saveIndex(); err != nil {
		return err
	}
	_ = os.Remove(r.activePath())
	return nil
}

func (r *runLogger) activeSummary() *runSummary {
	if r.active == nil {
		return nil
	}
	last := r.active.LastStats
	summary := runSummary{
		ID:            r.active.ID,
		SchemaVersion: r.active.SchemaVersion,
		ServerVersion: r.active.ServerVersion,
		APIVersion:    r.active.APIVersion,
		Seed:          r.active.Seed,
		StartedAt:     r.active.StartedAt,
		EndedAt:       "",
		Reason:        "",
		StepCount:     r.active.StepCount,
		MaxTick:       r.active.MaxTick,
		FinalTick:     last.Tick,
		FinalAlive:    last.AliveAgents,
		FinalEnergy:   last.AvgEnergy,
	}
	return &summary
}

func (r *runLogger) listRuns(limit int) []runSummary {
	total := len(r.runIndex)
	if total == 0 {
		return []runSummary{}
	}
	if limit > 0 && limit < total {
		total = limit
	}
	start := len(r.runIndex) - total
	out := make([]runSummary, 0, total)
	for i := len(r.runIndex) - 1; i >= start; i-- {
		out = append(out, r.runIndex[i])
	}
	return out
}

func (r *runLogger) getRun(id string) (runRecord, error) {
	path := r.runPath(id)
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runRecord{}, fmt.Errorf("run not found: %s", id)
		}
		return runRecord{}, err
	}
	var record runRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return runRecord{}, fmt.Errorf("decode run record: %w", err)
	}
	normalizeRunRecord(&record)
	return record, nil
}

func (r *runLogger) recoverActiveRun() error {
	payload, err := os.ReadFile(r.activePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var active activeRunRecord
	if err := json.Unmarshal(payload, &active); err != nil {
		return fmt.Errorf("decode active run: %w", err)
	}
	normalizeActiveRunRecord(&active)
	r.active = &active
	if err := r.finalize(runEndRecoveredRestart); err != nil {
		return err
	}
	return nil
}

func (r *runLogger) saveRunRecord(record runRecord) error {
	path := r.runPath(record.ID)
	return writeJSONFile(path, record)
}

func (r *runLogger) loadIndex() error {
	payload, err := os.ReadFile(r.indexPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.runIndex = []runSummary{}
			return nil
		}
		return err
	}
	var index []runSummary
	if err := json.Unmarshal(payload, &index); err != nil {
		return fmt.Errorf("decode run index: %w", err)
	}
	for i := range index {
		normalizeRunSummary(&index[i])
	}
	r.runIndex = index
	return nil
}

func (r *runLogger) saveIndex() error {
	return writeJSONFile(r.indexPath(), r.runIndex)
}

func (r *runLogger) saveActive() error {
	if r.active == nil {
		_ = os.Remove(r.activePath())
		return nil
	}
	return writeJSONFile(r.activePath(), r.active)
}

func (r *runLogger) indexPath() string {
	return filepath.Join(r.dir, "run_index.json")
}

func (r *runLogger) activePath() string {
	return filepath.Join(r.dir, "active_run.json")
}

func (r *runLogger) runPath(id string) string {
	return filepath.Join(r.dir, fmt.Sprintf("run_%s.json", id))
}

func summarizeRun(record runRecord) runSummary {
	return runSummary{
		ID:            record.ID,
		SchemaVersion: record.SchemaVersion,
		ServerVersion: record.ServerVersion,
		APIVersion:    record.APIVersion,
		Seed:          record.Seed,
		StartedAt:     record.StartedAt,
		EndedAt:       record.EndedAt,
		Reason:        record.Reason,
		StepCount:     record.StepCount,
		MaxTick:       record.MaxTick,
		FinalTick:     record.Final.Tick,
		FinalAlive:    record.Final.AliveAgents,
		FinalEnergy:   record.Final.AvgEnergy,
	}
}

func normalizeRunSummary(summary *runSummary) {
	if summary.SchemaVersion <= 0 {
		summary.SchemaVersion = 0
	}
	if summary.ServerVersion == "" {
		summary.ServerVersion = "legacy"
	}
	if summary.APIVersion == "" {
		summary.APIVersion = "legacy"
	}
}

func normalizeRunRecord(record *runRecord) {
	if record.SchemaVersion <= 0 {
		record.SchemaVersion = 0
	}
	if record.ServerVersion == "" {
		record.ServerVersion = "legacy"
	}
	if record.APIVersion == "" {
		record.APIVersion = "legacy"
	}
}

func normalizeActiveRunRecord(record *activeRunRecord) {
	if record.SchemaVersion <= 0 {
		record.SchemaVersion = runSchemaVersion
	}
	if record.ServerVersion == "" {
		record.ServerVersion = serverVersion
	}
	if record.APIVersion == "" {
		record.APIVersion = apiVersion
	}
}

func toTrace(stats sim.TickStats) runTracePoint {
	return runTracePoint{
		Tick:             stats.Tick,
		AliveAgents:      stats.AliveAgents,
		AvgEnergy:        stats.AvgEnergy,
		AvgInventory:     stats.AvgInventory,
		DeathsThisTick:   stats.DeathsThisTick,
		CollectedFruit:   stats.CollectedFruit,
		EatenFruit:       stats.EatenFruit,
		TradedFruit:      stats.TradedFruit,
		GroundFruitTotal: stats.GroundFruitTotal,
		DeficientAgents:  stats.DeficientAgents,
	}
}

func createRunID(seed uint32) string {
	now := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	return fmt.Sprintf("%s_%08x", now, seed)
}

func writeJSONFile(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}
