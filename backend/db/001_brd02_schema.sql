-- BRD-02 schema migration
-- All tables are project-scoped per FR-02-001

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    owner TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active', -- active, archived
    phase TEXT NOT NULL DEFAULT 'planning', -- planning, decomposition, execution, validation, acceptance, closed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);

CREATE TABLE IF NOT EXISTS orchestration_tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'todo', -- todo, in_progress, blocked, done, cancelled
    layer TEXT NOT NULL, -- A, B
    assignee TEXT DEFAULT '',
    required BOOLEAN NOT NULL DEFAULT false,
    priority INTEGER NOT NULL DEFAULT 0,
    stale BOOLEAN NOT NULL DEFAULT false,
    stale_threshold_minutes INTEGER,
    blocked_reason TEXT DEFAULT '',
    workspace_kind TEXT DEFAULT '',
    workspace_path TEXT DEFAULT '',
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Critical indexes for board read NFR (NFR-02-006: p50<300ms p95<500ms at 10k tasks)
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON orchestration_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON orchestration_tasks(project_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_execution_status ON orchestration_tasks(project_id, execution_status);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON orchestration_tasks(project_id, parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON orchestration_tasks(project_id, assignee);

CREATE TABLE IF NOT EXISTS task_parents (
    task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, parent_id)
);

CREATE TABLE IF NOT EXISTS task_children (
    task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    child_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, child_id)
);

CREATE TABLE IF NOT EXISTS task_dependencies (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    target_task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    type TEXT NOT NULL, -- blocks, depends_on, handoff
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, source_task_id, target_task_id)
);

CREATE INDEX IF NOT EXISTS idx_deps_project_id ON task_dependencies(project_id);
CREATE INDEX IF NOT EXISTS idx_deps_source ON task_dependencies(source_task_id);
CREATE INDEX IF NOT EXISTS idx_deps_target ON task_dependencies(target_task_id);

CREATE TABLE IF NOT EXISTS task_gates (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    phase TEXT NOT NULL, -- scope_review, architecture_review, implementation_review, code_review, qa_review, release_review
    state TEXT NOT NULL DEFAULT 'open', -- open, passed, blocked
    criteria TEXT[] DEFAULT '{}',
    blocking BOOLEAN NOT NULL DEFAULT true,
    passed_at TIMESTAMPTZ,
    passed_by TEXT DEFAULT '',
    override_note TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gates_project_id ON task_gates(project_id);
CREATE INDEX IF NOT EXISTS idx_gates_task_id ON task_gates(task_id);
CREATE INDEX IF NOT EXISTS idx_gates_state ON task_gates(project_id, state);

CREATE TABLE IF NOT EXISTS project_phase_gates (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phase_index INTEGER NOT NULL,
    phase TEXT NOT NULL, -- planning, decomposition, execution, validation, acceptance, closed
    state TEXT NOT NULL DEFAULT 'open', -- open, passed, blocked
    criteria TEXT[] DEFAULT '{}',
    pass_condition TEXT DEFAULT '',
    passed_at TIMESTAMPTZ,
    passed_by TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, phase_index)
);

CREATE TABLE IF NOT EXISTS decomposition_proposals (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    submitter TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'submitted', -- draft, submitted, accepted, rejected
    proposed_tasks JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_decomp_parent ON decomposition_proposals(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_decomp_state ON decomposition_proposals(parent_task_id, state);

CREATE TABLE IF NOT EXISTS webhook_registrations (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    events TEXT[] NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    secret_hash TEXT NOT NULL, -- stored HMAC secret hash, never returned raw
    delivery_last_attempt_at TIMESTAMPTZ,
    delivery_last_success_at TIMESTAMPTZ,
    delivery_failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_project_id ON webhook_registrations(project_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhook_registrations(project_id, active);

CREATE TABLE IF NOT EXISTS handoff_evidence (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES orchestration_tasks(id) ON DELETE CASCADE,
    from_agent TEXT NOT NULL,
    to_agent TEXT DEFAULT '',
    summary TEXT NOT NULL,
    artifacts TEXT[] DEFAULT '{}',
    validation_performed TEXT DEFAULT '',
    risks_or_residual_issues TEXT DEFAULT '',
    recommended_next_gate TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_handoff_project ON handoff_evidence(project_id);
CREATE INDEX IF NOT EXISTS idx_handoff_task ON handoff_evidence(task_id);

-- Audit events table (append-only per NFR-02-013)
CREATE TABLE IF NOT EXISTS audit_events (
    event_id TEXT PRIMARY KEY,
    schema_version TEXT NOT NULL DEFAULT 'v1alpha',
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    topic TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_role TEXT NOT NULL, -- human, layer_a, layer_b, system
    task_id TEXT,
    parent_task_id TEXT,
    gate_id TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_audit_project_id ON audit_events(project_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(project_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_event_id ON audit_events(event_id);
CREATE INDEX IF NOT EXISTS idx_audit_topic ON audit_events(project_id, topic);

-- Feature flags table
CREATE TABLE IF NOT EXISTS feature_flags (
    name TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Project SSE client tracking
CREATE TABLE IF NOT EXISTS sse_clients (
    client_id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    remote_addr TEXT DEFAULT '',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_event_id TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sse_project ON sse_clients(project_id);

-- Webhook delivery queue (in-process/DB-backed, no external broker per NFR-02-015)
CREATE TABLE IF NOT EXISTS webhook_delivery_queue (
    id SERIAL PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhook_registrations(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES audit_events(event_id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wdq_next_retry ON webhook_delivery_queue(next_retry_at);