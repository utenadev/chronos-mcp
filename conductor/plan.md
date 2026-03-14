# Implementation Plan: Chronos MCP Enhancements

## Objective
Implement the 4-layer memory architecture (v2.0) and time-awareness features (v2.1) for the chronos-mcp server based on the provided specifications, strictly adhering to Test-Driven Development (TDD). 
*(Note: The agent-hub singleton specification has been discarded as per user instruction.)*

## Scope & Impact
- **Database**: Add consolidation fields to memory tables and introduce a new `session_logs` table.
- **Domain Logic**: Update memory models to support persona anchors and session tracking.
- **Server/MCP**: Implement the "Time Awareness Hook" for all tool executions, enhance ambient context, and add session recording tools.

## Implementation Steps (TDD Approach)

### Phase 1: Database Layer (`internal/db`)
1.  **Write Tests**: Create `internal/db/db_test.go` to test schema creation, new fields, and `session_logs` operations.
2.  **Update Schema (`db.go`)**:
    - Add `is_persona_anchor` (INTEGER), `importance_score` (FLOAT), `causality_id` (TEXT), and `status_consolidated` (INTEGER) to the `memory_snapshots` table.
    - Create the `session_logs` table (`id`, `event_type`, `timestamp`, `summary`).
3.  **Implement Operations**:
    - Update `CreateSnapshot`, `GetSnapshot`, and `ListSnapshots` to handle the new memory fields.
    - Add `RecordSessionEvent(eventType, summary)` and `GetLatestSessionEvent()` functions.

### Phase 2: Memory Logic (`internal/memory`)
1.  **Write Tests**: Create `internal/memory/memory_test.go` for the updated memory models and session event logic.
2.  **Update Models (`memory.go`)**:
    - Add the new consolidation fields to the `Snapshot` struct.
    - Create a `SessionEvent` struct.
3.  **Implement Logic**:
    - Update `CreateSnapshot` and related functions to propagate the new fields.
    - Implement `RecordSessionEvent` and `GetTimeSinceLastActivity` functions.

### Phase 3: MCP Server & Time Awareness (`internal/mcp`)
1.  **Write Tests**: Create `internal/mcp/server_test.go` to test the Time Awareness Hook, the updated `get_ambient_context` tool, and the new `record_session_event` tool.
2.  **Time Awareness Hook (`server.go`)**:
    - Modify `HandleTool` to check the elapsed time since the last activity before executing any tool.
    - If the elapsed time exceeds a threshold (e.g., 1 hour), append a `[Chronos Awareness]` context string to the tool's result.
3.  **Ambient Context Enhancement (`server.go`)**:
    - Update the existing `get_ambient_context` tool to include the last active time and placeholder environmental information (e.g., mock weather/anniversary/trends) to fulfill the emotional context requirement.
4.  **Session Management Tool (`server.go`)**:
    - Add a new `record_session_event` tool that accepts `type` ("start" | "end") and records it via the Memory Manager.

## Verification
- Run `go test -v ./...` to ensure all newly written tests pass.
- Verify that `go build ./cmd/chronos-mcp` completes without errors.
- Test the new MCP tools and the Time Awareness Hook manually via the CLI.