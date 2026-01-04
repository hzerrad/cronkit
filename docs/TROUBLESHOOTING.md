# Troubleshooting Guide

This guide helps you resolve common issues when using Cronic.

## Common Errors

### Invalid Cron Expression

**Error:** `failed to parse expression: expected 5 fields`

**Cause:** The cron expression doesn't have exactly 5 fields.

**Solution:**
- Ensure your expression has 5 fields: `minute hour day-of-month month day-of-week`
- Example: `0 9 * * *` (not `0 9 * *`)

**Example:**
```bash
# Wrong
$ cronic explain "0 9 * *"
Error: failed to parse expression: expected 5 fields

# Correct
$ cronic explain "0 9 * * *"
At 09:00 daily
```

### Invalid Timezone

**Error:** `invalid timezone: unknown time zone <timezone>`

**Cause:** The timezone name is not a valid IANA timezone.

**Solution:**
- Use IANA timezone names (e.g., `America/New_York`, `Europe/London`, `UTC`)
- Common timezones:
  - `UTC` - Coordinated Universal Time
  - `America/New_York` - Eastern Time (US)
  - `America/Los_Angeles` - Pacific Time (US)
  - `Europe/London` - British Time
  - `Asia/Tokyo` - Japan Standard Time

**Example:**
```bash
# Wrong
$ cronic next "0 9 * * *" --timezone EST
Error: invalid timezone: unknown time zone EST

# Correct
$ cronic next "0 9 * * *" --timezone America/New_York
Next 10 runs for "0 9 * * *" (At 09:00 daily):
...
```

### File Not Found

**Error:** `failed to read crontab file: no such file or directory`

**Cause:** The specified crontab file doesn't exist.

**Solution:**
- Check the file path is correct
- Ensure you have read permissions for the file
- Use absolute paths if relative paths don't work

**Example:**
```bash
# Check if file exists
$ ls -l /etc/crontab

# Use absolute path
$ cronic list --file /etc/crontab
```

### Stdin Reading Issues

**Error:** `failed to read crontab from stdin: <error>`

**Cause:** Stdin is empty or contains invalid data.

**Solution:**
- Ensure data is being piped correctly
- Use `--stdin` flag explicitly if automatic detection fails
- Check that stdin is not a terminal (use pipes or redirection)

**Example:**
```bash
# Correct usage
$ echo "0 2 * * * /usr/bin/backup.sh" | cronic list --stdin

# Or with explicit flag
$ cat file.cron | cronic check --stdin
```

### Permission Denied

**Error:** `failed to read user crontab: permission denied`

**Cause:** Cannot execute `crontab -l` command.

**Solution:**
- Ensure `crontab` command is available in PATH
- Check user permissions
- Use `--file` flag to read from a file instead

**Example:**
```bash
# Use file instead of user crontab
$ cronic list --file /path/to/crontab
```

## Timezone Issues

### Times Don't Match Expected Values

**Symptom:** Next run times don't match what you expect.

**Cause:** Timezone mismatch between your system and the cron job's timezone.

**Solution:**
- Use `--timezone` flag to specify the correct timezone
- Check your system's timezone: `date`
- Verify the cron job's intended timezone

**Example:**
```bash
# Check system timezone
$ date
Mon Dec 28 12:00:00 PST 2025

# Use specific timezone
$ cronic next "0 9 * * *" --timezone UTC
```

### Daylight Saving Time Confusion

**Symptom:** Times shift unexpectedly during DST transitions.

**Cause:** IANA timezones handle DST automatically, which may cause confusion.

**Solution:**
- Use `UTC` for timezone-independent calculations
- Be aware that times will show in the specified timezone's current offset
- Check DST transition dates for your timezone

## Stdin Reading Problems

### Stdin Not Detected Automatically

**Symptom:** Command tries to read user crontab instead of stdin.

**Cause:** Stdin detection may fail in some environments.

**Solution:**
- Use `--stdin` flag explicitly
- Ensure stdin is not a terminal (use pipes)

**Example:**
```bash
# Explicit flag
$ cat file.cron | cronic list --stdin

# Automatic detection (should work)
$ cat file.cron | cronic list
```

### Empty Stdin

**Symptom:** Command succeeds but shows no jobs.

