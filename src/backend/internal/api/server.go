package api

import (
	"encoding/json"
	"net/http"
	"omega/backend/internal/bank"
	"omega/backend/internal/contracts"
	"omega/backend/internal/dashboard"
	"omega/backend/internal/government"
	"omega/backend/internal/observer"
	"omega/backend/internal/registry"
	"omega/backend/internal/store"
	"strconv"
)

type Server struct {
	db        *store.DB
	registry  *registry.Registry
	bank      *bank.Bank
	contracts *contracts.Contracts
	observer  *observer.Observer
	mux       *http.ServeMux
}

func NewServer(db *store.DB, reg *registry.Registry, bnk *bank.Bank, cts *contracts.Contracts, obs *observer.Observer) *Server {
	s := &Server{
		db:        db,
		registry:  reg,
		bank:      bnk,
		contracts: cts,
		observer:  obs,
		mux:       http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Dashboard
	s.mux.HandleFunc("GET /", dashboard.Handler())
	s.mux.HandleFunc("GET /dashboard", dashboard.Handler())

	// Health
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Government management
	s.mux.HandleFunc("POST /api/governments", s.handleCreateGovernment)
	s.mux.HandleFunc("GET /api/governments", s.handleListGovernments)
	s.mux.HandleFunc("GET /api/governments/{id}", s.handleGetGovernment)

	// Registry
	s.mux.HandleFunc("POST /api/governments/{id}/registry/register", s.handleRegister)
	s.mux.HandleFunc("GET /api/governments/{id}/registry/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/governments/{id}/registry/me", s.withAuth(s.handleMe))
	s.mux.HandleFunc("POST /api/governments/{id}/registry/firms", s.withAuth(s.handleCreateFirm))
	s.mux.HandleFunc("GET /api/governments/{id}/registry/firms", s.handleListFirms)

	// Bank
	s.mux.HandleFunc("GET /api/governments/{id}/bank/balance", s.withAuth(s.handleBalance))
	s.mux.HandleFunc("POST /api/governments/{id}/bank/transfer", s.withAuth(s.handleTransfer))
	s.mux.HandleFunc("GET /api/governments/{id}/bank/ledger", s.handleLedger)
	s.mux.HandleFunc("GET /api/governments/{id}/bank/supply", s.handleSupply)

	// Contracts
	s.mux.HandleFunc("POST /api/governments/{id}/contracts", s.withAuth(s.handleCreateContract))
	s.mux.HandleFunc("GET /api/governments/{id}/contracts", s.withAuth(s.handleListContracts))
	s.mux.HandleFunc("GET /api/governments/{id}/contracts/{cid}", s.handleGetContract)
	s.mux.HandleFunc("POST /api/governments/{id}/contracts/{cid}/sign", s.withAuth(s.handleSignContract))

	// Observer
	s.mux.HandleFunc("GET /api/governments/{id}/report", s.handleReport)
	s.mux.HandleFunc("GET /api/governments/{id}/events", s.handleEvents)
	s.mux.HandleFunc("POST /api/governments/{id}/snapshot", s.handleTakeSnapshot)
}

// Auth middleware: extracts agent from Bearer token
type authedHandler func(w http.ResponseWriter, r *http.Request, agent *registry.Agent)

func (s *Server) withAuth(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeErr(w, http.StatusUnauthorized, "missing authorization")
			return
		}

		agent, err := s.registry.Authenticate(auth, r.Method, r.URL.Path)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}

		govID := r.PathValue("id")
		if agent.GovID != govID {
			writeErr(w, http.StatusForbidden, "agent not in this government")
			return
		}

		h(w, r, agent)
	}
}

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- Government ---

