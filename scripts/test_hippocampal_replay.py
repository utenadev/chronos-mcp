#!/usr/bin/env python3
"""
Tests for Hippocampal Replay Script (v2.0 - Enhanced)

Tests new features:
1. Session-aware processing (get_current_session_id)
2. Persona anchor protection (importance=1.0, raw data emphasis)
3. GraphRAG-ready causality (JSON structure with parent_id)

Usage:
    python3 test_hippocampal_replay.py
"""

import argparse
import json
import os
import sys
import tempfile
import sqlite3
from datetime import datetime

# Import the main module
import hippocampal_replay as hr


class TestHippocampalReplay:
    """Test suite for hippocampal replay functionality (Enhanced v2.0)."""

    def setup_test_db(self):
        """Set up a test database."""
        self.temp_dir = tempfile.mkdtemp()
        self.db_path = os.path.join(self.temp_dir, "chronos.db")

        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Create tables
        cursor.execute("""
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
        """)

        cursor.execute("""
            CREATE TABLE conversation_turns (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                session_id TEXT NOT NULL,
                turn_number INTEGER NOT NULL,
                user_message TEXT NOT NULL,
                assistant_reply TEXT NOT NULL,
                context_snapshot TEXT DEFAULT '',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )
        """)

        cursor.execute("""
            CREATE TABLE session_logs (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                event_type TEXT NOT NULL,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
                summary TEXT
            )
        """)

        conn.commit()
        conn.close()

        return self.temp_dir

    def test_get_current_session_id(self):
        """Test getting current session ID from session_logs."""
        print("\nTest: get_current_session_id")

        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # No sessions yet
        session_id = replay.get_current_session_id()
        assert session_id is None, f"Expected None with no sessions, got {session_id}"
        print("  ✓ PASSED: Returns None when no sessions")

        # Insert session log
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute(
            "INSERT INTO session_logs (event_type, summary) VALUES (?, ?)",
            ("start", "test-session-001"),
        )
        conn.commit()
        conn.close()

        # Should return latest session
        session_id = replay.get_current_session_id()
        assert session_id == "test-session-001", (
            f"Expected 'test-session-001', got {session_id}"
        )
        print("  ✓ PASSED: Returns latest session ID")

        # Cleanup
        import shutil

        shutil.rmtree(temp_dir)

    def test_get_unconsolidated_snapshots(self):
        """Test retrieving unconsolidated snapshots."""
        print("\nTest: get_unconsolidated_snapshots")

        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Test content 1', 'test-env', 0, 0, NULL)
        """)
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Test content 2', 'test-env', 0, 1, 1)
        """)
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Test content 3', 'test-env', 1, 0, NULL)
        """)
        conn.commit()
        conn.close()

        snapshots = replay.get_unconsolidated_snapshots()

        assert len(snapshots) == 2, (
            f"Expected 2 unconsolidated snapshots, got {len(snapshots)}"
        )

        # Check persona anchor is retrieved
        persona_snap = [s for s in snapshots if s["is_persona_anchor"] == 1]
        assert len(persona_snap) == 1, "Should retrieve persona anchor snapshot"
        assert persona_snap[0]["parent_id"] == 1, "Should retrieve parent_id"

        print(
            "  ✓ PASSED: Retrieved unconsolidated snapshots with persona and parent info"
        )

        import shutil

        shutil.rmtree(temp_dir)

    def test_update_snapshot_consolidation(self):
        """Test updating snapshot consolidation with persona protection."""
        print("\nTest: update_snapshot_consolidation")

        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, importance_score)
            VALUES ('Test content', 'test-env', 0, 0.3)
        """)
        snapshot_id = cursor.lastrowid
        conn.commit()
        conn.close()

        # Test regular snapshot
        causality_json = json.dumps(
            {"pattern": "test", "parent_id": None, "is_persona_anchor": False}
        )
        success = replay.update_snapshot_consolidation(
            snapshot_id, causality_json, is_persona_anchor=0
        )
        assert success, "Update should succeed"

        # Verify importance_score is at least 0.5
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT importance_score, status_consolidated, causality_id
            FROM memory_snapshots WHERE id = ?
        """,
            (snapshot_id,),
        )
        row = cursor.fetchone()
        conn.close()

        assert row[0] == 0.5, f"Expected importance_score=0.5 (min), got {row[0]}"
        assert row[1] == 1, f"Expected status_consolidated=1, got {row[1]}"
        assert row[2] == causality_json, "causality_id should be set"
        print("  ✓ PASSED: Regular snapshot gets minimum 0.5 importance")

        # Test persona anchor
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, importance_score, is_persona_anchor)
            VALUES ('Persona content', 'test-env', 0, 0.0, 1)
        """)
        persona_id = cursor.lastrowid
        conn.commit()
        conn.close()

        causality_json = json.dumps(
            {"pattern": "persona-anchor", "parent_id": None, "is_persona_anchor": True}
        )
        success = replay.update_snapshot_consolidation(
            persona_id, causality_json, is_persona_anchor=1
        )
        assert success, "Persona update should succeed"

        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT importance_score FROM memory_snapshots WHERE id = ?
        """,
            (persona_id,),
        )
        row = cursor.fetchone()
        conn.close()

        assert row[0] == 1.0, f"Expected importance_score=1.0 for persona, got {row[0]}"
        print("  ✓ PASSED: Persona anchor gets 1.0 importance")

        import shutil

        shutil.rmtree(temp_dir)

    def test_causality_json_structure(self):
        """Test that causality_id is valid JSON with required fields."""
        print("\nTest: causality_json_structure")

        replay = hr.HippocampalReplay(data_dir="/tmp")

        test_cases = [
            {
                "content": "Fixed error in code",
                "is_persona_anchor": 0,
                "parent_id": 5,
                "expected_pattern": "error-resolution",
            },
            {
                "content": "Implemented new feature",
                "is_persona_anchor": 0,
                "parent_id": None,
                "expected_pattern": "implementation-complete",
            },
            {
                "content": "Persona definition",
                "is_persona_anchor": 1,
                "parent_id": None,
                "expected_pattern": "persona-anchor",
            },
        ]

        for tc in test_cases:
            snapshot = {
                "content": tc["content"],
                "environment": "test",
                "is_persona_anchor": tc["is_persona_anchor"],
                "parent_id": tc["parent_id"],
            }
            causality_json = replay.extract_causality_with_llm(snapshot, [])

            # Should be valid JSON
            causality_data = json.loads(causality_json)

            # Check required fields
            assert "pattern" in causality_data
            assert "patterns" in causality_data
            assert "timestamp" in causality_data
            assert "parent_id" in causality_data
            assert "is_persona_anchor" in causality_data

            # Check values
            assert causality_data["is_persona_anchor"] == (tc["is_persona_anchor"] == 1)
            assert causality_data["parent_id"] == tc["parent_id"]
            assert tc["expected_pattern"] in causality_data["patterns"]

            if tc["is_persona_anchor"]:
                assert causality_data["patterns"][0] == "persona-anchor"

        print("  ✓ PASSED: JSON structure valid with all required fields")

    def test_persona_anchor_extraction(self):
        """Test that persona anchors get special treatment."""
        print("\nTest: persona_anchor_extraction")

        replay = hr.HippocampalReplay(data_dir="/tmp")

        snapshot = {
            "content": "I am an AI assistant",
            "environment": "test",
            "is_persona_anchor": 1,
            "parent_id": None,
        }

        causality_json = replay.extract_causality_with_llm(snapshot, [])
        causality_data = json.loads(causality_json)

        # Persona anchor should be first pattern
        assert causality_data["patterns"][0] == "persona-anchor", (
            "persona-anchor should be first pattern"
        )
        assert causality_data["is_persona_anchor"] == True

        print("  ✓ PASSED: Persona anchor gets special pattern treatment")

    def test_perform_replay(self):
        """Test full replay process with enhanced features."""
        print("\nTest: perform_replay (with persona and parent)")

        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data with persona and parent
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Parent snapshot
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Parent snapshot', 'test-env', 0, 0, NULL)
        """)
        parent_id = cursor.lastrowid

        # Child snapshot
        cursor.execute(
            """
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Child snapshot', 'test-env', 0, 0, ?)
        """,
            (parent_id,),
        )

        # Persona anchor
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated, is_persona_anchor, parent_id)
            VALUES ('Persona definition', 'test-env', 0, 1, NULL)
        """)

        conn.commit()
        conn.close()

        # Run replay
        processed, succeeded = replay.perform_replay(dry_run=False)

        assert processed == 3, f"Expected 3 processed, got {processed}"
        assert succeeded == 3, f"Expected 3 succeeded, got {succeeded}"

        # Verify updates
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Check persona has importance 1.0
        cursor.execute("""
            SELECT importance_score, causality_id FROM memory_snapshots 
            WHERE is_persona_anchor = 1
        """)
        row = cursor.fetchone()
        assert row[0] == 1.0, f"Persona should have importance 1.0, got {row[0]}"

        # Check causality_id is valid JSON with parent_id
        causality_data = json.loads(row[1])
        assert "parent_id" in causality_data

        # Check child snapshot has parent_id in causality
        cursor.execute(
            """
            SELECT causality_id FROM memory_snapshots WHERE parent_id = ?
        """,
            (parent_id,),
        )
        child_row = cursor.fetchone()
        child_causality = json.loads(child_row[0])
        assert child_causality["parent_id"] == parent_id

        conn.close()

        print("  ✓ PASSED: Full replay with persona protection and parent tracking")

        import shutil

        shutil.rmtree(temp_dir)

    def run_all_tests(self):
        """Run all tests."""
        print("=" * 70)
        print("Hippocampal Replay Test Suite (v2.0 - Enhanced)")
        print("=" * 70)

        tests = [
            self.test_get_current_session_id,
            self.test_get_unconsolidated_snapshots,
            self.test_update_snapshot_consolidation,
            self.test_causality_json_structure,
            self.test_persona_anchor_extraction,
            self.test_perform_replay,
        ]

        passed = 0
        failed = 0

        for test in tests:
            try:
                test()
                passed += 1
            except AssertionError as e:
                print(f"\n✗ FAILED: {e}")
                failed += 1
            except Exception as e:
                print(f"\n✗ ERROR: {e}")
                import traceback

                traceback.print_exc()
                failed += 1

        print("\n" + "=" * 70)
        print(f"Results: {passed} passed, {failed} failed")
        print("=" * 70)

        if failed == 0:
            print("All tests PASSED ✓")
            return True
        else:
            print(f"{failed} test(s) FAILED")
            return False


def main():
    parser = argparse.ArgumentParser(
        description="Hippocampal Replay Test Suite (v2.0 Enhanced)"
    )

    args = parser.parse_args()

    test_suite = TestHippocampalReplay()
    success = test_suite.run_all_tests()

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
