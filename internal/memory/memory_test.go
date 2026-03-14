package memory

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/kench/chronos-mcp/internal/db"
)

func TestMemoryManager_SnapshotExtension(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-memory-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Create snapshot with extension fields
	content := "Test content"
	env := "test"
	tags := []string{"tag1", "tag2"}
	isPersona := 1
	importance := 0.8
	causality := "root-1"

	id, err := mm.CreateSnapshotExt(ctx, content, env, tags, nil, isPersona, importance, causality)
	if err != nil {
		t.Fatalf("failed to create snapshot ext: %v", err)
	}

	snap, err := mm.GetSnapshot(ctx, id)
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}

	if snap.IsPersonaAnchor != isPersona {
		t.Errorf("expected isPersona %d, got %d", isPersona, snap.IsPersonaAnchor)
	}
	if snap.ImportanceScore != importance {
		t.Errorf("expected importance %f, got %f", importance, snap.ImportanceScore)
	}
	if snap.CausalityID != causality {
		t.Errorf("expected causality %s, got %s", causality, snap.CausalityID)
	}
}

func TestMemoryManager_SessionEvents(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-memory-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	err = mm.RecordSessionEvent(ctx, "start", "Session Started")
	if err != nil {
		t.Fatalf("failed to record session event: %v", err)
	}

	duration, err := mm.GetTimeSinceLastActivity(ctx)
	if err != nil {
		t.Fatalf("failed to get time since last activity: %v", err)
	}
	if duration < 0 {
		t.Errorf("expected non-negative duration, got %v", duration)
	}
}

// Test CreateSnapshot and GetSnapshot
func TestMemoryManager_CreateSnapshot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-snapshot-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Create snapshot
	id, err := mm.CreateSnapshot(ctx, "Test content", "test-env", []string{"tag1", "tag2"}, nil)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	// Get snapshot
	snap, err := mm.GetSnapshot(ctx, id)
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}

	if snap.Content != "Test content" {
		t.Errorf("expected content 'Test content', got %s", snap.Content)
	}
	if snap.Environment != "test-env" {
		t.Errorf("expected environment 'test-env', got %s", snap.Environment)
	}
	if len(snap.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(snap.Tags))
	}
}

// Test ListSnapshots
func TestMemoryManager_ListSnapshots(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-list-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Create multiple snapshots
	for i := 0; i < 5; i++ {
		_, err := mm.CreateSnapshot(ctx, "Content "+string(rune('A'+i)), "test-env", []string{"test"}, nil)
		if err != nil {
			t.Fatalf("failed to create snapshot %d: %v", i, err)
		}
	}

	snapshots, err := mm.ListSnapshots(ctx, "test-env", 10)
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}

	if len(snapshots) != 5 {
		t.Errorf("expected 5 snapshots, got %d", len(snapshots))
	}

	// Test limit
	limited, err := mm.ListSnapshots(ctx, "test-env", 3)
	if err != nil {
		t.Fatalf("failed to list limited snapshots: %v", err)
	}
	if len(limited) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(limited))
	}

	// Test empty environment
	empty, err := mm.ListSnapshots(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("failed to list empty snapshots: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(empty))
	}
}

// Test CheckoutSnapshot
func TestMemoryManager_CheckoutSnapshot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-checkout-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// No snapshots yet
	snap, err := mm.CheckoutSnapshot(ctx, "test-env")
	if err != nil {
		t.Fatalf("failed to checkout snapshot: %v", err)
	}
	if snap != nil {
		t.Errorf("expected nil for empty env, got %v", snap)
	}

	// Create snapshot
	_, err = mm.CreateSnapshot(ctx, "Latest content", "test-env", []string{"tag"}, nil)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// Checkout should return the latest
	snap, err = mm.CheckoutSnapshot(ctx, "test-env")
	if err != nil {
		t.Fatalf("failed to checkout snapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snap.Content != "Latest content" {
		t.Errorf("expected content 'Latest content', got %s", snap.Content)
	}
}

