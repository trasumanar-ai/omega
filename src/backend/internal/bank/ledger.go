package bank

import "omega/backend/internal/store"

type TxType string

const (
	TxMint     TxType = "mint"
	TxTax      TxType = "tax"
	TxTransfer TxType = "transfer"
	TxFee      TxType = "fee"
	TxInterest TxType = "interest"
)

type LedgerEntry struct {
	ID        int64  `json:"id"`
	GovID     string `json:"government_id"`
	Type      TxType `json:"type"`
	FromID    string `json:"from_id"`
	ToID      string `json:"to_id"`
	Amount    float64 `json:"amount"`
	Memo      string `json:"memo"`
	CreatedAt string `json:"created_at"`
}

func recordTx(db *store.DB, govID string, txType TxType, from, to string, amount float64, memo string) error {
	_, err := db.Exec(
		`INSERT INTO ledger (government_id, type, from_id, to_id, amount, memo) VALUES (?, ?, ?, ?, ?, ?)`,
		govID, string(txType), from, to, amount, memo,
	)
	return err
}

func queryLedger(db *store.DB, govID string, limit int) ([]LedgerEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(
		`SELECT id, government_id, type, COALESCE(from_id,''), COALESCE(to_id,''), amount, memo, created_at
		 FROM ledger WHERE government_id = ? ORDER BY id DESC LIMIT ?`, govID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.ID, &e.GovID, &e.Type, &e.FromID, &e.ToID, &e.Amount, &e.Memo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
