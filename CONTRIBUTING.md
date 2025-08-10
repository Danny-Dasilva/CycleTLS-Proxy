# Contributing to CycleTLS-Proxy

Thank you for your interest in contributing to CycleTLS-Proxy! This document provides guidelines and information for contributors to help ensure a smooth collaboration process.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Adding Browser Profiles](#adding-browser-profiles)
- [Testing](#testing)
- [Code Style](#code-style)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)
- [Community](#community)

## Code of Conduct

This project adheres to a Code of Conduct that we expect all contributors to follow:

- Be respectful and inclusive in your communications
- Focus on constructive feedback and collaboration
- Respect different viewpoints and experiences
- Show empathy towards other community members
- Report unacceptable behavior to the maintainers

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- **Go 1.21+** installed
- **Git** for version control
- **Make** for build automation (optional)
- **Docker** for containerized testing (optional)
- A **GitHub account** for pull requests

### First Steps

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/CycleTLS-Proxy.git
   cd CycleTLS-Proxy
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/Danny-Dasilva/CycleTLS-Proxy.git
   ```
4. **Verify the setup**:
   ```bash
   go mod download
   go build ./cmd/proxy
   ./proxy --version
   ```

## Development Setup

### Local Development Environment

1. **Install dependencies**:
   ```bash
   # Install Go dependencies
   go mod download
   
   # Install development tools (optional)
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install github.com/cosmtrek/air@latest  # for hot reload
   ```

2. **Set up CycleTLS dependency**:
   ```bash
   # Clone CycleTLS alongside this project
   cd ..
   git clone https://github.com/Danny-Dasilva/CycleTLS.git
   cd CycleTLS-Proxy
   
   # The go.mod replace directive should handle the local dependency
   ```

3. **Run the development server**:
   ```bash
   # Standard build and run
   go run ./cmd/proxy/main.go
   
   # Or with hot reload (if air is installed)
   air -c .air.toml
   ```

4. **Test the setup**:
   ```bash
   # In another terminal
   curl -H "X-URL: https://httpbin.org/get" -H "X-IDENTIFIER: chrome" http://localhost:8080
   ```

### IDE Setup

#### VS Code

Recommended extensions:
- **Go** (Google) - Official Go extension
- **REST Client** - For testing HTTP requests
- **GitLens** - Enhanced Git integration

#### GoLand/IntelliJ IDEA

Configure Go modules support and set the project root correctly.

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-new-profile` - New features
- `fix/session-memory-leak` - Bug fixes
- `docs/update-readme` - Documentation updates
- `refactor/handler-structure` - Code refactoring
- `test/integration-coverage` - Testing improvements

### Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards

3. **Test your changes**:
   ```bash
   # Run unit tests
   go test ./...
   
   # Run integration tests
   ./examples/test_all.sh --core-only
   
   # Test manually with examples
   ./examples/curl.sh
   ```

4. **Commit your changes** with clear messages

5. **Push and create a pull request**

## Adding Browser Profiles

Browser profiles are one of the most common contributions. Here's how to add a new one:

### Step 1: Research the Target Browser

1. **Gather TLS fingerprint data**:
   - JA3 fingerprint string
   - JA4 fingerprint string (preferred)
   - User-Agent string
   - TLS version support
   - HTTP version preference

2. **Verify fingerprint accuracy**:
   - Use tools like [JA3er](https://ja3er.com/) or [TLS fingerprinting tools](https://tls.peet.ws/)
   - Test with actual browser requests
   - Ensure the fingerprint is current and widely used

### Step 2: Add the Profile

Edit `internal/fingerprints/profiles.go`:

```go
"new_browser_profile": {
    JA3:         "771,4865-4866-4867...", // Your JA3 string
    JA4:         "t13d1517h2_8daaf615...", // Your JA4 string
    UserAgent:   "Mozilla/5.0 (...)",      // Matching User-Agent
    HTTPVersion: "h2",                     // "h2" or "http/1.1"
    TLSVersion:  "1.3",                    // "1.3", "1.2", etc.
    Description: "Browser Name Version on Platform",
    Platform:    "Platform Name",          // Windows, macOS, Linux, iOS, Android
},
```

### Step 3: Add Tests

Add tests in `internal/fingerprints/profiles_test.go`:

```go
func TestNewBrowserProfile(t *testing.T) {
    profile, exists := GetProfile("new_browser_profile")
    assert.True(t, exists)
    assert.NotEmpty(t, profile.JA3)
    assert.NotEmpty(t, profile.JA4)
    assert.NotEmpty(t, profile.UserAgent)
    assert.Contains(t, profile.UserAgent, "Expected Browser String")
}
```

### Step 4: Update Documentation

1. Add the profile to the table in `README.md`
2. Include it in example scripts
3. Update any relevant documentation

### Step 5: Verify the Profile Works

```bash
# Test the new profile
curl -H "X-URL: https://httpbin.org/user-agent" \
     -H "X-IDENTIFIER: new_browser_profile" \
     http://localhost:8080

# Verify the User-Agent matches
curl -H "X-URL: https://httpbin.org/user-agent" \
     -H "X-IDENTIFIER: new_browser_profile" \
     http://localhost:8080 | jq -r '."user-agent"'
```

## Testing

### Running Tests

```bash
# Unit tests
go test ./...

# With coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration tests
./examples/test_all.sh

# Specific test categories
./examples/test_all.sh --core-only

# Load testing
./examples/test_all.sh --load-test
```

### Writing Tests

#### Unit Tests

Follow Go testing conventions:

```go
func TestHandlerMethod(t *testing.T) {
    // Arrange
    handler := NewHandler(logger)
    
    // Act
    result := handler.SomeMethod(input)
    
    // Assert
    assert.Equal(t, expected, result)
}
```

#### Integration Tests

Add integration tests to `examples/test_all.sh`:

```bash
# Test new functionality
run_test "New Feature Test" \
    "curl -s -H 'X-URL: https://httpbin.org/test' -H 'X-IDENTIFIER: chrome' '$PROXY_URL' | jq -e '.result'"
```

### Test Requirements

All contributions should include:
- Unit tests for new functions/methods
- Integration tests for new endpoints/features
- Performance tests for significant changes
- Error case testing

## Code Style

### Go Code Style

Follow standard Go conventions:

- Use `gofmt` for formatting
- Follow Go naming conventions (PascalCase for exported, camelCase for unexported)
- Write clear, descriptive function and variable names
- Add comments for exported functions
- Handle errors appropriately
- Use context for cancellation and timeouts

#### Linting

Run linters before submitting:

```bash
# Install golangci-lint if not already installed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
golangci-lint run

# Fix auto-fixable issues
golangci-lint run --fix
```

### Documentation Style

- Use clear, concise language
- Include code examples where helpful
- Keep line lengths reasonable (< 100 characters)
- Use proper Markdown formatting
- Include links to relevant resources

### Configuration Files

- Use consistent indentation (2 spaces for YAML, 4 for Go)
- Comment complex configurations
- Follow established patterns in the codebase

## Commit Guidelines

### Commit Message Format

Use the conventional commit format:

```
type(scope): brief description

More detailed explanation if needed.

- Bullet points for multiple changes
- Reference issues with #123
```

#### Types

- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Maintenance tasks

#### Examples

```bash
feat(profiles): add Chrome 121 fingerprint profile

- Add JA3 and JA4 fingerprints for Chrome 121
- Update user agent string to match latest version
- Add comprehensive tests for new profile

Closes #45

fix(handler): prevent session ID collision

Previously, concurrent requests could generate identical session IDs
leading to connection conflicts. Now using UUID v4 for guaranteed uniqueness.

docs(readme): update installation instructions

- Add pre-built binary installation option
- Update Docker compose example
- Fix typos in API documentation
```

## Pull Request Process

### Before Submitting

1. **Sync with upstream**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all tests**:
   ```bash
   go test ./...
   ./examples/test_all.sh
   ```

3. **Run linters**:
   ```bash
   golangci-lint run
   ```

4. **Update documentation** if needed

### Pull Request Template

When creating a PR, include:

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring

## Changes Made
- List of specific changes
- Include any breaking changes

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed
- [ ] New tests added for new functionality

## Documentation
- [ ] README updated
- [ ] Code comments added
- [ ] API documentation updated

## Screenshots (if applicable)
Add screenshots for UI changes

## Related Issues
Closes #123
Related to #456

## Additional Notes
Any additional context or notes for reviewers
```

### Review Process

1. **Automated checks** must pass (tests, linting)
2. **Code review** by maintainers
3. **Manual testing** for significant changes
4. **Documentation review** for user-facing changes
5. **Final approval** and merge

### Addressing Review Comments

- Respond to all comments
- Make requested changes in new commits
- Use `git commit --fixup` for small fixes
- Squash commits before merge if requested

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):

- `MAJOR.MINOR.PATCH`
- `MAJOR`: Breaking changes
- `MINOR`: New features (backward compatible)
- `PATCH`: Bug fixes (backward compatible)

### Release Candidates

Major releases may have release candidates:
- `v1.2.0-rc.1` - First release candidate
- `v1.2.0-rc.2` - Second release candidate
- `v1.2.0` - Final release

### Release Checklist

For maintainers preparing releases:

1. **Update version numbers** in relevant files
2. **Update CHANGELOG.md** with release notes
3. **Run full test suite**
4. **Build and test Docker images**
5. **Create GitHub release** with release notes
6. **Update documentation** if needed

## Community

### Getting Help

- **GitHub Issues** - Bug reports and feature requests
- **GitHub Discussions** - Questions and general discussion
- **Documentation** - Check README and examples first

### Communication Guidelines

- Be respectful and professional
- Search existing issues before creating new ones
- Provide detailed information for bug reports
- Include steps to reproduce issues
- Use clear, descriptive titles

### Issue Templates

When reporting bugs, include:

- **Environment**: OS, Go version, proxy version
- **Steps to reproduce**: Minimal example
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Logs**: Relevant error messages or logs

For feature requests, include:

- **Use case**: Why is this needed?
- **Proposed solution**: How should it work?
- **Alternatives**: Other approaches considered
- **Additional context**: Any other relevant information

### Recognition

Contributors are recognized in:
- GitHub contributors list
- Release notes for significant contributions
- Special recognition for major features

## Development Tips

### Debugging

1. **Enable debug logging**:
   ```bash
   LOG_LEVEL=debug ./proxy
   ```

2. **Use development tools**:
   ```bash
   # Hot reload during development
   air -c .air.toml
   
   # Profile performance
   go tool pprof http://localhost:8080/debug/pprof/profile
   ```

3. **Test with verbose output**:
   ```bash
   VERBOSE=true ./examples/curl.sh
   ```

### Common Pitfalls

- **Session management**: Test session persistence carefully
- **Header handling**: Verify X-* headers are properly filtered
- **Error handling**: Ensure all error cases are handled
- **Memory leaks**: Profile memory usage with long-running tests
- **Concurrency**: Test concurrent request handling

### Performance Considerations

- **Memory usage**: Monitor session storage growth
- **Connection pooling**: Reuse HTTP connections when possible
- **Timeouts**: Set appropriate timeouts for all operations
- **Resource limits**: Consider resource constraints in production

## Questions?

If you have questions not covered in this guide:

1. Check existing [GitHub Issues](https://github.com/Danny-Dasilva/CycleTLS-Proxy/issues)
2. Review the [README](README.md) and examples
3. Create a new issue with the "question" label
4. Join the discussion in [GitHub Discussions](https://github.com/Danny-Dasilva/CycleTLS-Proxy/discussions)

Thank you for contributing to CycleTLS-Proxy! 🚀