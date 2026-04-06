package registry

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"omega/backend/internal/store"
)

type Registry struct {
	db *store.DB
}

func New(db *store.DB) *Registry {
	return &Registry{db: db}
}

func (r *Registry) Name() string { return "registry" }

type Agent struct {
	ID         string `json:"id"`
	GovID      string `json:"government_id"`
	Name       string `json:"name"`
	APIKey     string `json:"api_key,omitempty"` // only returned on registration
	SoulHash   string `json:"soul_hash"`
	Model      string `json:"model"`
	ReferredBy string `json:"referred_by,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type RegisterInput struct {
	GovID      string `json:"government_id"`
	Name       string `json:"name"`
	SoulMD     string `json:"soul_md"`    // raw soul.md content — we hash it
	Model      string `json:"model"`
	ReferredBy string `json:"referred_by"` // optional agent ID
}

// Register creates a new agent identity in a government.
// Returns the agent with API key (only time the key is returned in full).
func (r *Registry) Register(input RegisterInput) (*Agent, error) {
	id, err := genID()
	if err != nil {
		return nil, err
	}
	apiKey, err := genAPIKey()
	if err != nil {
		return nil, err
	}

	soulHash := hashSoul(input.SoulMD)

	// Validate referrer exists if provided
	if input.ReferredBy != "" {
		var exists int
		err := r.db.QueryRow(
			`SELECT COUNT(*) FROM agents WHERE id = ? AND government_id = ?`,
			input.ReferredBy, input.GovID,
		).Scan(&exists)
		if err != nil || exists == 0 {
			return nil, fmt.Errorf("referrer %s not found in this government", input.ReferredBy)
		}
	}

	var refBy *string
	if input.ReferredBy != "" {
		refBy = &input.ReferredBy
	}

	_, err = r.db.Exec(
		`INSERT INTO agents (id, government_id, name, api_key, soul_hash, model, referred_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, input.GovID, input.Name, apiKey, soulHash, input.Model, refBy,
	)
	if err != nil {
		return nil, fmt.Errorf("insert agent: %w", err)
	}

	return &Agent{
		ID:         id,
		GovID:      input.GovID,
		Name:       input.Name,
		APIKey:     apiKey,
		SoulHash:   soulHash,
		Model:      input.Model,
		ReferredBy: input.ReferredBy,
	}, nil
}

// Authenticate returns the agent for a given API key, or error.
func (r *Registry) Authenticate(apiKey string) (*Agent, error) {
	var a Agent
	err := r.db.QueryRow(
		`SELECT id, government_id, name, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE api_key = ?`, apiKey,
	).Scan(&a.ID, &a.GovID, &a.Name, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}
	return &a, nil
}

// Get returns an agent by ID (public info, no API key).
func (r *Registry) Get(govID, agentID string) (*Agent, error) {
	var a Agent
	err := r.db.QueryRow(
		`SELECT id, government_id, name, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE id = ? AND government_id = ?`, agentID, govID,
	).Scan(&a.ID, &a.GovID, &a.Name, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	return &a, nil
}

// List returns all agents in a government (public info).
func (r *Registry) List(govID string) ([]Agent, error) {
	rows, err := r.db.Query(
		`SELECT id, government_id, name, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE government_id = ? ORDER BY created_at`, govID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.GovID, &a.Name, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// CountByGov returns citizen count for a government.
func (r *Registry) CountByGov(govID string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE government_id = ?`, govID).Scan(&n)
	return n, err
}

func hashSoul(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func genID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func genAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ogov_" + hex.EncodeToString(b), nil
}
