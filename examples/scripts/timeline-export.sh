#!/bin/bash
# Example script: Generate timeline and export to file
# Usage: ./timeline-export.sh [crontab-file] [output-file]

set -e

CRONTAB_FILE="${1:-/etc/crontab}"
OUTPUT_FILE="${2:-timeline-$(date +%Y%m%d-%H%M%S).txt}"

echo "Generating timeline for: $CRONTAB_FILE"
echo "Exporting to: $OUTPUT_FILE"

cronic timeline --file "$CRONTAB_FILE" \
    --view day \
    --show-overlaps \
    --export "$OUTPUT_FILE"

echo "✓ Timeline exported to $OUTPUT_FILE"

# Also generate JSON version
JSON_FILE="${OUTPUT_FILE%.txt}.json"
cronic timeline --file "$CRONTAB_FILE" \
    --view day \
    --show-overlaps \
    --json \
    --export "$JSON_FILE"

echo "✓ JSON timeline exported to $JSON_FILE"


