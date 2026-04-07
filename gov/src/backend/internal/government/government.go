package government

import (
	"encoding/json"
	"fmt"
	"omega/backend/internal/store"

	"crypto/rand"
	"encoding/hex"
)

type Config struct {
	// Bank
	InitialBalance float64 `json:"initial_balance"`
	InterestRate   float64 `json:"interest_rate"`   // per tick, applied to all balances
	MintPerTick    float64 `json:"mint_per_tick"`    // UBI: minted to each citizen per tick
	TransferTax    float64 `json:"transfer_tax"`     // fraction taken on transfers (0.0-1.0)

	// Registry
	OpenMembership bool `json:"open_membership"`
	MaxCitizens    int  `json:"max_citizens"` // 0 = unlimited

	// Governor
	GovernorMode string `json:"governor_mode"` // "algorithm" or "agent"
}

type Government struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Config Config `json:"config"`

	db *store.DB
}

func New(db *store.DB, name string, cfg Config) (*Government, error) {
	id, err := genID()
	if err != nil {
		return nil, err
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	_, err = db.Exec(
		`INSERT INTO governments (id, name, config) VALUES (?, ?, ?)`,
		id, name, string(cfgJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert government: %w", err)
	}

	// Create treasury account for the government itself
	_, err = db.Exec(
		`INSERT INTO accounts (id, government_id, owner_id, balance) VALUES (?, ?, ?, ?)`,
		"treasury-"+id, id, "treasury", 0,
	)
	if err != nil {
		return nil, fmt.Errorf("create treasury: %w", err)
	}

	return &Government{ID: id, Name: name, Config: cfg, db: db}, nil
}

func Load(db *store.DB, id string) (*Government, error) {
	var g Government
	var cfgJSON string
	err := db.QueryRow(
		`SELECT id, name, config FROM governments WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &cfgJSON)
	if err != nil {
		return nil, fmt.Errorf("load government %s: %w", id, err)
	}
	if err := json.Unmarshal([]byte(cfgJSON), &g.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	g.db = db
	return &g, nil
}

func List(db *store.DB) ([]*Government, error) {
	rows, err := db.Query(`SELECT id, name, config FROM governments ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var govs []*Government
	for rows.Next() {
		var g Government
		var cfgJSON string
		if err := rows.Scan(&g.ID, &g.Name, &cfgJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(cfgJSON), &g.Config); err != nil {
			return nil, err
		}
		g.db = db
		govs = append(govs, &g)
	}
	return govs, rows.Err()
}

func (g *Government) DB() *store.DB { return g.db }

func genID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
