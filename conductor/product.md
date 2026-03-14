# Product Definition: Chronos MCP Enhancements (v2.0, v2.1)

## Overview
Chronos MCP is a memory version control and conversation tracking server. These enhancements provide deeper memory integration and time-based context awareness.

## Key Features (Implemented)
1.  **4-Layer Memory Architecture (v2.0)**:
    - **Consolidation Layer**: Added fields for memory consolidation (`is_persona_anchor`, `importance_score`, `causality_id`, `status_consolidated`).
    - **Persona Anchors**: Specialized memory fields for user preferences and persona tracking.
2.  **Time Awareness & Ambient Context (v2.1)**:
    - **Time Awareness Hook**: Automatic context injection (`[Chronos Awareness]`) when elapsed time between tool calls exceeds 1 hour.
    - **Session Logs**: Real-time tracking of session events (`start`, `end`).
    - **Ambient Context**: Enhanced context providing placeholder weather, trends, and anniversaries.

## Target Audience
LLM-based agents requiring long-term memory integration and contextual continuity over time.
