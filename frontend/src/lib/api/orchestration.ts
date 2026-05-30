// BRD-02 — Platform-Native Orchestration Pipeline
// TypeScript types derived from contracts/openapi.yaml (0.2.0-brd02)
// and contracts/events.md (canonical event envelope, v1alpha)

// ============================================================================
// Enumerations
// ============================================================================

export type TaskStatus = 'todo' | 'in_progress' | 'blocked' | 'done' | 'cancelled';
export type Layer = 'A' | 'B';
export type GateState = 'open' | 'passed' | 'blocked';
export type ProjectPhase = 'planning' | 'decomposition' | 'execution' | 'validation' | 'acceptance' | 'closed';
export type DecompositionState = 'draft' | 'submitted' | 'accepted' | 'rejected';
export type DependencyType = 'blocks' | 'depends_on' | 'handoff';

// ============================================================================
// Event Envelope (events.md canonical, v1alpha)
// ============================================================================

export interface EventEnvelope {
  eventId: string;
  schemaVersion: 'v1alpha';
  projectId: string;
  topic: string;
  actorId: string;
  actorRole: 'human' | 'layer_a' | 'layer_b' | 'system';
  taskId: string | null;
  parentTaskId: string | null;
  gateId: string | null;
  timestamp: string; // ISO 8601: YYYY-MM-DDTHH:mm:ss.SSSZ
  payload: Record<string, unknown>;
}

// ============================================================================
// Project
// ============================================================================

export interface ProjectStatistics {
  totalTasks: number;
  todoTasks: number;
  inProgressTasks: number;
  doneTasks: number;
  blockedTasks: number;
  activeGateCount: number;
}

