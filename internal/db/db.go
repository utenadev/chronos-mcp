package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type MemorySnapshot struct {
	ID                 int64     `json:"id"`
	Content            string    `json:"content"`
	Environment        string    `json:"environment"`
	Tags               string    `json:"tags"`
	CreatedAt          time.Time `json:"created_at"`
	ParentID           *int64    `json:"parent_id,omitempty"`
	IsPersonaAnchor    int       `json:"is_persona_anchor"`
	ImportanceScore    float64   `json:"importance_score"`
	CausalityID        string    `json:"causality_id"`
	StatusConsolidated int       `json:"status_consolidated"`
}

type SessionLog struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
}

type ConsolidationMetadata struct {
	LastProcessedTurnID int64     `json:"last_processed_turn_id"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ConversationTurn struct {
	ID              int64     `json:"id"`
	SessionID       string    `json:"session_id"`
	TurnNumber      int       `json:"turn_number"`
	UserMessage     string    `json:"user_message"`
	AssistantReply  string    `json:"assistant_reply"`
	ContextSnapshot string    `json:"context_snapshot,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type ThoughtAnnotation struct {
	ID        int64     `json:"id"`
	TurnID    int64     `json:"turn_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func NewDB(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "chronos.db")
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return db, nil
}

func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memory_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		environment TEXT NOT NULL DEFAULT 'default',
		tags TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		parent_id INTEGER REFERENCES memory_snapshots(id),
		is_persona_anchor INTEGER DEFAULT 0,
		importance_score FLOAT DEFAULT 0.0,
		causality_id TEXT DEFAULT '',
		status_consolidated INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS conversation_turns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		turn_number INTEGER NOT NULL,
		user_message TEXT NOT NULL,
		assistant_reply TEXT NOT NULL,
		context_snapshot TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS thought_annotations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		turn_id INTEGER NOT NULL REFERENCES conversation_turns(id),
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS session_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		summary TEXT
	);

	CREATE TABLE IF NOT EXISTS consolidation_metadata (
		last_processed_turn_id INTEGER PRIMARY KEY,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Initialize metadata if empty
	INSERT OR IGNORE INTO consolidation_metadata (last_processed_turn_id) VALUES (0);

	CREATE INDEX IF NOT EXISTS idx_snapshots_env ON memory_snapshots(environment);
	CREATE INDEX IF NOT EXISTS idx_snapshots_created ON memory_snapshots(created_at);
	CREATE INDEX IF NOT EXISTS idx_turns_session ON conversation_turns(session_id);
	CREATE INDEX IF NOT EXISTS idx_annotations_turn ON thought_annotations(turn_id);
	CREATE INDEX IF NOT EXISTS idx_session_logs_timestamp ON session_logs(timestamp);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Memory Snapshot operations

func (db *DB) CreateSnapshot(ctx context.Context, m *MemorySnapshot) (int64, error) {
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO memory_snapshots (content, environment, tags, parent_id, is_persona_anchor, importance_score, causality_id, status_consolidated) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Content, m.Environment, m.Tags, m.ParentID, m.IsPersonaAnchor, m.ImportanceScore, m.CausalityID, m.StatusConsolidated)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) GetSnapshot(ctx context.Context, id int64) (*MemorySnapshot, error) {
	var m MemorySnapshot
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, content, environment, tags, created_at, parent_id, is_persona_anchor, importance_score, causality_id, status_consolidated 
		 FROM memory_snapshots WHERE id = ?`, id).
		Scan(&m.ID, &m.Content, &m.Environment, &m.Tags, &m.CreatedAt, &m.ParentID, &m.IsPersonaAnchor, &m.ImportanceScore, &m.CausalityID, &m.StatusConsolidated)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (db *DB) ListSnapshots(ctx context.Context, env string, limit int) ([]MemorySnapshot, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, content, environment, tags, created_at, parent_id, is_persona_anchor, importance_score, causality_id, status_consolidated 
		 FROM memory_snapshots WHERE environment = ? ORDER BY created_at DESC LIMIT ?`,
		env, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []MemorySnapshot
	for rows.Next() {
		var m MemorySnapshot
		if err := rows.Scan(&m.ID, &m.Content, &m.Environment, &m.Tags, &m.CreatedAt, &m.ParentID, &m.IsPersonaAnchor, &m.ImportanceScore, &m.CausalityID, &m.StatusConsolidated); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, m)
	}
	return snapshots, nil
}

func (db *DB) GetLatestSnapshot(ctx context.Context, env string) (*MemorySnapshot, error) {
	var m MemorySnapshot
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, content, environment, tags, created_at, parent_id, is_persona_anchor, importance_score, causality_id, status_consolidated 
		 FROM memory_snapshots WHERE environment = ? ORDER BY created_at DESC LIMIT 1`,
		env).Scan(&m.ID, &m.Content, &m.Environment, &m.Tags, &m.CreatedAt, &m.ParentID, &m.IsPersonaAnchor, &m.ImportanceScore, &m.CausalityID, &m.StatusConsolidated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Conversation Turn operations

func (db *DB) CreateTurn(ctx context.Context, t *ConversationTurn) (int64, error) {
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO conversation_turns (session_id, turn_number, user_message, assistant_reply, context_snapshot) 
		 VALUES (?, ?, ?, ?, ?)`,
		t.SessionID, t.TurnNumber, t.UserMessage, t.AssistantReply, t.ContextSnapshot)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) GetTurn(ctx context.Context, id int64) (*ConversationTurn, error) {
	var t ConversationTurn
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, session_id, turn_number, user_message, assistant_reply, context_snapshot, created_at 
		 FROM conversation_turns WHERE id = ?`, id).
		Scan(&t.ID, &t.SessionID, &t.TurnNumber, &t.UserMessage, &t.AssistantReply, &t.ContextSnapshot, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (db *DB) GetSessionTurns(ctx context.Context, sessionID string) ([]ConversationTurn, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, session_id, turn_number, user_message, assistant_reply, context_snapshot, created_at 
		 FROM conversation_turns WHERE session_id = ? ORDER BY turn_number ASC`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []ConversationTurn
	for rows.Next() {
		var t ConversationTurn
		if err := rows.Scan(&t.ID, &t.SessionID, &t.TurnNumber, &t.UserMessage, &t.AssistantReply, &t.ContextSnapshot, &t.CreatedAt); err != nil {
			return nil, err
		}
		turns = append(turns, t)
	}
	return turns, nil
}

func (db *DB) GetTurnCount(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(turn_number), 0) FROM conversation_turns WHERE session_id = ?`,
		sessionID).Scan(&count)
	return count, err
}

// Thought Annotation operations

func (db *DB) CreateAnnotation(ctx context.Context, a *ThoughtAnnotation) (int64, error) {
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO thought_annotations (turn_id, content) VALUES (?, ?)`,
		a.TurnID, a.Content)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Session Log operations

func (db *DB) RecordSessionEvent(ctx context.Context, eventType, summary string) (int64, error) {
	result, err := db.conn.ExecContext(ctx,
		`INSERT INTO session_logs (event_type, summary) VALUES (?, ?)`,
		eventType, summary)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) GetLatestSessionEvent(ctx context.Context) (*SessionLog, error) {
	var s SessionLog
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, event_type, timestamp, summary FROM session_logs ORDER BY timestamp DESC LIMIT 1`).
		Scan(&s.ID, &s.EventType, &s.Timestamp, &s.Summary)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) GetAnnotations(ctx context.Context, turnID int64) ([]ThoughtAnnotation, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, turn_id, content, created_at FROM thought_annotations WHERE turn_id = ?`,
		turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var annotations []ThoughtAnnotation
	for rows.Next() {
		var a ThoughtAnnotation
		if err := rows.Scan(&a.ID, &a.TurnID, &a.Content, &a.CreatedAt); err != nil {
			return nil, err
		}
		annotations = append(annotations, a)
	}
	return annotations, nil
}

// Analysis operations

type ThoughtEvolution struct {
	SessionID     string    `json:"session_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	TurnCount     int       `json:"turn_count"`
	TopicChanges  int       `json:"topic_changes"`
	AvgTurnLength float64   `json:"avg_turn_length"`
}

