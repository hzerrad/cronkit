# Cronic Examples

This directory contains example crontab files and scripts demonstrating how to use Cronic.

## Crontab Examples

### `simple.cron`
A basic crontab with common cron patterns:
- Daily jobs
- Interval patterns
- Day-of-week specifications

### `complex.cron`
A more comprehensive example showing:
- Environment variables
- Business hours patterns
- Cron aliases (@hourly, @daily)
- Complex commands with pipes

### `with-comments.cron`
An example demonstrating best practices for documenting crontabs:
- Section headers
- Inline comments
- Job descriptions

## Script Examples

### `validate-crontab.sh`
Validates a crontab file and exits with appropriate exit codes for CI/CD integration.

**Usage:**
```bash
./validate-crontab.sh /etc/crontab
# or
cat my-crontab.cron | cronic check --stdin
```

### `list-jobs.sh`
Lists jobs from a file or stdin in JSON format.

**Usage:**
```bash
./list-jobs.sh /etc/crontab
# or
cat my-crontab.cron | ./list-jobs.sh
```

### `timeline-export.sh`
Generates a timeline visualization and exports it to both text and JSON formats.

**Usage:**
```bash
./timeline-export.sh /etc/crontab timeline-output
```

## Using Examples

### Validate an Example Crontab
```bash
cronic check --file examples/crontabs/simple.cron
```

### Explain Jobs from an Example
```bash
cronic list --file examples/crontabs/complex.cron
```

### Generate Timeline
```bash
# Basic timeline
cronic timeline --file examples/crontabs/complex.cron

# Show overlap information (v0.2.0)
cronic timeline --file examples/crontabs/complex.cron --show-overlaps

# Timeline with timezone
cronic timeline --file examples/crontabs/complex.cron --timezone America/New_York
```

### Use with Stdin
```bash
cat examples/crontabs/simple.cron | cronic list --stdin
cat examples/crontabs/simple.cron | cronic check --stdin
```

### Advanced Validation (v0.2.0)
```bash
# Group issues by severity
cronic check --file examples/crontabs/complex.cron --group-by severity --verbose

# Group issues by line number
cronic check --file examples/crontabs/complex.cron --group-by line --verbose

# Fail on warnings in CI/CD
cronic check --file examples/crontabs/complex.cron --fail-on warn --verbose

# Check with JSON output
cronic check --file examples/crontabs/complex.cron --json --verbose
```

## CI/CD Integration Examples

### GitHub Actions
```yaml
- name: Validate crontab
  run: |
    cronic check --file .github/crontab --fail-on warn --verbose

# With grouped output
- name: Validate crontab with grouped issues
  run: |
    cronic check --file .github/crontab --group-by severity --fail-on warn --verbose
```

### Pre-commit Hook
```bash
#!/bin/bash
# Validate crontab before commit
cronic check --file .crontab --fail-on error

# Or with verbose output
cronic check --file .crontab --fail-on warn --verbose --group-by severity
```

### Automated Documentation
```bash
#!/bin/bash
# Generate documentation for all crontabs
for file in crontabs/*.cron; do
    echo "=== $file ==="
    cronic list --file "$file" --all
    echo ""
done > CRONTAB_DOCS.md
```

### Timeline Analysis (v0.2.0)
```bash
#!/bin/bash
# Generate timeline with overlap analysis for all crontabs
for file in crontabs/*.cron; do
    echo "=== Timeline for $file ==="
    cronic timeline --file "$file" --show-overlaps
    echo ""
done > TIMELINE_ANALYSIS.txt
```

### Validation with Grouping (v0.2.0)
```bash
#!/bin/bash
# Validate all crontabs and group issues by severity
for file in crontabs/*.cron; do
    echo "=== Validating $file ==="
    cronic check --file "$file" --group-by severity --verbose
    echo ""
done
```


