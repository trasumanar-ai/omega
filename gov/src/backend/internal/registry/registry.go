package registry

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"omega/backend/internal/store"
	"strconv"
	"strings"
	"time"
)

type Registry struct {
	db *store.DB
}

func New(db *store.DB) *Registry {
	return &Registry{db: db}
}

func (r *Registry) Name() string { return "registry" }

type Agent struct {
	ID        string `json:"id"`
	GovID     string `json:"government_id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	SoulHash  string `json:"soul_hash"`
	Model     string `json:"model"`
	ReferredBy string `json:"referred_by,omitempty"`
	CreatedAt string `json:"created_at"`
}

type RegisterInput struct {
	GovID     string `json:"government_id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"` // Ed25519 public key, hex encoded
	SoulMD    string `json:"soul_md"`
	Model     string `json:"model"`
	ReferredBy string `json:"referred_by"`
}

// Register creates a new agent identity using their public key.
func (r *Registry) Register(input RegisterInput) (*Agent, error) {
	// Validate public key format
	pubBytes, err := hex.DecodeString(input.PublicKey)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key: must be %d-byte Ed25519 key, hex encoded", ed25519.PublicKeySize)
	}

	// Check for duplicate public key
	var exists int
	r.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE public_key = ?`, input.PublicKey).Scan(&exists)
	if exists > 0 {
		return nil, fmt.Errorf("public key already registered")
	}

	id, err := genID()
	if err != nil {
		return nil, err
	}

	soulHash := hashSoul(input.SoulMD)

	if input.ReferredBy != "" {
		var refExists int
		r.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE id = ? AND government_id = ?`,
			input.ReferredBy, input.GovID).Scan(&refExists)
		if refExists == 0 {
			return nil, fmt.Errorf("referrer %s not found", input.ReferredBy)
		}
	}

	var refBy *string
	if input.ReferredBy != "" {
		refBy = &input.ReferredBy
	}

	_, err = r.db.Exec(
		`INSERT INTO agents (id, government_id, name, public_key, soul_hash, model, referred_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, input.GovID, input.Name, input.PublicKey, soulHash, input.Model, refBy,
	)
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}

	return &Agent{
		ID:        id,
		GovID:     input.GovID,
		Name:      input.Name,
		PublicKey:  input.PublicKey,
		SoulHash:  soulHash,
		Model:     input.Model,
		ReferredBy: input.ReferredBy,
	}, nil
}

// Authenticate verifies a signed request.
// authHeader format: "Signed <pubkey>:<timestamp>:<signature>"
// The signed message is: "<timestamp>:<METHOD>:<path>"
func (r *Registry) Authenticate(authHeader, method, path string) (*Agent, error) {
	if !strings.HasPrefix(authHeader, "Signed ") {
		return nil, fmt.Errorf("invalid auth: expected 'Signed <pubkey>:<ts>:<sig>'")
	}

	parts := strings.SplitN(strings.TrimPrefix(authHeader, "Signed "), ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid auth format")
	}

	pubHex, tsStr, sigHex := parts[0], parts[1], parts[2]

	// Verify timestamp (allow 5 minute window)
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp")
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return nil, fmt.Errorf("request expired (timestamp too old or too far in future)")
	}

	// Verify signature
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key")
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("invalid signature")
	}

	message := tsStr + ":" + strings.ToUpper(method) + ":" + path
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(message), sigBytes) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Look up agent by public key
	var a Agent
	err = r.db.QueryRow(
		`SELECT id, government_id, name, public_key, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE public_key = ?`, pubHex,
	).Scan(&a.ID, &a.GovID, &a.Name, &a.PublicKey, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("unknown public key")
	}

	return &a, nil
}

// Get returns an agent by ID (public info).
func (r *Registry) Get(govID, agentID string) (*Agent, error) {
	var a Agent
	err := r.db.QueryRow(
		`SELECT id, government_id, name, public_key, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE id = ? AND government_id = ?`, agentID, govID,
	).Scan(&a.ID, &a.GovID, &a.Name, &a.PublicKey, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	return &a, nil
}

// List returns all agents in a government.
func (r *Registry) List(govID string) ([]Agent, error) {
	rows, err := r.db.Query(
		`SELECT id, government_id, name, public_key, soul_hash, model, COALESCE(referred_by, ''), created_at
		 FROM agents WHERE government_id = ? ORDER BY created_at`, govID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.GovID, &a.Name, &a.PublicKey, &a.SoulHash, &a.Model, &a.ReferredBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

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
