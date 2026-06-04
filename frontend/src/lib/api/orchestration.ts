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

// ============================================================================
// BRD-03 — Client Portal Types
// ============================================================================

export interface ProjectsHealthSummary {
  onTrack: number;
  atRisk: number;
  blocked: number;
}

export interface ClientProjectSummary {
  id: string;
  name: string;
  health: 'on_track' | 'at_risk' | 'blocked';
  confidence: 'high' | 'medium' | 'low';
  completionPercent: number; // -1 when no active tasks (nil backend)
  nextMilestone: string | null;
  pendingDecisions: number;
  overdueDecisions: number;
  latestUpdate: string; // ISO 8601
}

export interface PortfolioDecisionSummary {
  totalPending: number;
  overdue: number;
  waitingOnClient: number;
  atRiskCount: number;
  blockedCount: number;
}

export interface ClientPortfolio {
  projectsSummary: ProjectsHealthSummary;
  projectList: ClientProjectSummary[];
  decisionSummary: PortfolioDecisionSummary;
  timestamp: string; // ISO 8601
}

export interface ClientTaskCard {
  id: string;
  projectId: string;
  projectName: string;
  title: string;
  description: string | null;
  status: TaskStatus;
  priority: number;
  layer: Layer;
  assignee: string | null;
  tags: string[];
  dueDate: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ClientTaskColumn {
  id: string;
  title: string;
  status: TaskStatus;
  taskCards: ClientTaskCard[];
  totalCount: number;
  sortOrder: number;
}

export interface ClientApprovalItem {
  id: string;
  projectId: string;
  projectName: string;
  decisionTitle: string;
  decisionType: 'milestone_approval' | 'gate_approval' | 'change_request' | 'budget_approval' | 'scope_approval';
  requestedAt: string;
  requestedBy: string;
  dueDate: string | null;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'pending' | 'approved' | 'rejected' | 'deferred';
  summary: string;
  affectedTasks: string[];
  affectedMilestones: string[];
  riskImpact: string | null;
  requiresSignOff: string[];
}

export interface ClientRiskItem {
  id: string;
  projectId: string;
  projectName: string;
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'open' | 'mitigated' | 'accepted' | 'transferred';
  raisedAt: string;
  raisedBy: string;
  mitigations: string[];
  affectedTasks: string[];
  dueDate: string | null;
  lastReviewedAt: string | null;
}

export interface ClientMilestoneItem {
  id: string;
  projectId: string;
  projectName: string;
  title: string;
  description: string | null;
  dueDate: string;
  status: 'upcoming' | 'at_risk' | 'completed' | 'missed';
  completionPercent: number;
  deliverables: string[];
  dependencies: string[];
  gateId: string | null;
}

export interface ClientComment {
  id: string;
  taskId: string;
  authorId: string;
  authorName: string;
  authorRole: 'client' | 'agent' | 'system';
  content: string;
  createdAt: string;
  updatedAt: string;
  parentCommentId: string | null;
  mentions: string[];
  attachments: string[];
}

export interface ClientProjectDetail {
  project: ClientProjectDetailProject;
  healthSummary: ProjectsHealthSummary;
  taskColumns: ClientTaskColumn[];
  upcomingMilestones: ClientMilestoneItem[];
  activeRisks: ClientRiskItem[];
  approvalsPending: ClientApprovalItem[];
  recentComments: ClientComment[];
}

/**
 * Project shape returned by GET /client-portal/projects/{projectId}.
 * Separate from ClientProjectSummary (portfolio list) — contains board,
 * approvals, risks, milestones, and comments at the full detail level.
 */
export interface ClientProjectDetailProject {
  id: string;
  name: string; // from project name in repo
  description: string | null; // nullable
  phase: ProjectPhase;
  health: 'on_track' | 'at_risk' | 'blocked';
  confidence: 'high' | 'medium' | 'low';
  taskCounts: { done: number; inProgress: number; todo: number; blocked: number };
  progressPercent: number; // computed: (done/(total - cancelled)) * 100, -1 when nil
  completionPercent: number; // -1 when nil (no active tasks)
  timestamp: string; // ISO 8601
  nextAction: string;
}

export interface ClientApprovalInbox {
  items: ClientApprovalItem[];
  totalCount: number;
  urgentCount: number;
  byPriority: {
    urgent: number;
    high: number;
    medium: number;
    low: number;
  };
  byType: Record<string, number>;
  oldestPending: string | null; // ISO date
}

export type ApprovalOutcome = 'approve' | 'reject' | 'request_changes' | 'need_more_information';

export interface ApprovalDecisionRequest {
  approvalId: string;
  outcome: ApprovalOutcome;
  comments?: string;
  signature?: string; // client identity verification
}

export interface ApprovalDecisionResponse {
  success: boolean;
  approvalId: string;
  outcome: ApprovalOutcome;
  decidedAt: string;
  decidedBy: string;
  notificationSent: boolean;
}

export interface ClientSearchResultItem {
  id: string;
  type: 'task' | 'project' | 'decision' | 'milestone' | 'risk' | 'comment';
  title: string;
  summary: string | null;
  projectId: string | null;
  projectName: string | null;
  matchedOn: string[]; // field names that matched
  highlightedText: string | null; // excerpt with match highlighted
  relevanceScore: number;
  url: string;
}

export interface ClientSearchResults {
  query: string;
  items: ClientSearchResultItem[];
  totalCount: number;
  searchDurationMs: number;
  filters: {
    type?: string[];
    projectId?: string;
    status?: TaskStatus[];
  };
}