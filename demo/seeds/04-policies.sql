-- Demo Policies
-- Pre-configured policy bundles for demonstration

INSERT INTO policy_bundles (id, name, version, description, active, rules) VALUES
(
    'p1000000-0000-0000-0000-000000000001',
    'default-security',
    '1.0',
    'Default security policies for workflow execution',
    true,
    '[
        {
            "name": "require-approval-for-production",
            "description": "Require human approval for production deployments",
            "enabled": true,
            "severity": "high",
            "rego": "package bridge.approval\n\ndefault requires_approval = false\n\nrequires_approval {\n    input.step.requires_approval == true\n}\n\nrequires_approval {\n    input.context.environment == \"production\"\n}"
        },
        {
            "name": "block-on-critical-vulnerabilities",
            "description": "Block workflow if critical vulnerabilities are detected",
            "enabled": true,
            "severity": "critical",
            "rego": "package bridge.security\n\ndefault allow = true\n\nallow = false {\n    some finding in input.step.output.findings\n    finding.severity == \"critical\"\n}"
        },
        {
            "name": "rate-limit-llm-calls",
            "description": "Limit LLM API calls per workflow run",
            "enabled": true,
            "severity": "medium",
            "rego": "package bridge.limits\n\ndefault allow = true\n\nmax_tokens := 50000\n\nallow = false {\n    input.run.total_tokens > max_tokens\n}"
        }
    ]'
),
(
    'p1000000-0000-0000-0000-000000000002',
    'strict-governance',
    '1.0',
    'Strict governance policies for regulated environments',
    false,
    '[
        {
            "name": "dual-approval-required",
            "description": "Require approval from two different users",
            "enabled": true,
            "severity": "high",
            "rego": "package bridge.approval\n\ndefault meets_approval = false\n\nmeets_approval {\n    count(input.approvals) >= 2\n    unique_approvers := {a | a := input.approvals[_].approver}\n    count(unique_approvers) >= 2\n}"
        },
        {
            "name": "audit-all-actions",
            "description": "Require audit logging for all actions",
            "enabled": true,
            "severity": "high",
            "rego": "package bridge.audit\n\ndefault audit_required = true"
        },
        {
            "name": "no-external-tools",
            "description": "Block execution of external tools",
            "enabled": true,
            "severity": "critical",
            "rego": "package bridge.tools\n\ndefault allow = true\n\nblocked_tools := [\"shell\", \"exec\", \"system\"]\n\nallow = false {\n    some tool in input.step.tools\n    tool.name == blocked_tools[_]\n}"
        }
    ]'
);
