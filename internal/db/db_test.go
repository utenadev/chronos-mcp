package db

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func TestDB_SchemaExtension(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-db-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// Check if new columns exist in memory_snapshots
	rows, err := db.conn.Query("PRAGMA table_info(memory_snapshots)")
	if err != nil {
		t.Fatalf("failed to query table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue *string
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		columns[name] = true
	}

	newCols := []string{"is_persona_anchor", "importance_score", "causality_id", "status_consolidated"}
	for _, col := range newCols {
		if !columns[col] {
			t.Errorf("column %s missing in memory_snapshots", col)
		}
	}

	// Check if session_logs table exists
	var tableName string
	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='session_logs'").Scan(&tableName)
	if err != nil {
		t.Errorf("session_logs table missing: %v", err)
	}
}

func TestDB_SessionEvents(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test recording start event
	id1, err := db.RecordSessionEvent(ctx, "start", "Session started")
	if err != nil {
		t.Fatalf("failed to record start event: %v", err)
	}
	if id1 <= 0 {
		t.Errorf("expected positive id, got %d", id1)
	}

	// Test recording end event
	_, err = db.RecordSessionEvent(ctx, "end", "Session ended")
	if err != nil {
		t.Fatalf("failed to record end event: %v", err)
	}

	// Test getting latest event
	latest, err := db.GetLatestSessionEvent(ctx)
	if err != nil {
		t.Fatalf("failed to get latest event: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest event, got nil")
	}
	if latest.EventType != "end" {
		t.Errorf("expected event_type 'end', got %s", latest.EventType)
	}
	if latest.Summary != "Session ended" {
		t.Errorf("expected summary 'Session ended', got %s", latest.Summary)
	}
}

func TestDB_SnapshotExtension(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-snapshot-ext-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	snap := &MemorySnapshot{
		Content:            "Persona Info",
		Environment:        "test",
		IsPersonaAnchor:    1,
		ImportanceScore:    0.95,
		CausalityID:        "cause-123",
		StatusConsolidated: 0,
	}

	id, err := db.CreateSnapshot(ctx, snap)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	retrieved, err := db.GetSnapshot(ctx, id)
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}

	if retrieved.IsPersonaAnchor != 1 {
		t.Errorf("expected IsPersonaAnchor 1, got %d", retrieved.IsPersonaAnchor)
	}
	if retrieved.ImportanceScore != 0.95 {
		t.Errorf("expected ImportanceScore 0.95, got %f", retrieved.ImportanceScore)
	}
	if retrieved.CausalityID != "cause-123" {
		t.Errorf("expected CausalityID 'cause-123', got %s", retrieved.CausalityID)
	}
}

// Table-driven tests for extended field edge cases
func TestDB_SnapshotExtendedFields_EdgeCases(t *testing.T) {
	tests := []struct {
		name               string
		isPersonaAnchor    int
		importanceScore    float64
		causalityID        string
		statusConsolidated int
	}{
		{
			name:               "zero values",
			isPersonaAnchor:    0,
			importanceScore:    0.0,
			causalityID:        "",
			statusConsolidated: 0,
		},
		{
			name:               "max importance score",
			isPersonaAnchor:    1,
			importanceScore:    1.0,
			causalityID:        "max-test",
			statusConsolidated: 1,
		},
		{
			name:               "negative score (edge)",
			isPersonaAnchor:    0,
			importanceScore:    -0.5,
			causalityID:        "negative",
			statusConsolidated: 0,
		},
		{
			name:               "long causality ID",
			isPersonaAnchor:    1,
			importanceScore:    0.5,
			causalityID:        "very-long-causality-id-12345-67890-abcdef-ghijkl",
			statusConsolidated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "chronos-edge-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			db, err := NewDB(tempDir)
			if err != nil {
				t.Fatalf("failed to create db: %v", err)
			}
			defer db.Close()

			ctx := context.Background()

			snap := &MemorySnapshot{
				Content:            "Test content",
				Environment:        "test",
				IsPersonaAnchor:    tt.isPersonaAnchor,
				ImportanceScore:    tt.importanceScore,
				CausalityID:        tt.causalityID,
				StatusConsolidated: tt.statusConsolidated,
			}

			id, err := db.CreateSnapshot(ctx, snap)
			if err != nil {
				t.Fatalf("failed to create snapshot: %v", err)
			}

			retrieved, err := db.GetSnapshot(ctx, id)
			if err != nil {
				t.Fatalf("failed to get snapshot: %v", err)
			}

			if retrieved.IsPersonaAnchor != tt.isPersonaAnchor {
				t.Errorf("IsPersonaAnchor: expected %d, got %d", tt.isPersonaAnchor, retrieved.IsPersonaAnchor)
			}
			if retrieved.ImportanceScore != tt.importanceScore {
				t.Errorf("ImportanceScore: expected %f, got %f", tt.importanceScore, retrieved.ImportanceScore)
			}
			if retrieved.CausalityID != tt.causalityID {
				t.Errorf("CausalityID: expected %s, got %s", tt.causalityID, retrieved.CausalityID)
			}
			if retrieved.StatusConsolidated != tt.statusConsolidated {
				t.Errorf("StatusConsolidated: expected %d, got %d", tt.statusConsolidated, retrieved.StatusConsolidated)
			}
		})
	}
}

