// BRD-02 — API client
// Base URL: VITE_API_BASE_URL ?? 'http://localhost:3001'
// All project-scoped endpoints require Authorization: Bearer <token>

import type {
  Project,
  ProjectCreateRequest,
  ProjectUpdateRequest,
  ProjectListResponse,
  OrchestrationTask,
  OrchestrationTaskCreateRequest,
  ProjectTaskListResponse,
  TaskCompleteRequest,
  BlockRequest,
  TaskDependencyGraph,
  TaskDependencyReplaceRequest,
  TaskGate,
  TaskGateCreateRequest,
  TaskGateUpdateRequest,
  ProjectPhaseGate,
  ProjectPhaseGateCreateRequest,
  ProjectPhaseGateUpdateRequest,
  ProjectPhaseGateListResponse,
  DecompositionProposal,
  DecompositionProposalSubmitRequest,
  WebhookRegistration,
  WebhookRegistrationRequest,
  WebhookRegistrationListResponse,
  HandoffEvidenceListResponse,
  HealthResponse,
  ReadyResponse,
  PlatformStatus,
  EventEnvelope,
  ApiError,
} from './orchestration';

function getBase(): string {
  return import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:3001';
}

function getHeaders(): HeadersInit {
  // Auth token from localStorage — set by auth.svelte.ts on login
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('app_token') : null;
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: res.status, message: res.statusText }));
    const err = body as ApiError;
    throw Object.assign(new Error(err.message ?? `HTTP ${res.status}`), { code: err.code, status: res.status });
  }
  return res.json() as Promise<T>;
}

// ============================================================================
// Projects
// ============================================================================

export async function listProjects(): Promise<ProjectListResponse> {
  const res = await fetch(`${getBase()}/projects`, { headers: getHeaders() });
  return handleResponse<ProjectListResponse>(res);
}

export async function createProject(data: ProjectCreateRequest): Promise<Project> {
  const res = await fetch(`${getBase()}/projects`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<Project>(res);
}

export async function getProject(projectId: string): Promise<Project> {
  const res = await fetch(`${getBase()}/projects/${projectId}`, { headers: getHeaders() });
  return handleResponse<Project>(res);
}

export async function updateProject(projectId: string, data: ProjectUpdateRequest): Promise<Project> {
  const res = await fetch(`${getBase()}/projects/${projectId}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<Project>(res);
}

export async function deleteProject(projectId: string): Promise<void> {
  const res = await fetch(`${getBase()}/projects/${projectId}`, {
    method: 'DELETE',
    headers: getHeaders(),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: res.status, message: res.statusText }));
    throw Object.assign(new Error(body.message ?? `HTTP ${res.status}`), { status: res.status });
  }
}

// ============================================================================
// Tasks
// ============================================================================

export async function listProjectTasks(
  projectId: string,
  params?: { status?: string; layer?: string; assignee?: string }
): Promise<ProjectTaskListResponse> {
  const q = new URLSearchParams();
  if (params?.status) q.set('status', params.status);
  if (params?.layer) q.set('layer', params.layer);
  if (params?.assignee) q.set('assignee', params.assignee);
  const qs = q.toString();
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks${qs ? `?${qs}` : ''}`, { headers: getHeaders() });
  return handleResponse<ProjectTaskListResponse>(res);
}

export async function createProjectTask(projectId: string, data: OrchestrationTaskCreateRequest): Promise<OrchestrationTask> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<OrchestrationTask>(res);
}

export async function getProjectTask(projectId: string, taskId: string): Promise<OrchestrationTask> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}`, { headers: getHeaders() });
  return handleResponse<OrchestrationTask>(res);
}

export async function promoteProjectTask(projectId: string, taskId: string): Promise<OrchestrationTask> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/promote`, {
    method: 'POST',
    headers: getHeaders(),
  });
  return handleResponse<OrchestrationTask>(res);
}

export async function completeProjectTask(
  projectId: string,
  taskId: string,
  data: TaskCompleteRequest
): Promise<OrchestrationTask> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/complete`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<OrchestrationTask>(res);
}

export async function blockProjectTask(projectId: string, taskId: string, data: BlockRequest): Promise<OrchestrationTask> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/block`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<OrchestrationTask>(res);
}

// ============================================================================
// Task Dependencies
// ============================================================================

export async function getTaskDependencies(projectId: string, taskId: string): Promise<TaskDependencyGraph> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/dependencies`, { headers: getHeaders() });
  return handleResponse<TaskDependencyGraph>(res);
}

export async function replaceTaskDependencies(
  projectId: string,
  taskId: string,
  data: TaskDependencyReplaceRequest
): Promise<TaskDependencyGraph> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/dependencies`, {
    method: 'PUT',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<TaskDependencyGraph>(res);
}

// ============================================================================
// Task Gates
// ============================================================================

export async function createTaskGate(
  projectId: string,
  taskId: string,
  data: TaskGateCreateRequest
): Promise<TaskGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/gate`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<TaskGate>(res);
}

export async function getTaskGate(projectId: string, taskId: string, gateId: string): Promise<TaskGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/gate/${gateId}`, { headers: getHeaders() });
  return handleResponse<TaskGate>(res);
}

export async function updateTaskGate(
  projectId: string,
  taskId: string,
  gateId: string,
  data: TaskGateUpdateRequest
): Promise<TaskGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/gate/${gateId}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<TaskGate>(res);
}

// ============================================================================
// Decomposition
// ============================================================================

