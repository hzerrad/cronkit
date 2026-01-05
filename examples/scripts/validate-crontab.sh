#!/bin/bash
# Example script: Validate a crontab file
# Usage: ./validate-crontab.sh [crontab-file]

set -e

CRONTAB_FILE="${1:-/etc/crontab}"

echo "Validating crontab: $CRONTAB_FILE"
echo "================================"

if cronic check --file "$CRONTAB_FILE" --verbose; then
    echo ""
    echo "✓ Crontab validation passed"
    exit 0
else
    EXIT_CODE=$?
    echo ""
    echo "✗ Crontab validation failed (exit code: $EXIT_CODE)"
    exit $EXIT_CODE
fi



