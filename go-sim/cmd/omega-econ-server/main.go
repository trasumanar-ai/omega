package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"omega/go-sim/internal/econ"
)

type stepRequest struct {
	Count int `json:"count"`
}

type stateResponse struct {
	Config       econ.Config         `json:"config"`
	Stats        econ.TickStats      `json:"stats"`
	World        econ.WorldSnapshot  `json:"world"`
	Market       econ.MarketSnapshot `json:"market"`
	Run          runInfo             `json:"run"`
	RecentEvents []string            `json:"recentEvents"`
	LLMMode      string              `json:"llmMode"`
	LLMModel     string              `json:"llmModel"`
}

type serverState struct {
	mu          sync.Mutex
	config      econ.Config
	engine      *econ.Simulation
	lab         *labManager
	archive     *labArchiveStore
	experiments *experimentStore
	stats       econ.TickStats
	market      econ.MarketSnapshot
	recorder    *runRecorder
	events      []string
	provider    econ.DecisionProvider
	llmMode     string
	llmModel    string
}

func newServerState(cfg econ.Config, provider econ.DecisionProvider, llmMode string, llmModel string) (*serverState, error) {
	engine, err := econ.NewSimulation(cfg)
	if err != nil {
		return nil, err
	}
	recorder, err := newRunRecorder("runs")
	if err != nil {
		return nil, err
	}
	archive, err := newLabArchiveStore("lab-results")
	if err != nil {
		return nil, err
	}
	experiments, err := newExperimentStore("experiment-results")
	if err != nil {
		return nil, err
	}
	lab, err := newLabManager(cfg, provider)
	if err != nil {
		return nil, err
	}
	state := &serverState{
		config:      cfg,
		engine:      engine,
		lab:         lab,
		archive:     archive,
		experiments: experiments,
		market:      engine.MarketSnapshot(),
		recorder:    recorder,
		provider:    provider,
		llmMode:     llmMode,
		llmModel:    llmModel,
		stats: econ.TickStats{
			Tick:            0,
			LivingCountries: len(cfg.Countries),
		},
	}
	initialWorld := engine.Snapshot()
	initialRecord := makeInitialRecord(state.stats, initialWorld, state.market, llmMode)
	state.events = append([]string(nil), initialRecord.Events...)
	if err := state.recorder.start(cfg, llmMode, llmModel, initialRecord); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *serverState) resetLocked(cfg econ.Config) error {
	if s.recorder != nil {
		_ = s.recorder.finalize("reset")
	}
	engine, err := econ.NewSimulation(cfg)
	if err != nil {
		return err
	}
	s.config = cfg
	s.engine = engine
	if s.lab == nil {
		s.lab, err = newLabManager(cfg, s.provider)
		if err != nil {
			return err
		}
	} else {
		if err := s.lab.rebase(cfg); err != nil {
			return err
		}
	}
	s.market = engine.MarketSnapshot()
	s.stats = econ.TickStats{
		Tick:            0,
		LivingCountries: len(cfg.Countries),
	}
	initialRecord := makeInitialRecord(s.stats, engine.Snapshot(), s.market, s.llmMode)
	s.events = append([]string(nil), initialRecord.Events...)
	if s.recorder != nil {
		if err := s.recorder.start(cfg, s.llmMode, s.llmModel, initialRecord); err != nil {
			return err
		}
	}
	return nil
}

func (s *serverState) responseLocked() stateResponse {
	return stateResponse{
		Config:       s.config,
		Stats:        s.stats,
		World:        s.engine.Snapshot(),
		Market:       s.market,
		Run:          s.recorder.currentInfo(),
		RecentEvents: append([]string(nil), s.events...),
		LLMMode:      s.llmMode,
		LLMModel:     s.llmModel,
	}
}