// Test RecordTurn and GetTurn
func TestMemoryManager_RecordTurn(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-turn-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Record turn
	id, err := mm.RecordTurn(ctx, "session-1", "Hello", "Hi there", "context-1")
	if err != nil {
		t.Fatalf("failed to record turn: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	// Get turn
	turn, err := mm.GetTurn(ctx, id)
	if err != nil {
		t.Fatalf("failed to get turn: %v", err)
	}
	if turn.SessionID != "session-1" {
		t.Errorf("expected sessionID 'session-1', got %s", turn.SessionID)
	}
	if turn.UserMessage != "Hello" {
		t.Errorf("expected user message 'Hello', got %s", turn.UserMessage)
	}
	if turn.AssistantReply != "Hi there" {
		t.Errorf("expected assistant reply 'Hi there', got %s", turn.AssistantReply)
	}
	if turn.TurnNumber != 1 {
		t.Errorf("expected turn number 1, got %d", turn.TurnNumber)
	}
}

// Test GetSessionTurns
func TestMemoryManager_GetSessionTurns(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-session-turns-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Record multiple turns
	for i := 1; i <= 3; i++ {
		_, err := mm.RecordTurn(ctx, "session-multi", "Message", "Reply", "")
		if err != nil {
			t.Fatalf("failed to record turn %d: %v", i, err)
		}
	}

	// Get session turns
	turns, err := mm.GetSessionTurns(ctx, "session-multi")
	if err != nil {
		t.Fatalf("failed to get session turns: %v", err)
	}
	if len(turns) != 3 {
		t.Errorf("expected 3 turns, got %d", len(turns))
	}

	// Verify turn numbers are sequential
	for i, turn := range turns {
		if turn.TurnNumber != i+1 {
			t.Errorf("turn %d: expected turn number %d, got %d", i, i+1, turn.TurnNumber)
		}
	}

	// Empty session
	empty, err := mm.GetSessionTurns(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("failed to get empty turns: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 turns, got %d", len(empty))
	}
}

// Test AddAnnotation and GetAnnotations
func TestMemoryManager_AddAnnotation(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-annotation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// First create a turn
	turnID, err := mm.RecordTurn(ctx, "annotate-session", "User message", "Assistant reply", "")
	if err != nil {
		t.Fatalf("failed to record turn: %v", err)
	}

	// Add annotations
	for i := 1; i <= 3; i++ {
		_, err := mm.AddAnnotation(ctx, turnID, "Annotation content")
		if err != nil {
			t.Fatalf("failed to add annotation %d: %v", i, err)
		}
	}

	// Get annotations
	annotations, err := mm.GetAnnotations(ctx, turnID)
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}
	if len(annotations) != 3 {
		t.Errorf("expected 3 annotations, got %d", len(annotations))
	}

	// No annotations
	empty, err := mm.GetAnnotations(ctx, 99999)
	if err != nil {
		t.Fatalf("failed to get empty annotations: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(empty))
	}
}

// Test PredictNearFuture
func TestMemoryManager_PredictNearFuture(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-predict-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// Test with no turns
	prediction, err := mm.PredictNearFuture(ctx, "empty-session")
	if err != nil {
		t.Fatalf("failed to predict future: %v", err)
	}
	expectedMsg := "Not enough data to predict. Need at least 2 conversation turns."
	if prediction != expectedMsg {
		t.Errorf("expected '%s', got '%s'", expectedMsg, prediction)
	}

	// Create only 1 turn
	_, err = mm.RecordTurn(ctx, "one-turn", "Message", "Reply", "")
	if err != nil {
		t.Fatalf("failed to record turn: %v", err)
	}

	prediction, err = mm.PredictNearFuture(ctx, "one-turn")
	if err != nil {
		t.Fatalf("failed to predict future: %v", err)
	}
	if prediction != expectedMsg {
		t.Errorf("expected '%s', got '%s'", expectedMsg, prediction)
	}

	// Create 2+ turns
	for i := 1; i <= 3; i++ {
		_, err := mm.RecordTurn(ctx, "multi-turn", "User message content", "Assistant reply content", "")
		if err != nil {
			t.Fatalf("failed to record turn %d: %v", i, err)
		}
	}

	prediction, err = mm.PredictNearFuture(ctx, "multi-turn")
	if err != nil {
		t.Fatalf("failed to predict future: %v", err)
	}
	// Prediction should contain "focusing on" or similar
	if len(prediction) == 0 {
		t.Error("expected non-empty prediction")
	}
}

// Test GetTimeSinceLastActivity edge cases
func TestMemoryManager_GetTimeSinceLastActivity_EdgeCases(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-activity-edge-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// No activity yet
	duration, err := mm.GetTimeSinceLastActivity(ctx)
	if err != nil {
		t.Fatalf("failed to get time since last activity: %v", err)
	}
	if duration != 0 {
		t.Errorf("expected 0 duration with no activity, got %v", duration)
	}

	// Record activity
	err = mm.RecordSessionEvent(ctx, "start", "Session started")
	if err != nil {
		t.Fatalf("failed to record session event: %v", err)
	}

	// Should return a small duration
	duration, err = mm.GetTimeSinceLastActivity(ctx)
	if err != nil {
		t.Fatalf("failed to get time since last activity: %v", err)
	}
	if duration < 0 {
		t.Errorf("expected non-negative duration, got %v", duration)
	}
	if duration > time.Minute {
		t.Errorf("expected duration less than 1 minute, got %v", duration)
	}
}

// Test helper functions
func TestMemoryManager_JoinTags(t *testing.T) {
	tests := []struct {
		tags     []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"single"}, "single"},
		{[]string{"tag1", "tag2", "tag3"}, "tag1,tag2,tag3"},
		{[]string{"with space"}, "with space"},
	}

	for _, tt := range tests {
		result := joinTags(tt.tags)
		if result != tt.expected {
			t.Errorf("joinTags(%v) = %s, expected %s", tt.tags, result, tt.expected)
		}
	}
}

