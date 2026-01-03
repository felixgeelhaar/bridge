# Bridge

AI Workflow Orchestration & Governance Platform

Bridge enables teams to define, execute, and govern AI-powered workflows with built-in policy enforcement, approval gates, and comprehensive audit trails.

## Features

- **Workflow Orchestration**: Define multi-step AI workflows in YAML with conditional logic and parallel execution
- **Multi-Provider LLM Support**: Anthropic Claude, OpenAI GPT, Google Gemini, and local Ollama models
- **Policy Governance**: OPA/Rego-based policy engine for approval requirements and compliance
- **Resilience Patterns**: Circuit breakers, retry logic, rate limiting, and timeouts via [fortify](https://github.com/felixgeelhaar/fortify)
- **State Machine Workflows**: Robust workflow state management via [statekit](https://github.com/felixgeelhaar/statekit)
- **MCP Integration**: Model Context Protocol support for tool bindings via [mcp-go](https://github.com/felixgeelhaar/mcp-go)
- **Structured Logging**: Consistent observability via [bolt](https://github.com/felixgeelhaar/bolt)
- **GitHub Integration**: PR review workflows with webhook support

## Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/felixgeelhaar/bridge/main/install.sh | bash
```

### Homebrew

```bash
brew install felixgeelhaar/tap/bridge
```

### Go Install

```bash
go install github.com/felixgeelhaar/bridge/cmd/bridge@latest
```

### Docker

```bash
docker run --rm ghcr.io/felixgeelhaar/bridge:latest --help
```

### From Source

```bash
git clone https://github.com/felixgeelhaar/bridge.git
cd bridge
make build
```

## Quick Start

### Initialize Configuration

```bash
bridge init
```

This creates a `.bridge/` directory with default configuration files.

### Validate a Workflow

```bash
bridge validate -w workflow.yaml
```

### Run a Workflow

```bash
bridge run -w workflow.yaml
```

### Check Workflow Status

```bash
bridge status <run-id>
```

### Approve a Pending Workflow

```bash
bridge approve <run-id>
```

## Example Workflow

```yaml
# workflows/pr-review.yaml
name: guarded-pr-review
version: "1.0"
description: AI-assisted PR review with approval gate

triggers:
  - type: github.pull_request
    events: [opened, synchronize]

steps:
  - name: analyze-changes
    agent: code-reviewer
    input:
      pr_number: ${{ trigger.pr.number }}
      repo: ${{ trigger.repo.full_name }}

  - name: security-scan
    agent: security-analyst
    input:
      diff: ${{ steps.analyze-changes.output.diff }}

  - name: generate-review
    agent: code-reviewer
    requires_approval: true
    input:
      analysis: ${{ steps.analyze-changes.output }}
      security: ${{ steps.security-scan.output }}

policies:
  - name: require-human-approval
    rule: steps.*.requires_approval == true
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BRIDGE_CONFIG` | Config file path | `.bridge/config.yaml` |
| `ANTHROPIC_API_KEY` | Anthropic API key | - |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `GOOGLE_API_KEY` | Google AI API key | - |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |
| `GITHUB_TOKEN` | GitHub API token | - |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `RABBITMQ_URL` | RabbitMQ connection string | - |

### Policy Configuration

Bridge uses OPA/Rego for policy evaluation:

```rego
# policies/approval.rego
package bridge.approval

default requires_approval = false

requires_approval {
    input.step.requires_approval == true
}

requires_approval {
    input.workflow.risk_level == "high"
}
```

## Architecture

```
bridge/
├── cmd/bridge/              # CLI entry point
├── internal/
│   ├── domain/              # DDD bounded contexts
│   │   ├── workflow/        # Workflow definitions and runs
│   │   ├── governance/      # Policies and approvals
│   │   ├── agents/          # Agent configurations
│   │   └── analytics/       # Telemetry and metrics
│   ├── application/
│   │   └── orchestrator/    # Workflow execution engine
│   └── infrastructure/
│       ├── persistence/     # PostgreSQL + in-memory repos
│       ├── messaging/       # RabbitMQ + event bus
│       ├── policy/          # OPA integration
│       ├── llm/             # LLM provider adapters
│       ├── mcp/             # MCP server and tools
│       └── github/          # GitHub API integration
├── pkg/
│   ├── config/              # Configuration schemas
│   └── types/               # Shared types and errors
├── policies/                # Default OPA policies
└── examples/                # Example workflows
```

## Development

### Prerequisites

- Go 1.23+
- Docker (optional, for PostgreSQL/RabbitMQ)
- OPA CLI (optional, for policy testing)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Test with Coverage

```bash
make test-coverage
```

### Lint

```bash
make lint
```

### Generate sqlc Code

```bash
make sqlc
```

### Run All Checks

```bash
make all
```

### Start Development Environment

```bash
docker-compose up -d
```

### Start Demo Environment

```bash
docker-compose -f docker-compose.demo.yml up -d
```

## LLM Providers

Bridge supports multiple LLM providers with automatic failover and load balancing:

### Anthropic Claude

```yaml
providers:
  - name: anthropic
    type: anthropic
    api_key: ${ANTHROPIC_API_KEY}
    model: claude-sonnet-4-20250514
    max_tokens: 4096
```

### OpenAI

```yaml
providers:
  - name: openai
    type: openai
    api_key: ${OPENAI_API_KEY}
    model: gpt-4
    max_tokens: 4096
```

### Google Gemini

```yaml
providers:
  - name: gemini
    type: gemini
    api_key: ${GOOGLE_API_KEY}
    model: gemini-pro
```

### Ollama (Local)

```yaml
providers:
  - name: ollama
    type: ollama
    host: http://localhost:11434
    model: llama2
```

## Resilience Configuration

Bridge uses [fortify](https://github.com/felixgeelhaar/fortify) for resilience patterns:

```yaml
resilience:
  circuit_breaker:
    failure_threshold: 5
    success_threshold: 2
    timeout: 30s
  retry:
    max_attempts: 3
    initial_interval: 100ms
    max_interval: 10s
    multiplier: 2.0
  rate_limit:
    requests_per_second: 10
    burst: 20
  timeout: 60s
```

## Security

- All API keys are passed via environment variables
- Supports TLS for database and message queue connections
- Policy-based access control for workflow execution
- Audit logging for all governance decisions
- Security scanning integrated in CI/CD pipeline

See [SECURITY.md](.github/SECURITY.md) for reporting vulnerabilities.

## Release Process

Bridge uses [relicta](https://github.com/felixgeelhaar/relicta) for release governance:

1. **Plan**: Analyze commits and suggest version bump
2. **Validate**: Run tests, security scans, coverage checks
3. **Approve**: Risk-based approval gates
4. **Publish**: Create GitHub release, Docker image, Homebrew formula

Releases are governed by:
- Minimum 50% test coverage
- No critical security vulnerabilities
- No leaked secrets
- Passing security scans (gosec, govulncheck, gitleaks)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure:
- All tests pass
- Code follows Go conventions
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [bolt](https://github.com/felixgeelhaar/bolt) - Structured logging
- [fortify](https://github.com/felixgeelhaar/fortify) - Resilience patterns
- [statekit](https://github.com/felixgeelhaar/statekit) - State machines
- [mcp-go](https://github.com/felixgeelhaar/mcp-go) - MCP protocol
- [relicta](https://github.com/felixgeelhaar/relicta) - Release governance
- [OPA](https://www.openpolicyagent.org/) - Policy engine
- [sqlc](https://sqlc.dev/) - Type-safe SQL
