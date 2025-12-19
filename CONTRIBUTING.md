# Contributing to Presto 🎵

Thank you for your interest in contributing to Presto! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/aras/presto/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Your environment (OS, Go version, Presto version)
   - Relevant logs or error messages

### Suggesting Features

1. Check [Discussions](https://github.com/aras/presto/discussions) for similar ideas
2. Create a new discussion or issue explaining:
   - The problem you're trying to solve
   - Your proposed solution
   - Why it would benefit Presto users

### Pull Requests

1. **Fork the repository**
   ```bash
   git clone https://github.com/aras/presto.git
   cd presto
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write clear, documented code
   - Follow Go best practices
   - Add tests for new functionality
   - Update documentation as needed

4. **Test your changes**
   ```bash
   make test
   make build
   ```

5. **Commit with clear messages**
   ```bash
   git commit -m "feat: add awesome new feature"
   ```

   Use conventional commits:
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `refactor:` - Code refactoring
   - `test:` - Adding tests
   - `chore:` - Maintenance tasks

6. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/aras/presto.git
cd presto

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Run Presto
./bin/presto --version
```

### Project Structure

```
presto/
├── cmd/presto/          # CLI entry point
├── internal/
│   ├── parser/          # composer.json/lock parser
│   ├── packagist/       # Packagist API client
│   ├── resolver/        # Dependency resolution
│   ├── downloader/      # Parallel package downloader
│   ├── autoload/        # Autoload file generator
│   └── security/        # Security vulnerability scanner
├── examples/            # Example projects
├── .github/             # GitHub Actions workflows
└── Makefile            # Build commands
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write clear comments for exported functions
- Keep functions small and focused

### Testing

- Write unit tests for new code
- Aim for >80% code coverage
- Test edge cases and error conditions
- Use table-driven tests where appropriate

### Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Update CHANGELOG.md for notable changes

## Release Process

Releases are automated via GitHub Actions:

1. Update version in `cmd/presto/main.go`
2. Update CHANGELOG.md
3. Create and push a tag:
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```
4. GitHub Actions will build and create the release

## Questions?

- Open a [Discussion](https://github.com/aras/presto/discussions)
- Join our community chat
- Email: presto@example.com

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Presto! 🎵⚡