// Test NULL parent_id handling
func TestDB_SnapshotWithParent(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-parent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create parent snapshot
	parentSnap := &MemorySnapshot{
		Content:     "Parent snapshot",
		Environment: "test",
	}
	parentID, err := db.CreateSnapshot(ctx, parentSnap)
	if err != nil {
		t.Fatalf("failed to create parent snapshot: %v", err)
	}

	// Create child snapshot with parent
	parentIDPtr := &parentID
	childSnap := &MemorySnapshot{
		Content:     "Child snapshot",
		Environment: "test",
		ParentID:    parentIDPtr,
	}
	childID, err := db.CreateSnapshot(ctx, childSnap)
	if err != nil {
		t.Fatalf("failed to create child snapshot: %v", err)
	}

	retrieved, err := db.GetSnapshot(ctx, childID)
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}

	if retrieved.ParentID == nil {
		t.Fatal("expected ParentID to be non-nil")
	}
	if *retrieved.ParentID != parentID {
		t.Errorf("expected ParentID %d, got %d", parentID, *retrieved.ParentID)
	}

	// Create snapshot without parent
	orphanSnap := &MemorySnapshot{
		Content:     "Orphan snapshot",
		Environment: "test",
	}
	orphanID, err := db.CreateSnapshot(ctx, orphanSnap)
	if err != nil {
		t.Fatalf("failed to create orphan snapshot: %v", err)
	}

	orphanRetrieved, err := db.GetSnapshot(ctx, orphanID)
	if err != nil {
		t.Fatalf("failed to get orphan snapshot: %v", err)
	}

	if orphanRetrieved.ParentID != nil {
		t.Errorf("expected nil ParentID, got %v", *orphanRetrieved.ParentID)
	}
}

// Test ListSnapshots
func TestDB_ListSnapshots(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-list-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		snap := &MemorySnapshot{
			Content:            "Content " + string(rune('A'+i)),
			Environment:        "test-env",
			IsPersonaAnchor:    i % 2,
			ImportanceScore:    float64(i) * 0.2,
			CausalityID:        "cause-" + string(rune('0'+i)),
			StatusConsolidated: i % 2,
		}
		_, err := db.CreateSnapshot(ctx, snap)
		if err != nil {
			t.Fatalf("failed to create snapshot %d: %v", i, err)
		}
	}

	snapshots, err := db.ListSnapshots(ctx, "test-env", 10)
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(snapshots))
	}

	// Just verify extended fields exist in returned data
	for _, snap := range snapshots {
		_ = snap.IsPersonaAnchor
		_ = snap.ImportanceScore
		_ = snap.CausalityID
		_ = snap.StatusConsolidated
	}
}

// Test GetLatestSnapshot with extended fields
func TestDB_GetLatestSnapshot_WithExtendedFields(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-latest-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create snapshot with extended fields
	snap := &MemorySnapshot{
		Content:            "Latest snapshot",
		Environment:        "latest-env",
		IsPersonaAnchor:    1,
		ImportanceScore:    0.99,
		CausalityID:        "latest-cause",
		StatusConsolidated: 1,
	}
	_, err = db.CreateSnapshot(ctx, snap)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	latest, err := db.GetLatestSnapshot(ctx, "latest-env")
	if err != nil {
		t.Fatalf("failed to get latest snapshot: %v", err)
	}

	if latest == nil {
		t.Fatal("expected latest snapshot, got nil")
	}

	if latest.IsPersonaAnchor != 1 {
		t.Errorf("expected IsPersonaAnchor 1, got %d", latest.IsPersonaAnchor)
	}
	if latest.ImportanceScore != 0.99 {
		t.Errorf("expected ImportanceScore 0.99, got %f", latest.ImportanceScore)
	}
	if latest.CausalityID != "latest-cause" {
		t.Errorf("expected CausalityID 'latest-cause', got %s", latest.CausalityID)
	}
}