func (db *DB) AnalyzeSession(ctx context.Context, sessionID string) (*ThoughtEvolution, error) {
	var evolution ThoughtEvolution
	evolution.SessionID = sessionID

	var (
		startTimeStr sql.NullString
		endTimeStr   sql.NullString
		turnCount    int
		avgLength    sql.NullFloat64
	)

	err := db.conn.QueryRowContext(ctx,
		`SELECT MIN(created_at), MAX(created_at), COUNT(*), AVG(LENGTH(user_message) + LENGTH(assistant_reply))
		 FROM conversation_turns WHERE session_id = ?`,
		sessionID).Scan(&startTimeStr, &endTimeStr, &turnCount, &avgLength)
	if err != nil {
		return nil, err
	}

	if turnCount == 0 {
		return nil, nil
	}

	if startTimeStr.Valid {
		evolution.StartTime, _ = time.Parse("2006-01-02 15:04:05", startTimeStr.String)
	}
	if endTimeStr.Valid {
		evolution.EndTime, _ = time.Parse("2006-01-02 15:04:05", endTimeStr.String)
	}
	evolution.TurnCount = turnCount
	evolution.AvgTurnLength = avgLength.Float64

	return &evolution, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Consolidation Metadata operations

func (db *DB) GetConsolidationMetadata(ctx context.Context) (*ConsolidationMetadata, error) {
	var m ConsolidationMetadata
	var updatedAtStr string
	err := db.conn.QueryRowContext(ctx,
		`SELECT last_processed_turn_id, updated_at FROM consolidation_metadata LIMIT 1`).
		Scan(&m.LastProcessedTurnID, &updatedAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	return &m, nil
}

func (db *DB) UpdateConsolidationMetadata(ctx context.Context, lastTurnID int64) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE consolidation_metadata SET last_processed_turn_id = ?, updated_at = CURRENT_TIMESTAMP`,
		lastTurnID)
	return err
}
