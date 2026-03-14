#!/bin/bash
# Setup cron job for hippocampal replay
# Runs daily at 3:00 AM during sleep cycle

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CRON_JOB="0 3 * * * cd $SCRIPT_DIR && /usr/bin/python3 hippocampal_replay.py >> /var/log/chronos-hippocampal-replay.log 2>&1"

echo "Setting up hippocampal replay cron job..."
echo "Script location: $SCRIPT_DIR/hippocampal_replay.py"
echo "Schedule: Daily at 3:00 AM"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "hippocampal_replay.py"; then
    echo "Cron job already exists."
    echo "Current entry:"
    crontab -l | grep "hippocampal_replay.py"
    read -p "Do you want to replace it? (y/N): " replace
    if [[ ! $replace =~ ^[Yy]$ ]]; then
        echo "Setup cancelled."
        exit 0
    fi
    # Remove existing entry
    crontab -l 2>/dev/null | grep -v "hippocampal_replay.py" | crontab -
fi

# Add new cron job
(crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -

echo "Cron job installed successfully!"
echo ""
echo "To verify, run: crontab -l"
echo "To remove, run: crontab -l | grep -v hippocampal_replay.py | crontab -"
