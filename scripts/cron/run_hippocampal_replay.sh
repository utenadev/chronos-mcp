#!/bin/bash
# Run Hippocampal Replay
# Usage: ./run_hippocampal_replay.sh [--dry-run] [--verbose]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DATA_DIR="${DATA_DIR:-$HOME/.chronos}"
LOG_FILE="/var/log/chronos-hippocampal-replay.log"

# Parse arguments
DRY_RUN=""
VERBOSE=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --dry-run)
      DRY_RUN="--dry-run"
      shift
      ;;
    --verbose)
      VERBOSE="1"
      shift
      ;;
    --data-dir)
      DATA_DIR="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 [--dry-run] [--verbose] [--data-dir DIR]"
      echo ""
      echo "Options:"
      echo "  --dry-run      Show what would be done without making changes"
      echo "  --verbose      Enable verbose output"
      echo "  --data-dir DIR Specify data directory (default: ~/.chronos)"
      echo "  --help         Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Check if running during sleep cycle (3:00-4:00 AM)
HOUR=$(date +%H)
if [[ "$HOUR" -ge 3 && "$HOUR" -lt 4 ]]; then
  IN_SLEEP_CYCLE="1"
else
  IN_SLEEP_CYCLE="0"
fi

# Log start
echo "========================================" | tee -a "$LOG_FILE"
echo "Hippocampal Replay - $(date '+%Y-%m-%d %H:%M:%S')" | tee -a "$LOG_FILE"
echo "Sleep cycle: $([ "$IN_SLEEP_CYCLE" = "1" ] && echo "YES (3:00-4:00 AM)" || echo "NO")" | tee -a "$LOG_FILE"
echo "Data directory: $DATA_DIR" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"

# Check Python availability
if ! command -v python3 &> /dev/null; then
  echo "ERROR: python3 not found" | tee -a "$LOG_FILE"
  exit 1
fi

# Run hippocampal replay
cd "$PROJECT_DIR"
if [[ "$VERBOSE" = "1" ]]; then
  python3 scripts/hippocampal_replay.py --data-dir "$DATA_DIR" $DRY_RUN 2>&1 | tee -a "$LOG_FILE"
else
  python3 scripts/hippocampal_replay.py --data-dir "$DATA_DIR" $DRY_RUN >> "$LOG_FILE" 2>&1
fi

EXIT_CODE=${PIPESTATUS[0]}

# Log end
echo "" | tee -a "$LOG_FILE"
echo "Exit code: $EXIT_CODE" | tee -a "$LOG_FILE"
echo "Completed at: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"

exit $EXIT_CODE