// Test empty environment
func TestDB_GetLatestSnapshot_EmptyEnvironment(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-empty-env-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// No snapshots in this environment
	latest, err := db.GetLatestSnapshot(ctx, "nonexistent-env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if latest != nil {
		t.Errorf("expected nil for nonexistent environment, got %v", latest)
	}
}

// Test SessionLogs edge cases
func TestDB_SessionEvents_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		summary   string
	}{
		{
			name:      "empty summary",
			eventType: "start",
			summary:   "",
		},
		{
			name:      "long summary",
			eventType: "end",
			summary:   "This is a very long summary that contains a lot of information about the session and what happened during it",
		},
		{
			name:      "special characters in summary",
			eventType: "start",
			summary:   "Session with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "chronos-session-edge-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			db, err := NewDB(tempDir)
			if err != nil {
				t.Fatalf("failed to create db: %v", err)
			}
			defer db.Close()

			ctx := context.Background()

			id, err := db.RecordSessionEvent(ctx, tt.eventType, tt.summary)
			if err != nil {
				t.Fatalf("failed to record event: %v", err)
			}

			if id <= 0 {
				t.Errorf("expected positive id, got %d", id)
			}

			latest, err := db.GetLatestSessionEvent(ctx)
			if err != nil {
				t.Fatalf("failed to get latest event: %v", err)
			}

			if latest.Summary != tt.summary {
				t.Errorf("expected summary '%s', got '%s'", tt.summary, latest.Summary)
			}
		})
	}
}

// Test GetLatestSessionEvent with no events
func TestDB_GetLatestSessionEvent_NoEvents(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-no-events-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	latest, err := db.GetLatestSessionEvent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if latest != nil {
		t.Errorf("expected nil when no events exist, got %v", latest)
	}
}

// Test GetSnapshot with non-existent ID
func TestDB_GetSnapshot_NonExistent(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-nonexist-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.GetSnapshot(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent snapshot, got nil")
	}
}

