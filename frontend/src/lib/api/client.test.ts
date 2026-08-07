/**
 * Unit tests for BRD-02 API client functions
 * Tests: webhooks CRUD, task lifecycle, decomposition, gates, SSE connection
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type {
  WebhookRegistration,
  WebhookRegistrationRequest,
  WebhookRegistrationListResponse,
  OrchestrationTask,
  ProjectTaskListResponse,
  ProjectPhaseGate,
  ProjectPhaseGateListResponse,
} from './orchestration';

// ---------------------------------------------------------------------------
// Mock localStorage — used by getHeaders() in client.ts
// ---------------------------------------------------------------------------
const localStorageMock = {
  getItem: vi.fn().mockReturnValue(null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true,
  configurable: true,
});

// ---------------------------------------------------------------------------
// Mock fetch helper
// ---------------------------------------------------------------------------
function makeMockResponse<T>(data: T, init?: ResponseInit & { status?: number }): Response {
  const status = init?.status ?? 200;
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (init?.headers) {
    for (const [k, v] of Object.entries(init.headers as Record<string, string>)) {
      headers[k] = v;
    }
  }
  return new Response(JSON.stringify(data), { status, headers } as ResponseInit);
}

function makeErrorResponse(status: number, message = 'Error'): Response {
  return new Response(JSON.stringify({ code: `ERR_${status}`, message }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  } as ResponseInit);
}

function makeEmptyResponse(status: number): Response {
  return new Response(null, { status } as ResponseInit);
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------
const MOCK_PROJECT_ID = 'proj-alpha';
const MOCK_WEBHOOK: WebhookRegistration = {
  id: 'wh-1',
  projectId: MOCK_PROJECT_ID,
  url: 'https://example.com/webhook',
  events: ['task.created', 'task.completed'],
  active: true,
  secret: '***',
  deliveryStatus: {
    lastAttemptAt: '2026-05-28T12:00:00Z',
    lastSuccessAt: '2026-05-28T11:00:00Z',
    failureCount: 0,
  },
  createdAt: '2026-05-28T10:00:00Z',
};

const MOCK_TASK: OrchestrationTask = {
  id: 'task-x',
  projectId: MOCK_PROJECT_ID,
  title: 'Test task',
  body: null,
  status: 'in_progress',
  layer: 'B',
  assignee: 'layer_b:agent1',
  parents: [],
  children: [],
  dependencies: [],
  gates: [],
  priority: 0,
  createdAt: '2026-05-28T10:00:00Z',
  updatedAt: '2026-05-28T10:00:00Z',
  completedAt: null,
  blockedReason: null,
  metadata: {},
};

const MOCK_PHASE_GATE: ProjectPhaseGate = {
  id: 'gate-1',
  projectId: MOCK_PROJECT_ID,
  phaseIndex: 0,
  phase: 'planning',
  state: 'open',
  criteria: [],
  passCondition: null,
  passedAt: null,
  passedBy: null,
  createdAt: '2026-05-28T10:00:00Z',
};

// ---------------------------------------------------------------------------
// Webhook tests — FR-02-024, FR-02-025
// ---------------------------------------------------------------------------
describe('listProjectWebhooks', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns webhooks array from API', async () => {
    const { listProjectWebhooks } = await import('./client');
    const mock = makeMockResponse<WebhookRegistrationListResponse>({ webhooks: [MOCK_WEBHOOK] });
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(mock);

    const result = await listProjectWebhooks(MOCK_PROJECT_ID);

    expect(result.webhooks).toHaveLength(1);
    expect(result.webhooks[0]!.id).toBe('wh-1');
    expect(result.webhooks[0]!.url).toBe('https://example.com/webhook');
  });

  it('throws on non-OK response', async () => {
    const { listProjectWebhooks } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(makeErrorResponse(401, 'Unauthorized'));

    await expect(listProjectWebhooks(MOCK_PROJECT_ID)).rejects.toThrow('Unauthorized');
  });
});

describe('registerProjectWebhook', () => {
  beforeEach(() => vi.clearAllMocks());

  it('posts registration payload and returns created webhook', async () => {
    const { registerProjectWebhook } = await import('./client');
    const payload: WebhookRegistrationRequest = {
      url: 'https://example.com/webhook',
      events: ['task.created'],
      secret: 'my-secret',
    };
    const mock = makeMockResponse<WebhookRegistration>({
      ...MOCK_WEBHOOK,
      url: payload.url,
      events: payload.events,
    });
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(mock);

    const result = await registerProjectWebhook(MOCK_PROJECT_ID, payload);

    expect(result.url).toBe(payload.url);
    expect(result.events).toEqual(['task.created']);

    // Verify request body
    const [, opts] = vi.mocked(global.fetch).mock.calls[0]!;
    const body = JSON.parse((opts as RequestInit).body as string);
    expect(body.url).toBe(payload.url);
    expect(body.events).toEqual(['task.created']);
    expect(body.secret).toBe(payload.secret);
  });

  it('throws on duplicate registration conflict', async () => {
    const { registerProjectWebhook } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(makeErrorResponse(409, 'Webhook already exists'));

    await expect(
      registerProjectWebhook(MOCK_PROJECT_ID, { url: 'https://dup.com', events: ['task.*'], secret: 'x' })
    ).rejects.toThrow('Webhook already exists');
  });
});

describe('deleteProjectWebhook', () => {
  beforeEach(() => vi.clearAllMocks());

  it('DELETE /projects/:id/webhooks/:whId and returns void on 204', async () => {
    const { deleteProjectWebhook } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(makeEmptyResponse(204));

    await expect(deleteProjectWebhook(MOCK_PROJECT_ID, 'wh-1')).resolves.toBeUndefined();

    const [url, opts] = vi.mocked(global.fetch).mock.calls[0]!;
    expect(url).toContain(`/projects/${MOCK_PROJECT_ID}/webhooks/wh-1`);
    expect((opts as RequestInit).method).toBe('DELETE');
  });

  it('throws on not-found', async () => {
    const { deleteProjectWebhook } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(makeErrorResponse(404, 'Webhook not found'));

    await expect(deleteProjectWebhook(MOCK_PROJECT_ID, 'wh-missing')).rejects.toThrow('Webhook not found');
  });
});

// ---------------------------------------------------------------------------
// Task lifecycle — FR-02-003, FR-02-015, FR-02-021
// ---------------------------------------------------------------------------
describe('completeProjectTask — structured handoff per FR-02-015', () => {
  beforeEach(() => vi.clearAllMocks());

  it('sends all required handoff fields', async () => {
    const { completeProjectTask } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse<OrchestrationTask>({ ...MOCK_TASK, status: 'done' })
    );

    await completeProjectTask(MOCK_PROJECT_ID, 'task-x', {
      summary: 'Completed auth service',
      validationPerformed: 'Unit tests pass, integration tests pass',
      risksOrResidualIssues: 'Production monitoring recommended',
    });

    const [, opts] = vi.mocked(global.fetch).mock.calls[0]!;
    const body = JSON.parse((opts as RequestInit).body as string);
    expect(body.summary).toBe('Completed auth service');
    expect(body.validationPerformed).toBe('Unit tests pass, integration tests pass');
    expect(body.risksOrResidualIssues).toBe('Production monitoring recommended');
  });

  it('throws 422 when required summary is missing', async () => {
    const { completeProjectTask } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeErrorResponse(422, 'summary is required')
    );

    await expect(
      completeProjectTask(MOCK_PROJECT_ID, 'task-x', {
        summary: '',
        validationPerformed: 'tests',
      } as any)
    ).rejects.toThrow('summary is required');
  });
});

describe('blockProjectTask — requires reason per FR-02-021', () => {
  beforeEach(() => vi.clearAllMocks());

  it('sends reason in block request', async () => {
    const { blockProjectTask } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse<OrchestrationTask>({ ...MOCK_TASK, status: 'blocked' })
    );

    await blockProjectTask(MOCK_PROJECT_ID, 'task-x', { reason: 'Waiting on external dependency' });

    const [, opts] = vi.mocked(global.fetch).mock.calls[0]!;
    const body = JSON.parse((opts as RequestInit).body as string);
    expect(body.reason).toBe('Waiting on external dependency');
  });
});

describe('listProjectTasks — project isolation per FR-02-001', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns only tasks scoped to the specified project', async () => {
    const { listProjectTasks } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse<ProjectTaskListResponse>({ tasks: [MOCK_TASK] })
    );

    const result = await listProjectTasks('proj-alpha');
    expect(result.tasks.every((t) => t.projectId === 'proj-alpha')).toBe(true);
  });

  it('appends status filter to query string', async () => {
    const { listProjectTasks } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse<ProjectTaskListResponse>({ tasks: [] })
    );

    await listProjectTasks('proj-alpha', { status: 'done' });
    const [url] = vi.mocked(global.fetch).mock.calls[0]!;
    expect(url.toString()).toContain('status=done');
  });
});

// ---------------------------------------------------------------------------
// Phase gates — FR-02-016, FR-02-017
// ---------------------------------------------------------------------------
describe('listProjectPhaseGates', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns phase gates for the project', async () => {
    const { listProjectPhaseGates } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse<ProjectPhaseGateListResponse>({ gates: [MOCK_PHASE_GATE] })
    );

    const result = await listProjectPhaseGates(MOCK_PROJECT_ID);
    expect(result.gates).toHaveLength(1);
    expect(result.gates[0]!.phase).toBe('planning');
  });
});

// ---------------------------------------------------------------------------
// SSE — FR-02-023
// ---------------------------------------------------------------------------
// Builds a fetch Response whose body is a live ReadableStream of SSE bytes.
function makeSSEResponse(frames: string[]): Response {
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const f of frames) controller.enqueue(new TextEncoder().encode(f));
      controller.close();
    },
  });
  return new Response(stream, {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  });
}

describe('createSSEConnection — FR-02-023', () => {
  beforeEach(() => vi.clearAllMocks());

  it('parses NAMED SSE events and dispatches the envelope (FR-02-023)', async () => {
    // Backend emits named events (event: <topic>), which never reach the
    // default 'message' handler — verify the client parses them directly.
    const envelope = {
      eventId: 'evt-1',
      schemaVersion: 'v1alpha',
      projectId: 'proj-alpha',
      topic: 'task.created',
      actorId: 'actor-1',
      actorRole: 'layer_a',
      taskId: 'task-1',
      parentTaskId: null,
      gateId: null,
      timestamp: '2026-05-28T12:00:00.000Z',
      payload: { kind: 'task' },
    };
    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeSSEResponse([
        `event: task.created\nid: ${envelope.eventId}\ndata: ${JSON.stringify(envelope)}\n\n`,
      ])
    );

    const { createSSEConnection } = await import('./client');
    const onEvent = vi.fn();
    const conn = createSSEConnection({
      projectId: 'proj-alpha',
      onEvent,
      onError: vi.fn(),
      onConnect: vi.fn(),
      onDisconnect: vi.fn(),
    });

    expect(conn).toBeDefined();
    expect(typeof conn.close).toBe('function');
    // Named event (task.created) must hit onEvent even though there is no 'message' listener.
    await vi.waitFor(() => expect(onEvent).toHaveBeenCalledWith(expect.objectContaining({ topic: 'task.created', eventId: 'evt-1' })));
    // Correct endpoint path (was previously the 404 '/stream').
    expect(fetchSpy.mock.calls[0]?.[0]).toContain('/projects/proj-alpha/events/stream');
    conn.close();
  });
});

// ---------------------------------------------------------------------------
// Health/readiness — NFR-02-012, NFR-02-014
// ---------------------------------------------------------------------------
describe('getReady — platform availability per NFR-02-012', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns ReadyResponse with status=ready', async () => {
    const { getReady } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse({ status: 'ready', timestamp: '2026-05-28T12:00:00Z' })
    );

    const result = await getReady();
    expect(result.status).toBe('ready');
    expect(result.timestamp).toBeDefined();
  });

  it('returns degraded with failing subsystem keys', async () => {
    const { getReady } = await import('./client');
    vi.spyOn(global, 'fetch').mockResolvedValueOnce(
      makeMockResponse({
        status: 'degraded',
        subsystems: { webhookQueue: 'unavailable' },
        timestamp: '2026-05-28T12:00:00Z',
      })
    );

    const result = await getReady();
    expect(result.status).toBe('degraded');
    expect(result.subsystems?.webhookQueue).toBe('unavailable');
  });
});
