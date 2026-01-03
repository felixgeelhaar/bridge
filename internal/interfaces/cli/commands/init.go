package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/output"
	"github.com/urfave/cli/v2"
)

// InitCommand returns the init command.
func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new Bridge project",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Template to use (basic, pr-review)",
				Value:   "basic",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing configuration",
			},
		},
		Action: runInit,
	}
}

func runInit(c *cli.Context) error {
	formatter := output.NewFormatter(c.String("output"))
	template := c.String("template")
	force := c.Bool("force")

	// Create .bridge directory
	bridgeDir := ".bridge"
	if err := os.MkdirAll(bridgeDir, 0755); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create .bridge directory: %v", err))
		return err
	}

	// Create config file
	configPath := filepath.Join(bridgeDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil && !force {
		formatter.Error("Configuration already exists. Use --force to overwrite.")
		return fmt.Errorf("configuration exists")
	}

	configContent := getConfigTemplate()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create config: %v", err))
		return err
	}

	// Create workflows directory
	workflowsDir := filepath.Join(bridgeDir, "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create workflows directory: %v", err))
		return err
	}

	// Create policies directory
	policiesDir := filepath.Join(bridgeDir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create policies directory: %v", err))
		return err
	}

	// Create example workflow based on template
	workflowPath := filepath.Join(workflowsDir, "example.yaml")
	workflowContent := getWorkflowTemplate(template)
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create workflow: %v", err))
		return err
	}

	// Create default policy
	policyPath := filepath.Join(policiesDir, "default.rego")
	policyContent := getDefaultPolicy()
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		formatter.Error(fmt.Sprintf("Failed to create policy: %v", err))
		return err
	}

	formatter.Success("Bridge project initialized")
	formatter.Info(fmt.Sprintf("Configuration: %s", configPath))
	formatter.Info(fmt.Sprintf("Example workflow: %s", workflowPath))
	formatter.Info(fmt.Sprintf("Default policy: %s", policyPath))
	formatter.Info("\nNext steps:")
	formatter.Info("  1. Edit .bridge/config.yaml with your LLM provider settings")
	formatter.Info("  2. Customize .bridge/workflows/example.yaml")
	formatter.Info("  3. Run: bridge validate -w .bridge/workflows/example.yaml")
	formatter.Info("  4. Run: bridge run -w .bridge/workflows/example.yaml")

	return nil
}

func getConfigTemplate() string {
	return `# Bridge Configuration
version: "1.0"

# LLM Provider Configuration
providers:
  anthropic:
    enabled: true
    api_key: ${ANTHROPIC_API_KEY}
    default_model: claude-sonnet-4-20250514

  openai:
    enabled: false
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o

  gemini:
    enabled: false
    api_key: ${GEMINI_API_KEY}
    default_model: gemini-1.5-pro

  ollama:
    enabled: false
    base_url: http://localhost:11434
    default_model: llama3

# Default provider to use
default_provider: anthropic

# Governance settings
governance:
  require_approval_for:
    - file-write
    - shell-exec
    - git-push

  blocked_paths:
    - .env
    - .env.*
    - secrets/
    - .ssh/
    - credentials

# Audit settings
audit:
  enabled: true
  log_path: .bridge/audit.log

# Resilience settings
resilience:
  circuit_breaker:
    failure_threshold: 5
    success_threshold: 2
    timeout: 30s

  retry:
    max_attempts: 3
    initial_delay: 1s
    max_delay: 30s

  rate_limit:
    requests_per_minute: 60
`
}

func getWorkflowTemplate(template string) string {
	switch template {
	case "pr-review":
		return getPRReviewTemplate()
	default:
		return getBasicTemplate()
	}
}

func getBasicTemplate() string {
	return `# Basic Workflow Template
name: basic-workflow
version: "1.0"
description: A basic workflow example

triggers:
  - type: manual
    description: Manually triggered workflow

steps:
  - name: analyze
    agent: code-reviewer
    description: Analyze the input
    input:
      prompt: "Analyze the following: ${{ trigger.input }}"

  - name: summarize
    agent: summarizer
    description: Summarize the analysis
    input:
      analysis: ${{ steps.analyze.output.content }}

policies:
  - name: default
    path: .bridge/policies/default.rego
`
}

func getPRReviewTemplate() string {
	return `# PR Review Workflow Template
name: guarded-pr-review
version: "1.0"
description: AI-assisted PR review with approval gate

triggers:
  - type: github.pull_request
    events:
      - opened
      - synchronize
    config:
      repo: ${{ env.GITHUB_REPOSITORY }}

steps:
  - name: fetch-changes
    agent: code-reviewer
    description: Fetch and analyze PR changes
    input:
      pr_number: ${{ trigger.pr.number }}
      repo: ${{ trigger.repo.full_name }}

  - name: security-scan
    agent: security-analyst
    description: Scan for security issues
    input:
      diff: ${{ steps.fetch-changes.output.diff }}
      files: ${{ steps.fetch-changes.output.files }}

  - name: code-review
    agent: code-reviewer
    description: Perform detailed code review
    input:
      diff: ${{ steps.fetch-changes.output.diff }}
      security_findings: ${{ steps.security-scan.output }}

  - name: post-review
    agent: github-commenter
    description: Post review comments
    requires_approval: true
    input:
      pr_number: ${{ trigger.pr.number }}
      review: ${{ steps.code-review.output }}
      security: ${{ steps.security-scan.output }}

policies:
  - name: require-human-approval
    path: .bridge/policies/default.rego
`
}

func getDefaultPolicy() string {
	return `package bridge.policy

# Default allow
default allowed = true

# Default no approval required
default requires_approval = false

# Require approval for file write capabilities
requires_approval {
    input.capabilities[_] == "file-write"
}

# Require approval for shell execution
requires_approval {
    input.capabilities[_] == "shell-exec"
}

# Block access to sensitive paths
allowed = false {
    path := input.context.path
    contains(path, ".env")
}

allowed = false {
    path := input.context.path
    contains(path, "secrets/")
}

# Violation messages
violation[msg] {
    not allowed
    path := input.context.path
    msg := sprintf("Access to sensitive path blocked: %s", [path])
}
`
}