func (s *Server) handleCreateGovernment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string            `json:"name"`
		Config government.Config `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if req.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	gov, err := government.New(s.db, req.Name, req.Config)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, gov)
}

func (s *Server) handleListGovernments(w http.ResponseWriter, r *http.Request) {
	govs, err := government.List(s.db)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, govs)
}

func (s *Server) handleGetGovernment(w http.ResponseWriter, r *http.Request) {
	gov, err := government.Load(s.db, r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}

	citizenCount, _ := s.registry.CountByGov(gov.ID)
	supply, _ := s.bank.TotalSupply(gov.ID)
	gini, _ := s.bank.GiniCoefficient(gov.ID)

	writeJSON(w, 200, map[string]any{
		"government":    gov,
		"citizen_count": citizenCount,
		"money_supply":  supply,
		"gini":          gini,
	})
}

// --- Registry ---

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	govID := r.PathValue("id")

	// Verify government exists and check membership
	gov, err := government.Load(s.db, govID)
	if err != nil {
		writeErr(w, 404, "government not found")
		return
	}
	if !gov.Config.OpenMembership {
		writeErr(w, 403, "this government does not accept new members")
		return
	}
	if gov.Config.MaxCitizens > 0 {
		count, _ := s.registry.CountByGov(govID)
		if count >= gov.Config.MaxCitizens {
			writeErr(w, 403, "government is full")
			return
		}
	}

	var req struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
		SoulMD    string `json:"soul_md"`
		Model     string `json:"model"`
		ReferredBy string `json:"referred_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if req.PublicKey == "" {
		writeErr(w, 400, "public_key required (Ed25519, hex encoded)")
		return
	}

	input := registry.RegisterInput{
		GovID:     govID,
		Name:      req.Name,
		PublicKey:  req.PublicKey,
		SoulMD:    req.SoulMD,
		Model:     req.Model,
		ReferredBy: req.ReferredBy,
	}

	agent, err := s.registry.Register(input)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}

	_, err = s.bank.CreateAccount(govID, agent.ID, gov.Config.InitialBalance)
	if err != nil {
		writeErr(w, 500, "failed to create bank account: "+err.Error())
		return
	}

	s.observer.LogEvent(govID, agent.ID, "register", "", map[string]any{
		"name": agent.Name, "model": req.Model, "initial_balance": gov.Config.InitialBalance,
		"public_key": req.PublicKey[:16] + "...",
	}, "ok")

	writeJSON(w, 201, agent)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.registry.List(r.PathValue("id"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if agents == nil {
		agents = []registry.Agent{}
	}
	writeJSON(w, 200, agents)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	balance, _ := s.bank.Balance(agent.GovID, agent.ID)
	writeJSON(w, 200, map[string]any{
		"agent":   agent,
		"balance": balance,
	})
}

func (s *Server) handleCreateFirm(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	firm, err := s.registry.CreateFirm(agent.GovID, agent.ID, req.Name)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.observer.LogEvent(agent.GovID, agent.ID, "create_firm", firm.ID,
		map[string]any{"name": req.Name}, "ok")
	writeJSON(w, 201, firm)
}

func (s *Server) handleListFirms(w http.ResponseWriter, r *http.Request) {
	firms, err := s.registry.ListFirms(r.PathValue("id"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if firms == nil {
		firms = []registry.Firm{}
	}
	writeJSON(w, 200, firms)
}

// --- Bank ---

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	balance, err := s.bank.Balance(agent.GovID, agent.ID)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"balance": balance})
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	var req struct {
		ToID   string  `json:"to_id"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if req.ToID == "" || req.Amount <= 0 {
		writeErr(w, 400, "to_id and positive amount required")
		return
	}

	gov, err := government.Load(s.db, agent.GovID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	err = s.bank.Transfer(agent.GovID, agent.ID, req.ToID, req.Amount, gov.Config.TransferTax)
	if err != nil {
		s.observer.LogEvent(agent.GovID, agent.ID, "transfer", req.ToID,
			map[string]any{"amount": req.Amount}, err.Error())
		writeErr(w, 400, err.Error())
		return
	}
	s.observer.LogEvent(agent.GovID, agent.ID, "transfer", req.ToID,
		map[string]any{"amount": req.Amount, "tax": req.Amount * gov.Config.TransferTax}, "ok")
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleLedger(w http.ResponseWriter, r *http.Request) {
	entries, err := s.bank.Ledger(r.PathValue("id"), 100)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if entries == nil {
		entries = []bank.LedgerEntry{}
	}
	writeJSON(w, 200, entries)
}

func (s *Server) handleSupply(w http.ResponseWriter, r *http.Request) {
	govID := r.PathValue("id")
	supply, err := s.bank.TotalSupply(govID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	gini, _ := s.bank.GiniCoefficient(govID)
	citizenCount, _ := s.registry.CountByGov(govID)
	writeJSON(w, 200, map[string]any{
		"total_supply":  supply,
		"gini":          gini,
		"citizen_count": citizenCount,
	})
}

// --- Contracts ---

func (s *Server) handleCreateContract(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	var req struct {
		Type    contracts.ContractType `json:"type"`
		Terms   contracts.Terms        `json:"terms"`
		MyRole  string                 `json:"my_role"`
		Parties []contracts.Party      `json:"parties"` // other parties
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	ct, err := s.contracts.Create(agent.GovID, agent.ID, req.MyRole, req.Type, req.Terms, req.Parties)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.observer.LogEvent(agent.GovID, agent.ID, "create_contract", ct.ID,
		map[string]any{"type": req.Type, "terms": req.Terms, "parties": req.Parties}, "ok")
	writeJSON(w, 201, ct)
}

func (s *Server) handleListContracts(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	cts, err := s.contracts.ListByAgent(agent.GovID, agent.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if cts == nil {
		cts = []contracts.Contract{}
	}
	writeJSON(w, 200, cts)
}

func (s *Server) handleGetContract(w http.ResponseWriter, r *http.Request) {
	ct, err := s.contracts.Get(r.PathValue("id"), r.PathValue("cid"))
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, ct)
}

func (s *Server) handleSignContract(w http.ResponseWriter, r *http.Request, agent *registry.Agent) {
	cid := r.PathValue("cid")
	err := s.contracts.Sign(agent.GovID, cid, agent.ID)
	if err != nil {
		s.observer.LogEvent(agent.GovID, agent.ID, "sign_contract", cid, nil, err.Error())
		writeErr(w, 400, err.Error())
		return
	}
	s.observer.LogEvent(agent.GovID, agent.ID, "sign_contract", cid, nil, "ok")
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- Observer ---

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	govID := r.PathValue("id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("events"))
	if limit <= 0 {
		limit = 500
	}
	report, err := s.observer.GetReport(govID, limit)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, report)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	govID := r.PathValue("id")
	actor := r.URL.Query().Get("actor")
	action := r.URL.Query().Get("action")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := s.observer.GetEvents(govID, actor, action, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if events == nil {
		events = []observer.Event{}
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleTakeSnapshot(w http.ResponseWriter, r *http.Request) {
	govID := r.PathValue("id")
	if err := s.observer.TakeSnapshot(govID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
