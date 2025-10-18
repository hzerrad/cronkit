# Testing Guidelines

This document provides comprehensive guidelines for testing in the cronic project. We follow Test-Driven Development (TDD) and Behavior-Driven Development (BDD) practices.

## Table of Contents

- [Testing Philosophy](#testing-philosophy)
- [Test Types](#test-types)
- [Test Structure](#test-structure)
- [Writing Tests](#writing-tests)
- [Running Tests](#running-tests)
- [Coverage Goals](#coverage-goals)
- [Best Practices](#best-practices)

## Testing Philosophy

### Test-Driven Development (TDD)

We follow the TDD cycle:

1. **Red**: Write a failing test first
2. **Green**: Write minimal code to make the test pass
3. **Refactor**: Improve the code while keeping tests green

### Behavior-Driven Development (BDD)

We use BDD for integration and E2E tests to:
- Write tests in natural language
- Focus on user behavior and outcomes
- Create living documentation

## Test Types

### Unit Tests

**Location**: `internal/*/` (alongside source code)
**Framework**: Go testing + testify
**Purpose**: Test individual functions and methods in isolation

**Example:**
```go
func TestVersionCommand(t *testing.T) {
    t.Run("should have correct name", func(t *testing.T) {
        assert.Equal(t, "version", versionCmd.Use)
    })
}
```

**When to write:**
- Testing pure functions
- Testing business logic
- Testing error handling
- Testing edge cases

### Integration Tests

**Location**: `test/integration/`
**Framework**: Ginkgo + Gomega
**Purpose**: Test how components work together

**Example:**
```go
Describe("CLI Integration Tests", func() {
    Context("when running 'cronic version'", func() {
        It("should display version information", func() {
            command := exec.Command(pathToCLI, "version")
            session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            Eventually(session).Should(gexec.Exit(0))
        })
    })
})
```

**When to write:**
- Testing command execution
- Testing flag parsing
- Testing output formatting
- Testing command interactions

### End-to-End (E2E) Tests

**Location**: `test/e2e/`
**Framework**: Ginkgo + Gomega
**Purpose**: Test complete user workflows

**Example:**
```go
Describe("Complete User Workflow", func() {
    Context("when a new user runs the CLI", func() {
        It("should complete a full workflow", func() {
            By("checking version first")
            // ... test steps

            By("running example command")
            // ... test steps
        })
    })
})
```

**When to write:**
- Testing user scenarios
- Testing multi-step workflows
- Testing system interactions
- Testing error recovery

## Test Structure

### Unit Test Structure

```go
package cmd

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeature(t *testing.T) {
    // Setup (Arrange)
    input := "test"

    // Exercise (Act)
    result := SomeFunction(input)

    // Verify (Assert)
    assert.Equal(t, expected, result)
}
```

### Table-Driven Tests

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := SomeFunction(tt.input)

            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### BDD Test Structure

```go
var _ = Describe("Feature", func() {
    var (
        // Shared variables
        result string
    )

    BeforeEach(func() {
        // Setup before each test
    })

    AfterEach(func() {
        // Cleanup after each test
    })

    Describe("Scenario", func() {
        Context("when condition is met", func() {
            It("should produce expected outcome", func() {
                // Test implementation
                Expect(result).To(Equal("expected"))
            })
        })
    })
})
```

## Writing Tests

### TDD Workflow

1. **Write the test first:**
   ```go
   func TestNewFeature(t *testing.T) {
       result := NewFeature()
       assert.NotNil(t, result)
   }
   ```

2. **Run the test (it should fail):**
   ```bash
   make test-unit
   ```

3. **Write minimal implementation:**
   ```go
   func NewFeature() *Feature {
       return &Feature{}
   }
   ```

4. **Run tests again (should pass):**
   ```bash
   make test-unit
   ```

5. **Refactor if needed**

### BDD Workflow

1. **Write the scenario in natural language:**
   ```go
   Describe("User greeting", func() {
       Context("when user provides a name", func() {
           It("should greet the user by name", func() {
               // Test will be written here
           })
       })
   })
   ```

2. **Implement the test:**
   ```go
   It("should greet the user by name", func() {
       command := exec.Command(pathToCLI, "greet", "--name", "Alice")
       session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
       Expect(err).NotTo(HaveOccurred())
       Eventually(session).Should(gexec.Exit(0))
       Expect(session.Out).To(gbytes.Say("Hello, Alice!"))
   })
   ```

3. **Run the test (should fail)**

4. **Implement the feature**

5. **Run tests again (should pass)**

## Running Tests

### Quick Reference

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run only E2E tests
make test-e2e

# Run all BDD tests
make test-bdd

# Run with coverage
make test-coverage

# Watch mode (auto-run on file changes)
make test-watch
```

### Running Specific Tests

```bash
# Run specific test file
go test -v ./internal/cmd/root_test.go

# Run specific test function
go test -v -run TestRootCommand ./internal/cmd/

# Run specific BDD spec
ginkgo -focus="Version Command" ./test/integration/
```

### Test Flags

```bash
# Verbose output
go test -v ./...

# Race detection
go test -race ./...

# Short mode (skip long tests)
go test -short ./...

# Coverage
go test -cover ./...

# Parallel execution
go test -parallel 4 ./...
```

## Coverage Goals

### Coverage Targets

- **Overall Coverage**: 80% minimum
- **Critical Paths**: 90% minimum
- **New Code**: 100% (all new code must have tests)

### Checking Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage in terminal
go test -cover ./...

# Detailed coverage by function
go tool cover -func=bin/coverage.out

# HTML coverage report
# Opens in browser: bin/coverage.html
```

### Coverage Best Practices

1. **Focus on critical paths first**
2. **Don't test generated code**
3. **Test error cases**
4. **Test edge cases and boundaries**
5. **Don't test third-party code**

## Best Practices

### General Guidelines

1. **Write tests first (TDD)**
   - Define expected behavior before implementation
   - Tests serve as specification and documentation

2. **Keep tests simple and readable**
   - One assertion per test (when practical)
   - Clear test names describing what's being tested

3. **Test behavior, not implementation**
   - Focus on what the code does, not how it does it
   - This makes refactoring easier

4. **Maintain test independence**
   - Tests should not depend on each other
   - Tests should be able to run in any order

5. **Use descriptive names**
   ```go
   // Good
   func TestVersionCommand_ShouldReturnVersionString(t *testing.T)

   // Bad
   func TestVersion(t *testing.T)
   ```

### Unit Test Best Practices

1. **Use testify for assertions**
   ```go
   assert.Equal(t, expected, actual, "optional message")
   require.NoError(t, err) // Stops test on failure
   ```

2. **Use table-driven tests for multiple cases**

3. **Test error cases explicitly**
   ```go
   t.Run("should return error for invalid input", func(t *testing.T) {
       _, err := Function(invalidInput)
       require.Error(t, err)
       assert.Contains(t, err.Error(), "expected message")
   })
   ```

4. **Use subtests for organization**
   ```go
   func TestFeature(t *testing.T) {
       t.Run("positive case", func(t *testing.T) { ... })
       t.Run("negative case", func(t *testing.T) { ... })
       t.Run("edge case", func(t *testing.T) { ... })
   }
   ```

### BDD Best Practices

1. **Use descriptive Context and It blocks**
   ```go
   Context("when user is authenticated", func() {
       It("should allow access to protected resources", func() {
           // Test implementation
       })
   })
   ```

2. **Use By() for multi-step scenarios**
   ```go
   It("should complete workflow", func() {
       By("logging in")
       // ... login steps

       By("performing action")
       // ... action steps

       By("verifying result")
       // ... verification steps
   })
   ```

3. **Use BeforeEach/AfterEach for setup/cleanup**
   ```go
   BeforeEach(func() {
       // Common setup
   })

   AfterEach(func() {
       // Common cleanup
   })
   ```

4. **Use Eventually for async operations**
   ```go
   Eventually(session).Should(gexec.Exit(0))
   Eventually(func() bool {
       return checkCondition()
   }).Should(BeTrue())
   ```

### Mocking and Test Doubles

1. **Use interfaces for dependencies**
   ```go
   type DataStore interface {
       Get(key string) (string, error)
   }

   // Easy to mock in tests
   type MockDataStore struct {}
   func (m *MockDataStore) Get(key string) (string, error) {
       return "mocked value", nil
   }
   ```

2. **Consider using testify/mock for complex mocks**

3. **Keep mocks simple and focused**

### Performance Testing

```go
func BenchmarkFeature(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Feature()
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

## Continuous Improvement

1. **Review test coverage regularly**
2. **Update tests when requirements change**
3. **Refactor tests as you refactor code**
4. **Share testing knowledge with the team**
5. **Keep tests fast (unit tests < 100ms)**

## Resources

- [Go Testing Package](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)

## Getting Help

If you have questions about testing:
1. Check this documentation
2. Review existing tests for examples
3. Ask in team discussions
4. Refer to external resources linked above
