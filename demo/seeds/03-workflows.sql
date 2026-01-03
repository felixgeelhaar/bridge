-- Demo Workflows
-- Pre-configured workflow definitions for demonstration

INSERT INTO workflow_definitions (id, name, version, description, config) VALUES
(
    'w1000000-0000-0000-0000-000000000001',
    'pr-review',
    '1.0',
    'Comprehensive pull request review workflow with code review and security analysis',
    '{
        "name": "pr-review",
        "version": "1.0",
        "description": "Comprehensive PR review with code review, security analysis, and automated feedback",
        "triggers": [
            {
                "type": "github.pull_request",
                "events": ["opened", "synchronize"]
            }
        ],
        "steps": [
            {
                "name": "fetch-changes",
                "agent": "file-reader",
                "description": "Fetch changed files from the pull request"
            },
            {
                "name": "code-review",
                "agent": "code-reviewer",
                "description": "Perform comprehensive code review"
            },
            {
                "name": "security-scan",
                "agent": "security-analyst",
                "description": "Analyze code for security vulnerabilities"
            },
            {
                "name": "generate-summary",
                "agent": "code-reviewer",
                "requires_approval": true,
                "description": "Generate final review summary"
            },
            {
                "name": "post-review",
                "agent": "github-commenter",
                "description": "Post review comments to GitHub"
            }
        ],
        "policies": [
            {
                "name": "require-security-pass",
                "rule": "security-scan.severity != \"critical\""
            }
        ]
    }'
),
(
    'w1000000-0000-0000-0000-000000000002',
    'code-generation',
    '1.0',
    'AI-assisted code generation workflow with review and testing',
    '{
        "name": "code-generation",
        "version": "1.0",
        "description": "Generate code from specifications with automated review and testing",
        "triggers": [
            {
                "type": "manual",
                "events": ["requested"]
            }
        ],
        "steps": [
            {
                "name": "analyze-requirements",
                "agent": "code-generator",
                "description": "Analyze and clarify requirements"
            },
            {
                "name": "generate-code",
                "agent": "code-generator",
                "description": "Generate implementation code"
            },
            {
                "name": "generate-tests",
                "agent": "test-generator",
                "description": "Generate test suite for the implementation"
            },
            {
                "name": "review-output",
                "agent": "code-reviewer",
                "requires_approval": true,
                "description": "Review generated code and tests"
            },
            {
                "name": "create-pr",
                "agent": "github-commenter",
                "description": "Create pull request with generated code"
            }
        ]
    }'
),
(
    'w1000000-0000-0000-0000-000000000003',
    'security-audit',
    '1.0',
    'Comprehensive security audit workflow',
    '{
        "name": "security-audit",
        "version": "1.0",
        "description": "Full security audit of codebase with vulnerability reporting",
        "triggers": [
            {
                "type": "schedule",
                "cron": "0 0 * * 0"
            },
            {
                "type": "manual",
                "events": ["requested"]
            }
        ],
        "steps": [
            {
                "name": "scan-dependencies",
                "agent": "security-analyst",
                "description": "Scan dependencies for known vulnerabilities"
            },
            {
                "name": "scan-secrets",
                "agent": "security-analyst",
                "description": "Scan for exposed secrets and credentials"
            },
            {
                "name": "scan-code",
                "agent": "security-analyst",
                "description": "Static application security testing"
            },
            {
                "name": "generate-report",
                "agent": "documentation-writer",
                "description": "Generate comprehensive security report"
            },
            {
                "name": "create-issues",
                "agent": "github-commenter",
                "requires_approval": true,
                "description": "Create GitHub issues for findings"
            }
        ],
        "policies": [
            {
                "name": "block-critical",
                "rule": "!contains(scan-code.findings, \"critical\")"
            }
        ]
    }'
);
