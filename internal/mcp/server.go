package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kench/chronos-mcp/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

type ChronosMCPServer struct {
	mm        *memory.MemoryManager
	tools     []mcp.Tool
	resources []mcp.Resource
}

func NewChronosMCPServer(mm *memory.MemoryManager) *ChronosMCPServer {
	s := &ChronosMCPServer{mm: mm}
	s.initTools()
	return s
}

func (s *ChronosMCPServer) initTools() {
	s.tools = []mcp.Tool{
		mcp.NewTool("create_snapshot",
			mcp.WithDescription("Create a memory snapshot (like git commit)"),
			mcp.WithString("content", mcp.Required(), mcp.Description("The memory content to snapshot")),
			mcp.WithString("environment", mcp.Description("Environment identifier (default: 'default')")),
			mcp.WithString("tags", mcp.Description("Comma-separated tags")),
		),
		mcp.NewTool("checkout_snapshot",
			mcp.WithDescription("Checkout the latest snapshot (like git checkout)"),
			mcp.WithString("environment", mcp.Description("Environment identifier (default: 'default')")),
		),
		mcp.NewTool("list_snapshots",
			mcp.WithDescription("List memory snapshots"),
			mcp.WithString("environment", mcp.Description("Environment identifier (default: 'default')")),
			mcp.WithNumber("limit", mcp.Description("Number of snapshots to retrieve (default: 10)")),
		),
		mcp.NewTool("get_snapshot",
			mcp.WithDescription("Get a specific snapshot by ID"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Snapshot ID")),
		),
		mcp.NewTool("record_turn",
			mcp.WithDescription("Record a conversation turn"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
			mcp.WithString("user_message", mcp.Required(), mcp.Description("User's message")),
			mcp.WithString("assistant_reply", mcp.Required(), mcp.Description("Assistant's reply")),
			mcp.WithString("context_snapshot", mcp.Description("Context snapshot at this turn")),
		),
		mcp.NewTool("get_session_turns",
			mcp.WithDescription("Get all turns for a session"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		mcp.NewTool("get_turn",
			mcp.WithDescription("Get a specific turn by ID"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Turn ID")),
		),
		mcp.NewTool("add_annotation",
			mcp.WithDescription("Add a thought annotation to a turn"),
			mcp.WithNumber("turn_id", mcp.Required(), mcp.Description("Turn ID to annotate")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Annotation content")),
		),
		mcp.NewTool("get_annotations",
			mcp.WithDescription("Get annotations for a turn"),
			mcp.WithNumber("turn_id", mcp.Required(), mcp.Description("Turn ID")),
		),
		mcp.NewTool("analyze_evolution",
			mcp.WithDescription("Analyze thought evolution for a session"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		mcp.NewTool("predict_future",
			mcp.WithDescription("Predict near-future based on past inertia"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		mcp.NewTool("get_ambient_context",
			mcp.WithDescription("Get ambient context for injection (Chronos signature)"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		mcp.NewTool("record_session_event",
			mcp.WithDescription("Record a session event (start or end)"),
			mcp.WithString("type", mcp.Required(), mcp.Description("Event type (start or end)")),
			mcp.WithString("summary", mcp.Description("Summary of the session")),
		),
	}
	s.resources = []mcp.Resource{}
}

func (s *ChronosMCPServer) GetTools() []mcp.Tool         { return s.tools }
func (s *ChronosMCPServer) GetResources() []mcp.Resource { return s.resources }

func (s *ChronosMCPServer) HandleTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// Time Awareness Hook
	awareness := ""
	diff, err := s.mm.GetTimeSinceLastActivity(ctx)
	if err == nil && diff > 1*time.Hour {
		awareness = fmt.Sprintf("\n\n[Chronos Awareness] It has been %v since your last activity. You were previously working on a session.", diff.Round(time.Minute))
	}

	var result interface{}
	var toolErr error

	switch name {
	case "create_snapshot":
		result, toolErr = s.handleCreateSnapshot(ctx, args)
	case "checkout_snapshot":
		result, toolErr = s.handleCheckoutSnapshot(ctx, args)
	case "list_snapshots":
		result, toolErr = s.handleListSnapshots(ctx, args)
	case "get_snapshot":
		result, toolErr = s.handleGetSnapshot(ctx, args)
	case "record_turn":
		result, toolErr = s.handleRecordTurn(ctx, args)
	case "get_session_turns":
		result, toolErr = s.handleGetSessionTurns(ctx, args)
	case "get_turn":
		result, toolErr = s.handleGetTurn(ctx, args)
	case "add_annotation":
		result, toolErr = s.handleAddAnnotation(ctx, args)
	case "get_annotations":
		result, toolErr = s.handleGetAnnotations(ctx, args)
	case "analyze_evolution":
		result, toolErr = s.handleAnalyzeEvolution(ctx, args)
	case "predict_future":
		result, toolErr = s.handlePredictFuture(ctx, args)
	case "get_ambient_context":
		result, toolErr = s.handleGetAmbientContext(ctx, args)
	case "record_session_event":
		result, toolErr = s.handleRecordSessionEvent(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	if toolErr != nil {
		return nil, toolErr
	}

	if awareness != "" {
		return fmt.Sprintf("%v%s", result, awareness), nil
	}
	return result, nil
}

func (s *ChronosMCPServer) handleRecordSessionEvent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	eventType := getString(args, "type", "")
	summary := getString(args, "summary", "")
	if eventType == "" {
		return nil, fmt.Errorf("type is required")
	}
	err := s.mm.RecordSessionEvent(ctx, eventType, summary)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Session event '%s' recorded successfully.", eventType), nil
}

func (s *ChronosMCPServer) handleCreateSnapshot(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	content := getString(args, "content", "")
	env := getString(args, "environment", "default")
	tagsStr := getString(args, "tags", "")
	tags := strings.Split(tagsStr, ",")
	if tagsStr == "" {
		tags = []string{}
	}
	id, err := s.mm.CreateSnapshot(ctx, content, env, tags, nil)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Snapshot created with ID: %d", id), nil
}

func (s *ChronosMCPServer) handleCheckoutSnapshot(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	env := getString(args, "environment", "default")
	snapshot, err := s.mm.CheckoutSnapshot(ctx, env)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return "No snapshots found for this environment.", nil
	}
	return formatSnapshot(snapshot), nil
}

func (s *ChronosMCPServer) handleListSnapshots(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	env := getString(args, "environment", "default")
	limit := int(getInt(args, "limit", 10))
	snapshots, err := s.mm.ListSnapshots(ctx, env, limit)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return "No snapshots found.", nil
	}
	var result string
	for _, snap := range snapshots {
		result += formatSnapshot(&snap) + "\n---\n"
	}
	return result, nil
}

func (s *ChronosMCPServer) handleGetSnapshot(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt(args, "id", 0)
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}
	snapshot, err := s.mm.GetSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	return formatSnapshot(snapshot), nil
}

func (s *ChronosMCPServer) handleRecordTurn(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID := getString(args, "session_id", "")
	userMsg := getString(args, "user_message", "")
	assistantReply := getString(args, "assistant_reply", "")
	contextSnapshot := getString(args, "context_snapshot", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	id, err := s.mm.RecordTurn(ctx, sessionID, userMsg, assistantReply, contextSnapshot)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Turn recorded with ID: %d", id), nil
}

func (s *ChronosMCPServer) handleGetSessionTurns(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID := getString(args, "session_id", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	turns, err := s.mm.GetSessionTurns(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(turns) == 0 {
		return "No turns found for this session.", nil
	}
	var result string
	for _, turn := range turns {
		result += formatTurn(&turn) + "\n---\n"
	}
	return result, nil
}

func (s *ChronosMCPServer) handleGetTurn(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id := getInt(args, "id", 0)
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}
	turn, err := s.mm.GetTurn(ctx, id)
	if err != nil {
		return nil, err
	}
	if turn == nil {
		return nil, fmt.Errorf("turn not found")
	}
	return formatTurn(turn), nil
}

func (s *ChronosMCPServer) handleAddAnnotation(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	turnID := getInt(args, "turn_id", 0)
	content := getString(args, "content", "")
	if turnID == 0 {
		return nil, fmt.Errorf("turn_id is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	id, err := s.mm.AddAnnotation(ctx, turnID, content)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Annotation added with ID: %d", id), nil
}

func (s *ChronosMCPServer) handleGetAnnotations(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	turnID := getInt(args, "turn_id", 0)
	if turnID == 0 {
		return nil, fmt.Errorf("turn_id is required")
	}
	annotations, err := s.mm.GetAnnotations(ctx, turnID)
	if err != nil {
		return nil, err
	}
	if len(annotations) == 0 {
		return "No annotations found for this turn.", nil
	}
	var result string
	for _, a := range annotations {
		result += fmt.Sprintf("[%d] %s\n", a.ID, a.Content)
	}
	return result, nil
}

func (s *ChronosMCPServer) handleAnalyzeEvolution(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID := getString(args, "session_id", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	evolution, err := s.mm.AnalyzeEvolution(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if evolution == nil {
		return "No data found for this session.", nil
	}
	return formatEvolution(evolution), nil
}

func (s *ChronosMCPServer) handlePredictFuture(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID := getString(args, "session_id", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	prediction, err := s.mm.PredictNearFuture(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return "[chronos]: " + prediction, nil
}

func (s *ChronosMCPServer) handleGetAmbientContext(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID := getString(args, "session_id", "")
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	// Last activity info
	diff, _ := s.mm.GetTimeSinceLastActivity(ctx)
	lastActivity := fmt.Sprintf("[chronos] Last activity: %v ago.", diff.Round(time.Minute))

	// Placeholder Environmental info (Ambient context)
	weather := "Cloudy, 18°C"
	anniversary := "March 14: Pi Day"
	trend := "Go 1.25 release, AI memory architectures"

	ambient := fmt.Sprintf("\n[Ambient Context]\n- Weather: %s\n- Today: %s\n- Trends: %s", weather, anniversary, trend)

	turns, err := s.mm.GetSessionTurns(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(turns) == 0 {
		return fmt.Sprintf("%s\n%s\n[chronos] No previous context found for this session.", lastActivity, ambient), nil
	}
	snapshot, _ := s.mm.CheckoutSnapshot(ctx, "default")
	var context string
	if snapshot != nil {
		context = fmt.Sprintf("[chronos] Context from latest snapshot: %s\n", snapshot.Content)
	}
	if len(turns) >= 2 {
		recent := turns[len(turns)-2:]
		context += "[chronos] Recent turns: "
		for _, t := range recent {
			context += fmt.Sprintf("\n- Turn %d: %s", t.TurnNumber, truncate(t.UserMessage, 100))
		}
	}
	return fmt.Sprintf("%s\n%s\n%s", lastActivity, ambient, context), nil
}

func getString(args map[string]interface{}, key, def string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return def
}

func getInt(args map[string]interface{}, key string, def int) int64 {
	if v, ok := args[key].(float64); ok {
		return int64(v)
	}
	return int64(def)
}

func formatSnapshot(s *memory.Snapshot) string {
	tags := ""
	if len(s.Tags) > 0 {
		tags = fmt.Sprintf(" [Tags: %s]", strings.Join(s.Tags, ", "))
	}
	return fmt.Sprintf("Snapshot #%d%s\nEnvironment: %s\nCreated: %s\n\n%s",
		s.ID, tags, s.Environment, s.CreatedAt.Format("2006-01-02 15:04:05"), s.Content)
}

func formatTurn(t *memory.Turn) string {
	return fmt.Sprintf("Turn #%d (Session: %s)\nTime: %s\n\nUser: %s\n\nAssistant: %s",
		t.TurnNumber, t.SessionID, t.CreatedAt.Format("2006-01-02 15:04:05"),
		truncate(t.UserMessage, 200), truncate(t.AssistantReply, 200))
}

func formatEvolution(e *memory.Evolution) string {
	return fmt.Sprintf(`[chronos] Thought Evolution Analysis

Session: %s
Duration: %s to %s
Total Turns: %d
Avg Turn Length: %.1f characters
Pattern: %s`,
		e.SessionID, e.StartTime.Format("2006-01-02 15:04"), e.EndTime.Format("2006-01-02 15:04"),
		e.TurnCount, e.AvgTurnLength, getEvolutionPattern(e))
}

func getEvolutionPattern(e *memory.Evolution) string {
	if e.AvgTurnLength > 500 {
		return "Detail-oriented"
	} else if e.AvgTurnLength < 100 {
		return "Concise"
	}
	return "Balanced"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