// Test CreateTurn and GetTurn
func TestDB_CreateTurnAndGetTurn(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-turn-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	turn := &ConversationTurn{
		SessionID:       "test-session-1",
		TurnNumber:      1,
		UserMessage:     "Hello",
		AssistantReply:  "Hi there",
		ContextSnapshot: "context-1",
	}

	id, err := db.CreateTurn(ctx, turn)
	if err != nil {
		t.Fatalf("failed to create turn: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	retrieved, err := db.GetTurn(ctx, id)
	if err != nil {
		t.Fatalf("failed to get turn: %v", err)
	}

	if retrieved.SessionID != "test-session-1" {
		t.Errorf("expected SessionID 'test-session-1', got %s", retrieved.SessionID)
	}
	if retrieved.TurnNumber != 1 {
		t.Errorf("expected TurnNumber 1, got %d", retrieved.TurnNumber)
	}
	if retrieved.UserMessage != "Hello" {
		t.Errorf("expected UserMessage 'Hello', got %s", retrieved.UserMessage)
	}
}

// Test GetSessionTurns
func TestDB_GetSessionTurns(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-session-turns-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create multiple turns for same session
	for i := 1; i <= 3; i++ {
		turn := &ConversationTurn{
			SessionID:       "session-multi",
			TurnNumber:      i,
			UserMessage:     "User message " + string(rune('0'+i)),
			AssistantReply:  "Assistant reply " + string(rune('0'+i)),
			ContextSnapshot: "",
		}
		_, err := db.CreateTurn(ctx, turn)
		if err != nil {
			t.Fatalf("failed to create turn %d: %v", i, err)
		}
	}

	turns, err := db.GetSessionTurns(ctx, "session-multi")
	if err != nil {
		t.Fatalf("failed to get session turns: %v", err)
	}

	if len(turns) != 3 {
		t.Errorf("expected 3 turns, got %d", len(turns))
	}

	// Verify order (should be by turn_number ASC)
	for i, turn := range turns {
		if turn.TurnNumber != i+1 {
			t.Errorf("turn %d: expected TurnNumber %d, got %d", i, i+1, turn.TurnNumber)
		}
	}

	// Test empty session
	emptyTurns, err := db.GetSessionTurns(ctx, "nonexistent-session")
	if err != nil {
		t.Fatalf("failed to get empty session turns: %v", err)
	}
	if len(emptyTurns) != 0 {
		t.Errorf("expected 0 turns for nonexistent session, got %d", len(emptyTurns))
	}
}

// Test GetTurnCount
func TestDB_GetTurnCount(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-turn-count-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// No turns yet
	count, err := db.GetTurnCount(ctx, "new-session")
	if err != nil {
		t.Fatalf("failed to get turn count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Create turns
	for i := 1; i <= 5; i++ {
		turn := &ConversationTurn{
			SessionID:      "new-session",
			TurnNumber:       i,
			UserMessage:     "msg",
			AssistantReply:  "reply",
		}
		_, err := db.CreateTurn(ctx, turn)
		if err != nil {
			t.Fatalf("failed to create turn: %v", err)
		}
	}

	count, err = db.GetTurnCount(ctx, "new-session")
	if err != nil {
		t.Fatalf("failed to get turn count: %v", err)
	}
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

// Test CreateAnnotation and GetAnnotations
func TestDB_CreateAnnotationAndGetAnnotations(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-annotation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// First create a turn
	turn := &ConversationTurn{
		SessionID:      "annotate-session",
		TurnNumber:     1,
		UserMessage:    "User",
		AssistantReply: "Assistant",
	}
	turnID, err := db.CreateTurn(ctx, turn)
	if err != nil {
		t.Fatalf("failed to create turn: %v", err)
	}

	// Create annotations
	for i := 1; i <= 3; i++ {
		annotation := &ThoughtAnnotation{
			TurnID:  turnID,
			Content: "Annotation " + string(rune('0'+i)),
		}
		_, err := db.CreateAnnotation(ctx, annotation)
		if err != nil {
			t.Fatalf("failed to create annotation %d: %v", i, err)
		}
	}

	// Get annotations
	annotations, err := db.GetAnnotations(ctx, turnID)
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}

	if len(annotations) != 3 {
		t.Errorf("expected 3 annotations, got %d", len(annotations))
	}

	// Test no annotations
	emptyAnnotations, err := db.GetAnnotations(ctx, 99999)
	if err != nil {
		t.Fatalf("failed to get empty annotations: %v", err)
	}
	if len(emptyAnnotations) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(emptyAnnotations))
	}
}

// Test Consolidation Metadata
func TestDB_ConsolidationMetadata(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-meta-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Initial metadata (should be 0)
	meta, err := db.GetConsolidationMetadata(ctx)
	if err != nil {
		t.Fatalf("GetConsolidationMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatal("expected initial metadata, got nil")
	}
	if meta.LastProcessedTurnID != 0 {
		t.Errorf("expected LastProcessedTurnID 0, got %d", meta.LastProcessedTurnID)
	}

	// 2. Update metadata
	err = db.UpdateConsolidationMetadata(ctx, 42)
	if err != nil {
		t.Fatalf("UpdateConsolidationMetadata failed: %v", err)
	}

	// 3. Verify update
	meta, err = db.GetConsolidationMetadata(ctx)
	if err != nil {
		t.Fatalf("GetConsolidationMetadata after update failed: %v", err)
	}
	if meta.LastProcessedTurnID != 42 {
		t.Errorf("expected LastProcessedTurnID 42, got %d", meta.LastProcessedTurnID)
	}
}

// Test AnalyzeSession
func TestDB_AnalyzeSession(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-analyze-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Test empty data
	analysis, err := db.AnalyzeSession(ctx, "empty-session")
	if err != nil {
		t.Fatalf("AnalyzeSession failed for empty data: %v", err)
	}
	if analysis != nil {
		t.Errorf("expected nil analysis for empty session, got %v", analysis)
	}

	// 2. Test normal data
	sessionID := "active-session"
	for i := 1; i <= 3; i++ {
		turn := &ConversationTurn{
			SessionID:      sessionID,
			TurnNumber:     i,
			UserMessage:    "Short",     // 5 chars
			AssistantReply: "Very long", // 9 chars
		}
		_, err := db.CreateTurn(ctx, turn)
		if err != nil {
			t.Fatalf("failed to create turn %d: %v", i, err)
		}
	}

	analysis, err = db.AnalyzeSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("AnalyzeSession failed for normal data: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis for active session, got nil")
	}

	if analysis.TurnCount != 3 {
		t.Errorf("expected TurnCount 3, got %d", analysis.TurnCount)
	}
	expectedAvg := 14.0 // (5+9) * 3 / 3
	if analysis.AvgTurnLength != expectedAvg {
		t.Errorf("expected AvgTurnLength %f, got %f", expectedAvg, analysis.AvgTurnLength)
	}
}
