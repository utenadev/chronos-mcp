#!/usr/bin/env python3

import os
import sqlite3

import metabolism


def setup_test_db(tmp_path):
    db_path = os.path.join(tmp_path, "chronos.db")
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    cursor.execute(
        """
        CREATE TABLE consolidation_metadata (
            last_processed_turn_id INTEGER PRIMARY KEY,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
        """
    )
    cursor.execute(
        """
        CREATE TABLE conversation_turns (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id TEXT NOT NULL,
            turn_number INTEGER NOT NULL,
            user_message TEXT NOT NULL,
            assistant_reply TEXT NOT NULL,
            context_snapshot TEXT DEFAULT '',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
        """
    )
    cursor.execute(
        """
        CREATE TABLE memory_snapshots (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            content TEXT NOT NULL,
            environment TEXT NOT NULL DEFAULT 'default',
            tags TEXT DEFAULT '',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            parent_id INTEGER REFERENCES memory_snapshots(id),
            is_persona_anchor INTEGER DEFAULT 0,
            importance_score FLOAT DEFAULT 0.0,
            causality_id TEXT DEFAULT '',
            status_consolidated INTEGER DEFAULT 0
        )
        """
    )

    cursor.execute(
        "INSERT INTO consolidation_metadata (last_processed_turn_id) VALUES (0)"
    )

    conn.commit()
    conn.close()
    return db_path


def test_get_last_processed_turn_id(tmp_path):
    db_path = setup_test_db(tmp_path)
    processor = metabolism.ChronosMetabolism(db_path=db_path)

    assert processor.get_last_processed_turn_id() == 0

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute("DELETE FROM consolidation_metadata")
    cursor.execute(
        "INSERT INTO consolidation_metadata (last_processed_turn_id) VALUES (7)"
    )
    conn.commit()
    conn.close()

    assert processor.get_last_processed_turn_id() == 7


def test_get_new_turns(tmp_path):
    db_path = setup_test_db(tmp_path)
    processor = metabolism.ChronosMetabolism(db_path=db_path)

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute(
        """
        INSERT INTO conversation_turns (session_id, turn_number, user_message, assistant_reply)
        VALUES (?, ?, ?, ?)
        """,
        ("s1", 1, "hello", "hi"),
    )
    cursor.execute(
        """
        INSERT INTO conversation_turns (session_id, turn_number, user_message, assistant_reply)
        VALUES (?, ?, ?, ?)
        """,
        ("s1", 2, "implement feature", "done"),
    )
    conn.commit()
    conn.close()

    turns = processor.get_new_turns(1)
    assert len(turns) == 1
    assert turns[0]["turn_number"] == 2


def test_analyze_turns(tmp_path):
    db_path = setup_test_db(tmp_path)
    processor = metabolism.ChronosMetabolism(db_path=db_path)

    turns = [
        {
            "id": 1,
            "session_id": "s1",
            "turn_number": 1,
            "user_message": "We should implement tests",
            "assistant_reply": "I will test and verify",
            "context_snapshot": "",
            "created_at": "2026-01-01",
        }
    ]

    analysis = processor.analyze_turns(turns)
    assert analysis["turn_count"] == 1
    assert "testing" in analysis["patterns"]
    assert analysis["environment"] == "s1"


def test_full_flow(tmp_path):
    db_path = setup_test_db(tmp_path)
    processor = metabolism.ChronosMetabolism(db_path=db_path)

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute(
        """
        INSERT INTO conversation_turns (session_id, turn_number, user_message, assistant_reply)
        VALUES (?, ?, ?, ?)
        """,
        ("session-a", 1, "fix error", "fixed with patch"),
    )
    cursor.execute(
        """
        INSERT INTO conversation_turns (session_id, turn_number, user_message, assistant_reply)
        VALUES (?, ?, ?, ?)
        """,
        ("session-a", 2, "implement endpoint", "implemented"),
    )
    conn.commit()
    conn.close()

    result = processor.process_once(dry_run=False)
    assert result.processed_turns == 2
    assert result.last_turn_id == 2
    assert result.snapshot_id is not None

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute("SELECT COUNT(*) FROM memory_snapshots")
    snapshot_count = cursor.fetchone()[0]
    cursor.execute(
        "SELECT last_processed_turn_id FROM consolidation_metadata ORDER BY updated_at DESC LIMIT 1"
    )
    checkpoint = cursor.fetchone()[0]
    conn.close()

    assert snapshot_count == 1
    assert checkpoint == 2
