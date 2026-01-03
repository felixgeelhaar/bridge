-- Demo Sample Runs
-- Pre-created workflow runs to demonstrate different states

-- Completed PR review run
INSERT INTO workflow_runs (id, workflow_id, workflow_name, status, triggered_by, trigger_data, context, started_at, completed_at) VALUES
(
    'r1000000-0000-0000-0000-000000000001',
    'w1000000-0000-0000-0000-000000000001',
    'pr-review',
    'completed',
    'github-webhook',
    '{"event": "pull_request", "action": "opened", "pr": {"number": 42, "title": "Add user authentication", "author": "alice"}}',
    '{"review_score": 8.5, "issues_found": 3, "security_passed": true}',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour 45 minutes'
);

INSERT INTO step_runs (id, run_id, name, agent_id, status, tokens_in, tokens_out, step_order, started_at, completed_at) VALUES
('s1000000-0000-0000-0000-000000000001', 'r1000000-0000-0000-0000-000000000001', 'fetch-changes', 'file-reader', 'completed', 500, 2000, 0, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour 58 minutes'),
('s1000000-0000-0000-0000-000000000002', 'r1000000-0000-0000-0000-000000000001', 'code-review', 'code-reviewer', 'completed', 3000, 1500, 1, NOW() - INTERVAL '1 hour 58 minutes', NOW() - INTERVAL '1 hour 52 minutes'),
('s1000000-0000-0000-0000-000000000003', 'r1000000-0000-0000-0000-000000000001', 'security-scan', 'security-analyst', 'completed', 3000, 800, 2, NOW() - INTERVAL '1 hour 52 minutes', NOW() - INTERVAL '1 hour 48 minutes'),
('s1000000-0000-0000-0000-000000000004', 'r1000000-0000-0000-0000-000000000001', 'generate-summary', 'code-reviewer', 'completed', 2000, 1000, 3, NOW() - INTERVAL '1 hour 48 minutes', NOW() - INTERVAL '1 hour 46 minutes'),
('s1000000-0000-0000-0000-000000000005', 'r1000000-0000-0000-0000-000000000001', 'post-review', 'github-commenter', 'completed', 1000, 200, 4, NOW() - INTERVAL '1 hour 46 minutes', NOW() - INTERVAL '1 hour 45 minutes');

-- Running code generation workflow
INSERT INTO workflow_runs (id, workflow_id, workflow_name, status, triggered_by, trigger_data, current_step_index, started_at) VALUES
(
    'r1000000-0000-0000-0000-000000000002',
    'w1000000-0000-0000-0000-000000000002',
    'code-generation',
    'executing',
    'manual',
    '{"request": "Implement user profile API endpoint", "requirements": ["GET /api/users/:id", "PATCH /api/users/:id", "Input validation", "Rate limiting"]}',
    2,
    NOW() - INTERVAL '15 minutes'
);

INSERT INTO step_runs (id, run_id, name, agent_id, status, tokens_in, tokens_out, step_order, started_at, completed_at) VALUES
('s1000000-0000-0000-0000-000000000006', 'r1000000-0000-0000-0000-000000000002', 'analyze-requirements', 'code-generator', 'completed', 1000, 1500, 0, NOW() - INTERVAL '15 minutes', NOW() - INTERVAL '12 minutes'),
('s1000000-0000-0000-0000-000000000007', 'r1000000-0000-0000-0000-000000000002', 'generate-code', 'code-generator', 'completed', 2500, 4000, 1, NOW() - INTERVAL '12 minutes', NOW() - INTERVAL '5 minutes'),
('s1000000-0000-0000-0000-000000000008', 'r1000000-0000-0000-0000-000000000002', 'generate-tests', 'test-generator', 'running', 4000, 0, 2, NOW() - INTERVAL '5 minutes', NULL),
('s1000000-0000-0000-0000-000000000009', 'r1000000-0000-0000-0000-000000000002', 'review-output', 'code-reviewer', 'pending', 0, 0, 3, NULL, NULL),
('s1000000-0000-0000-0000-000000000010', 'r1000000-0000-0000-0000-000000000002', 'create-pr', 'github-commenter', 'pending', 0, 0, 4, NULL, NULL);

-- Awaiting approval
INSERT INTO workflow_runs (id, workflow_id, workflow_name, status, triggered_by, trigger_data, current_step_index, started_at) VALUES
(
    'r1000000-0000-0000-0000-000000000003',
    'w1000000-0000-0000-0000-000000000003',
    'security-audit',
    'awaiting_approval',
    'schedule',
    '{"scheduled_at": "2024-01-07T00:00:00Z", "scope": "full"}',
    4,
    NOW() - INTERVAL '1 hour'
);

INSERT INTO step_runs (id, run_id, name, agent_id, status, tokens_in, tokens_out, step_order, started_at, completed_at) VALUES
('s1000000-0000-0000-0000-000000000011', 'r1000000-0000-0000-0000-000000000003', 'scan-dependencies', 'security-analyst', 'completed', 1500, 2000, 0, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '55 minutes'),
('s1000000-0000-0000-0000-000000000012', 'r1000000-0000-0000-0000-000000000003', 'scan-secrets', 'security-analyst', 'completed', 2000, 500, 1, NOW() - INTERVAL '55 minutes', NOW() - INTERVAL '50 minutes'),
('s1000000-0000-0000-0000-000000000013', 'r1000000-0000-0000-0000-000000000003', 'scan-code', 'security-analyst', 'completed', 5000, 3000, 2, NOW() - INTERVAL '50 minutes', NOW() - INTERVAL '35 minutes'),
('s1000000-0000-0000-0000-000000000014', 'r1000000-0000-0000-0000-000000000003', 'generate-report', 'documentation-writer', 'completed', 6000, 4000, 3, NOW() - INTERVAL '35 minutes', NOW() - INTERVAL '20 minutes'),
('s1000000-0000-0000-0000-000000000015', 'r1000000-0000-0000-0000-000000000003', 'create-issues', 'github-commenter', 'pending', 0, 0, 4, NULL, NULL);

INSERT INTO approval_requests (id, run_id, step_name, status, requested_by, expires_at) VALUES
(
    'ap100000-0000-0000-0000-000000000001',
    'r1000000-0000-0000-0000-000000000003',
    'create-issues',
    'pending',
    'system',
    NOW() + INTERVAL '24 hours'
);

-- Failed run
INSERT INTO workflow_runs (id, workflow_id, workflow_name, status, triggered_by, trigger_data, error, current_step_index, started_at, completed_at) VALUES
(
    'r1000000-0000-0000-0000-000000000004',
    'w1000000-0000-0000-0000-000000000001',
    'pr-review',
    'failed',
    'github-webhook',
    '{"event": "pull_request", "action": "synchronize", "pr": {"number": 38, "title": "Refactor database layer"}}',
    'Security scan detected critical vulnerability: SQL injection in query builder',
    2,
    NOW() - INTERVAL '3 hours',
    NOW() - INTERVAL '2 hours 50 minutes'
);

INSERT INTO step_runs (id, run_id, name, agent_id, status, tokens_in, tokens_out, error, step_order, started_at, completed_at) VALUES
('s1000000-0000-0000-0000-000000000016', 'r1000000-0000-0000-0000-000000000004', 'fetch-changes', 'file-reader', 'completed', 500, 1500, NULL, 0, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours 58 minutes'),
('s1000000-0000-0000-0000-000000000017', 'r1000000-0000-0000-0000-000000000004', 'code-review', 'code-reviewer', 'completed', 2000, 1000, NULL, 1, NOW() - INTERVAL '2 hours 58 minutes', NOW() - INTERVAL '2 hours 54 minutes'),
('s1000000-0000-0000-0000-000000000018', 'r1000000-0000-0000-0000-000000000004', 'security-scan', 'security-analyst', 'failed', 2000, 800, 'Critical vulnerability detected: SQL injection', 2, NOW() - INTERVAL '2 hours 54 minutes', NOW() - INTERVAL '2 hours 50 minutes');

-- Sample audit events
INSERT INTO audit_events (id, type, actor, resource_type, resource_id, action, details, timestamp) VALUES
('ae100000-0000-0000-0000-000000000001', 'workflow.started', 'github-webhook', 'workflow_run', 'r1000000-0000-0000-0000-000000000001', 'start', '{"workflow_name": "pr-review", "trigger": "pull_request"}', NOW() - INTERVAL '2 hours'),
('ae100000-0000-0000-0000-000000000002', 'step.completed', 'system', 'step_run', 's1000000-0000-0000-0000-000000000002', 'complete', '{"step_name": "code-review", "tokens_used": 4500}', NOW() - INTERVAL '1 hour 52 minutes'),
('ae100000-0000-0000-0000-000000000003', 'policy.evaluated', 'system', 'workflow_run', 'r1000000-0000-0000-0000-000000000001', 'evaluate', '{"policy_name": "require-approval-for-production", "result": "pass"}', NOW() - INTERVAL '1 hour 48 minutes'),
('ae100000-0000-0000-0000-000000000004', 'approval.requested', 'system', 'workflow_run', 'r1000000-0000-0000-0000-000000000003', 'request', '{"step_name": "create-issues", "expires_at": "' || (NOW() + INTERVAL '24 hours')::text || '"}', NOW() - INTERVAL '20 minutes'),
('ae100000-0000-0000-0000-000000000005', 'workflow.failed', 'system', 'workflow_run', 'r1000000-0000-0000-0000-000000000004', 'fail', '{"error": "Security scan detected critical vulnerability", "step": "security-scan"}', NOW() - INTERVAL '2 hours 50 minutes');
