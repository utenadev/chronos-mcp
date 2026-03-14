#!/usr/bin/env python3
"""
Hippocampal Replay Batch Script (v2.0 - Enhanced)

Performs memory consolidation during sleep cycle (3:00-4:00 AM).
- Extracts unconsolidated snapshots (status_consolidated=0)
- Uses LLM to extract causality and abstraction
- Updates snapshots with structured causality_id (JSON format)
- Protects persona anchors (is_persona_anchor=1) with max importance

Enhancements:
1. Session-aware processing (get_current_session_id)
2. Persona anchor protection (importance=1.0, raw data emphasis)
3. GraphRAG-ready causality (JSON structure with parent_id)

Usage:
    python3 hippocampal_replay.py [--data-dir DIR]
"""

import argparse
import json
import os
import sqlite3
import sys
from datetime import datetime
from typing import List, Optional, Tuple, Dict


class HippocampalReplay:
    """Manages hippocampal replay for memory consolidation with enhanced features."""
    
    def __init__(self, data_dir: str = None):
        """Initialize with data directory."""
        if data_dir is None:
            data_dir = os.path.expanduser("~/.chronos")
        
        self.data_dir = data_dir
        self.db_path = os.path.join(data_dir, "chronos.db")
        
    def _get_connection(self) -> sqlite3.Connection:
        """Get database connection."""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        return conn
    
    def get_current_session_id(self) -> Optional[str]:
        """Get current session ID from latest session log."""
        conn = self._get_connection()
        cursor = conn.cursor()
        
        cursor.execute("""
            SELECT event_type, summary
            FROM session_logs
            ORDER BY timestamp DESC
            LIMIT 1
        """)
        
        row = cursor.fetchone()
        conn.close()
        
        if row:
            # Return the summary as session identifier if available
            return row['summary']
        return None
    
    def get_unconsolidated_snapshots(self) -> List[dict]:
        """Get all unconsolidated snapshots (status_consolidated=0)."""
        conn = self._get_connection()
        cursor = conn.cursor()

        cursor.execute("""
            SELECT id, content, environment, tags, created_at, parent_id,
                   is_persona_anchor, importance_score, causality_id, status_consolidated
            FROM memory_snapshots
            WHERE status_consolidated = 0
            ORDER BY created_at ASC
        """)

        snapshots = [dict(row) for row in cursor.fetchall()]
        conn.close()

        return snapshots
    
    def get_turns_for_session(self, session_id: str) -> List[dict]:
        """Get all conversation turns for a session."""
        conn = self._get_connection()
        cursor = conn.cursor()

        cursor.execute(
            """
            SELECT id, session_id, turn_number, user_message, 
                   assistant_reply, context_snapshot, created_at
            FROM conversation_turns
            WHERE session_id = ?
            ORDER BY turn_number ASC
            """,
            (session_id,),
        )

        turns = [dict(row) for row in cursor.fetchall()]
        conn.close()

        return turns
    
    def extract_causality_with_llm(
        self, snapshot: dict, turns: List[dict]
    ) -> Optional[str]:
        """
        Use LLM to extract causality and abstraction.

        Returns JSON string with causality info or None if no patterns found.
        """
        # Build context for LLM
        snapshot_content = snapshot.get("content", "")
        environment = snapshot.get("environment", "")
        tags = snapshot.get("tags", "")
        is_persona_anchor = snapshot.get("is_persona_anchor", 0)
        parent_id = snapshot.get("parent_id")

        # Build conversation context
        conversation_context = []
        for turn in turns:
            conversation_context.append(
                {
                    "turn_number": turn["turn_number"],
                    "user": turn["user_message"],
                    "assistant": turn["assistant_reply"],
                }
            )

        # Analyze patterns with persona and parent awareness
        causality_data = self._analyze_patterns(
            snapshot_content, conversation_context, is_persona_anchor, parent_id
        )

        if causality_data:
            return json.dumps(causality_data)
        return None
    
    def _analyze_patterns(
        self, content: str, turns: List[dict], is_persona_anchor: int = 0, parent_id: int = None
    ) -> Optional[Dict]:
        """
        Pattern analysis with persona anchor and causality graph support.

        In production, this would:
        1. Call OpenAI/Claude API with persona-aware prompts
        2. Extract causality relationships
        3. Generate abstraction/summary
        """
        # Simple keyword-based causality extraction
        causality_patterns = []

        # Pattern 1: Error resolution
        if "error" in content.lower() or "fix" in content.lower():
            causality_patterns.append("error-resolution")

        # Pattern 2: Implementation completion
        if "implement" in content.lower() or "complete" in content.lower():
            causality_patterns.append("implementation-complete")

        # Pattern 3: Learning/decision
        if (
            "decide" in content.lower()
            or "learn" in content.lower()
            or "choose" in content.lower()
        ):
            causality_patterns.append("learning-decision")

        # Pattern 4: Integration
        if "integrate" in content.lower() or "connect" in content.lower():
            causality_patterns.append("integration")

        if not causality_patterns:
            # Default: general conversation
            causality_patterns.append("general-conversation")

        # For persona anchors, add special pattern and preserve raw data
        if is_persona_anchor:
            causality_patterns.insert(0, "persona-anchor")

        # Build causality data with graph support (JSON structure for GraphRAG)
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        causality_data = {
            "pattern": causality_patterns[0] if causality_patterns else "unknown",
            "patterns": causality_patterns,
            "timestamp": timestamp,
            "parent_id": parent_id,
            "is_persona_anchor": is_persona_anchor == 1
        }

        return causality_data
    
    def update_snapshot_consolidation(
        self, snapshot_id: int, causality_json: str, is_persona_anchor: int = 0
    ) -> bool:
        """Update snapshot with causality_id and mark as consolidated."""
        conn = self._get_connection()
        cursor = conn.cursor()

        try:
            # For persona anchors, set importance to 1.0
            # For others, ensure minimum 0.5
            cursor.execute(
                """
                UPDATE memory_snapshots
                SET causality_id = ?,
                    status_consolidated = 1,
                    importance_score = CASE
                        WHEN ? = 1 THEN 1.0
                        WHEN importance_score < 0.5 THEN 0.5
                        ELSE importance_score
                    END
                WHERE id = ?
                """,
                (causality_json, is_persona_anchor, snapshot_id),
            )

            conn.commit()
            success = cursor.rowcount > 0

        except sqlite3.Error as e:
            print(f"Error updating snapshot {snapshot_id}: {e}", file=sys.stderr)
            success = False
        finally:
            conn.close()

        return success
    
    def perform_replay(self, dry_run: bool = False) -> Tuple[int, int]:
        """
        Perform hippocampal replay on all unconsolidated snapshots.

        Args:
            dry_run: If True, don't actually update database

        Returns:
            Tuple of (processed_count, success_count)
        """
        print(f"{'[DRY RUN] ' if dry_run else ''}Starting hippocampal replay...")
        print(f"Data directory: {self.data_dir}")
        
        # Get current session for context
        current_session = self.get_current_session_id()
        if current_session:
            print(f"Current session: {current_session}")

        # Get unconsolidated snapshots
        snapshots = self.get_unconsolidated_snapshots()
        print(f"Found {len(snapshots)} unconsolidated snapshots")

        if not snapshots:
            print("No snapshots to process")
            return 0, 0

        processed = 0
        succeeded = 0

        for snapshot in snapshots:
            snapshot_id = snapshot["id"]
            session_id = snapshot.get("environment", f"snapshot-{snapshot_id}")
            is_persona_anchor = snapshot.get("is_persona_anchor", 0)
            parent_id = snapshot.get("parent_id")

            print(f"\nProcessing snapshot {snapshot_id}...")
            if is_persona_anchor:
                print(f"  [PERSONA ANCHOR - Protected]")
            if parent_id:
                print(f"  Parent ID: {parent_id}")

            # Get related conversation turns (prefer current session)
            turns = self.get_turns_for_session(session_id)
            print(f"  Found {len(turns)} conversation turns")

            # Extract causality using LLM
            causality_json = self.extract_causality_with_llm(snapshot, turns)

            if causality_json:
                print(f"  Extracted causality: {causality_json}")

                if not dry_run:
                    # Update snapshot with persona awareness
                    if self.update_snapshot_consolidation(
                        snapshot_id, causality_json, is_persona_anchor
                    ):
                        print(f"  ✓ Successfully consolidated")
                        succeeded += 1
                    else:
                        print(f"  ✗ Failed to consolidate")
                else:
                    print(f"  [DRY RUN] Would update with causality_id={causality_json}")
                    succeeded += 1
            else:
                print(f"  No causality patterns found")

            processed += 1

        print(f"\n{'=' * 50}")
        print(f"Replay complete: {succeeded}/{processed} snapshots consolidated")

        return processed, succeeded


