package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/kench/chronos-mcp/internal/db"
)

// MemoryManager handles memory version control and thought tracking
type MemoryManager struct {
	db *db.DB
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(database *db.DB) *MemoryManager {
	return &MemoryManager{db: database}
}

// Snapshot represents a memory snapshot with version control
type Snapshot struct {
	ID                 int64     `json:"id"`
	Content            string    `json:"content"`
	Environment        string    `json:"environment"`
	Tags               []string  `json:"tags"`
	CreatedAt          time.Time `json:"created_at"`
	ParentID           *int64    `json:"parent_id,omitempty"`
	IsPersonaAnchor    int       `json:"is_persona_anchor"`
	ImportanceScore    float64   `json:"importance_score"`
	CausalityID        string    `json:"causality_id"`
	StatusConsolidated int       `json:"status_consolidated"`
}

// Turn represents a conversation turn
type Turn struct {
	ID              int64     `json:"id"`
	SessionID       string    `json:"session_id"`
	TurnNumber      int       `json:"turn_number"`
	UserMessage     string    `json:"user_message"`
	AssistantReply  string    `json:"assistant_reply"`
	ContextSnapshot string    `json:"context_snapshot,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Annotation represents a thought annotation
type Annotation struct {
	ID        int64     `json:"id"`
	TurnID    int64     `json:"turn_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Evolution represents thought evolution analysis
type Evolution struct {
	SessionID     string    `json:"session_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	TurnCount     int       `json:"turn_count"`
	TopicChanges  int       `json:"topic_changes"`
	AvgTurnLength float64   `json:"avg_turn_length"`
}

// CreateSnapshot creates a new memory snapshot
func (m *MemoryManager) CreateSnapshot(ctx context.Context, content, env string, tags []string, parentID *int64) (int64, error) {
	snapshot := &db.MemorySnapshot{
		Content:     content,
		Environment: env,
		Tags:        joinTags(tags),
		ParentID:    parentID,
	}
	return m.db.CreateSnapshot(ctx, snapshot)
}

// CreateSnapshotExt creates a new memory snapshot with extension fields
func (m *MemoryManager) CreateSnapshotExt(ctx context.Context, content, env string, tags []string, parentID *int64, isPersona int, importance float64, causalityID string) (int64, error) {
	snapshot := &db.MemorySnapshot{
		Content:            content,
		Environment:        env,
		Tags:               joinTags(tags),
		ParentID:           parentID,
		IsPersonaAnchor:    isPersona,
		ImportanceScore:    importance,
		CausalityID:        causalityID,
		StatusConsolidated: 0,
	}
	return m.db.CreateSnapshot(ctx, snapshot)
}

// RecordSessionEvent records a session start or end
func (m *MemoryManager) RecordSessionEvent(ctx context.Context, eventType, summary string) error {
	_, err := m.db.RecordSessionEvent(ctx, eventType, summary)
	return err
}

// GetTimeSinceLastActivity returns the duration since the last session event or conversation turn
func (m *MemoryManager) GetTimeSinceLastActivity(ctx context.Context) (time.Duration, error) {
	// Check latest session event
	event, err := m.db.GetLatestSessionEvent(ctx)
	if err != nil {
		return 0, err
	}

	// For simplicity in this implementation, we just check the latest session log.
	// In a real scenario, we might also check the latest conversation turn.
	if event == nil {
		return 0, nil
	}

	return time.Since(event.Timestamp), nil
}

// GetSnapshot retrieves a snapshot by ID
func (m *MemoryManager) GetSnapshot(ctx context.Context, id int64) (*Snapshot, error) {
	s, err := m.db.GetSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		ID:                 s.ID,
		Content:            s.Content,
		Environment:        s.Environment,
		Tags:               splitTags(s.Tags),
		CreatedAt:          s.CreatedAt,
		ParentID:           s.ParentID,
		IsPersonaAnchor:    s.IsPersonaAnchor,
		ImportanceScore:    s.ImportanceScore,
		CausalityID:        s.CausalityID,
		StatusConsolidated: s.StatusConsolidated,
	}, nil
}

// ListSnapshots lists snapshots for an environment
func (m *MemoryManager) ListSnapshots(ctx context.Context, env string, limit int) ([]Snapshot, error) {
	snapshots, err := m.db.ListSnapshots(ctx, env, limit)
	if err != nil {
		return nil, err
	}
	result := make([]Snapshot, len(snapshots))
	for i, s := range snapshots {
		result[i] = Snapshot{
			ID:                 s.ID,
			Content:            s.Content,
			Environment:        s.Environment,
			Tags:               splitTags(s.Tags),
			CreatedAt:          s.CreatedAt,
			ParentID:           s.ParentID,
			IsPersonaAnchor:    s.IsPersonaAnchor,
			ImportanceScore:    s.ImportanceScore,
			CausalityID:        s.CausalityID,
			StatusConsolidated: s.StatusConsolidated,
		}
	}
	return result, nil
}

// CheckoutSnapshot retrieves the latest snapshot (like git checkout)
func (m *MemoryManager) CheckoutSnapshot(ctx context.Context, env string) (*Snapshot, error) {
	s, err := m.db.GetLatestSnapshot(ctx, env)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &Snapshot{
		ID:                 s.ID,
		Content:            s.Content,
		Environment:        s.Environment,
		Tags:               splitTags(s.Tags),
		CreatedAt:          s.CreatedAt,
		ParentID:           s.ParentID,
		IsPersonaAnchor:    s.IsPersonaAnchor,
		ImportanceScore:    s.ImportanceScore,
		CausalityID:        s.CausalityID,
		StatusConsolidated: s.StatusConsolidated,
	}, nil
}

// RecordTurn records a conversation turn
func (m *MemoryManager) RecordTurn(ctx context.Context, sessionID, userMsg, assistantReply, contextSnapshot string) (int64, error) {
	turnNum, err := m.db.GetTurnCount(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	turnNum++ // Next turn number

	turn := &db.ConversationTurn{
		SessionID:       sessionID,
		TurnNumber:      turnNum,
		UserMessage:     userMsg,
		AssistantReply:  assistantReply,
		ContextSnapshot: contextSnapshot,
	}
	return m.db.CreateTurn(ctx, turn)
}

// GetTurn retrieves a turn by ID
func (m *MemoryManager) GetTurn(ctx context.Context, id int64) (*Turn, error) {
	t, err := m.db.GetTurn(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Turn{
		ID:              t.ID,
		SessionID:       t.SessionID,
		TurnNumber:      t.TurnNumber,
		UserMessage:     t.UserMessage,
		AssistantReply:  t.AssistantReply,
		ContextSnapshot: t.ContextSnapshot,
		CreatedAt:       t.CreatedAt,
	}, nil
}

// GetSessionTurns retrieves all turns for a session
func (m *MemoryManager) GetSessionTurns(ctx context.Context, sessionID string) ([]Turn, error) {
	turns, err := m.db.GetSessionTurns(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	result := make([]Turn, len(turns))
	for i, t := range turns {
		result[i] = Turn{
			ID:              t.ID,
			SessionID:       t.SessionID,
			TurnNumber:      t.TurnNumber,
			UserMessage:     t.UserMessage,
			AssistantReply:  t.AssistantReply,
			ContextSnapshot: t.ContextSnapshot,
			CreatedAt:       t.CreatedAt,
		}
	}
	return result, nil
}

// AddAnnotation adds a thought annotation to a turn
func (m *MemoryManager) AddAnnotation(ctx context.Context, turnID int64, content string) (int64, error) {
	annotation := &db.ThoughtAnnotation{
		TurnID:  turnID,
		Content: content,
	}
	return m.db.CreateAnnotation(ctx, annotation)
}

// GetAnnotations retrieves annotations for a turn
func (m *MemoryManager) GetAnnotations(ctx context.Context, turnID int64) ([]Annotation, error) {
	annotations, err := m.db.GetAnnotations(ctx, turnID)
	if err != nil {
		return nil, err
	}
	result := make([]Annotation, len(annotations))
	for i, a := range annotations {
		result[i] = Annotation{
			ID:        a.ID,
			TurnID:    a.TurnID,
			Content:   a.Content,
			CreatedAt: a.CreatedAt,
		}
	}
	return result, nil
}

// AnalyzeEvolution analyzes thought evolution for a session
func (m *MemoryManager) AnalyzeEvolution(ctx context.Context, sessionID string) (*Evolution, error) {
	e, err := m.db.AnalyzeSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, nil
	}
	return &Evolution{
		SessionID:     e.SessionID,
		StartTime:     e.StartTime,
		EndTime:       e.EndTime,
		TurnCount:     e.TurnCount,
		TopicChanges:  e.TopicChanges,
		AvgTurnLength: e.AvgTurnLength,
	}, nil
}

// PredictNearFuture predicts near-future based on past inertia
func (m *MemoryManager) PredictNearFuture(ctx context.Context, sessionID string) (string, error) {
	turns, err := m.db.GetSessionTurns(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if len(turns) < 2 {
		return "Not enough data to predict. Need at least 2 conversation turns.", nil
	}

	// Simple prediction: look at recent topics
	recentTurns := turns
	if len(recentTurns) > 5 {
		recentTurns = turns[len(turns)-5:]
	}

	// Analyze recent message patterns
	var userMessages []string
	for _, t := range recentTurns {
		if len(t.UserMessage) > 0 {
			userMessages = append(userMessages, t.UserMessage)
		}
	}

	if len(userMessages) == 0 {
		return "No user messages found in recent turns.", nil
	}

	prediction := fmt.Sprintf("Based on your recent %d turns, you're focusing on: %s. ",
		len(userMessages), summarizeTopic(userMessages[len(userMessages)-1]))

	// Check for patterns
	if len(turns) >= 3 {
		pattern := detectPattern(turns)
		if pattern != "" {
			prediction += pattern
		}
	}

	return prediction, nil
}

// Helper functions

func joinTags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}

func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	var result []string
	for _, t := range splitComma(tags) {
		if t := trim(t); t != "" {
			result = append(result, t)
		}
	}
	return result
}

func splitComma(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func summarizeTopic(message string) string {
	// Simple keyword extraction - take first 50 chars
	if len(message) > 50 {
		return message[:50] + "..."
	}
	return message
}

func detectPattern(turns []db.ConversationTurn) string {
	// Simple pattern detection
	if len(turns) < 3 {
		return ""
	}

	// Check for increasing/decreasing turn length
	var lengths []int
	for _, t := range turns {
		lengths = append(lengths, len(t.UserMessage)+len(t.AssistantReply))
	}

	increasing := true
	decreasing := true
	for i := 1; i < len(lengths); i++ {
		if lengths[i] <= lengths[i-1] {
			increasing = false
		}
		if lengths[i] >= lengths[i-1] {
			decreasing = false
		}
	}

	if increasing {
		return "Your conversations are getting more detailed over time."
	}
	if decreasing {
		return "Your conversations are becoming more concise."
	}

	return "Your conversation patterns appear stable."
}
