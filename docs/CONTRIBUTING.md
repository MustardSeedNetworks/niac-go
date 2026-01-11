# Contributing to NIAC-Go

Thank you for your interest in contributing to NIAC-Go! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, constructive, and professional. We're all here to build great software together.

## Getting Started

### Prerequisites

- **Go**: 1.21 or later
- **libpcap**: For packet capture
  - macOS: `brew install libpcap` (usually pre-installed)
  - Linux: `sudo apt-get install libpcap-dev`
  - Windows: Install [Npcap](https://npcap.com/)
- **Git**: For version control

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/niac-go
   cd niac-go
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build the project**
   ```bash
   go build -o niac ./cmd/niac
   ```

4. **Run tests**
   ```bash
   go test ./...
   ```

5. **Install pre-commit hooks** (optional but recommended)
   ```bash
   ./scripts/install-hooks.sh
   ```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description
```

Branch naming conventions:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions/improvements

### 2. Make Changes

- Write clean, idiomatic Go code
- Follow existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Check test coverage for specific package
go test -coverprofile=coverage.out ./pkg/yourpackage
go tool cover -html=coverage.out
```

### 4. Format and Lint

```bash
# Format code
go fmt ./...

# Run linter (if golangci-lint installed)
golangci-lint run

# Run go vet
go vet ./...
```

### 5. Commit Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git commit -m "feat: add new VLAN tagging support"
git commit -m "fix: correct IPv6 address parsing in config"
git commit -m "docs: update README with new examples"
git commit -m "test: add unit tests for DHCP handler"
```

Commit message format:
```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `chore`: Maintenance tasks

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:
- Clear description of changes
- Reference to related issues (`Fixes #123`)
- Screenshots/examples if applicable

## Code Style Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Keep functions focused and concise (< 50 lines preferred)
- Use meaningful variable names
- Add comments for exported functions (godoc format)

### Example

```go
// HandlePacket processes incoming network packets for the device.
// It returns an error if the packet cannot be processed.
func (d *Device) HandlePacket(packet gopacket.Packet) error {
    if packet == nil {
        return fmt.Errorf("packet cannot be nil")
    }

    // Process packet layers
    ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
    if ethernetLayer == nil {
        return nil // Not an Ethernet frame
    }

    // ... rest of implementation
    return nil
}
```

### Testing

- Write table-driven tests where applicable
- Use meaningful test names: `TestFunctionName_Condition_ExpectedResult`
- Test both success and failure cases
- Mock external dependencies

Example:
```go
func TestDevice_HandlePacket_ValidEthernet_ReturnsNil(t *testing.T) {
    tests := []struct {
        name    string
        packet  gopacket.Packet
        wantErr bool
    }{
        {
            name:    "valid ethernet packet",
            packet:  createValidEthernetPacket(),
            wantErr: false,
        },
        {
            name:    "nil packet",
            packet:  nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            d := &Device{}
            err := d.HandlePacket(tt.packet)
            if (err != nil) != tt.wantErr {
                t.Errorf("HandlePacket() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Project Structure

```
niac-go/
├── cmd/
│   ├── niac/           # Main CLI application
│   └── niac-convert/   # Configuration converter
├── pkg/
│   ├── capture/        # Packet capture (gopacket/libpcap)
│   ├── config/         # Configuration parsing
│   ├── device/         # Device simulation
│   ├── errors/         # Error injection
│   ├── interactive/    # TUI (Bubble Tea)
│   ├── logging/        # Colored logging
│   ├── protocols/      # Protocol handlers (ARP, LLDP, etc)
│   └── snmp/           # SNMP agent
├── examples/           # Configuration examples
├── docs/               # Documentation
└── tests/              # Integration tests (future)
```

## Adding New Features

### New Protocol Handler

1. Create handler file: `pkg/protocols/yourprotocol.go`
2. Implement the protocol logic
3. Add tests: `pkg/protocols/yourprotocol_test.go`
4. Update `pkg/protocols/stack.go` to register handler
5. Add configuration support in `pkg/config/`
6. Add example config in `examples/`
7. Update documentation

### New CLI Command

1. Create command file: `cmd/niac/yourcommand.go`
2. Define cobra.Command with proper help/examples
3. Add tests (if testable without live interface)
4. Register in `cmd/niac/root.go`
5. Update CLI_REFERENCE.md
6. Regenerate man pages: `niac man`

## Testing Requirements

### Unit Tests

- Required for all new code
- Aim for >70% coverage for new packages
- Focus on edge cases and error conditions

### Integration Tests

- Add to `tests/integration/` (when directory exists)
- Test end-to-end workflows
- Use real or mocked network interfaces

## Documentation

- Update README.md for user-facing changes
- Update docs/ARCHITECTURE.md for architectural changes
- Add/update examples in `examples/` directory
- Regenerate man pages if CLI changes
- Update CHANGELOG.md (maintainers will help)

## Pull Request Process

1. **Ensure tests pass**: `go test ./...`
2. **Update documentation**: README, examples, etc.
3. **Keep PRs focused**: One feature/fix per PR
4. **Respond to feedback**: Address review comments promptly
5. **Squash commits**: Before merging (or maintainer will do it)

### PR Checklist

- [ ] Tests added/updated and passing
- [ ] Documentation updated
- [ ] Code formatted (`go fmt`)
- [ ] No new linter warnings
- [ ] Commit messages follow convention
- [ ] PR description is clear and complete

## Release Process

Maintainers follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Getting Help

- **Questions**: Open a [Discussion](https://github.com/krisarmstrong/niac-go/discussions)
- **Bugs**: Open an [Issue](https://github.com/krisarmstrong/niac-go/issues)
- **Features**: Open an Issue with `[Feature Request]` prefix

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Recognition

Contributors will be recognized in:
- CHANGELOG.md for their contributions
- GitHub's contributor graph
- Special mentions for significant contributions

Thank you for contributing to NIAC-Go! 🎉