func main() {
	port := flag.Int("port", 8090, "HTTP port")
	flag.Parse()

	provider := econ.DecisionProvider(nil)
	llmMode := "heuristic"
	llmModel := ""
	if llmCfg, err := econ.LoadOpenRouterConfigFromEnv(); err == nil {
		provider = econ.NewOpenRouterProvider(llmCfg).ProviderFunc()
		llmMode = "openrouter"
		llmModel = llmCfg.Model
		log.Printf("using OpenRouter LLM provider with model %s", llmModel)
	} else {
		log.Printf("OpenRouter disabled, falling back to heuristic decisions: %v", err)
	}

	state, err := newServerState(econ.DefaultConfig(), provider, llmMode, llmModel)
	if err != nil {
		log.Fatalf("failed to initialize econ simulation: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":       true,
			"service":  "omega-econ",
			"llmMode":  state.llmMode,
			"llmModel": state.llmModel,
		})
	})

	mux.HandleFunc("/api/econ/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		resp := state.responseLocked()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/step", func(w http.ResponseWriter, r *http.Request) {
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
		if req.Count > 5000 {
			req.Count = 5000
		}

		state.mu.Lock()
		for i := 0; i < req.Count; i++ {
			prevWorld := state.engine.Snapshot()
			state.stats = state.engine.Step(state.provider)
			world := state.engine.Snapshot()
			state.market = state.engine.MarketSnapshot()
			state.events = deriveEvents(prevWorld, world)
			if state.recorder != nil {
				_ = state.recorder.append(tickRecord{
					Tick:       state.stats.Tick,
					RecordedAt: time.Now().UTC().Format(time.RFC3339),
					Stats:      state.stats,
					World:      world,
					Market:     state.market,
					Events:     append([]string(nil), state.events...),
				})
			}
		}
		resp := state.responseLocked()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		err := state.resetLocked(state.config)
		resp := state.responseLocked()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var cfg econ.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
			return
		}
		state.mu.Lock()
		err := state.resetLocked(cfg)
		resp := state.responseLocked()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/compare", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		req := compareRequest{Count: 120}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		state.mu.Lock()
		cfg := state.config
		provider := state.provider
		state.mu.Unlock()

		resp, err := runComparisons(cfg, provider, req.Count, req.Algorithms)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/lab", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		resp := state.lab.snapshot()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/lab/step", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		req := stepRequest{Count: 1}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		state.mu.Lock()
		state.lab.stepAll(req.Count)
		resp := state.lab.snapshot()
		state.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/lab/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		err := state.lab.resetAll()
		resp := state.lab.snapshot()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/lab/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		req := labCreateRequest{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		state.mu.Lock()
		_, err := state.lab.createSession(req.Name, req.Presets, req.Algorithms)
		resp := state.lab.snapshot()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/api/econ/lab/archive", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		records, err := state.archive.list()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, records)
	})

	mux.HandleFunc("/api/econ/lab/archive/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/econ/lab/archive/"), "/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing archive id")
			return
		}
		state.mu.Lock()
		record, err := state.archive.get(id)
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, record)
	})

	mux.HandleFunc("/api/econ/experiments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		records, err := state.experiments.list()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, records)
	})

	mux.HandleFunc("/api/econ/experiments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/econ/experiments/"), "/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing experiment id")
			return
		}
		state.mu.Lock()
		record, err := state.experiments.get(id)
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, record)
	})

	mux.HandleFunc("/api/econ/lab/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/econ/lab/sessions/")
		path = strings.Trim(path, "/")
		if path == "" {
			writeError(w, http.StatusBadRequest, "missing session id")
			return
		}

		parts := strings.Split(path, "/")
		sessionID := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}

		switch {
		case r.Method == http.MethodDelete && action == "":
			state.mu.Lock()
			err := state.lab.deleteSession(sessionID)
			resp := state.lab.snapshot()
			state.mu.Unlock()
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case r.Method == http.MethodPost && action == "step":
			req := stepRequest{Count: 1}
			if r.Body != nil {
				_ = json.NewDecoder(r.Body).Decode(&req)
			}
			state.mu.Lock()
			err := state.lab.stepSession(sessionID, req.Count)
			resp := state.lab.snapshot()
			state.mu.Unlock()
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case r.Method == http.MethodPost && action == "reset":
			req := labResetRequest{}
			if r.Body != nil {
				_ = json.NewDecoder(r.Body).Decode(&req)
			}
			state.mu.Lock()
			err := state.lab.resetSession(sessionID, req.Presets, req.Algorithms)
			resp := state.lab.snapshot()
			state.mu.Unlock()
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case r.Method == http.MethodPost && action == "save":
			req := labArchiveSaveRequest{}
			if r.Body != nil {
				_ = json.NewDecoder(r.Body).Decode(&req)
			}
			state.mu.Lock()
			session, err := state.lab.findSession(sessionID)
			if err == nil {
				_, err = state.archive.save(session.snapshot(), req.Name, req.Notes)
			}
			state.mu.Unlock()
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			writeError(w, http.StatusMethodNotAllowed, "unsupported lab session operation")
		}
	})

	mux.HandleFunc("/api/econ/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		state.mu.Lock()
		runs, err := state.recorder.list()
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, runs)
	})

	mux.HandleFunc("/api/econ/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/econ/runs/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing run id")
			return
		}
		state.mu.Lock()
		var (
			record *runRecord
			err    error
		)
		if id == "current" {
			record = state.recorder.currentRecord()
		} else {
			record, err = state.recorder.get(id)
		}
		state.mu.Unlock()
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if record == nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, record)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("omega econ server listening on http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}
