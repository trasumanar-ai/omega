package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS governments (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	config      TEXT NOT NULL DEFAULT '{}',
	created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agents (
	id              TEXT PRIMARY KEY,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	name            TEXT NOT NULL,
	public_key      TEXT NOT NULL UNIQUE,
	soul_hash       TEXT NOT NULL DEFAULT '',
	model           TEXT NOT NULL DEFAULT '',
	referred_by     TEXT DEFAULT NULL REFERENCES agents(id),
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_agents_gov ON agents(government_id);
CREATE INDEX IF NOT EXISTS idx_agents_pubkey ON agents(public_key);

CREATE TABLE IF NOT EXISTS firms (
	id              TEXT PRIMARY KEY,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	name            TEXT NOT NULL,
	owner_id        TEXT NOT NULL REFERENCES agents(id),
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS firm_members (
	firm_id     TEXT NOT NULL REFERENCES firms(id),
	agent_id    TEXT NOT NULL REFERENCES agents(id),
	role        TEXT NOT NULL DEFAULT 'member',
	joined_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (firm_id, agent_id)
);

CREATE TABLE IF NOT EXISTS accounts (
	id              TEXT PRIMARY KEY,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	owner_id        TEXT NOT NULL,
	balance         REAL NOT NULL DEFAULT 0,
	CHECK(balance >= 0)
);
CREATE INDEX IF NOT EXISTS idx_accounts_gov ON accounts(government_id);
CREATE INDEX IF NOT EXISTS idx_accounts_owner ON accounts(owner_id);

CREATE TABLE IF NOT EXISTS ledger (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	type            TEXT NOT NULL,
	from_id         TEXT,
	to_id           TEXT,
	amount          REAL NOT NULL,
	memo            TEXT NOT NULL DEFAULT '',
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ledger_gov ON ledger(government_id);

CREATE TABLE IF NOT EXISTS contracts (
	id              TEXT PRIMARY KEY,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	type            TEXT NOT NULL,
	terms           TEXT NOT NULL,
	state           TEXT NOT NULL DEFAULT 'pending',
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	executed_at     DATETIME
);

CREATE TABLE IF NOT EXISTS contract_parties (
	contract_id TEXT NOT NULL REFERENCES contracts(id),
	agent_id    TEXT NOT NULL REFERENCES agents(id),
	role        TEXT NOT NULL DEFAULT 'party',
	signed      INTEGER NOT NULL DEFAULT 0,
	signed_at   DATETIME,
	PRIMARY KEY (contract_id, agent_id)
);

CREATE TABLE IF NOT EXISTS events (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	government_id   TEXT NOT NULL REFERENCES governments(id),
	timestamp       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	actor           TEXT NOT NULL,
	action          TEXT NOT NULL,
	target          TEXT NOT NULL DEFAULT '',
	details         TEXT NOT NULL DEFAULT '{}',
	result          TEXT NOT NULL DEFAULT 'ok',
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_gov ON events(government_id);
CREATE INDEX IF NOT EXISTS idx_events_actor ON events(government_id, actor);
CREATE INDEX IF NOT EXISTS idx_events_action ON events(government_id, action);

CREATE TABLE IF NOT EXISTS snapshots (
	id                INTEGER PRIMARY KEY AUTOINCREMENT,
	government_id     TEXT NOT NULL REFERENCES governments(id),
	timestamp         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	citizen_count     INTEGER NOT NULL DEFAULT 0,
	money_supply      REAL NOT NULL DEFAULT 0,
	treasury_balance  REAL NOT NULL DEFAULT 0,
	gini              REAL NOT NULL DEFAULT 0,
	tx_count          INTEGER NOT NULL DEFAULT 0,
	tx_volume         REAL NOT NULL DEFAULT 0,
	contracts_created INTEGER NOT NULL DEFAULT 0,
	contracts_active  INTEGER NOT NULL DEFAULT 0,
	firm_count        INTEGER NOT NULL DEFAULT 0,
	balances          TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_snapshots_gov ON snapshots(government_id);
`
