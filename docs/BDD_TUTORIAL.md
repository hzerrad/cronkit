# BDD with Ginkgo Tutorial for Cronic

A practical guide to writing Behavior-Driven Development (BDD) tests using Ginkgo and Gomega for the cronic CLI project.

## Table of Contents

- [What is BDD?](#what-is-bdd)
- [Quick Start](#quick-start)
- [Writing Your First BDD Test](#writing-your-first-bdd-test)
- [Ginkgo Structure](#ginkgo-structure)
- [Practical Examples](#practical-examples)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)

## What is BDD?

BDD (Behavior-Driven Development) focuses on describing **behavior** in natural language before writing tests. With Ginkgo, we write tests that read like specifications:

```
Given a user wants to greet someone
When they run 'cronic greet --name Alice'
Then they should see 'Hello, Alice!'
```

## Quick Start

### 1. Create a New BDD Test Suite

```bash
# Navigate to where you want to create tests
cd test/integration

# Generate a new test suite
ginkgo bootstrap

# Generate a spec file for your feature
ginkgo generate user_greeting
```

This creates:
- `integration_suite_test.go` - Test suite bootstrap
- `user_greeting_test.go` - Your feature spec

### 2. Run the Tests

```bash
# Run all BDD tests
make test-bdd

# Run specific suite
make test-integration

# Run with Ginkgo directly
ginkgo -v ./test/integration/
```

## Writing Your First BDD Test

Let's add a new "greet" command to cronic using BDD/TDD.

### Step 1: Write the BDD Specification First

Create `test/integration/greet_command_test.go`:

```go
package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Greet Command", func() {
	var (
		pathToCLI string
	)

	BeforeSuite(func() {
		var err error
		pathToCLI, err = gexec.Build("github.com/hzerrad/cronic/cmd/cronic")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	Describe("greeting users", func() {
		Context("when a user provides their name", func() {
			It("should greet them by name", func() {
				command := exec.Command(pathToCLI, "greet", "--name", "Alice")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Hello, Alice!"))
			})
		})

		Context("when no name is provided", func() {
			It("should show an error message", func() {
				command := exec.Command(pathToCLI, "greet")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("name is required"))
			})
		})

		Context("when using different greetings", func() {
			It("should support custom greeting styles", func() {
				command := exec.Command(pathToCLI, "greet",
					"--name", "Bob",
					"--style", "formal")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("Good day, Bob!"))
			})
		})
	})
})
```

### Step 2: Run the Test (It Should Fail - RED)

```bash
make test-integration
# ❌ Should fail because 'greet' command doesn't exist yet
```

### Step 3: Implement the Feature

Create `internal/cmd/greet.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Greet a user by name",
	Long:  `Greet command provides personalized greetings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("name is required")
		}

		style, _ := cmd.Flags().GetString("style")

		var greeting string
		switch style {
		case "formal":
			greeting = fmt.Sprintf("Good day, %s!", name)
		default:
			greeting = fmt.Sprintf("Hello, %s!", name)
		}

		fmt.Fprintln(cmd.OutOrStdout(), greeting)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
	greetCmd.Flags().StringP("name", "n", "", "Name to greet")
	greetCmd.Flags().StringP("style", "s", "casual", "Greeting style (casual, formal)")
}
```

### Step 4: Run the Test Again (Should Pass - GREEN)

```bash
make test-integration
# ✅ All tests should pass!
```

### Step 5: Add Unit Tests (TDD)

Create `internal/cmd/greet_test.go`:

```go
package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreetCommand(t *testing.T) {
	t.Run("should greet user by name", func(t *testing.T) {
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"greet", "--name", "Alice"})

		err := rootCmd.Execute()
		require.NoError(t, err)

		assert.Contains(t, buf.String(), "Hello, Alice!")
	})

	t.Run("should return error when name is missing", func(t *testing.T) {
		rootCmd.SetArgs([]string{"greet"})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("should support formal greeting style", func(t *testing.T) {
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"greet", "--name", "Bob", "--style", "formal"})

		err := rootCmd.Execute()
		require.NoError(t, err)

		assert.Contains(t, buf.String(), "Good day, Bob!")
	})
}
```

## Ginkgo Structure

### Core Building Blocks

```go
Describe("Feature Name", func() {
    // Top-level: What are we testing?

    Context("when specific condition exists", func() {
        // Context: Under what circumstances?

        It("should produce expected outcome", func() {
            // It: What should happen?

            // Arrange - Setup
            command := exec.Command(pathToCLI, "version")

            // Act - Execute
            session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())

            // Assert - Verify
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("cronic"))
        })
    })
})
```

### Setup and Teardown

```go
var _ = Describe("Feature", func() {
    var tempDir string

    // Runs ONCE before all tests in this Describe
    BeforeSuite(func() {
        // Build CLI binary, setup database, etc.
    })

    // Runs ONCE after all tests in this Describe
    AfterSuite(func() {
        // Cleanup build artifacts, close connections, etc.
    })

    // Runs BEFORE each It() block
    BeforeEach(func() {
        var err error
        tempDir, err = os.MkdirTemp("", "test-*")
        Expect(err).NotTo(HaveOccurred())
    })

    // Runs AFTER each It() block
    AfterEach(func() {
        os.RemoveAll(tempDir)
    })

    It("test something", func() {
        // tempDir is available here
    })
})
```

## Practical Examples

### Example 1: Testing CLI Flags

```go
var _ = Describe("Config Command", func() {
    Describe("setting configuration values", func() {
        Context("when setting a valid key-value pair", func() {
            It("should save the configuration", func() {
                command := exec.Command(pathToCLI,
                    "config", "set",
                    "--key", "api_token",
                    "--value", "secret123")
                session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

                Expect(err).NotTo(HaveOccurred())
                Eventually(session).Should(gexec.Exit(0))
                Expect(session.Out).To(gbytes.Say("Configuration saved"))
            })
        })

        Context("when key is missing", func() {
            It("should show an error", func() {
                command := exec.Command(pathToCLI, "config", "set", "--value", "secret")
                session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

                Expect(err).NotTo(HaveOccurred())
                Eventually(session).Should(gexec.Exit(1))
                Expect(session.Err).To(gbytes.Say("key is required"))
            })
        })
    })
})
```

### Example 2: Testing File Operations

```go
var _ = Describe("Init Command", func() {
    var tempDir string

    BeforeEach(func() {
        var err error
        tempDir, err = os.MkdirTemp("", "cronic-test-*")
        Expect(err).NotTo(HaveOccurred())
    })

    AfterEach(func() {
        os.RemoveAll(tempDir)
    })

    Describe("initializing a new project", func() {
        Context("when run in an empty directory", func() {
            It("should create a config file", func() {
                command := exec.Command(pathToCLI, "init")
                command.Dir = tempDir
                session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

                Expect(err).NotTo(HaveOccurred())
                Eventually(session).Should(gexec.Exit(0))

                // Verify file was created
                configPath := filepath.Join(tempDir, ".cronic.yaml")
                Expect(configPath).To(BeAnExistingFile())

                // Verify file contents
                content, err := os.ReadFile(configPath)
                Expect(err).NotTo(HaveOccurred())
                Expect(string(content)).To(ContainSubstring("version:"))
            })
        })

        Context("when config file already exists", func() {
            It("should not overwrite existing config", func() {
                // Create existing config
                configPath := filepath.Join(tempDir, ".cronic.yaml")
                err := os.WriteFile(configPath, []byte("existing: true"), 0644)
                Expect(err).NotTo(HaveOccurred())

                command := exec.Command(pathToCLI, "init")
                command.Dir = tempDir
                session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

                Expect(err).NotTo(HaveOccurred())
                Eventually(session).Should(gexec.Exit(1))
                Expect(session.Err).To(gbytes.Say("already initialized"))

                // Verify original content preserved
                content, err := os.ReadFile(configPath)
                Expect(err).NotTo(HaveOccurred())
                Expect(string(content)).To(ContainSubstring("existing: true"))
            })
        })
    })
})
```

### Example 3: Testing Multi-Step Workflows

```go
var _ = Describe("User Workflow", func() {
    Describe("complete task management flow", func() {
        It("should allow creating, listing, and completing tasks", func() {
            By("creating a new task")
            cmd := exec.Command(pathToCLI, "task", "add", "Buy groceries")
            session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("Task added"))

            By("listing all tasks")
            cmd = exec.Command(pathToCLI, "task", "list")
            session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("Buy groceries"))
            Expect(session.Out).To(gbytes.Say("pending"))

            By("completing the task")
            cmd = exec.Command(pathToCLI, "task", "complete", "1")
            session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            Eventually(session).Should(gexec.Exit(0))

            By("verifying task is marked complete")
            cmd = exec.Command(pathToCLI, "task", "list")
            session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("completed"))
        })
    })
})
```

### Example 4: Testing with Different Matchers

```go
var _ = Describe("Output Validation", func() {
    It("demonstrates various Gomega matchers", func() {
        command := exec.Command(pathToCLI, "stats")
        session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
        Expect(err).NotTo(HaveOccurred())

        Eventually(session).Should(gexec.Exit(0))

        // String matching
        Expect(session.Out).To(gbytes.Say("Total:"))
        Expect(session.Out).NotTo(gbytes.Say("Error"))

        // Regular expressions
        Expect(session.Out).To(gbytes.Say("Total: \\d+"))

        // Multiple conditions
        Expect(session.Out).To(
            And(
                gbytes.Say("Total:"),
                gbytes.Say("Active:"),
            ),
        )

        // Get full output as string
        output := string(session.Out.Contents())
        Expect(output).To(ContainSubstring("statistics"))
        Expect(output).To(HavePrefix("Statistics"))
        Expect(output).To(HaveSuffix("Done\n"))
    })
})
```

## Best Practices

### 1. Use Descriptive Names

```go
// ❌ Bad
Describe("Test 1", func() {
    It("works", func() { ... })
})

