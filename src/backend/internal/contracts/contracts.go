package contracts

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"omega/backend/internal/bank"
	"omega/backend/internal/store"
)

type State string

const (
	StatePending  State = "pending"
	StateActive   State = "active"
	StateComplete State = "complete"
	StateBreach   State = "breach"
)

type ContractType string

const (
	// Escrow: both parties lock funds, released when conditions met
	TypeEscrow ContractType = "escrow"
	// Payment: recurring payment from A to B
	TypePayment ContractType = "payment"
)

type Contract struct {
	ID        string       `json:"id"`
	GovID     string       `json:"government_id"`
	Type      ContractType `json:"type"`
	Terms     Terms        `json:"terms"`
	State     State        `json:"state"`
	CreatedAt string       `json:"created_at"`
	Parties   []Party      `json:"parties,omitempty"`
}

type Terms struct {
	Amount      float64 `json:"amount"`
	Interval    int     `json:"interval,omitempty"`    // ticks between payments (for payment contracts)
	Duration    int     `json:"duration,omitempty"`     // total ticks (for payment contracts)
	Description string  `json:"description"`
}

type Party struct {
	ContractID string `json:"contract_id"`
	AgentID    string `json:"agent_id"`
	Role       string `json:"role"` // "payer", "payee", "depositor", etc.
	Signed     bool   `json:"signed"`
	SignedAt   string `json:"signed_at,omitempty"`
}

type Contracts struct {
	db   *store.DB
	bank *bank.Bank
}

func New(db *store.DB, b *bank.Bank) *Contracts {
	return &Contracts{db: db, bank: b}
}

func (c *Contracts) Name() string { return "contracts" }

// Create a new contract. The creator is automatically a party.
func (c *Contracts) Create(govID, creatorID, role string, ctype ContractType, terms Terms, otherParties []Party) (*Contract, error) {
	id, err := genID()
	if err != nil {
		return nil, err
	}

	termsJSON, err := json.Marshal(terms)
	if err != nil {
		return nil, err
	}

	_, err = c.db.Exec(
		`INSERT INTO contracts (id, government_id, type, terms, state) VALUES (?, ?, ?, ?, ?)`,
		id, govID, string(ctype), string(termsJSON), string(StatePending),
	)
	if err != nil {
		return nil, fmt.Errorf("create contract: %w", err)
	}

	// Add creator as signed party
	_, err = c.db.Exec(
		`INSERT INTO contract_parties (contract_id, agent_id, role, signed, signed_at)
		 VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP)`,
		id, creatorID, role,
	)
	if err != nil {
		return nil, err
	}

	// Add other parties (unsigned)
	for _, p := range otherParties {
		_, err = c.db.Exec(
			`INSERT INTO contract_parties (contract_id, agent_id, role, signed) VALUES (?, ?, ?, 0)`,
			id, p.AgentID, p.Role,
		)
		if err != nil {
			return nil, err
		}
	}

	return c.Get(govID, id)
}

