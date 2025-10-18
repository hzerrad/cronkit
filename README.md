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

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt
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
