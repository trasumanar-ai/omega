package observer

import (
	"encoding/json"
	"fmt"
	"omega/backend/internal/store"
	"time"
)

type Observer struct {
	db *store.DB
}

func New(db *store.DB) *Observer {
	return &Observer{db: db}
}

// Event represents a single observable action in the government.
type Event struct {
	ID        int64  `json:"id"`
	GovID     string `json:"government_id"`
	Timestamp string `json:"timestamp"`
	Actor     string `json:"actor"`     // agent ID or "system"
	Action    string `json:"action"`    // e.g. "transfer", "register", "create_contract"
	Target    string `json:"target"`    // target agent/entity ID
	Details   string `json:"details"`   // JSON blob with full context
	Result    string `json:"result"`    // "ok" or error message
}

// Snapshot captures the full state of a government at a point in time.
type Snapshot struct {
	ID           int64   `json:"id"`
	GovID        string  `json:"government_id"`
	Timestamp    string  `json:"timestamp"`
	CitizenCount int     `json:"citizen_count"`
	MoneySupply  float64 `json:"money_supply"`
	TreasuryBal  float64 `json:"treasury_balance"`
	Gini         float64 `json:"gini"`
	TxCount      int     `json:"tx_count"`      // total ledger entries since last snapshot
	TxVolume     float64 `json:"tx_volume"`     // total transfer volume since last snapshot
	ContractsCrt int     `json:"contracts_created"`
	ContractsAct int     `json:"contracts_active"`
	FirmCount    int     `json:"firm_count"`
	Balances     string  `json:"balances"` // JSON: map[agentID]balance
}

// Report is a full experiment report for a government.
type Report struct {
	Government   GovSummary      `json:"government"`
	Timeline     []Snapshot      `json:"timeline"`
	Events       []Event         `json:"events"`
	Citizens     []CitizenReport `json:"citizens"`
	ActiveContracts int          `json:"active_contracts"`
	TotalTx      int             `json:"total_transactions"`
	Duration     string          `json:"duration"`
}

type GovSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Config    string `json:"config"`
	CreatedAt string `json:"created_at"`
}

type CitizenReport struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Balance    float64 `json:"balance"`
	TxSent     int     `json:"tx_sent"`
	TxReceived int     `json:"tx_received"`
	VolumeSent float64 `json:"volume_sent"`
	VolumeRecv float64 `json:"volume_received"`
	Contracts  int     `json:"contracts"`
	JoinedAt   string  `json:"joined_at"`
}

// LogEvent records an observable event.
func (o *Observer) LogEvent(govID, actor, action, target string, details any, result string) {
	detailsJSON, _ := json.Marshal(details)
	o.db.Exec(
		`INSERT INTO events (government_id, actor, action, target, details, result)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		govID, actor, action, target, string(detailsJSON), result,
	)
}

// TakeSnapshot captures current government state.
func (o *Observer) TakeSnapshot(govID string) error {
	var citizenCount int
	o.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE government_id = ?`, govID).Scan(&citizenCount)

	var moneySupply float64
	o.db.QueryRow(
		`SELECT COALESCE(SUM(balance), 0) FROM accounts WHERE government_id = ? AND owner_id != 'treasury'`,
		govID,
	).Scan(&moneySupply)

	var treasuryBal float64
	o.db.QueryRow(
		`SELECT COALESCE(balance, 0) FROM accounts WHERE government_id = ? AND owner_id = 'treasury'`,
		govID,
	).Scan(&treasuryBal)

	gini := calcGini(o.db, govID)

	// Tx stats since last snapshot
	var lastSnapshotTime string
	err := o.db.QueryRow(
		`SELECT COALESCE(MAX(timestamp), '1970-01-01') FROM snapshots WHERE government_id = ?`, govID,
	).Scan(&lastSnapshotTime)
	if err != nil {
		lastSnapshotTime = "1970-01-01"
	}

	var txCount int
	var txVolume float64
	o.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM ledger
		 WHERE government_id = ? AND type = 'transfer' AND created_at > ?`,
		govID, lastSnapshotTime,
	).Scan(&txCount, &txVolume)

	var contractsCreated int
	o.db.QueryRow(
		`SELECT COUNT(*) FROM contracts WHERE government_id = ? AND created_at > ?`,
		govID, lastSnapshotTime,
	).Scan(&contractsCreated)

	var contractsActive int
	o.db.QueryRow(
		`SELECT COUNT(*) FROM contracts WHERE government_id = ? AND state = 'active'`,
		govID,
	).Scan(&contractsActive)

	var firmCount int
	o.db.QueryRow(`SELECT COUNT(*) FROM firms WHERE government_id = ?`, govID).Scan(&firmCount)

	// Per-agent balances
	rows, err := o.db.Query(
		`SELECT owner_id, balance FROM accounts WHERE government_id = ? AND owner_id != 'treasury'`,
		govID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	balances := map[string]float64{}
	for rows.Next() {
		var id string
		var bal float64
		rows.Scan(&id, &bal)
		balances[id] = bal
	}
	balancesJSON, _ := json.Marshal(balances)

	_, err = o.db.Exec(
		`INSERT INTO snapshots (government_id, citizen_count, money_supply, treasury_balance,
		 gini, tx_count, tx_volume, contracts_created, contracts_active, firm_count, balances)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		govID, citizenCount, moneySupply, treasuryBal, gini,
		txCount, txVolume, contractsCreated, contractsActive, firmCount, string(balancesJSON),
	)
	return err
}

