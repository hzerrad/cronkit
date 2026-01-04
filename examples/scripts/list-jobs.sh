#!/bin/bash
# Example script: List jobs from stdin or file
# Usage: ./list-jobs.sh [crontab-file] OR cat file.cron | ./list-jobs.sh

set -e

if [ -t 0 ]; then
    # stdin is a terminal, use file argument
    if [ -z "$1" ]; then
        echo "Usage: $0 [crontab-file] OR cat file.cron | $0"
        exit 1
    fi
    cronic list --file "$1" --json
else
    # stdin is piped, read from stdin
    cronic list --stdin --json
fi


