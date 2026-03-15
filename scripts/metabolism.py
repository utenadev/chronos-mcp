#!/usr/bin/env python3
"""Chronos Metabolism v1.0 - incremental memory metabolism.

This script continuously (or one-shot) reads newly recorded conversation turns,
extracts lightweight keyword-based patterns, and stores the result as a new
memory snapshot. It maintains a checkpoint in consolidation_metadata so each
turn is processed exactly once.
"""

from __future__ import annotations

import argparse
import os
import sqlite3
import sys
import time
from collections import Counter
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Dict, List


DEFAULT_DATA_DIR = os.path.expanduser("~/.chronos")
DEFAULT_DB_PATH = os.path.join(DEFAULT_DATA_DIR, "chronos.db")
POLL_INTERVAL_SECONDS = 30


@dataclass
class ProcessingResult:
    """Container for one metabolism processing run."""

    processed_turns: int
    last_turn_id: int
    snapshot_id: int | None
    dry_run: bool


class ChronosMetabolism:
    """Incrementally processes conversation turns into memory snapshots."""

    def __init__(self, db_path: str = DEFAULT_DB_PATH) -> None:
        """Initialize metabolism processor with SQLite database path."""
        self.db_path = db_path

    def _get_connection(self) -> sqlite3.Connection:
        """Create a SQLite connection with row access by column name."""
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        return conn

    def get_last_processed_turn_id(self) -> int:
        """Return checkpoint from consolidation_metadata or 0 when missing."""
        conn = self._get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(
                """
                SELECT COALESCE(MAX(last_processed_turn_id), 0) AS last_processed_turn_id
                FROM consolidation_metadata
                """
            )
            row = cursor.fetchone()
            return int(row["last_processed_turn_id"]) if row else 0
        finally:
            conn.close()

    def get_new_turns(self, since_id: int) -> List[Dict[str, Any]]:
        """Fetch conversation turns whose id is greater than the checkpoint."""
        conn = self._get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(
                """
                SELECT id, session_id, turn_number, user_message, assistant_reply,
                       context_snapshot, created_at
                FROM conversation_turns
                WHERE id > ?
                ORDER BY id ASC
                """,
                (since_id,),
            )
            return [dict(row) for row in cursor.fetchall()]
        finally:
            conn.close()

    def analyze_turns(self, turns: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Extract simple keyword patterns from turns (LLM-ready placeholder)."""
        combined_text = "\n".join(
            f"U:{turn['user_message']}\nA:{turn['assistant_reply']}" for turn in turns
        ).lower()

        keyword_groups = {
            "error-resolution": ["error", "bug", "fix", "failure", "exception"],
            "implementation": ["implement", "build", "create", "develop", "code"],
            "planning": ["plan", "strategy", "roadmap", "approach", "design"],
            "decision-making": ["decide", "choose", "tradeoff", "option", "select"],
            "testing": ["test", "assert", "validate", "verify", "coverage"],
        }

        matched_patterns: List[str] = []
        keyword_counts: Counter[str] = Counter()
        for pattern, keywords in keyword_groups.items():
            count = sum(combined_text.count(keyword) for keyword in keywords)
            if count > 0:
                matched_patterns.append(pattern)
                keyword_counts[pattern] = count

        if not matched_patterns:
            matched_patterns.append("general-conversation")

        session_ids = sorted({turn["session_id"] for turn in turns})
        analysis_time = datetime.now(timezone.utc).isoformat(timespec="seconds")
        lead_text = (
            f"Metabolism summary at {analysis_time}: "
            f"processed {len(turns)} turn(s), patterns={', '.join(matched_patterns)}"
        )

        return {
            "content": lead_text,
            "environment": session_ids[-1] if session_ids else "default",
            "tags": ",".join(matched_patterns),
            "patterns": matched_patterns,
            "pattern_counts": dict(keyword_counts),
            "turn_count": len(turns),
            "session_ids": session_ids,
        }

    def create_snapshot(
        self, analysis: Dict[str, Any], dry_run: bool = False
    ) -> int | None:
        """Insert metabolism output into memory_snapshots and return snapshot id."""
        if dry_run:
            return None

        conn = self._get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(
                """
                INSERT INTO memory_snapshots (
                    content,
                    environment,
                    tags,
                    importance_score,
                    status_consolidated
                ) VALUES (?, ?, ?, ?, ?)
                """,
                (
                    analysis.get("content", "Metabolism summary"),
                    analysis.get("environment", "default"),
                    analysis.get("tags", ""),
                    0.6,
                    1,
                ),
            )
            conn.commit()
            row_id = cursor.lastrowid
            return int(row_id) if row_id is not None else None
        finally:
            conn.close()

    def update_checkpoint(self, turn_id: int, dry_run: bool = False) -> None:
        """Persist latest processed turn id into consolidation_metadata."""
        if dry_run:
            return

        conn = self._get_connection()
        try:
            cursor = conn.cursor()
            cursor.execute(
                """
                UPDATE consolidation_metadata
                SET last_processed_turn_id = ?,
                    updated_at = CURRENT_TIMESTAMP
                """,
                (turn_id,),
            )

            if cursor.rowcount == 0:
                cursor.execute(
                    """
                    INSERT INTO consolidation_metadata (last_processed_turn_id)
                    VALUES (?)
                    """,
                    (turn_id,),
                )

            conn.commit()
        finally:
            conn.close()

    def process_once(self, dry_run: bool = False) -> ProcessingResult:
        """Run one incremental metabolism cycle."""
        last_processed = self.get_last_processed_turn_id()
        turns = self.get_new_turns(last_processed)

        if not turns:
            return ProcessingResult(0, last_processed, None, dry_run)

        analysis = self.analyze_turns(turns)
        snapshot_id = self.create_snapshot(analysis, dry_run=dry_run)
        latest_turn_id = int(turns[-1]["id"])
        self.update_checkpoint(latest_turn_id, dry_run=dry_run)

        return ProcessingResult(len(turns), latest_turn_id, snapshot_id, dry_run)

    def run_daemon(
        self, dry_run: bool = False, poll_interval: int = POLL_INTERVAL_SECONDS
    ) -> None:
        """Continuously poll for new turns and process them every interval."""
        print(
            f"Starting Chronos metabolism daemon (interval={poll_interval}s, dry_run={dry_run})"
        )
        print(f"Database: {self.db_path}")

        while True:
            try:
                result = self.process_once(dry_run=dry_run)
                if result.processed_turns > 0:
                    print(
                        "Processed "
                        f"{result.processed_turns} turn(s), "
                        f"checkpoint={result.last_turn_id}, "
                        f"snapshot_id={result.snapshot_id}"
                    )
                else:
                    print("No new turns")
            except KeyboardInterrupt:
                print("Stopping metabolism daemon")
                return
            except sqlite3.Error as exc:
                print(f"Database error during daemon cycle: {exc}", file=sys.stderr)
            except Exception as exc:
                print(f"Unexpected error during daemon cycle: {exc}", file=sys.stderr)

            time.sleep(poll_interval)


def parse_args() -> argparse.Namespace:
    """Parse command line options for metabolism execution."""
    parser = argparse.ArgumentParser(
        description="Chronos Metabolism v1.0 - incremental memory metabolism"
    )
    mode_group = parser.add_mutually_exclusive_group()
    mode_group.add_argument(
        "--once",
        action="store_true",
        help="Run one metabolism cycle and exit",
    )
    mode_group.add_argument(
        "--daemon",
        action="store_true",
        help="Run continuous polling mode (every 30 seconds)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Analyze turns but do not modify the database",
    )
    parser.add_argument(
        "--db-path",
        default=DEFAULT_DB_PATH,
        help=f"SQLite database path (default: {DEFAULT_DB_PATH})",
    )
    return parser.parse_args()


def main() -> None:
    """Entry point for CLI usage."""
    args = parse_args()

    metabolism = ChronosMetabolism(db_path=args.db_path)
    if not os.path.exists(metabolism.db_path):
        print(f"Error: Database not found at {metabolism.db_path}", file=sys.stderr)
        sys.exit(1)

    try:
        if args.daemon:
            metabolism.run_daemon(dry_run=args.dry_run)
            return

        result = metabolism.process_once(dry_run=args.dry_run)
        if result.processed_turns == 0:
            print("No new turns to process")
        else:
            action = "Would process" if args.dry_run else "Processed"
            print(
                f"{action} {result.processed_turns} turn(s), "
                f"checkpoint={result.last_turn_id}, snapshot_id={result.snapshot_id}"
            )
    except sqlite3.Error as exc:
        print(f"Database error: {exc}", file=sys.stderr)
        sys.exit(1)
    except Exception as exc:
        print(f"Unexpected error: {exc}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