export interface Project {
  id: string;
  name: string;
  description: string | null;
  owner: string;
  status: 'active' | 'archived';
  phase: ProjectPhase;
  phaseGates: ProjectPhaseGate[];
  statistics: ProjectStatistics;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectCreateRequest {
  name: string;
  description?: string;
  phaseGates?: ProjectPhaseGateCreateRequest[];
}

export interface ProjectUpdateRequest {
  name?: string;
  description?: string;
  phaseGates?: ProjectPhaseGateCreateRequest[];
}

export interface ProjectListResponse {
  projects: Project[];
}

// ============================================================================
// OrchestrationTask
// ============================================================================

export interface TaskDependency {
  id: string;
  projectId: string;
  sourceTaskId: string;
  targetTaskId: string;
  type: DependencyType;
  createdAt: string;
}

export interface TaskDependencyGraph {
  taskId: string;
  parents: TaskDependency[];
  children: TaskDependency[];
}

export interface TaskDependencyReplaceRequest {
  dependencies: Array<{
    targetTaskId: string;
    type: DependencyType;
  }>;
}

export interface TaskGate {
  id: string;
  projectId: string;
  taskId: string;
  phase: string;
  state: GateState;
  criteria: string[];
  passedAt: string | null;
  passedBy: string | null;
  overrideNote: string | null;
  createdAt: string;
}

export interface OrchestrationTask {
  id: string;
  projectId: string;
  title: string;
  body: string | null;
  status: TaskStatus;
  layer: Layer;
  assignee: string | null;
  parents: string[];
  children: string[];
  dependencies: TaskDependency[];
  gates: TaskGate[];
  priority: number;
  createdAt: string;
  updatedAt: string;
  completedAt: string | null;
  blockedReason: string | null;
  metadata: Record<string, unknown>;
}

export interface OrchestrationTaskCreateRequest {
  title: string;
  body?: string;
  assignee: string;
  layer: Layer;
  parents?: string[];
  priority?: number;
  workspaceKind?: 'scratch' | 'dir' | 'worktree';
  tags?: string[];
  phaseGate?: string;
}

export interface ProjectTaskListResponse {
  tasks: OrchestrationTask[];
}

// ============================================================================
// Task Gate
// ============================================================================

export interface TaskGateCreateRequest {
  phase: string;
  criteria?: string[];
}

export interface TaskGateUpdateRequest {
  state?: GateState;
  criteria?: string[];
  overrideNote?: string;
}

// ============================================================================
// ProjectPhaseGate
// ============================================================================

export interface ProjectPhaseGate {
  id: string;
  projectId: string;
  phaseIndex: number;
  phase: ProjectPhase;
  state: GateState;
  criteria: string[];
  passCondition: string | null;
  passedAt: string | null;
  passedBy: string | null;
  createdAt: string;
}

export interface ProjectPhaseGateCreateRequest {
  phaseIndex: number;
  phase: ProjectPhase;
  criteria?: string[];
  passCondition?: string;
}

export interface ProjectPhaseGateUpdateRequest {
  state?: GateState;
  criteria?: string[];
  passCondition?: string;
  passedBy?: string;
}

export interface ProjectPhaseGateListResponse {
  gates: ProjectPhaseGate[];
}

// ============================================================================
// Decomposition Proposal
// ============================================================================

export interface DecomposedTaskSpec {
  title: string;
  body?: string | null;
  layer: Layer;
  assignee?: string | null;
  order: number;
  dependencies?: string[];
  tags?: string[];
}

export interface DecompositionProposal {
  id: string;
  projectId: string;
  parentTaskId: string;
  submitter: string;
  state: DecompositionState;
  proposedTasks: DecomposedTaskSpec[];
  createdAt: string;
  updatedAt: string;
}

export interface DecompositionProposalSubmitRequest {
  submitter: string;
  proposedTasks: DecomposedTaskSpec[];
  override?: boolean;
}

// ============================================================================
// Webhook Registration
// ============================================================================

export interface WebhookDeliveryStatus {
  lastAttemptAt: string | null;
  lastSuccessAt: string | null;
  failureCount: number;
}

export interface WebhookRegistration {
  id: string;
  projectId: string;
  url: string;
  events: string[];
  active: boolean;
  secret: string; // never returned after creation — placeholder '***'
  deliveryStatus: WebhookDeliveryStatus;
  createdAt: string;
}

export interface WebhookRegistrationRequest {
  url: string;
  events: string[];
  secret: string;
}

export interface WebhookRegistrationListResponse {
  webhooks: WebhookRegistration[];
}

// ============================================================================
// Handoff Evidence
// ============================================================================

export interface HandoffEvidence {
  id: string;
  projectId: string;
  taskId: string;
  fromAgent: string;
  toAgent: string | null;
  summary: string;
  metadata: Record<string, unknown>;
  createdAt: string;
}

export interface HandoffEvidenceListResponse {
  records: HandoffEvidence[];
}

// ============================================================================
// Task Actions
// ============================================================================

export interface TaskCompleteRequest {
  summary: string;
  artifacts?: string[];
  validationPerformed: string;
  risksOrResidualIssues?: string | null;
  recommendedNextGate?: string | null;
}

export interface BlockRequest {
  reason: string;
}

// ============================================================================
// Health / Platform Status
// ============================================================================

export interface HealthResponse {
  status: 'healthy' | 'degraded' | 'maintenance';
  timestamp: string;
}

export interface ReadyResponse {
  status: 'ready' | 'degraded';
  subsystems?: {
    storage?: string;
    currentState?: string;
    auditPersistence?: string;
    webhookQueue?: string;
  };
  timestamp: string;
}

export interface PlatformStatus {
  phase: 'phase-0' | 'phase-1' | 'phase-2';
  gates: GateStatus[];
  featureFlags: Record<string, boolean>;
  timestamp: string;
}

export interface GateStatus {
  gateId: string;
  name: string;
  state: 'open' | 'passed' | 'blocked';
}

// ============================================================================
// Agent Registry
// ============================================================================

export interface AgentProfile {
  name: string;
  layer: 'A' | 'B';
  status: 'active' | 'idle' | 'blocked';
  currentTaskId: string | null;
  skills: string[];
}

export interface AgentRegistry {
  agents: AgentProfile[];
  timestamp: string;
}

// ============================================================================
// SSE Event Stream
// ============================================================================

export interface SSEConnectionState {
  connected: boolean;
  lastEventId: string | null;
  reconnectAttempts: number;
}

// ============================================================================
// Error
// ============================================================================

export interface ApiError {
  code: string;
  message: string;
}