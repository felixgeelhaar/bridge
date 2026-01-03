# Contributing to Bridge

Thank you for your interest in contributing to Bridge! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful and constructive in all interactions. We're building something together.

## Getting Started

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Make

### Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/bridge.git
   cd bridge
   ```

3. Install dependencies:
   ```bash
   go mod download
   make tools
   ```

4. Start development environment:
   ```bash
   docker-compose up -d
   ```

5. Run tests:
   ```bash
   make test
   ```

## Development Workflow

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test improvements

Example: `feature/add-azure-provider`

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting (no code change)
- `refactor` - Code refactoring
- `perf` - Performance improvement
- `test` - Adding tests
- `chore` - Maintenance tasks
- `ci` - CI/CD changes

Examples:
```
feat(llm): add Azure OpenAI provider support
fix(workflow): handle nil pointer in step execution
docs(readme): add installation instructions
test(governance): add policy evaluation tests
```

### Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Run `make lint` to check for issues
- Keep functions small and focused
- Write meaningful comments for exported functions

### Testing

- Write tests for all new functionality
- Maintain test coverage above 50%
- Use table-driven tests where appropriate
- Run `make test` before submitting PR

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "foo", "FOO", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Run all checks:
   ```bash
   make all
   ```
4. Push your branch and create a PR
5. Fill in the PR template
6. Wait for review

### PR Checklist

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code is formatted (`make fmt`)
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventions
- [ ] No breaking changes (or documented)

## Architecture Guidelines

### Domain-Driven Design

Bridge follows DDD principles:

- **Bounded Contexts**: workflow, governance, agents, analytics
- **Aggregates**: WorkflowDefinition, WorkflowRun, PolicyBundle
- **Entities**: Step, Approval, Agent
- **Value Objects**: IDs, configurations
- **Domain Events**: WorkflowStarted, StepCompleted, etc.

### Hexagonal Architecture

```
┌─────────────────────────────────────────────────┐
│                  Interfaces                      │
│              (CLI, HTTP, Webhooks)              │
├─────────────────────────────────────────────────┤
│                 Application                      │
│               (Orchestrator)                    │
├─────────────────────────────────────────────────┤
│                   Domain                         │
│     (Workflow, Governance, Agents)              │
├─────────────────────────────────────────────────┤
│               Infrastructure                     │
│  (PostgreSQL, RabbitMQ, LLM, GitHub, MCP)       │
└─────────────────────────────────────────────────┘
```

### Adding a New LLM Provider

1. Create provider in `internal/infrastructure/llm/`
2. Implement the `Provider` interface
3. Add configuration struct
4. Register in provider registry
5. Add tests
6. Update documentation

```go
type MyProvider struct {
    config MyProviderConfig
    client *myclient.Client
    logger *bolt.Logger
}

func (p *MyProvider) Name() string { return "myprovider" }

func (p *MyProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    // Implementation
}
```

### Adding a New MCP Tool

1. Create tool in `internal/infrastructure/mcp/tools/`
2. Implement tool handler
3. Register with MCP server
4. Add tests
5. Update documentation

## Reporting Issues

### Bug Reports

Include:
- Bridge version
- Go version
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### Feature Requests

Include:
- Use case description
- Proposed solution
- Alternatives considered
- Impact on existing functionality

## Security

Report security vulnerabilities privately. See [SECURITY.md](.github/SECURITY.md).

## Questions?

- Check existing issues
- Read the documentation
- Open a discussion

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
