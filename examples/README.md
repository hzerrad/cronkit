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
cronic timeline --file examples/crontabs/complex.cron --show-overlaps
```

### Use with Stdin
```bash
cat examples/crontabs/simple.cron | cronic list --stdin
cat examples/crontabs/simple.cron | cronic check --stdin
```

## CI/CD Integration Examples

### GitHub Actions
```yaml
- name: Validate crontab
  run: |
    cronic check --file .github/crontab --fail-on warn
```

### Pre-commit Hook
```bash
#!/bin/bash
# Validate crontab before commit
cronic check --file .crontab --fail-on error
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


