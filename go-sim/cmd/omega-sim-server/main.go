package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
}

func newServerState(initial sim.SimulationConfig) (*serverState, error) {
	s := &serverState{config: initial}
	if err := s.rebuildLocked(initial); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *serverState) rebuildLocked(cfg sim.SimulationConfig) error {
	seed := uint32(time.Now().UnixNano())
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
	return nil
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
	flag.Parse()

	state, err := newServerState(sim.DefaultConfig())
	if err != nil {
		log.Fatalf("failed to initialize simulation: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		err := state.rebuildLocked(state.config)
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
		err := state.rebuildLocked(cfg)
		resp := state.stateResponseLocked()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
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

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("omega sim server listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
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