// ✅ Good
Describe("User Authentication", func() {
    Context("when credentials are valid", func() {
        It("should grant access to the system", func() { ... })
    })
})
```

### 2. One Assertion Per It (When Possible)

```go
// ✅ Good - Each test has a clear purpose
Context("when validating input", func() {
    It("should accept valid email", func() { ... })
    It("should reject invalid email", func() { ... })
    It("should require email field", func() { ... })
})
```

### 3. Use By() for Multi-Step Tests

```go
It("should complete registration workflow", func() {
    By("submitting registration form")
    // ... registration code

    By("receiving confirmation email")
    // ... email verification code

    By("activating account")
    // ... activation code
})
```

### 4. Use Table-Driven Tests with DescribeTable

```go
DescribeTable("greeting styles",
    func(style string, expectedOutput string) {
        command := exec.Command(pathToCLI, "greet",
            "--name", "Alice",
            "--style", style)
        session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
        Expect(err).NotTo(HaveOccurred())
        Eventually(session).Should(gexec.Exit(0))
        Expect(session.Out).To(gbytes.Say(expectedOutput))
    },
    Entry("casual style", "casual", "Hello, Alice!"),
    Entry("formal style", "formal", "Good day, Alice!"),
    Entry("friendly style", "friendly", "Hey Alice!"),
)
```

### 5. Use Focused and Pending Tests

```go
// Focus on specific test during development
FIt("should test this one thing", func() { ... })