// Sign a contract. When all parties have signed, the contract becomes active.
func (c *Contracts) Sign(govID, contractID, agentID string) error {
	// Verify party exists on this contract
	var exists int
	err := c.db.QueryRow(
		`SELECT COUNT(*) FROM contract_parties cp
		 JOIN contracts ct ON cp.contract_id = ct.id
		 WHERE cp.contract_id = ? AND cp.agent_id = ? AND ct.government_id = ?`,
		contractID, agentID, govID,
	).Scan(&exists)
	if err != nil || exists == 0 {
		return fmt.Errorf("not a party to this contract")
	}

	_, err = c.db.Exec(
		`UPDATE contract_parties SET signed = 1, signed_at = CURRENT_TIMESTAMP
		 WHERE contract_id = ? AND agent_id = ?`,
		contractID, agentID,
	)
	if err != nil {
		return err
	}

	// Check if all parties have signed
	var unsigned int
	err = c.db.QueryRow(
		`SELECT COUNT(*) FROM contract_parties WHERE contract_id = ? AND signed = 0`,
		contractID,
	).Scan(&unsigned)
	if err != nil {
		return err
	}

	if unsigned == 0 {
		_, err = c.db.Exec(
			`UPDATE contracts SET state = ? WHERE id = ?`,
			string(StateActive), contractID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns a contract with its parties.
func (c *Contracts) Get(govID, contractID string) (*Contract, error) {
	var ct Contract
	var termsJSON string
	err := c.db.QueryRow(
		`SELECT id, government_id, type, terms, state, created_at
		 FROM contracts WHERE id = ? AND government_id = ?`,
		contractID, govID,
	).Scan(&ct.ID, &ct.GovID, &ct.Type, &termsJSON, &ct.State, &ct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("contract %s not found", contractID)
	}
	if err := json.Unmarshal([]byte(termsJSON), &ct.Terms); err != nil {
		return nil, err
	}

	rows, err := c.db.Query(
		`SELECT contract_id, agent_id, role, signed, COALESCE(signed_at, '')
		 FROM contract_parties WHERE contract_id = ?`, contractID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Party
		if err := rows.Scan(&p.ContractID, &p.AgentID, &p.Role, &p.Signed, &p.SignedAt); err != nil {
			return nil, err
		}
		ct.Parties = append(ct.Parties, p)
	}

	return &ct, rows.Err()
}

// ListByAgent returns all contracts an agent is party to.
func (c *Contracts) ListByAgent(govID, agentID string) ([]Contract, error) {
	rows, err := c.db.Query(
		`SELECT ct.id, ct.government_id, ct.type, ct.terms, ct.state, ct.created_at
		 FROM contracts ct
		 JOIN contract_parties cp ON ct.id = cp.contract_id
		 WHERE ct.government_id = ? AND cp.agent_id = ?
		 ORDER BY ct.created_at DESC`, govID, agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cts []Contract
	for rows.Next() {
		var ct Contract
		var termsJSON string
		if err := rows.Scan(&ct.ID, &ct.GovID, &ct.Type, &termsJSON, &ct.State, &ct.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(termsJSON), &ct.Terms); err != nil {
			return nil, err
		}
		cts = append(cts, ct)
	}
	return cts, rows.Err()
}

// ExecutePayments processes active payment contracts.
// Called by the government tick loop.
func (c *Contracts) ExecutePayments(govID string, taxRate float64) error {
	rows, err := c.db.Query(
		`SELECT ct.id, ct.terms FROM contracts ct
		 WHERE ct.government_id = ? AND ct.type = ? AND ct.state = ?`,
		govID, string(TypePayment), string(StateActive),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type pending struct {
		id    string
		terms Terms
	}
	var toProcess []pending
	for rows.Next() {
		var p pending
		var termsJSON string
		if err := rows.Scan(&p.id, &termsJSON); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(termsJSON), &p.terms); err != nil {
			return err
		}
		toProcess = append(toProcess, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range toProcess {
		parties, err := c.getParties(p.id)
		if err != nil {
			continue
		}

		var payer, payee string
		for _, party := range parties {
			switch party.Role {
			case "payer":
				payer = party.AgentID
			case "payee":
				payee = party.AgentID
			}
		}
		if payer == "" || payee == "" {
			continue
		}

		err = c.bank.Transfer(govID, payer, payee, p.terms.Amount, taxRate)
		if err != nil {
			// Breach: payer can't pay
			c.db.Exec(`UPDATE contracts SET state = ? WHERE id = ?`, string(StateBreach), p.id)
			continue
		}
	}

	return nil
}

func (c *Contracts) getParties(contractID string) ([]Party, error) {
	rows, err := c.db.Query(
		`SELECT contract_id, agent_id, role, signed, COALESCE(signed_at, '')
		 FROM contract_parties WHERE contract_id = ?`, contractID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parties []Party
	for rows.Next() {
		var p Party
		if err := rows.Scan(&p.ContractID, &p.AgentID, &p.Role, &p.Signed, &p.SignedAt); err != nil {
			return nil, err
		}
		parties = append(parties, p)
	}
	return parties, rows.Err()
}

func genID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("ct_%x", b), nil
}