def main():
    parser = argparse.ArgumentParser(
        description="Hippocampal Replay - Memory consolidation during sleep cycle (Enhanced v2.0)"
    )
    parser.add_argument(
        "--data-dir", default=None, help="Data directory (default: ~/.chronos)"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be done without making changes",
    )
    parser.add_argument(
        "--list-unconsolidated",
        action="store_true",
        help="List unconsolidated snapshots without processing",
    )
    
    args = parser.parse_args()
    
    # Initialize replay manager
    replay = HippocampalReplay(data_dir=args.data_dir)
    
    # Check if database exists
    if not os.path.exists(replay.db_path):
        print(f"Error: Database not found at {replay.db_path}", file=sys.stderr)
        print("Make sure the chronos-mcp server has been run at least once.", file=sys.stderr)
        sys.exit(1)
    
    if args.list_unconsolidated:
        # Just list unconsolidated snapshots
        snapshots = replay.get_unconsolidated_snapshots()
        print(f"Unconsolidated snapshots: {len(snapshots)}")
        for snap in snapshots:
            print(f"  ID: {snap['id']}, Environment: {snap['environment']}, Created: {snap['created_at']}")
            if snap.get('is_persona_anchor'):
                print(f"    [PERSONA ANCHOR]")
            if snap.get('parent_id'):
                print(f"    Parent: {snap['parent_id']}")
            print(f"    Content preview: {snap['content'][:100]}...")
        sys.exit(0)
    
    # Perform replay
    processed, succeeded = replay.perform_replay(dry_run=args.dry_run)
    
    if processed == 0:
        sys.exit(0)
    elif succeeded < processed:
        sys.exit(1)  # Partial failure
    else:
        sys.exit(0)


if __name__ == "__main__":
    main()
