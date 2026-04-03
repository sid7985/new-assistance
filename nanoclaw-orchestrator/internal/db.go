package internal

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	Conn *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	queries := []string{
		// ── Missions ──
		`CREATE TABLE IF NOT EXISTS missions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			goal TEXT,
			status TEXT DEFAULT 'active',
			tokens_used INTEGER DEFAULT 0,
			api_calls INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		);`,

		// ── Persistent Memory (Key-Value) ──
		`CREATE TABLE IF NOT EXISTS memory (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// ── Full Audit Trail (every action the agent takes) ──
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mission_id INTEGER,
			action_type TEXT,
			action_detail TEXT,
			source TEXT DEFAULT 'minimax',
			tokens_used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// ── Budget Configuration ──
		`CREATE TABLE IF NOT EXISTS budget (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			monthly_limit_cents INTEGER DEFAULT 1000,
			current_month_spend_cents INTEGER DEFAULT 0,
			last_reset DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// ── Mem0: Layered Personalized Memory ──
		`CREATE TABLE IF NOT EXISTS mem0_entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT,    -- 'user', 'session', 'agent'
			entity_id TEXT,      -- e.g. telegram_user_123
			fact TEXT,           -- the actual memory/preference
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// ── PageIndex: Vectorless RAG Tree ──
		`CREATE TABLE IF NOT EXISTS pageindex_nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT,
			parent_node_id INTEGER,
			node_id TEXT,
			title TEXT,
			summary TEXT,
			content TEXT,        -- The actual underlying chunk, if any
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// Seed budget row if not exists
		`INSERT OR IGNORE INTO budget (id, monthly_limit_cents) VALUES (1, 1000);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return nil, err
		}
	}

	return &Database{Conn: db}, nil
}

// ═══════════════════════════════════════════
// Memory (Key-Value Store)
// ═══════════════════════════════════════════

func (d *Database) StoreVariable(key, value string) error {
	query := `INSERT OR REPLACE INTO memory (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := d.Conn.Exec(query, key, value)
	return err
}

func (d *Database) GetVariable(key string) (string, error) {
	var value string
	err := d.Conn.QueryRow("SELECT value FROM memory WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// ═══════════════════════════════════════════
// Mission Lifecycle
// ═══════════════════════════════════════════

func (d *Database) CreateMission(goal string) (int64, error) {
	res, err := d.Conn.Exec("INSERT INTO missions (goal, status) VALUES (?, 'active')", goal)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *Database) CompleteMission(id int64) error {
	_, err := d.Conn.Exec("UPDATE missions SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (d *Database) FailMission(id int64, reason string) error {
	_, err := d.Conn.Exec("UPDATE missions SET status = 'failed', completed_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return err
	}
	return d.LogAction(id, "MISSION_FAILED", reason, "system", 0)
}

func (d *Database) GetActiveMission() (int64, string, error) {
	var id int64
	var goal string
	err := d.Conn.QueryRow("SELECT id, goal FROM missions WHERE status = 'active' ORDER BY created_at DESC LIMIT 1").Scan(&id, &goal)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	return id, goal, err
}

// ═══════════════════════════════════════════
// Audit Trail
// ═══════════════════════════════════════════

func (d *Database) LogAction(missionID int64, actionType, detail, source string, tokensUsed int) error {
	_, err := d.Conn.Exec(
		"INSERT INTO audit_log (mission_id, action_type, action_detail, source, tokens_used) VALUES (?, ?, ?, ?, ?)",
		missionID, actionType, detail, source, tokensUsed,
	)
	return err
}

func (d *Database) GetMissionLog(missionID int64) ([]string, error) {
	rows, err := d.Conn.Query(
		"SELECT created_at, action_type, action_detail FROM audit_log WHERE mission_id = ? ORDER BY created_at ASC", missionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var ts, actionType, detail string
		if err := rows.Scan(&ts, &actionType, &detail); err != nil {
			continue
		}
		logs = append(logs, fmt.Sprintf("[%s] %s: %s", ts, actionType, detail))
	}
	return logs, nil
}

// ═══════════════════════════════════════════
// Budget Tracking (Paperclip-inspired)
// ═══════════════════════════════════════════

func (d *Database) RecordSpend(amountCents int) error {
	// Reset monthly counter if we've crossed into a new month
	var lastReset time.Time
	err := d.Conn.QueryRow("SELECT last_reset FROM budget WHERE id = 1").Scan(&lastReset)
	if err == nil {
		now := time.Now()
		if now.Month() != lastReset.Month() || now.Year() != lastReset.Year() {
			d.Conn.Exec("UPDATE budget SET current_month_spend_cents = 0, last_reset = CURRENT_TIMESTAMP WHERE id = 1")
		}
	}

	_, err = d.Conn.Exec("UPDATE budget SET current_month_spend_cents = current_month_spend_cents + ? WHERE id = 1", amountCents)
	return err
}

func (d *Database) GetBudgetStatus() (limitCents int, spentCents int, err error) {
	err = d.Conn.QueryRow("SELECT monthly_limit_cents, current_month_spend_cents FROM budget WHERE id = 1").Scan(&limitCents, &spentCents)
	return
}

func (d *Database) SetMonthlyBudget(limitCents int) error {
	_, err := d.Conn.Exec("UPDATE budget SET monthly_limit_cents = ? WHERE id = 1", limitCents)
	return err
}

func (d *Database) IsBudgetExceeded() bool {
	limit, spent, err := d.GetBudgetStatus()
	if err != nil {
		return false // fail-open on DB error
	}
	return spent >= limit
}

// ═══════════════════════════════════════════
// Mission Token Accounting
// ═══════════════════════════════════════════

func (d *Database) AddMissionTokens(missionID int64, tokens int) error {
	_, err := d.Conn.Exec("UPDATE missions SET tokens_used = tokens_used + ?, api_calls = api_calls + 1 WHERE id = ?", tokens, missionID)
	return err
}

func (d *Database) GetMissionStats(missionID int64) (tokensUsed int, apiCalls int, err error) {
	err = d.Conn.QueryRow("SELECT tokens_used, api_calls FROM missions WHERE id = ?", missionID).Scan(&tokensUsed, &apiCalls)
	return
}

// ═══════════════════════════════════════════
// Mem0: Layered Personalized Memory
// ═══════════════════════════════════════════

func (d *Database) AddEntityMemory(entityType, entityID, fact string) error {
	_, err := d.Conn.Exec("INSERT INTO mem0_entities (entity_type, entity_id, fact) VALUES (?, ?, ?)", entityType, entityID, fact)
	return err
}

func (d *Database) GetEntityMemories(entityType, entityID string) ([]string, error) {
	rows, err := d.Conn.Query("SELECT fact FROM mem0_entities WHERE entity_type = ? AND entity_id = ? ORDER BY created_at ASC", entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []string
	for rows.Next() {
		var fact string
		if err := rows.Scan(&fact); err == nil {
			memories = append(memories, fact)
		}
	}
	return memories, nil
}
