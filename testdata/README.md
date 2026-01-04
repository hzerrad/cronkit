# Test Data

This directory contains test fixtures used by unit, integration, and E2E tests.

## Structure

```
testdata/
├── crontab/
│   ├── valid/          # Valid crontab files
│   ├── invalid/        # Invalid crontab files (for error testing)
│   ├── edge-cases/     # Edge case scenarios
│   ├── performance/    # Large crontabs for performance testing
│   ├── sample.cron     # Sample crontab with various patterns
│   ├── empty.cron      # Empty crontab
│   └── invalid.cron    # Crontab with invalid entries
└── expressions.json    # Test cron expressions
```

## Usage in Tests

### Loading Test Fixtures

```go
import "github.com/hzerrad/cronic/internal/testutil"

// Load a test crontab
path := testutil.LoadTestCrontab("sample.cron")

// Create temporary crontab
file, cleanup := testutil.CreateTempCrontab(t, "0 2 * * * /usr/bin/backup.sh")
defer cleanup()
```

### Direct File Access

```go
// In tests, use relative paths from test file location
testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
```

## Test Fixture Guidelines

- **Valid fixtures**: Should contain valid cron expressions for positive testing
- **Invalid fixtures**: Should contain various types of invalid entries for error testing
- **Edge cases**: Should cover boundary conditions and unusual but valid patterns
- **Performance fixtures**: Should be large (100+ jobs) for performance benchmarking

## Adding New Fixtures

1. Place files in appropriate subdirectory
2. Use descriptive names (e.g., `dom-dow-conflict.cron`)
3. Add comments explaining the test scenario
4. Update this README if adding new categories


