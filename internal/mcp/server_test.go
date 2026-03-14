package mcp

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/kench/chronos-mcp/internal/db"
	"github.com/kench/chronos-mcp/internal/memory"
)

func TestChronosMCPServer_TimeAwarenessHook(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// 1. Record an event
	_, err = database.RecordSessionEvent(ctx, "start", "Current Session")
	if err != nil {
		t.Fatalf("failed to record event: %v", err)
	}

	// Just test if the tool runs without awareness when time is short
	args := map[string]interface{}{"session_id": "test"}
	result, err := server.HandleTool(ctx, "get_ambient_context", args)
	if err != nil {
		t.Fatalf("tool failed: %v", err)
	}
	if strings.Contains(result.(string), "[Chronos Awareness]") {
		t.Error("expected no awareness for recent activity")
	}
}

func TestChronosMCPServer_RecordSessionEventTool(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-tool-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	args := map[string]interface{}{
		"type":    "start",
		"summary": "New session for testing",
	}
	result, err := server.HandleTool(ctx, "record_session_event", args)
	if err != nil {
		t.Fatalf("failed to handle record_session_event: %v", err)
	}

	resultStr := result.(string)
	if !strings.Contains(resultStr, "recorded") {
		t.Errorf("expected result to contain 'recorded', got %s", resultStr)
	}

	// Verify in DB
	event, _ := database.GetLatestSessionEvent(ctx)
	if event == nil || event.EventType != "start" {
		t.Error("event not correctly recorded in DB")
	}
}

// Test create_snapshot tool
func TestChronosMCPServer_CreateSnapshot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-snapshot-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	args := map[string]interface{}{
		"content":     "Test snapshot content",
		"environment": "test-env",
		"tags":        "tag1,tag2",
	}
	result, err := server.HandleTool(ctx, "create_snapshot", args)
	if err != nil {
		t.Fatalf("failed to handle create_snapshot: %v", err)
	}

	resultStr := result.(string)
	if !strings.Contains(resultStr, "Snapshot created") {
		t.Errorf("expected result to contain 'Snapshot created', got %s", resultStr)
	}

	// Verify snapshot was created
	snapshots, _ := mm.ListSnapshots(ctx, "test-env", 10)
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snapshots))
	}
}

// Test checkout_snapshot tool
func TestChronosMCPServer_CheckoutSnapshot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-checkout-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// No snapshots yet
	args := map[string]interface{}{"environment": "test-env"}
	result, err := server.HandleTool(ctx, "checkout_snapshot", args)
	if err != nil {
		t.Fatalf("failed to handle checkout_snapshot: %v", err)
	}
	if !strings.Contains(result.(string), "No snapshots") {
		t.Errorf("expected 'No snapshots' message, got %s", result)
	}

	// Create snapshot
	mm.CreateSnapshot(ctx, "Test content", "test-env", []string{}, nil)

	result, err = server.HandleTool(ctx, "checkout_snapshot", args)
	if err != nil {
		t.Fatalf("failed to handle checkout_snapshot: %v", err)
	}
	if !strings.Contains(result.(string), "Test content") {
		t.Errorf("expected snapshot content in result, got %s", result)
	}
}

// Test list_snapshots tool
func TestChronosMCPServer_ListSnapshots(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-list-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Empty list
	args := map[string]interface{}{"environment": "test-env"}
	result, err := server.HandleTool(ctx, "list_snapshots", args)
	if err != nil {
		t.Fatalf("failed to handle list_snapshots: %v", err)
	}
	if !strings.Contains(result.(string), "No snapshots") {
		t.Errorf("expected 'No snapshots' message, got %s", result)
	}

	// Create snapshots
	for i := 0; i < 3; i++ {
		mm.CreateSnapshot(ctx, "Content", "test-env", []string{}, nil)
	}

	result, err = server.HandleTool(ctx, "list_snapshots", args)
	if err != nil {
		t.Fatalf("failed to handle list_snapshots: %v", err)
	}
	if !strings.Contains(result.(string), "---") {
		t.Errorf("expected snapshots in result, got %s", result)
	}
}

// Test record_turn tool
func TestChronosMCPServer_RecordTurn(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-turn-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	args := map[string]interface{}{
		"session_id":      "test-session",
		"user_message":    "Hello",
		"assistant_reply": "Hi there",
	}
	result, err := server.HandleTool(ctx, "record_turn", args)
	if err != nil {
		t.Fatalf("failed to handle record_turn: %v", err)
	}
	if !strings.Contains(result.(string), "Turn recorded") {
		t.Errorf("expected 'Turn recorded' message, got %s", result)
	}

	// Verify turn was recorded
	turns, _ := mm.GetSessionTurns(ctx, "test-session")
	if len(turns) != 1 {
		t.Errorf("expected 1 turn, got %d", len(turns))
	}
}

// Test get_session_turns tool
func TestChronosMCPServer_GetSessionTurns(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-session-turns-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Empty session
	args := map[string]interface{}{"session_id": "empty-session"}
	result, err := server.HandleTool(ctx, "get_session_turns", args)
	if err != nil {
		t.Fatalf("failed to handle get_session_turns: %v", err)
	}
	if !strings.Contains(result.(string), "No turns") {
		t.Errorf("expected 'No turns' message, got %s", result)
	}

	// Create turns
	for i := 1; i <= 2; i++ {
		mm.RecordTurn(ctx, "test-session", "User", "Assistant", "")
	}

	args = map[string]interface{}{"session_id": "test-session"}
	result, err = server.HandleTool(ctx, "get_session_turns", args)
	if err != nil {
		t.Fatalf("failed to handle get_session_turns: %v", err)
	}
	if !strings.Contains(result.(string), "Turn #") {
		t.Errorf("expected turns in result, got %s", result)
	}
}

