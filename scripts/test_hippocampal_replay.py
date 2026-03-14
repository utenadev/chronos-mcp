#!/usr/bin/env python3
"""
Tests for Hippocampal Replay Script

Usage:
    python3 test_hippocampal_replay.py [--test-database] [--test-llm] [--integration]
"""

import argparse
import os
import sys
import tempfile
import sqlite3
from datetime import datetime

# Import the main module
import hippocampal_replay as hr


class TestHippocampalReplay:
    """Test suite for hippocampal replay functionality."""

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

    def test_get_unconsolidated_snapshots(self):
        """Test retrieving unconsolidated snapshots."""
        print("\nTest: get_unconsolidated_snapshots")

        # Setup
        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated)
            VALUES ('Test content 1', 'test-env', 0)
        """)
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated)
            VALUES ('Test content 2', 'test-env', 0)
        """)
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated)
            VALUES ('Test content 3', 'test-env', 1)
        """)
        conn.commit()
        conn.close()

        # Test
        snapshots = replay.get_unconsolidated_snapshots()

        assert len(snapshots) == 2, (
            f"Expected 2 unconsolidated snapshots, got {len(snapshots)}"
        )
        for snap in snapshots:
            assert snap["status_consolidated"] == 0, (
                "All returned snapshots should be unconsolidated"
            )

        print("  ✓ PASSED: Retrieved only unconsolidated snapshots")

        # Cleanup
        import shutil

        shutil.rmtree(temp_dir)

    def test_update_snapshot_consolidation(self):
        """Test updating snapshot consolidation status."""
        print("\nTest: update_snapshot_consolidation")

        # Setup
        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO memory_snapshots (content, environment, status_consolidated)
            VALUES ('Test content', 'test-env', 0)
        """)
        snapshot_id = cursor.lastrowid
        conn.commit()
        conn.close()

        # Test
        causality_id = "test-causality-20240314"
        success = replay.update_snapshot_consolidation(snapshot_id, causality_id)

        assert success, "Update should succeed"

        # Verify
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT causality_id, status_consolidated
            FROM memory_snapshots WHERE id = ?
        """,
            (snapshot_id,),
        )
        row = cursor.fetchone()
        conn.close()

        assert row[0] == causality_id, (
            f"Expected causality_id={causality_id}, got {row[0]}"
        )
        assert row[1] == 1, f"Expected status_consolidated=1, got {row[1]}"

        print("  ✓ PASSED: Successfully updated snapshot consolidation")

        # Cleanup
        import shutil

        shutil.rmtree(temp_dir)

    def test_extract_causality_patterns(self):
        """Test causality pattern extraction."""
        print("\nTest: extract_causality_patterns")

        replay = hr.HippocampalReplay(data_dir="/tmp")

        # Test different patterns
        test_cases = [
            ("Fixed error in authentication", "error-resolution"),
            ("Implemented new feature", "implementation-complete"),
            ("Decided to use Python", "learning-decision"),
            ("General conversation about nothing", "general-conversation"),
        ]

        for content, expected_pattern in test_cases:
            snapshot = {"content": content, "environment": "test"}
            causality_id = replay.extract_causality_with_llm(snapshot, [])

            assert expected_pattern in causality_id, (
                f"Expected '{expected_pattern}' in causality_id, got {causality_id}"
            )

        print("  ✓ PASSED: Correctly extracted causality patterns")

    def test_perform_replay(self):
        """Test full replay process."""
        print("\nTest: perform_replay (dry run)")

        # Setup
        temp_dir = self.setup_test_db()
        replay = hr.HippocampalReplay(data_dir=temp_dir)

        # Insert test data
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        for i in range(3):
            cursor.execute(
                """
                INSERT INTO memory_snapshots (content, environment, status_consolidated)
                VALUES (?, ?, 0)
            """,
                (f"Test content {i}", f"test-env-{i}"),
            )
        conn.commit()
        conn.close()

        # Test (dry run)
        processed, succeeded = replay.perform_replay(dry_run=True)

        assert processed == 3, f"Expected 3 processed, got {processed}"
        assert succeeded == 3, f"Expected 3 succeeded, got {succeeded}"

        print("  ✓ PASSED: Dry run processed all snapshots")

        # Test (actual run)
        processed, succeeded = replay.perform_replay(dry_run=False)

        assert processed == 3, f"Expected 3 processed, got {processed}"
        assert succeeded == 3, f"Expected 3 succeeded, got {succeeded}"

        # Verify updates
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        cursor.execute(
            "SELECT COUNT(*) FROM memory_snapshots WHERE status_consolidated = 0"
        )
        unconsolidated = cursor.fetchone()[0]
        cursor.execute(
            "SELECT COUNT(*) FROM memory_snapshots WHERE status_consolidated = 1"
        )
        consolidated = cursor.fetchone()[0]
        conn.close()

        assert unconsolidated == 0, f"Expected 0 unconsolidated, got {unconsolidated}"
        assert consolidated == 3, f"Expected 3 consolidated, got {consolidated}"

        print("  ✓ PASSED: Successfully consolidated all snapshots")

        # Cleanup
        import shutil

        shutil.rmtree(temp_dir)

    def run_all_tests(self):
        """Run all tests."""
        print("=" * 60)
        print("Hippocampal Replay Test Suite")
        print("=" * 60)

        try:
            self.test_get_unconsolidated_snapshots()
            self.test_update_snapshot_consolidation()
            self.test_extract_causality_patterns()
            self.test_perform_replay()

            print("\n" + "=" * 60)
            print("All tests PASSED ✓")
            print("=" * 60)
            return True

        except AssertionError as e:
            print(f"\n✗ Test FAILED: {e}")
            return False
        except Exception as e:
            print(f"\n✗ Test ERROR: {e}")
            import traceback

            traceback.print_exc()
            return False


def main():
    parser = argparse.ArgumentParser(description="Hippocampal Replay Test Suite")
    parser.add_argument(
        "--data-dir", default=None, help="Data directory (default: ~/.chronos)"
    )

    args = parser.parse_args()

    # Run tests
    test_suite = TestHippocampalReplay()
    success = test_suite.run_all_tests()

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