// GetReport generates a full experiment report for a government.
func (o *Observer) GetReport(govID string, eventLimit int) (*Report, error) {
	if eventLimit <= 0 {
		eventLimit = 500
	}

	// Government summary
	var gs GovSummary
	err := o.db.QueryRow(
		`SELECT id, name, config, created_at FROM governments WHERE id = ?`, govID,
	).Scan(&gs.ID, &gs.Name, &gs.Config, &gs.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("government %s not found", govID)
	}

	// Timeline (snapshots)
	snapRows, err := o.db.Query(
		`SELECT id, government_id, timestamp, citizen_count, money_supply, treasury_balance,
		 gini, tx_count, tx_volume, contracts_created, contracts_active, firm_count, balances
		 FROM snapshots WHERE government_id = ? ORDER BY timestamp`, govID,
	)
	if err != nil {
		return nil, err
	}
	defer snapRows.Close()

	var timeline []Snapshot
	for snapRows.Next() {
		var s Snapshot
		if err := snapRows.Scan(&s.ID, &s.GovID, &s.Timestamp, &s.CitizenCount,
			&s.MoneySupply, &s.TreasuryBal, &s.Gini, &s.TxCount, &s.TxVolume,
			&s.ContractsCrt, &s.ContractsAct, &s.FirmCount, &s.Balances); err != nil {
			return nil, err
		}
		timeline = append(timeline, s)
	}

	// Events
	eventRows, err := o.db.Query(
		`SELECT id, government_id, timestamp, actor, action, target, details, result
		 FROM events WHERE government_id = ? ORDER BY id DESC LIMIT ?`, govID, eventLimit,
	)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()

	var events []Event
	for eventRows.Next() {
		var e Event
		if err := eventRows.Scan(&e.ID, &e.GovID, &e.Timestamp, &e.Actor,
			&e.Action, &e.Target, &e.Details, &e.Result); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	// Per-citizen report
	citizenRows, err := o.db.Query(
		`SELECT a.id, a.name, COALESCE(ac.balance, 0), a.created_at
		 FROM agents a
		 LEFT JOIN accounts ac ON ac.government_id = a.government_id AND ac.owner_id = a.id
		 WHERE a.government_id = ?`, govID,
	)
	if err != nil {
		return nil, err
	}
	defer citizenRows.Close()

	var citizens []CitizenReport
	for citizenRows.Next() {
		var c CitizenReport
		if err := citizenRows.Scan(&c.ID, &c.Name, &c.Balance, &c.JoinedAt); err != nil {
			return nil, err
		}

		// Tx stats per citizen
		o.db.QueryRow(
			`SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM ledger
			 WHERE government_id = ? AND from_id = ? AND type = 'transfer'`,
			govID, c.ID,
		).Scan(&c.TxSent, &c.VolumeSent)

		o.db.QueryRow(
			`SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM ledger
			 WHERE government_id = ? AND to_id = ? AND type = 'transfer'`,
			govID, c.ID,
		).Scan(&c.TxReceived, &c.VolumeRecv)

		o.db.QueryRow(
			`SELECT COUNT(*) FROM contract_parties cp
			 JOIN contracts ct ON cp.contract_id = ct.id
			 WHERE ct.government_id = ? AND cp.agent_id = ?`,
			govID, c.ID,
		).Scan(&c.Contracts)

		citizens = append(citizens, c)
	}

	// Totals
	var totalTx int
	o.db.QueryRow(`SELECT COUNT(*) FROM ledger WHERE government_id = ?`, govID).Scan(&totalTx)

	var activeContracts int
	o.db.QueryRow(
		`SELECT COUNT(*) FROM contracts WHERE government_id = ? AND state = 'active'`, govID,
	).Scan(&activeContracts)

	// Duration
	duration := ""
	if gs.CreatedAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", gs.CreatedAt); err == nil {
			duration = time.Since(t).Round(time.Second).String()
		}
	}

	return &Report{
		Government:      gs,
		Timeline:        timeline,
		Events:          events,
		Citizens:        citizens,
		ActiveContracts: activeContracts,
		TotalTx:         totalTx,
		Duration:        duration,
	}, nil
}

// GetEvents returns filtered events.
func (o *Observer) GetEvents(govID string, actor string, action string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, government_id, timestamp, actor, action, target, details, result
	          FROM events WHERE government_id = ?`
	args := []any{govID}

	if actor != "" {
		query += ` AND actor = ?`
		args = append(args, actor)
	}
	if action != "" {
		query += ` AND action = ?`
		args = append(args, action)
	}

	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := o.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.GovID, &e.Timestamp, &e.Actor,
			&e.Action, &e.Target, &e.Details, &e.Result); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func calcGini(db *store.DB, govID string) float64 {
	rows, err := db.Query(
		`SELECT balance FROM accounts WHERE government_id = ? AND owner_id != 'treasury' ORDER BY balance`,
		govID,
	)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var balances []float64
	for rows.Next() {
		var b float64
		rows.Scan(&b)
		balances = append(balances, b)
	}

	n := len(balances)
	if n == 0 {
		return 0
	}

	var sumDiff, sumTotal float64
	for _, bi := range balances {
		for _, bj := range balances {
			if bi > bj {
				sumDiff += bi - bj
			} else {
				sumDiff += bj - bi
			}
		}
		sumTotal += bi
	}
	if sumTotal == 0 {
		return 0
	}
	return sumDiff / (2 * float64(n) * sumTotal)
}
