# cronic

A CLI application built with Go.

## Installation

### From Source

```bash
git clone https://github.com/hzerrad/cronic.git
cd cronic
make install
```

### Using Go Install

```bash
go install github.com/hzerrad/cronic/cmd/cronic@latest
```

## Usage

```bash
cronic [command]
```

### Available Commands

- `version` - Print the version number of cronic
- `help` - Help about any command

### Flags

- `-h, --help` - Help for cronic
- `-v, --version` - Version for cronic

## Development

### Prerequisites

- Go 1.25.2 or higher

### Building

```bash
# Build the binary
make build

# Build for all platforms
make build-all
```

### Testing

This project follows **Test-Driven Development (TDD)** and **Behavior-Driven Development (BDD)** practices. See [TESTING.md](TESTING.md) for comprehensive testing guidelines.

```bash
# Run all tests (unit + BDD)
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run only E2E tests
make test-e2e

# Run all BDD tests
make test-bdd

# Run tests with coverage
make test-coverage

# Watch mode (auto-run on changes)
make test-watch
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Run go vet
make vet
```

### Git Hooks

The project includes pre-commit hooks to enforce code quality standards. The hooks will automatically run:
- `go fmt` - Ensures all code is properly formatted
- `go vet` - Checks for common Go programming errors
- `golangci-lint` - Runs comprehensive linting (if installed)
- `go test` - Runs unit tests (when test files are modified)

To install the pre-commit hooks:

```bash
make setup-hooks
```

After installation, these checks will run automatically on every commit. If any check fails, the commit will be blocked until the issues are fixed.

**Configuration:**
- Skip tests temporarily: `SKIP_TESTS=1 git commit`
- Always run tests: `RUN_TESTS=1 git commit`

**Note:** If you don't have `golangci-lint` installed, the hook will skip it with a warning. Install it for comprehensive linting:

```bash
# macOS
brew install golangci-lint

# Linux/macOS
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or visit: https://golangci-lint.run/usage/install/
```

## Project Structure

```
cronic/
├── cmd/
│   └── cronic/          # Application entry point
│       └── main.go
├── internal/            # Private application code
│   └── cmd/            # Command implementations
│       ├── root.go     # Root command
│       └── version.go  # Version command
├── pkg/                # Public libraries
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── Makefile            # Build automation
├── LICENSE             # Apache 2.0 License
└── README.md           # This file
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Author

[hzerrad](https://github.com/hzerrad)
