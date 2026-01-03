-- Bridge Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflow Definitions Table
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    checksum VARCHAR(64),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_definitions_name ON workflow_definitions(name);
CREATE INDEX idx_workflow_definitions_updated_at ON workflow_definitions(updated_at);

-- Workflow Runs Table
CREATE TABLE IF NOT EXISTS workflow_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflow_definitions(id) ON DELETE CASCADE,
    workflow_name VARCHAR(255) NOT NULL,
    workflow_version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    current_step_index INTEGER NOT NULL DEFAULT 0,
    context JSONB DEFAULT '{}',
    triggered_by VARCHAR(255),
    trigger_data JSONB DEFAULT '{}',
    error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_runs_workflow_id ON workflow_runs(workflow_id);
CREATE INDEX idx_workflow_runs_status ON workflow_runs(status);
CREATE INDEX idx_workflow_runs_created_at ON workflow_runs(created_at);
CREATE INDEX idx_workflow_runs_active ON workflow_runs(status) WHERE status NOT IN ('completed', 'failed', 'cancelled');

-- Step Runs Table
CREATE TABLE IF NOT EXISTS step_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input JSONB DEFAULT '{}',
    output JSONB DEFAULT '{}',
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    timeout_seconds INTEGER DEFAULT 300,
    max_retries INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    error TEXT,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    step_order INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_step_runs_run_id ON step_runs(run_id);
CREATE INDEX idx_step_runs_status ON step_runs(status);
CREATE UNIQUE INDEX idx_step_runs_run_step ON step_runs(run_id, step_index);

-- Agents Table
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(255) NOT NULL,
    system_prompt TEXT,
    max_tokens INTEGER DEFAULT 4096,
    temperature DECIMAL(3,2) DEFAULT 0.7,
    capabilities TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_active ON agents(active);

-- Policy Bundles Table
CREATE TABLE IF NOT EXISTS policy_bundles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL DEFAULT '[]',
    checksum VARCHAR(64),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_bundles_name ON policy_bundles(name);
CREATE INDEX idx_policy_bundles_active ON policy_bundles(active);

-- Approval Requests Table
CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    requested_by VARCHAR(255),
    approved_by VARCHAR(255),
    rejected_by VARCHAR(255),
    reason TEXT,
    expires_at TIMESTAMPTZ,
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_requests_run_id ON approval_requests(run_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_approval_requests_pending ON approval_requests(status, expires_at) WHERE status = 'pending';

-- Audit Events Table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(100) NOT NULL,
    actor VARCHAR(255),
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    details JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_type ON audit_events(type);
CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp);
CREATE INDEX idx_audit_events_actor ON audit_events(actor);

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers
CREATE TRIGGER update_workflow_definitions_updated_at
    BEFORE UPDATE ON workflow_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_runs_updated_at
    BEFORE UPDATE ON workflow_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policy_bundles_updated_at
    BEFORE UPDATE ON policy_bundles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