export async function proposeDecomposition(
  projectId: string,
  taskId: string,
  data: DecompositionProposalSubmitRequest
): Promise<DecompositionProposal> {
  const res = await fetch(`${getBase()}/projects/${projectId}/tasks/${taskId}/decomposition`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<DecompositionProposal>(res);
}

// ============================================================================
// Phase Gates
// ============================================================================

export async function listProjectPhaseGates(projectId: string): Promise<ProjectPhaseGateListResponse> {
  const res = await fetch(`${getBase()}/projects/${projectId}/phase-gates`, { headers: getHeaders() });
  return handleResponse<ProjectPhaseGateListResponse>(res);
}

export async function createProjectPhaseGate(
  projectId: string,
  data: ProjectPhaseGateCreateRequest
): Promise<ProjectPhaseGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/phase-gates`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<ProjectPhaseGate>(res);
}

export async function getProjectPhaseGate(projectId: string, gateId: string): Promise<ProjectPhaseGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/phase-gates/${gateId}`, { headers: getHeaders() });
  return handleResponse<ProjectPhaseGate>(res);
}

export async function updateProjectPhaseGate(
  projectId: string,
  gateId: string,
  data: ProjectPhaseGateUpdateRequest
): Promise<ProjectPhaseGate> {
  const res = await fetch(`${getBase()}/projects/${projectId}/phase-gates/${gateId}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<ProjectPhaseGate>(res);
}

// ============================================================================
// Webhooks
// ============================================================================

export async function listProjectWebhooks(projectId: string): Promise<WebhookRegistrationListResponse> {
  const res = await fetch(`${getBase()}/projects/${projectId}/webhooks`, { headers: getHeaders() });
  return handleResponse<WebhookRegistrationListResponse>(res);
}

export async function registerProjectWebhook(
  projectId: string,
  data: WebhookRegistrationRequest
): Promise<WebhookRegistration> {
  const res = await fetch(`${getBase()}/projects/${projectId}/webhooks`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<WebhookRegistration>(res);
}

export async function deleteProjectWebhook(projectId: string, webhookId: string): Promise<void> {
  const res = await fetch(`${getBase()}/projects/${projectId}/webhooks/${webhookId}`, {
    method: 'DELETE',
    headers: getHeaders(),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: res.status, message: res.statusText }));
    throw Object.assign(new Error(body.message ?? `HTTP ${res.status}`), { status: res.status });
  }
}

// ============================================================================
// Handoff Evidence
// ============================================================================

export async function listHandoffEvidence(
  projectId: string,
  params?: { taskId?: string; startTime?: string; endTime?: string }
): Promise<HandoffEvidenceListResponse> {
  const q = new URLSearchParams();
  if (params?.taskId) q.set('taskId', params.taskId);
  if (params?.startTime) q.set('startTime', params.startTime);
  if (params?.endTime) q.set('endTime', params.endTime);
  const qs = q.toString();
  const res = await fetch(`${getBase()}/projects/${projectId}/handoff-evidence${qs ? `?${qs}` : ''}`, {
    headers: getHeaders(),
  });
  return handleResponse<HandoffEvidenceListResponse>(res);
}

// ============================================================================
// Health / Platform Status
// ============================================================================

export async function getHealth(): Promise<HealthResponse> {
  const res = await fetch(`${getBase()}/health`);
  return handleResponse<HealthResponse>(res);
}

export async function getReady(): Promise<ReadyResponse> {
  const res = await fetch(`${getBase()}/ready`);
  return handleResponse<ReadyResponse>(res);
}

export async function getStatus(): Promise<PlatformStatus> {
  const res = await fetch(`${getBase()}/status`, { headers: getHeaders() });
  return handleResponse<PlatformStatus>(res);
}

// ============================================================================
// SSE Event Stream
// ============================================================================

export type SSEEventHandler = (envelope: EventEnvelope) => void;

export interface SSEOptions {
  projectId: string;
  events?: string[]; // filter by event types
  onEvent: SSEEventHandler;
  onError?: (err: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
}

export function createSSEConnection(opts: SSEOptions): { close: () => void } {
  const base = getBase();
  const params = new URLSearchParams();
  if (opts.events?.length) params.set('events', opts.events.join(','));
  const url = `${base}/projects/${opts.projectId}/stream${params.toString() ? `?${params}` : ''}`;

  let es: EventSource;
  let lastEventId: string | null = null;
  let reconnectAttempts = 0;
  let closed = false;
  let reconnectTimer: ReturnType<typeof setTimeout>;

  function connect() {
    const headers: Record<string, string> = {
      Accept: 'text/event-stream',
    };
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('app_token') : null;
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (lastEventId) headers['Last-Event-ID'] = lastEventId;

    // Use fetch + ReadableStream for SSE (EventSource doesn't support custom headers)
    // Fall back to EventSource for simplicity when no auth required
    es = new EventSource(url);
    opts.onConnect?.();

    es.addEventListener('message', (e: MessageEvent) => {
      try {
        const envelope = JSON.parse(e.data) as EventEnvelope;
        lastEventId = envelope.eventId;
        reconnectAttempts = 0;
        opts.onEvent(envelope);
      } catch {
        // ignore parse errors
      }
    });

    es.addEventListener('error', () => {
      es.close();
      if (closed) return;
      reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      reconnectTimer = setTimeout(connect, delay);
      opts.onError?.(new Error(`SSE disconnected, reconnecting in ${delay}ms (attempt ${reconnectAttempts})`));
    });

    es.addEventListener('open', () => {
      reconnectAttempts = 0;
    });
  }

  connect();

  return {
    close() {
      closed = true;
      clearTimeout(reconnectTimer);
      es?.close();
      opts.onDisconnect?.();
    },
  };
}