// Mark test as pending/TODO
PIt("should implement this feature later", func() { ... })

// Skip a test
XIt("skip this test temporarily", func() { ... })
```

### 6. Use Eventually for Async Operations

```go
// ✅ Good - Wait for async operation
Eventually(func() bool {
    return fileExists(configPath)
}).Should(BeTrue())

// With custom timeout and polling interval
Eventually(func() string {
    return readLogFile()
}, 5*time.Second, 100*time.Millisecond).Should(ContainSubstring("ready"))
```

## Common Patterns

### Pattern 1: Shared Test Fixtures

```go
// test/integration/fixtures.go
package integration_test

func CreateTestUser() map[string]string {
    return map[string]string{
        "name":  "Test User",
        "email": "test@example.com",
    }
}

// Use in tests
It("should work with test user", func() {
    user := CreateTestUser()
    // ... use user in test
})
```

### Pattern 2: Custom Matchers

```go
func BeValidJSON() types.GomegaMatcher {
    return WithTransform(func(s string) bool {
        var js json.RawMessage
        return json.Unmarshal([]byte(s), &js) == nil
    }, BeTrue())
}

// Usage
It("should output valid JSON", func() {
    output := string(session.Out.Contents())
    Expect(output).To(BeValidJSON())
})
```

### Pattern 3: Reusable Test Helpers

```go
func RunCommand(args ...string) *gexec.Session {
    command := exec.Command(pathToCLI, args...)
    session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
    Expect(err).NotTo(HaveOccurred())
    return session
}

// Usage
It("should work", func() {
    session := RunCommand("greet", "--name", "Alice")
    Eventually(session).Should(gexec.Exit(0))
})
```

## Quick Reference

### Common Matchers

```go
// Equality
Expect(actual).To(Equal(expected))
Expect(actual).To(BeIdenticalTo(expected))

// Comparison
Expect(5).To(BeNumerically(">", 3))
Expect(10).To(BeNumerically("~", 11, 2)) // Within delta

// Strings
Expect(str).To(ContainSubstring("hello"))
Expect(str).To(HavePrefix("Hello"))
Expect(str).To(HaveSuffix("world"))
Expect(str).To(MatchRegexp("\\d+"))

// Collections
Expect(slice).To(ContainElement("item"))
Expect(slice).To(HaveLen(5))
Expect(slice).To(BeEmpty())
Expect(map).To(HaveKey("key"))
Expect(map).To(HaveKeyWithValue("key", "value"))

// Errors
Expect(err).NotTo(HaveOccurred())
Expect(err).To(MatchError("specific error"))

// Files
Expect(path).To(BeAnExistingFile())
Expect(path).To(BeADirectory())

// Async
Eventually(func() bool { ... }).Should(BeTrue())
Consistently(func() bool { ... }).Should(BeTrue())
```

### Running Tests

```bash
# Run all tests
ginkgo -r

# Run with verbosity
ginkgo -v -r

# Run specific suite
ginkgo ./test/integration

# Run focused tests only
ginkgo -focus="greet command" ./test/integration

# Skip certain tests
ginkgo -skip="slow tests" -r

# Run tests in parallel
ginkgo -p -r

# Generate coverage
ginkgo -cover -r
```

## Conclusion

BDD with Ginkgo helps you:
- **Write tests in natural language** that non-developers can understand
- **Document behavior** through executable specifications
- **Catch issues early** by defining expected behavior first
- **Build confidence** in your code through comprehensive testing

Start with simple scenarios and gradually build up to complex workflows. Remember: the best test is one that clearly communicates intent and fails with helpful messages!

For more examples, check out:
- `test/integration/cli_integration_test.go`
- `test/e2e/e2e_scenarios_test.go`
- [TESTING.md](../TESTING.md) for the full testing guidelines
