#!/usr/bin/env python3
"""
Hippocampal Replay Batch Script

Performs memory consolidation during sleep cycle (3:00-4:00 AM).
- Extracts unconsolidated snapshots (status_consolidated=0)
- Uses LLM to extract causality and abstraction
- Updates snapshots with causality_id and marks as consolidated

Usage:
    python3 hippocampal_replay.py [--data-dir DIR]
"""

import argparse
import json
import os
import sqlite3
import sys
from datetime import datetime
from typing import List, Optional, Tuple


class HippocampalReplay:
    """Manages hippocampal replay for memory consolidation."""

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

        Returns causality_id string or None if no patterns found.
        """
        # Build context for LLM
        snapshot_content = snapshot.get("content", "")
        environment = snapshot.get("environment", "")
        tags = snapshot.get("tags", "")

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

        # For now, use a simple rule-based approach
        # In production, this would call an actual LLM API
        causality_id = self._analyze_patterns(snapshot_content, conversation_context)

        return causality_id

    def _analyze_patterns(self, content: str, turns: List[dict]) -> Optional[str]:
        """
        Simple pattern analysis (placeholder for LLM integration).

        In production, this would:
        1. Call OpenAI/Claude API
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

        # Generate causality ID
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        causality_id = f"{'-'.join(causality_patterns)}-{timestamp}"

        return causality_id

    def update_snapshot_consolidation(
        self, snapshot_id: int, causality_id: str
    ) -> bool:
        """Update snapshot with causality_id and mark as consolidated."""
        conn = self._get_connection()
        cursor = conn.cursor()

        try:
            cursor.execute(
                """
                UPDATE memory_snapshots
                SET causality_id = ?,
                    status_consolidated = 1,
                    importance_score = CASE 
                        WHEN importance_score < 0.5 THEN 0.5 
                        ELSE importance_score 
                    END
                WHERE id = ?
            """,
                (causality_id, snapshot_id),
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

            print(f"\nProcessing snapshot {snapshot_id}...")

            # Get related conversation turns
            turns = self.get_turns_for_session(session_id)
            print(f"  Found {len(turns)} conversation turns")

            # Extract causality using LLM
            causality_id = self.extract_causality_with_llm(snapshot, turns)

            if causality_id:
                print(f"  Extracted causality: {causality_id}")

                if not dry_run:
                    # Update snapshot
                    if self.update_snapshot_consolidation(snapshot_id, causality_id):
                        print(f"  ✓ Successfully consolidated")
                        succeeded += 1
                    else:
                        print(f"  ✗ Failed to consolidate")
                else:
                    print(f"  [DRY RUN] Would update with causality_id={causality_id}")
                    succeeded += 1
            else:
                print(f"  No causality patterns found")

            processed += 1

        print(f"\n{'=' * 50}")
        print(f"Replay complete: {succeeded}/{processed} snapshots consolidated")

        return processed, succeeded


def main():
    parser = argparse.ArgumentParser(
        description="Hippocampal Replay - Memory consolidation during sleep cycle"
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
        print(
            "Make sure the chronos-mcp server has been run at least once.",
            file=sys.stderr,
        )
        sys.exit(1)

    if args.list_unconsolidated:
        # Just list unconsolidated snapshots
        snapshots = replay.get_unconsolidated_snapshots()
        print(f"Unconsolidated snapshots: {len(snapshots)}")
        for snap in snapshots:
            print(
                f"  ID: {snap['id']}, Environment: {snap['environment']}, Created: {snap['created_at']}"
            )
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