func TestMemoryManager_SplitTags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"single", []string{"single"}},
		{"tag1,tag2,tag3", []string{"tag1", "tag2", "tag3"}},
		{" with , spaces ", []string{"with", "spaces"}},
	}

	for _, tt := range tests {
		result := splitTags(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitTags(%s) returned %d items, expected %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, tag := range result {
			if tag != tt.expected[i] {
				t.Errorf("splitTags(%s)[%d] = %s, expected %s", tt.input, i, tag, tt.expected[i])
			}
		}
	}
}

// Test AnalyzeEvolution
func TestMemoryManager_AnalyzeEvolution(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-evolution-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := NewMemoryManager(database)
	ctx := context.Background()

	// 1. Empty data
	evo, err := mm.AnalyzeEvolution(ctx, "empty-session")
	if err != nil {
		t.Fatalf("AnalyzeEvolution failed: %v", err)
	}
	if evo != nil {
		t.Errorf("expected nil for empty data, got %v", evo)
	}

	// 2. Normal data
	sessionID := "evolution-session"
	for i := 1; i <= 2; i++ {
		_, err := mm.RecordTurn(ctx, sessionID, "Short", "Long answer", "")
		if err != nil {
			t.Fatalf("failed to record turn: %v", err)
		}
	}

	evo, err = mm.AnalyzeEvolution(ctx, sessionID)
	if err != nil {
		t.Fatalf("AnalyzeEvolution failed: %v", err)
	}
	if evo == nil {
		t.Fatal("expected evolution data, got nil")
	}
	if evo.TurnCount != 2 {
		t.Errorf("expected TurnCount 2, got %d", evo.TurnCount)
	}
}
