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
	"sync"
	"syscall"
	"time"

	"omega/backend/internal/agents"
	"omega/backend/internal/world"
)

type stepRequest struct {
	Count int `json:"count"`
}

type stateResponse struct {
	Config   world.Config        `json:"config"`
	Stats    world.TickStats     `json:"stats"`
	World    world.WorldSnapshot `json:"world"`
	LLMMode  string              `json:"llmMode"`
	LLMModel string              `json:"llmModel"`
}

type serverState struct {
	mu             sync.Mutex
	config         world.Config
	runtime        *world.Runtime
	llmProvider    world.DecisionProvider
	decisionWindow time.Duration
	llmMode        string
	llmModel       string
}

func main() {
	port := flag.Int("port", 8090, "HTTP port")
	decisionWindowMS := flag.Int("decision-window-ms", 25, "time budget for async agent actions per tick")
	flag.Parse()

	llmProvider := world.DecisionProvider(nil)
	llmMode := "heuristic"
	llmModel := ""
	if llmCfg, err := agents.LoadOpenRouterConfigFromEnv(); err == nil {
		llmProvider = agents.NewOpenRouterProvider(llmCfg).ProviderFunc()
		llmMode = "openrouter"
		llmModel = llmCfg.Model
		log.Printf("using LLM provider: %s", llmModel)
	} else {
		log.Printf("LLM disabled, heuristic mode: %v", err)
	}

	cfg := world.DefaultConfig()
	provider := agents.FallbackProvider(cfg, llmProvider)
	runtime, err := world.NewRuntime(cfg, provider, time.Duration(*decisionWindowMS)*time.Millisecond)
	if err != nil {
		log.Fatalf("init failed: %v", err)
	}
	defer runtime.Close()

	state := &serverState{
		config:         cfg,
		runtime:        runtime,
		llmProvider:    llmProvider,
		decisionWindow: time.Duration(*decisionWindowMS) * time.Millisecond,
		llmMode:        llmMode,
		llmModel:       llmModel,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "llmMode": state.llmMode, "llmModel": state.llmModel})
	})

	mux.HandleFunc("/api/econ/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		state.mu.Lock()
		resp := state.response()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	})

	mux.HandleFunc("/api/econ/step", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
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
		state.runtime.RunSteps(req.Count)
		resp := state.response()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	})

	mux.HandleFunc("/api/econ/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		state.mu.Lock()
		provider := agents.FallbackProvider(state.config, state.llmProvider)
		runtime, err := world.NewRuntime(state.config, provider, state.decisionWindow)
		if err == nil {
			state.runtime.Close()
			state.runtime = runtime
		}
		state.mu.Unlock()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		state.mu.Lock()
		resp := state.response()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	})

	mux.HandleFunc("/api/econ/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		var cfg world.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, 400, fmt.Sprintf("bad config: %v", err))
			return
		}
		state.mu.Lock()
		provider := agents.FallbackProvider(cfg, state.llmProvider)
		runtime, err := world.NewRuntime(cfg, provider, state.decisionWindow)
		if err == nil {
			state.runtime.Close()
			state.config = cfg
			state.runtime = runtime
		}
		state.mu.Unlock()
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		resp := state.response()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: corsMiddleware(mux),
	}

	go func() {
		log.Printf("omega server listening on http://localhost:%d", *port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	srv.Shutdown(context.Background())
}

func (s *serverState) response() stateResponse {
	return stateResponse{
		Config:   s.config,
		Stats:    s.runtime.Stats(),
		World:    s.runtime.Snapshot(),
		LLMMode:  s.llmMode,
		LLMModel: s.llmModel,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
