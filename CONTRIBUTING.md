# Contributing to cronkit

Thank you for considering contributing to cronkit! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful and inclusive. We welcome contributions from everyone.

## How to Contribute

### Reporting Bugs

- Check if the bug has already been reported in the Issues section
- If not, create a new issue with a clear title and description
- Include steps to reproduce the bug
- Include your environment details (OS, Go version, etc.)

### Suggesting Enhancements

- Open an issue with your enhancement suggestion
- Clearly describe the feature and its benefits
- Explain how it would work

### Pull Requests

1. Fork the repository
2. Create a new branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. Make your changes:
   - Write clear, concise commit messages
   - Follow Go coding conventions
   - Add tests for new functionality
   - Update documentation as needed

4. Ensure your code passes all checks:
   ```bash
   make test
   make lint
   make fmt
   ```

5. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Open a Pull Request with:
   - A clear title and description
   - Reference to any related issues
   - Description of changes made
   - Screenshots if applicable

## Development Setup

### Prerequisites

- Go 1.25.2 or higher
- Git
- Make

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/hzerrad/cronkit.git
   cd cronkit
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up git hooks (recommended):
   ```bash
   make setup-hooks
   ```

   This installs pre-commit hooks that automatically enforce code quality standards.

4. Build the project:
   ```bash
   make build
   ```

5. Run tests:
   ```bash
   make test
   ```

## Code Style

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Run `make fmt` before committing
- Run `make vet` to check for common errors
- Run `make lint` to check for issues
- Write meaningful variable and function names
- Add comments for exported functions and complex logic

### Pre-commit Hooks

The project includes pre-commit hooks that automatically enforce code quality. Install them with:

```bash
make setup-hooks
```

The hooks will automatically run on every commit:
- **go fmt**: Ensures code is properly formatted
- **go vet**: Checks for common Go programming errors
- **golangci-lint**: Runs comprehensive linting (if installed)

If any check fails, the commit will be blocked. This helps maintain code quality and catches issues early.

## Adding New Commands

To add a new command:

1. Create a new file in `internal/cmd/` (e.g., `mycommand.go`)
2. Define your command using cobra:
   ```go
   package cmd

   import (
       "fmt"
       "github.com/spf13/cobra"
   )

   var myCmd = &cobra.Command{
       Use:   "mycommand",
       Short: "Brief description",
       Long:  `Longer description`,
       Run: func(cmd *cobra.Command, args []string) {
           fmt.Println("mycommand called")
       },
   }

   func init() {
       rootCmd.AddCommand(myCmd)
   }
   ```

3. Test your command
4. Add tests in a corresponding `_test.go` file

## Testing

- Write unit tests for new functionality
- Ensure all tests pass with `make test`
- Aim for good test coverage
- Run `make test-coverage` to view coverage

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Update CHANGELOG.md (if present) with your changes

## Questions?

If you have questions, feel free to open an issue for discussion.

Thank you for contributing!