**Cause:** Stdin is empty or contains only whitespace.

**Solution:**
- Verify the input source has content
- Check for hidden characters or encoding issues

**Example:**
```bash
# Check if stdin has content
$ cat file.cron | wc -l

# Try with explicit stdin
$ cat file.cron | cronic list --stdin --all
```

## Performance Issues with Large Crontabs

### Slow Processing

**Symptom:** Commands take a long time with large crontabs (100+ jobs).

**Cause:** Large crontabs require more processing time.

**Solution:**
- Use `--json` flag for faster output (no formatting)
- Process specific files instead of user crontab
- Consider splitting large crontabs into multiple files

**Example:**
```bash
# Faster JSON output
$ cronic list --file large.cron --json > output.json

# Check specific file
$ cronic check --file specific-jobs.cron
```

### Memory Issues

**Symptom:** Out of memory errors with very large crontabs.

**Cause:** Very large crontabs (1000+ jobs) may consume significant memory.

**Solution:**
- Process crontabs in smaller chunks
- Use streaming processing where possible
- Report the issue if it persists

## JSON Parsing Errors

### Invalid JSON Output

**Symptom:** JSON output cannot be parsed.

**Cause:** Output may contain non-JSON content (errors, warnings).

**Solution:**
- Check exit codes (non-zero may indicate errors)
- Redirect stderr to separate file
- Validate JSON with `jq` or similar tools

**Example:**
```bash
# Separate stdout and stderr
$ cronic list --file file.cron --json > output.json 2> errors.txt

# Validate JSON
$ cronic list --file file.cron --json | jq .
```

### Schema Mismatch

**Symptom:** JSON structure doesn't match expected schema.

**Cause:** Version mismatch or schema changes.

**Solution:**
- Check Cronic version: `cronic version`
- Review JSON schema documentation: `docs/JSON_SCHEMAS.md`
- Update your parsing code if needed

## Validation False Positives/Negatives

### DOM/DOW Conflict Not Detected

**Symptom:** Expression with both DOM and DOW doesn't show warning.

**Cause:** `--verbose` flag not used, or severity filtering.

**Solution:**
- Use `--verbose` flag to see warnings
- Check `--fail-on` setting
- Review diagnostic codes documentation

**Example:**
```bash
# Show warnings
$ cronic check "0 0 1 * 1" --verbose

# Check with specific fail-on level
$ cronic check "0 0 1 * 1" --fail-on warn
```

### Empty Schedule Not Detected

**Symptom:** Expression that never runs shows as valid.

**Cause:** Empty schedule detection may have limitations.

**Solution:**
- Report the specific expression
- Use `--verbose` flag
- Check if expression actually runs with `next` command

**Example:**
```bash
# Check if expression runs
$ cronic next "problematic-expression" --count 100

# Validate with verbose
$ cronic check "problematic-expression" --verbose
```

## Getting Help

If you encounter issues not covered here:

1. **Check the documentation:**
   - `README.md` - Main documentation
   - `docs/JSON_SCHEMAS.md` - JSON output schemas
   - `docs/DEVELOPMENT.md` - Development guide

2. **Run with verbose output:**
   ```bash
   cronic <command> --verbose
   ```

3. **Check exit codes:**
   - `0` - Success
   - `1` - Error
   - `2` - Warning (with `--fail-on warn` or `--verbose`)

4. **Report issues:**
   - Include the command and arguments used
   - Include error messages
   - Include your system information (OS, Go version)
   - Include sample crontab if applicable

## Common Patterns

### Validating Before Deployment

```bash
# Validate crontab before deploying
cronic check --file new-crontab.cron --fail-on warn

# Exit code 0 = safe to deploy
# Exit code 1 = errors found, don't deploy
# Exit code 2 = warnings found (configurable)
```

### CI/CD Integration

```bash
# In CI pipeline
if ! cronic check --file .github/crontab --fail-on error; then
    echo "Crontab validation failed"
    exit 1
fi
```

### Debugging Cron Issues

```bash
# Explain what the expression does
cronic explain "problematic-expression"

# Show when it will run
cronic next "problematic-expression" --count 20

# Check for issues
cronic check "problematic-expression" --verbose

# Visualize schedule
cronic timeline "problematic-expression" --view day
```