// Test get_turn tool
func TestChronosMCPServer_GetTurn(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-get-turn-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Create turn
	turnID, _ := mm.RecordTurn(ctx, "test-session", "User", "Assistant", "")

	args := map[string]interface{}{"id": float64(turnID)}
	result, err := server.HandleTool(ctx, "get_turn", args)
	if err != nil {
		t.Fatalf("failed to handle get_turn: %v", err)
	}
	if !strings.Contains(result.(string), "User:") {
		t.Errorf("expected turn details in result, got %s", result)
	}
}

// Test add_annotation tool
func TestChronosMCPServer_AddAnnotation(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-annotation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Create turn
	turnID, _ := mm.RecordTurn(ctx, "test-session", "User", "Assistant", "")

	args := map[string]interface{}{
		"turn_id": float64(turnID),
		"content": "This is an annotation",
	}
	result, err := server.HandleTool(ctx, "add_annotation", args)
	if err != nil {
		t.Fatalf("failed to handle add_annotation: %v", err)
	}
	if !strings.Contains(result.(string), "Annotation added") {
		t.Errorf("expected 'Annotation added' message, got %s", result)
	}

	// Verify annotation
	annotations, _ := mm.GetAnnotations(ctx, turnID)
	if len(annotations) != 1 {
		t.Errorf("expected 1 annotation, got %d", len(annotations))
	}
}

// Test get_annotations tool
func TestChronosMCPServer_GetAnnotations(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-get-annotations-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Create turn and annotations
	turnID, _ := mm.RecordTurn(ctx, "test-session", "User", "Assistant", "")
	mm.AddAnnotation(ctx, turnID, "Annotation 1")
	mm.AddAnnotation(ctx, turnID, "Annotation 2")

	args := map[string]interface{}{"turn_id": float64(turnID)}
	result, err := server.HandleTool(ctx, "get_annotations", args)
	if err != nil {
		t.Fatalf("failed to handle get_annotations: %v", err)
	}
	if !strings.Contains(result.(string), "Annotation 1") {
		t.Errorf("expected annotations in result, got %s", result)
	}
}

// Test predict_future tool
func TestChronosMCPServer_PredictFuture(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-predict-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Not enough data
	args := map[string]interface{}{"session_id": "empty-session"}
	result, err := server.HandleTool(ctx, "predict_future", args)
	if err != nil {
		t.Fatalf("failed to handle predict_future: %v", err)
	}
	if !strings.Contains(result.(string), "[chronos]:") {
		t.Errorf("expected '[chronos]:' prefix, got %s", result)
	}
}

// Test get_ambient_context tool with turns
func TestChronosMCPServer_GetAmbientContext(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-ambient-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Create snapshot
	mm.CreateSnapshot(ctx, "Snapshot content", "default", []string{}, nil)

	// Create turns
	for i := 1; i <= 3; i++ {
		mm.RecordTurn(ctx, "ambient-session", "User message", "Assistant reply", "")
	}

	args := map[string]interface{}{"session_id": "ambient-session"}
	result, err := server.HandleTool(ctx, "get_ambient_context", args)
	if err != nil {
		t.Fatalf("failed to handle get_ambient_context: %v", err)
	}
	if !strings.Contains(result.(string), "[chronos]") {
		t.Errorf("expected '[chronos]' in result, got %s", result)
	}
}

// Test get_snapshot tool
func TestChronosMCPServer_GetSnapshot(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-get-snapshot-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// Create snapshot
	snapID, _ := mm.CreateSnapshot(ctx, "Snapshot content", "test-env", []string{"tag"}, nil)

	args := map[string]interface{}{"id": float64(snapID)}
	result, err := server.HandleTool(ctx, "get_snapshot", args)
	if err != nil {
		t.Fatalf("failed to handle get_snapshot: %v", err)
	}
	if !strings.Contains(result.(string), "Snapshot #") {
		t.Errorf("expected snapshot in result, got %s", result)
	}
}

// Test unknown tool
func TestChronosMCPServer_UnknownTool(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-unknown-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	args := map[string]interface{}{}
	_, err = server.HandleTool(ctx, "unknown_tool", args)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' error, got %v", err)
	}
}

// Test analyze_evolution tool
func TestChronosMCPServer_AnalyzeEvolution(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "chronos-mcp-analyze-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.NewDB(tempDir)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer database.Close()

	mm := memory.NewMemoryManager(database)
	server := NewChronosMCPServer(mm)
	ctx := context.Background()

	// 1. Empty data
	args := map[string]interface{}{"session_id": "empty-session"}
	result, err := server.HandleTool(ctx, "analyze_evolution", args)
	if err != nil {
		t.Fatalf("analyze_evolution failed: %v", err)
	}
	if !strings.Contains(result.(string), "No data found for this session.") {
		t.Errorf("expected 'No data found for this session.' for empty session, got %s", result)
	}

	// 2. Normal data
	sessionID := "active-session"
	for i := 1; i <= 2; i++ {
		mm.RecordTurn(ctx, sessionID, "Short message", "Assistant response", "")
	}

	args = map[string]interface{}{"session_id": sessionID}
	result, err = server.HandleTool(ctx, "analyze_evolution", args)
	if err != nil {
		t.Fatalf("analyze_evolution failed: %v", err)
	}
	if !strings.Contains(result.(string), "[chronos] Thought Evolution Analysis") {
		t.Errorf("expected header in result, got %s", result)
	}
}
