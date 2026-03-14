# Tech Stack: Chronos MCP

## Core Technologies
- **Language**: Go 1.25
- **Database**: SQLite (WAL mode)
- **MCP SDK**: `github.com/mark3labs/mcp-go`

## Key Libraries
- `github.com/mattn/go-sqlite3`: SQLite driver with WAL support.
- `github.com/mark3labs/mcp-go`: MCP server implementation SDK.

## Architecture
- **Layered Memory**: DB -> Domain (Memory) -> Server (MCP).
- **Hook Pattern**: Middleware-like execution in `HandleTool` for time awareness.
- **BBS Coordination**: Multi-agent orchestration via `agent-hub` (SQLite BBS).
