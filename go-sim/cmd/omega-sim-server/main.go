package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"omega/go-sim/internal/sim"
)

type stepRequest struct {
	Count int `json:"count"`
}

type stateResponse struct {
	Config sim.SimulationConfig `json:"config"`
	Stats  sim.TickStats        `json:"stats"`
	World  sim.WorldSnapshot    `json:"world"`
}

type serverState struct {
	mu     sync.Mutex
	config sim.SimulationConfig
	seed   uint32
	engine *sim.Simulation
	stats  sim.TickStats
	runLog *runLogger
}

func newServerState(initial sim.SimulationConfig, runLog *runLogger) (*serverState, error) {
	s := &serverState{config: initial, runLog: runLog}
	if err := s.rebuildLocked(initial, runEndConfigChanged); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *serverState) rebuildLocked(cfg sim.SimulationConfig, reason runEndReason) error {
	seed := uint32(time.Now().UnixNano())
	return s.rebuildWithSeedLocked(cfg, seed, reason)
}

func (s *serverState) rebuildWithSeedLocked(cfg sim.SimulationConfig, seed uint32, reason runEndReason) error {
	if s.engine != nil {
		if err := s.finalizeRunLocked(reason); err != nil {
			log.Printf("warn: failed to finalize previous run: %v", err)
		}
	}

	engine, err := sim.NewSimulation(cfg, seed)
	if err != nil {
		return err
	}
	if err := engine.Validate(); err != nil {
		return err
	}
	s.config = cfg
	s.seed = seed
	s.engine = engine
	s.stats = sim.TickStats{
		Tick:        0,
		AliveAgents: cfg.AgentCount,
		Occupancy:   float64(cfg.AgentCount) / float64(maxInt(1, cfg.Width*cfg.Height)),
		AvgEnergy:   cfg.InitialEnergy,
	}
	if s.runLog != nil {
		if err := s.runLog.start(cfg, seed, s.stats); err != nil {
			log.Printf("warn: failed to start run logging: %v", err)
		}
	}
	return nil
}

func (s *serverState) finalizeRunLocked(reason runEndReason) error {
	if s.runLog == nil {
		return nil
	}
	return s.runLog.finalize(reason)
}

func (s *serverState) stateResponseLocked() stateResponse {
	return stateResponse{
		Config: s.config,
		Stats:  s.stats,
		World:  s.engine.Snapshot(),
	}
}

func main() {
	port := flag.Int("port", 8080, "HTTP port")
	runsDir := flag.String("runs-dir", "./runs", "directory for persisted run logs")
	flag.Parse()

	runLog, err := newRunLogger(*runsDir)
	if err != nil {
		log.Fatalf("failed to initialize run logger: %v", err)
	}

	state, err := newServerState(sim.DefaultConfig(), runLog)
	if err != nil {
		log.Fatalf("failed to initialize simulation: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"version": versionPayload(),
		})
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, versionPayload())
	})

	mux.HandleFunc("/api/sim/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		resp := state.stateResponseLocked()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/sim/step", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		req := stepRequest{Count: 1}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		if req.Count <= 0 {
			req.Count = 1
		}
		if req.Count > 10000 {
			req.Count = 10000
		}

		state.mu.Lock()
		for i := 0; i < req.Count; i++ {
			state.stats = state.engine.Step(nil)
			if state.runLog != nil {
				if err := state.runLog.recordStep(state.stats); err != nil {
					log.Printf("warn: failed to record run step: %v", err)
				}
			}
		}
		resp := state.stateResponseLocked()
		state.mu.Unlock()

		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/sim/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		state.mu.Lock()
		err := state.rebuildLocked(state.config, runEndManualReset)
		resp := state.stateResponseLocked()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/sim/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var cfg sim.SimulationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
			return
		}

		state.mu.Lock()
		err := state.rebuildLocked(cfg, runEndConfigChanged)
		resp := state.stateResponseLocked()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/sim/replay", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req struct {
			Seed   uint32               `json:"seed"`
			Config sim.SimulationConfig `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid replay request: %v", err))
			return
		}

		state.mu.Lock()
		err := state.rebuildWithSeedLocked(req.Config, req.Seed, runEndReplayRequest)
		if err != nil {
			state.mu.Unlock()
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		resp := state.stateResponseLocked()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/sim/agent/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/sim/agent/")
		agentID, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid agent id")
			return
		}

		state.mu.Lock()
		detail, err := state.engine.GetAgentDetail(agentID)
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, detail)
	})

	mux.HandleFunc("/api/sim/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		limit := 100
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				writeError(w, http.StatusBadRequest, "invalid limit")
				return
			}
			limit = parsed
		}

		state.mu.Lock()
		dir := ""
		runs := []runSummary{}
		active := (*runSummary)(nil)
		if state.runLog != nil {
			dir = state.runLog.directory()
			runs = state.runLog.listRuns(limit)
			active = state.runLog.activeSummary()
		}
		state.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"version":   versionPayload(),
			"directory": dir,
			"activeRun": active,
			"runs":      runs,
		})
	})

	mux.HandleFunc("/api/sim/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/sim/runs/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing run id")
			return
		}

		state.mu.Lock()
		if state.runLog == nil {
			state.mu.Unlock()
			writeError(w, http.StatusNotFound, "run logger not configured")
			return
		}
		record, err := state.runLog.getRun(id)
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, record)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("omega sim server listening on %s", addr)
	log.Printf("server version: %s (%s, run schema %d)", serverVersion, apiVersion, runSchemaVersion)
	log.Printf("run logs directory: %s", runLog.directory())

	server := &http.Server{
		Addr:    addr,
		Handler: withCORS(mux),
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("shutdown signal received")
		state.mu.Lock()
		if err := state.finalizeRunLocked(runEndServerShutdown); err != nil {
			log.Printf("warn: finalize on shutdown failed: %v", err)
		}
		state.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("warn: server shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	state.mu.Lock()
	if err := state.finalizeRunLocked(runEndServerShutdown); err != nil {
		log.Printf("warn: final run persist failed: %v", err)
	}
	state.mu.Unlock()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func versionPayload() map[string]any {
	return map[string]any{
		"serverVersion":    serverVersion,
		"apiVersion":       apiVersion,
		"runSchemaVersion": runSchemaVersion,
	}
}
