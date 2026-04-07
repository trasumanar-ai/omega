package bank

import (
	"fmt"
	"omega/backend/internal/store"
)

type Bank struct {
	db *store.DB
}

func New(db *store.DB) *Bank {
	return &Bank{db: db}
}

func (b *Bank) Name() string { return "bank" }

type Account struct {
	ID      string  `json:"id"`
	GovID   string  `json:"government_id"`
	OwnerID string  `json:"owner_id"`
	Balance float64 `json:"balance"`
}

// CreateAccount opens an account for an agent in a government.
func (b *Bank) CreateAccount(govID, ownerID string, initialBalance float64) (*Account, error) {
	id := govID + ":" + ownerID
	_, err := b.db.Exec(
		`INSERT INTO accounts (id, government_id, owner_id, balance) VALUES (?, ?, ?, ?)`,
		id, govID, ownerID, initialBalance,
	)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if initialBalance > 0 {
		if err := recordTx(b.db, govID, TxMint, "treasury", ownerID, initialBalance, "initial balance"); err != nil {
			return nil, err
		}
	}

	return &Account{ID: id, GovID: govID, OwnerID: ownerID, Balance: initialBalance}, nil
}

// Balance returns the balance for an agent.
func (b *Bank) Balance(govID, ownerID string) (float64, error) {
	var balance float64
	err := b.db.QueryRow(
		`SELECT balance FROM accounts WHERE government_id = ? AND owner_id = ?`,
		govID, ownerID,
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("account not found for %s", ownerID)
	}
	return balance, nil
}

// Transfer moves money from one agent to another, applying transfer tax.
// Tax goes to treasury.
func (b *Bank) Transfer(govID, fromID, toID string, amount float64, taxRate float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check sender balance
	var senderBalance float64
	err = tx.QueryRow(
		`SELECT balance FROM accounts WHERE government_id = ? AND owner_id = ?`,
		govID, fromID,
	).Scan(&senderBalance)
	if err != nil {
		return fmt.Errorf("sender account not found")
	}
	if senderBalance < amount {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", senderBalance, amount)
	}

	tax := amount * taxRate
	net := amount - tax

	// Debit sender
	_, err = tx.Exec(
		`UPDATE accounts SET balance = balance - ? WHERE government_id = ? AND owner_id = ?`,
		amount, govID, fromID,
	)
	if err != nil {
		return err
	}

	// Credit receiver
	_, err = tx.Exec(
		`UPDATE accounts SET balance = balance + ? WHERE government_id = ? AND owner_id = ?`,
		net, govID, toID,
	)
	if err != nil {
		return err
	}

	// Credit treasury with tax
	if tax > 0 {
		_, err = tx.Exec(
			`UPDATE accounts SET balance = balance + ? WHERE government_id = ? AND owner_id = 'treasury'`,
			tax, govID,
		)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Record in ledger (outside tx for simplicity — ledger is append-only)
	recordTx(b.db, govID, TxTransfer, fromID, toID, net, "")
	if tax > 0 {
		recordTx(b.db, govID, TxFee, fromID, "treasury", tax, "transfer tax")
	}

	return nil
}

// Mint creates money from nothing into an agent's account.
func (b *Bank) Mint(govID, toID string, amount float64, memo string) error {
	if amount <= 0 {
		return fmt.Errorf("mint amount must be positive")
	}
	_, err := b.db.Exec(
		`UPDATE accounts SET balance = balance + ? WHERE government_id = ? AND owner_id = ?`,
		amount, govID, toID,
	)
	if err != nil {
		return err
	}
	return recordTx(b.db, govID, TxMint, "treasury", toID, amount, memo)
}

// TotalSupply returns the total money in circulation for a government.
func (b *Bank) TotalSupply(govID string) (float64, error) {
	var total float64
	err := b.db.QueryRow(
		`SELECT COALESCE(SUM(balance), 0) FROM accounts WHERE government_id = ? AND owner_id != 'treasury'`,
		govID,
	).Scan(&total)
	return total, err
}

// Ledger returns recent transactions.
func (b *Bank) Ledger(govID string, limit int) ([]LedgerEntry, error) {
	return queryLedger(b.db, govID, limit)
}

// ApplyInterest applies interest rate to all non-treasury accounts.
// Positive rate = money creation (inflationary), negative = destruction (deflationary).
func (b *Bank) ApplyInterest(govID string, rate float64) error {
	if rate == 0 {
		return nil
	}

	rows, err := b.db.Query(
		`SELECT owner_id, balance FROM accounts WHERE government_id = ? AND owner_id != 'treasury' AND balance > 0`,
		govID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type adj struct {
		ownerID string
		amount  float64
	}
	var adjustments []adj
	for rows.Next() {
		var ownerID string
		var balance float64
		if err := rows.Scan(&ownerID, &balance); err != nil {
			return err
		}
		interest := balance * rate
		if interest != 0 {
			adjustments = append(adjustments, adj{ownerID, interest})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, a := range adjustments {
		if a.amount > 0 {
			b.db.Exec(
				`UPDATE accounts SET balance = balance + ? WHERE government_id = ? AND owner_id = ?`,
				a.amount, govID, a.ownerID,
			)
			recordTx(b.db, govID, TxInterest, "treasury", a.ownerID, a.amount, "interest")
		} else {
			// Negative interest: take from account, give to treasury
			abs := -a.amount
			b.db.Exec(
				`UPDATE accounts SET balance = MAX(0, balance - ?) WHERE government_id = ? AND owner_id = ?`,
				abs, govID, a.ownerID,
			)
			b.db.Exec(
				`UPDATE accounts SET balance = balance + ? WHERE government_id = ? AND owner_id = 'treasury'`,
				abs, govID,
			)
			recordTx(b.db, govID, TxInterest, a.ownerID, "treasury", abs, "negative interest")
		}
	}
	return nil
}

// GiniCoefficient calculates wealth inequality (0 = perfect equality, 1 = maximum inequality).
func (b *Bank) GiniCoefficient(govID string) (float64, error) {
	rows, err := b.db.Query(
		`SELECT balance FROM accounts WHERE government_id = ? AND owner_id != 'treasury' ORDER BY balance`,
		govID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var balances []float64
	for rows.Next() {
		var bal float64
		if err := rows.Scan(&bal); err != nil {
			return 0, err
		}
		balances = append(balances, bal)
	}

	n := len(balances)
	if n == 0 {
		return 0, nil
	}

	var sumDiff, sumTotal float64
	for i, bi := range balances {
		for _, bj := range balances {
			if bi > bj {
				sumDiff += bi - bj
			} else {
				sumDiff += bj - bi
			}
		}
		_ = i
		sumTotal += bi
	}

	if sumTotal == 0 {
		return 0, nil
	}
	return sumDiff / (2 * float64(n) * sumTotal), nil
}
