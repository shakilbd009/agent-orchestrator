/**
 * Component tests for Webhooks page (+page.svelte)
 * E2E contract: evals/e2e/brd-02-platform-orchestration-webhooks.md
 *
 * Verifies: webhook registration form, delete confirmation, project isolation,
 * event chip suggestions (all 18 types), HMAC signature badge.
 *
 * Architecture: ALL vi.mock calls are at TOP LEVEL of this file (hoisted by Vitest).
 * vitest-setup.ts only provides jsdom globals (localStorage, EventSource).
 * SvelteKit virtual modules ($app/navigation, $app/state, $lib/api/client) are mocked here
 * so the mock is registered before any module graph is resolved.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/svelte/svelte5';
import { tick } from 'svelte';

// ---------------------------------------------------------------------------
// All mock data + fn refs created together inside vi.hoisted() so both are
// available before vi.mock() factories run and the module graph resolves.
// pageState is a plain mutable object that backs the $app/state mock Proxy —
// tests mutate it in beforeEach to control the URL before rendering.
// ---------------------------------------------------------------------------
const {
  listWebhooksFn,
  registerWebhookFn,
  deleteWebhookFn,
  MOCK_WEBHOOKS,
  pageState,
} = vi.hoisted(() => {
  const webhooks = [
    {
      id: 'wh-1',
      projectId: 'proj-alpha',
      url: 'https://consumer.example.com/webhook',
      events: ['task.created', 'task.completed'],
      active: true,
      secret: '***',
      deliveryStatus: {
        lastAttemptAt: '2026-05-28T12:00:00Z',
        lastSuccessAt: '2026-05-28T11:00:00Z',
        failureCount: 0,
      },
      createdAt: '2026-05-28T10:00:00Z',
    },
    {
      id: 'wh-2',
      projectId: 'proj-alpha',
      url: 'https://other.example.com/hook',
      events: ['gate.approved'],
      active: false,
      secret: '***',
      deliveryStatus: {
        lastAttemptAt: '2026-05-28T09:00:00Z',
        lastSuccessAt: null,
        failureCount: 3,
      },
      createdAt: '2026-05-28T08:00:00Z',
    },
  ];

  // Mutable state object — tests modify .url in-place before rendering
  const pageState = {
    url: new URL('http://localhost/orchestration/webhooks?project=proj-alpha'),
  };

  return {
    MOCK_WEBHOOKS: webhooks,
    listWebhooksFn: vi.fn<any>().mockResolvedValue({ webhooks }),
    registerWebhookFn: vi.fn<any>().mockResolvedValue({
      ...webhooks[0]!,
      id: 'wh-new',
      url: 'https://new.example.com/hook',
    }),
    deleteWebhookFn: vi.fn<any>().mockResolvedValue(undefined),
    pageState,
  };
});

// ---------------------------------------------------------------------------
// ALL vi.mock calls at top level — hoisted before any module resolution
// ---------------------------------------------------------------------------

// $app/state — SvelteKit 5 runes-compatible mock.
// page.url is wrapped in a Proxy so that Svelte 5's $derived macro can
// intercept and track the reactive read of page.url.searchParams.
// The underlying URL object (pageState.url) is mutated in-place by tests.
vi.mock('$app/state', () => ({
  page: new Proxy(
    { __brand: 'page' },
    {
      get(_target: any, prop: string) {
        if (prop === 'url') {
          return pageState.url; // plain URL — Svelte 5 Proxy traps this read
        }
        return undefined;
      },
    },
  ),
}));

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
  preloadData: vi.fn(),
  preloadRoute: vi.fn(),
  invalidate: vi.fn(),
  invalidateAll: vi.fn(),
  onNavigate: vi.fn(),
  afterNavigate: vi.fn(),
  beforeNavigate: vi.fn(),
}));

vi.mock('$app/forms', () => ({
  enhance: vi.fn(() => () => {}),
}));

// $app/stores is kept for any code that imports it, but the webhooks
// component uses $app/state so this is not the active mock.
vi.mock('$app/stores', () => ({
  page: {
    subscribe: vi.fn((fn: (v: unknown) => void) => {
      fn({ url: pageState.url });
      return () => {};
    }),
  },
  navigating: { subscribe: vi.fn(() => () => {}) },
  updated: { subscribe: vi.fn(() => () => {}) },
}));

vi.mock('$lib/api/client', () => ({
  listProjectWebhooks: listWebhooksFn,
  registerProjectWebhook: registerWebhookFn,
  deleteProjectWebhook: deleteWebhookFn,
  listProjects: vi.fn().mockResolvedValue({ projects: [] }),
  createProject: vi.fn().mockResolvedValue({ id: 'proj-new', name: 'New Project' }),
  deleteProject: vi.fn().mockResolvedValue(undefined),
  listProjectTasks: vi.fn().mockResolvedValue({ tasks: [] }),
  getProject: vi.fn().mockResolvedValue({ id: 'proj-alpha', name: 'Alpha' }),
  completeProjectTask: vi.fn().mockResolvedValue({}),
  blockProjectTask: vi.fn().mockResolvedValue({}),
  promoteProjectTask: vi.fn().mockResolvedValue({}),
  createTaskGate: vi.fn().mockResolvedValue({}),
  updateTaskGate: vi.fn().mockResolvedValue({}),
  listProjectPhaseGates: vi.fn().mockResolvedValue({ gates: [] }),
  updateProjectPhaseGate: vi.fn().mockResolvedValue({}),
  proposeDecomposition: vi.fn().mockResolvedValue({ id: 'prop-1', state: 'submitted', proposedTasks: [] }),
  getReady: vi.fn().mockResolvedValue({ status: 'ready', timestamp: '2026-05-28T00:00:00Z' }),
  getHealth: vi.fn().mockResolvedValue({ status: 'healthy', timestamp: '2026-05-28T00:00:00Z' }),
  getStatus: vi.fn().mockResolvedValue({ phase: 'phase-1', gates: [], featureFlags: {}, timestamp: '2026-05-28T00:00:00Z' }),
  createSSEConnection: vi.fn(() => ({ close: vi.fn() })),
  listHandoffEvidence: vi.fn().mockResolvedValue({ records: [] }),
  getTaskDependencies: vi.fn().mockResolvedValue({ taskId: '', parents: [], children: [] }),
  replaceTaskDependencies: vi.fn().mockResolvedValue({ taskId: '', parents: [], children: [] }),
  getTaskGate: vi.fn().mockResolvedValue({}),
  getProjectPhaseGate: vi.fn().mockResolvedValue({}),
  updateProject: vi.fn().mockResolvedValue({}),
  getProjectTask: vi.fn().mockResolvedValue({}),
}));

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

async function renderPage() {
  const Page = (await import('./+page.svelte')).default;
  const result = render(Page);
  // Flush reactive updates — wait for microtasks + tick for onMount + reactive state
  await Promise.resolve();
  await tick();
  await Promise.resolve();
  await tick();
  return result;
}

// FR-02-024 — Webhook registration form
describe('Webhook registration form — FR-02-024', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cleanup();
    // Ensure default URL has ?project=proj-alpha (in-place mutation)
    pageState.url = new URL('http://localhost/orchestration/webhooks?project=proj-alpha');
    listWebhooksFn.mockResolvedValue({ webhooks: MOCK_WEBHOOKS });
    registerWebhookFn.mockResolvedValue({
      ...MOCK_WEBHOOKS[0]!,
      id: 'wh-new',
      url: 'https://new.example.com/hook',
    });
    deleteWebhookFn.mockResolvedValue(undefined);
  });

  it('shows register webhook button when project is selected', async () => {
    await renderPage();
    expect(screen.getByRole('button', { name: /register webhook/i })).toBeTruthy();
  });

  it('opens register form when button is clicked', async () => {
    await renderPage();
    await fireEvent.click(screen.getByRole('button', { name: /register webhook/i }));
    expect(screen.getByRole('form')).toBeTruthy();
  });

  it('calls registerProjectWebhook with comma-parsed events on submit', async () => {
    await renderPage();
    await fireEvent.click(screen.getByRole('button', { name: /register webhook/i }));

    await fireEvent.input(screen.getByLabelText(/endpoint url/i), {
      target: { value: 'https://new.example.com/hook' },
    });
    await fireEvent.input(screen.getByLabelText(/events/i), {
      target: { value: 'task.created, task.completed' },
    });
    await fireEvent.input(screen.getByLabelText(/secret/i), {
      target: { value: 'my-secret' },
    });

    await fireEvent.submit(screen.getByRole('form'));

    expect(registerWebhookFn).toHaveBeenCalledWith('proj-alpha', {
      url: 'https://new.example.com/hook',
      events: ['task.created', 'task.completed'],
      secret: 'my-secret',
    });
  });

  it('closes form after successful registration', async () => {
    await renderPage();
    await fireEvent.click(screen.getByRole('button', { name: /register webhook/i }));

    await fireEvent.input(screen.getByLabelText(/endpoint url/i), {
      target: { value: 'https://new.example.com/hook' },
    });
    await fireEvent.input(screen.getByLabelText(/events/i), {
      target: { value: 'task.created' },
    });
    await fireEvent.input(screen.getByLabelText(/secret/i), {
      target: { value: 's' },
    });

    await fireEvent.submit(screen.getByRole('form'));

    // Wait for async handleRegister to complete (API call + form reset).
    await waitFor(() => {
      expect(screen.queryByRole('form')).toBeNull();
    }, { timeout: 1000 });
  });

  it('does not call API when URL is empty', async () => {
    await renderPage();
    await fireEvent.click(screen.getByRole('button', { name: /register webhook/i }));

    await fireEvent.input(screen.getByLabelText(/events/i), {
      target: { value: 'task.created' },
    });
    await fireEvent.input(screen.getByLabelText(/secret/i), {
      target: { value: 's' },
    });

    await fireEvent.submit(screen.getByRole('form'));

    expect(registerWebhookFn).not.toHaveBeenCalled();
  });
});

// FR-02-024 — Webhook list display
describe('Webhook cards — FR-02-024', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cleanup();
    pageState.url = new URL('http://localhost/orchestration/webhooks?project=proj-alpha');
    listWebhooksFn.mockResolvedValue({ webhooks: MOCK_WEBHOOKS });
  });

  it('renders all registered webhooks', async () => {
    await renderPage();
    // findBy* queries poll until the element appears (waits for async loadWebhooks)
    expect(await screen.findByText('https://consumer.example.com/webhook')).toBeTruthy();
    expect(await screen.findByText('https://other.example.com/hook')).toBeTruthy();
  });

  it('shows X-Webhook-Signature badge on each card (HMAC per ADR-02-003)', async () => {
    await renderPage();
    const badges = await screen.findAllByText('X-Webhook-Signature');
    expect(badges.length).toBeGreaterThanOrEqual(2);
  });

  it('shows Active/Inactive badge per registration', async () => {
    await renderPage();
    const activeBadges = await screen.findAllByText('Active');
    const inactiveBadges = await screen.findAllByText('Inactive');
    expect(activeBadges.length).toBeGreaterThanOrEqual(1);
    expect(inactiveBadges.length).toBeGreaterThanOrEqual(1);
  });

  it('renders all subscribed event tags', async () => {
    await renderPage();
    expect(await screen.findByText('task.created')).toBeTruthy();
    expect(await screen.findByText('task.completed')).toBeTruthy();
    expect(await screen.findByText('gate.approved')).toBeTruthy();
  });

  it('shows delivery failure count for failed webhooks', async () => {
    await renderPage();
    expect(await screen.findByText(/3/)).toBeTruthy();
  });
});

// FR-02-024 — Delete confirmation
describe('Delete webhook — FR-02-024', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cleanup();
    pageState.url = new URL('http://localhost/orchestration/webhooks?project=proj-alpha');
    listWebhooksFn.mockResolvedValue({ webhooks: MOCK_WEBHOOKS });
    deleteWebhookFn.mockResolvedValue(undefined);
  });

  it('opens confirmation modal when delete button is clicked', async () => {
    await renderPage();
    const deleteBtns = await screen.findAllByRole('button', { name: /^delete$/i });
    await fireEvent.click(deleteBtns[0]!);
    expect(screen.getByRole('dialog')).toBeTruthy();
  });

  it('calls deleteProjectWebhook with correct IDs on confirm', async () => {
    await renderPage();
    const deleteBtns = await screen.findAllByRole('button', { name: /^delete$/i });
    await fireEvent.click(deleteBtns[0]!);
    // Wait for the modal to appear then click the confirm button inside it
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).toBeTruthy();
    });
    const dialog = screen.getByRole('dialog');
    // All delete buttons visible — use the one inside the dialog
    const allDeleteBtns = screen.getAllByRole('button', { name: /^delete$/i });
    const modalDeleteBtn = allDeleteBtns.find((btn) => dialog.contains(btn));
    await fireEvent.click(modalDeleteBtn!);
    expect(deleteWebhookFn).toHaveBeenCalledWith('proj-alpha', 'wh-1');
  });

  it('closes modal without calling delete on cancel', async () => {
    await renderPage();
    const deleteBtns = await screen.findAllByRole('button', { name: /^delete$/i });
    await fireEvent.click(deleteBtns[0]!);
    await fireEvent.click(screen.getByRole('button', { name: /^cancel$/i }));
    expect(deleteWebhookFn).not.toHaveBeenCalled();
  });
});

// Project isolation — FR-02-001
describe('Project isolation — FR-02-001', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cleanup();
    // Remove ?project= to exercise the no-project guard
    pageState.url = new URL('http://localhost/orchestration/webhooks');
  });

  it('shows "select a project" message when ?project= is absent', async () => {
    await renderPage();
    expect(screen.getByText(/select a project/i)).toBeTruthy();
  });
});

// All 18 event type chips — FR-02-024
describe('Event chip suggestions — FR-02-024', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cleanup();
    pageState.url = new URL('http://localhost/orchestration/webhooks?project=proj-alpha');
    listWebhooksFn.mockResolvedValue({ webhooks: MOCK_WEBHOOKS });
  });

  it('renders all 18 available event type chips', async () => {
    await renderPage();
    await fireEvent.click(screen.getByRole('button', { name: /register webhook/i }));
    // Now the form should be visible — find it by id since findByRole('form') would
    // have timed out (form was hidden before the click above)
    const form = screen.getByRole('form');

    const expected = [
      'task.created', 'task.status.changed', 'task.stale.detected', 'task.blocked',
      'task.cancelled', 'task.completed', 'task.decomposition.proposed',
      'task.decomposition.approved', 'task.decomposition.rejected',
      'gate.opened', 'gate.approved', 'gate.rejected',
      'project.created', 'agent.activated', 'agent.idle',
      'handoff.submitted', 'auth.mutation.denied', 'webhook.delivery.failed',
    ];

    // Get all event chip buttons within the form
    const chipButtons = Array.from(form.querySelectorAll<HTMLButtonElement>('.event-chip'));
    const chipTexts = chipButtons.map((btn) => btn.textContent?.trim() ?? '');
    for (const ev of expected) {
      expect(chipTexts).toContain(ev);
    }
  });
});
