# Chronos MCP: Memory Version Control & Time Awareness

Chronos MCP is an MCP (Model Context Protocol) server designed to transform agent conversation logs into long-term structured "wisdom." Featuring a SQLite backend, it provides version-controlled memory snapshots and active "Time Awareness."

## Key Features

### 1. 4-Layer Memory Architecture (v2.0)
Based on neuroscientific insights, memory is managed across four distinct layers:
- **Short-term Memory (Simple Memory)**: Raw conversation logs.
- **Semantic Memory**: Recall via vector search (Future extension).
- **Relational Memory (GraphRAG)**: Causal relationships between knowledge pieces.
- **Consolidation Layer [NEW]**: Summarization and abstraction of memories via sleep cycles.

### 2. Time Awareness Hook (v2.1)
Enables agents to perceive the passage of time:
- **Time Awareness Hook**: Automatically injects elapsed time context (`[Chronos Awareness]`) when a gap of more than 1 hour exists between tool executions.
- **Ambient Context**: Provides background info like weather, trends, and anniversaries for more natural interactions.

### 3. Session Management
- Explicitly tracks session `start` and `end` events to analyze the evolution of thoughts over time.

## Architecture

The system features a hybrid architecture, separating the real-time server layer from the compute-intensive analysis layer.

- **Go (MCP Server / Core)**:
    - Provides MCP tools (via stdio).
    - Direct SQLite DB access and persistence.
    - Time Awareness Hook for real-time context injection.
    - Core business logic and high-performance data handling.
- **Python (Consolidation Batch / Analyzer)**:
    - Background processing during "Sleep Cycles" (AM 3:00 - 4:00).
    - LLM-based summarization and abstraction of conversation logs.
    - Identification of causal links between memories.
    - Extraction of Persona Anchors for deeper user understanding.

## MCP Tools Reference

- `create_snapshot`: Save current state as a snapshot.
- `checkout_snapshot`: Retrieve the latest snapshot.
- `record_turn`: Log a conversation turn (user/assistant pair).
- `analyze_evolution`: Analyze the evolution of thoughts within a session.
- `get_ambient_context`: Retrieve background context including time gaps and environmental data.
- `record_session_event`: Log session lifecycle events.

## Project Status
- **Go Core**: Implemented with >80% unit test coverage.
- **Memory Consolidation Batch (Python)**: Currently being implemented by OpenCode.

## License
MIT License
