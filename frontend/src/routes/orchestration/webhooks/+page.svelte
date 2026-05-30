<script lang="ts">
  import { page } from '$app/stores';
  import { listProjectWebhooks, registerProjectWebhook, deleteProjectWebhook } from '$lib/api/client';
  import type { WebhookRegistration, WebhookRegistrationRequest } from '$lib/api/orchestration';

  // Use $app/stores for test compatibility. The $effect tracks the page store
  // reactively; on every projectId change (including initial), it clears webhooks
  // and loads fresh data. No onMount timing issues.
  let projectId = $derived($page.url.searchParams.get('project') ?? '');
  let loadedProjectId = '';

  $effect(() => {
    const currentProject = projectId;
    if (currentProject && currentProject !== loadedProjectId) {
      loadedProjectId = currentProject;
      webhooks = [];
      Promise.resolve().then(() => loadWebhooks());
    }
  });

  let webhooks = $state<WebhookRegistration[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Register form
  let showRegister = $state(false);
  let formUrl = $state('');
  let formEvents = $state('');
  let formSecret = $state('');
  let formLoading = $state(false);
  let formErr = $state<string | null>(null);

  // Delete confirmation
  let deleteTarget = $state<WebhookRegistration | null>(null);
  let deleteLoading = $state(false);

  const AVAILABLE_EVENTS = [
    'task.created',
    'task.status.changed',
    'task.stale.detected',
    'task.blocked',
    'task.cancelled',
    'task.completed',
    'task.decomposition.proposed',
    'task.decomposition.approved',
    'task.decomposition.rejected',
    'gate.opened',
    'gate.approved',
    'gate.rejected',
    'project.created',
    'agent.activated',
    'agent.idle',
    'handoff.submitted',
    'auth.mutation.denied',
    'webhook.delivery.failed',
  ];

  async function loadWebhooks() {
    console.error('loadWebhooks running, projectId:', projectId, 'loading was:', loading);
    if (!projectId) {
      console.error('loadWebhooks: no projectId, setting error');
      error = 'No project selected. Go to Projects and select one.';
      loading = false;
      console.error('loadWebhooks: error set, loading now:', loading);
      return;
    }
    console.error('loadWebhooks: calling API with projectId:', projectId);
    loading = true;
    error = null;
    try {
      const res = await listProjectWebhooks(projectId);
      console.error('loadWebhooks: API returned, res:', res, 'webhooks:', res?.webhooks?.length);
      webhooks = res.webhooks;
      console.error('loadWebhooks: webhooks state set, loading:', loading);
    } catch (e) {
      console.error('loadWebhooks: API error:', e);
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
      console.error('loadWebhooks: finally, loading now:', loading);
    }
  }

  async function handleRegister() {
    formErr = null;
    const events = formEvents
      .split(',')
      .map((e) => e.trim())
      .filter((e) => e.length > 0);

    if (!formUrl) {
      formErr = 'URL is required.';
      return;
    }
    if (events.length === 0) {
      formErr = 'At least one event is required.';
      return;
    }
    if (!formSecret) {
      formErr = 'Secret is required for HMAC signing.';
      return;
    }

    const payload: WebhookRegistrationRequest = {
      url: formUrl,
      events,
      secret: formSecret,
    };

    formLoading = true;
    try {
      const created = await registerProjectWebhook(projectId, payload);
      webhooks = [...webhooks, created];
    } catch (e) {
      formErr = e instanceof Error ? e.message : String(e);
      formLoading = false;
      return;
    } finally {
      formLoading = false;
    }
    // Only close form and reset fields on success
    showRegister = false;
    formUrl = '';
    formEvents = '';
    formSecret = '';
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    deleteLoading = true;
    try {
      await deleteProjectWebhook(projectId, deleteTarget.id);
      webhooks = webhooks.filter((w) => w.id !== deleteTarget!.id);
      deleteTarget = null;
    } catch (e) {
      // keep confirmation open, show inline error
      error = e instanceof Error ? e.message : String(e);
    } finally {
      deleteLoading = false;
    }
  }

  function formatDate(iso: string | null): string {
    if (!iso) return 'Never';
    return new Date(iso).toLocaleString();
  }
</script>

<svelte:head>
  <title>Webhooks{projectId ? ` — Project ${projectId}` : ''}</title>
</svelte:head>

<div class="page">
  <div class="page-header">
    <div class="title-row">
      <h1>Webhooks</h1>
      {#if !projectId}
        <span class="no-project-badge">No project selected</span>
      {/if}
    </div>
    <p class="subtitle">
      Register URLs to receive signed webhook events via
      <code>X-Webhook-Signature</code> (HMAC-SHA256).
    </p>
  </div>

  {#if loading}
    <div class="state-block">
      <p>Loading webhooks...</p>
    </div>
  {:else if error && webhooks.length === 0}
    <div class="state-block error">
      <p>{error}</p>
      <button class="btn-secondary" onclick={loadWebhooks}>Retry</button>
    </div>
  {:else if !projectId}
    <div class="state-block">
      <p>Select a project to manage its webhooks.</p>
    </div>
  {:else}
    <!-- Action bar -->
    <div class="action-bar">
      <button
        class="btn-primary"
        id="register-webhook-btn"
        name="register-webhook-btn"
        onclick={() => { showRegister = !showRegister; formErr = null; }}
      >
        {showRegister ? 'Cancel' : '+ Register Webhook'}
      </button>
      <button class="btn-secondary" id="refresh-btn" name="refresh-btn" onclick={loadWebhooks}>
        Refresh
      </button>
    </div>

    <!-- Register form -->
    {#if showRegister}
      <div class="form-card">
        <h2>Register Webhook</h2>
        <form
          id="register-webhook-form"
          name="register-webhook-form"
          onsubmit={(e) => { e.preventDefault(); handleRegister(); }}
        >
          <div class="field">
            <label for="webhook-url">Endpoint URL <span class="required">*</span></label>
            <input
              type="url"
              id="webhook-url"
              name="webhook-url"
              placeholder="https://your-service.example.com/webhook"
              bind:value={formUrl}
              required
            />
          </div>

          <div class="field">
            <label for="webhook-events">
              Events <span class="required">*</span>
              <span class="hint">(comma-separated)</span>
            </label>
            <input
              type="text"
              id="webhook-events"
              name="webhook-events"
              placeholder="task.created, task.completed, gate.approved"
              bind:value={formEvents}
              required
            />
            <div class="event-suggestions">
              {#each AVAILABLE_EVENTS as ev}
                <button
                  type="button"
                  class="event-chip"
                  onclick={() => {
                    const current = formEvents
                      .split(',')
                      .map((s) => s.trim())
                      .filter((s) => s.length > 0);
                    if (!current.includes(ev)) {
                      formEvents = [...current, ev].join(', ');
                    }
                  }}
                >
                  {ev}
                </button>
              {/each}
            </div>
          </div>

          <div class="field">
            <label for="webhook-secret">
              Secret <span class="required">*</span>
              <span class="x-sig-badge" title="Payload signed with HMAC-SHA256 using this secret">
                X-Webhook-Signature
              </span>
            </label>
            <input
              type="password"
              id="webhook-secret"
              name="webhook-secret"
              placeholder="Shared secret for HMAC signing"
              bind:value={formSecret}
              required
            />
            <p class="field-hint">
              Used to compute the <code>X-Webhook-Signature</code> header on every delivery.
            </p>
          </div>

          {#if formErr}
            <p class="form-error">{formErr}</p>
          {/if}

          <div class="form-actions">
            <button
              type="submit"
              class="btn-primary"
              id="submit-webhook-btn"
              name="submit-webhook-btn"
              disabled={formLoading}
            >
              {formLoading ? 'Registering...' : 'Register'}
            </button>
            <button
              type="button"
              class="btn-secondary"
              onclick={() => { showRegister = false; formErr = null; }}
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    {/if}

    <!-- Webhook list -->
    {#if webhooks.length === 0 && !showRegister}
      <div class="empty-state">
        <p>No webhooks registered for this project.</p>
        <p>Click "Register Webhook" to add one.</p>
      </div>
    {:else}
      <div class="webhook-list">
        {#each webhooks as wh (wh.id)}
          <div class="webhook-card" class:inactive={!wh.active}>
            <div class="wh-header">
              <div class="wh-url-row">
                <span class="wh-url">{wh.url}</span>
                <span
                  class="wh-active-badge"
                  class:active={wh.active}
                  class:inactive={!wh.active}
                >
                  {wh.active ? 'Active' : 'Inactive'}
                </span>
                <span class="x-sig-indicator" title="HMAC-SHA256 signed">
                  X-Webhook-Signature
                </span>
              </div>
              <button
                class="btn-danger-sm"
                id="delete-{wh.id}"
                name="delete-{wh.id}"
                onclick={() => { deleteTarget = wh; }}
              >
                Delete
              </button>
            </div>

            <div class="wh-events">
              {#each wh.events as ev}
                <span class="event-tag">{ev}</span>
              {/each}
            </div>

            <div class="wh-footer">
              <div class="wh-meta">
                <span class="meta-item">
                  <span class="meta-label">Created</span>
                  {formatDate(wh.createdAt)}
                </span>
                <span class="meta-item">
                  <span class="meta-label">Last attempt</span>
                  {formatDate(wh.deliveryStatus.lastAttemptAt)}
                </span>
                <span class="meta-item">
                  <span class="meta-label">Last success</span>
                  {formatDate(wh.deliveryStatus.lastSuccessAt)}
                </span>
              </div>
              <div class="wh-stats">
                <span class="stat success" title="Successful deliveries">
                  &#10003; {wh.deliveryStatus.failureCount === 0 && wh.deliveryStatus.lastSuccessAt ? 'OK' : '—'}
                </span>
                <span class="stat failure" title="Failed deliveries">
                  &#10007; {wh.deliveryStatus.failureCount}
                </span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<!-- Delete confirmation modal -->
{#if deleteTarget}
  <div class="modal-overlay" onclick={() => { if (!deleteLoading) deleteTarget = null; }}>
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
      <h3>Delete Webhook</h3>
      <p>
        Delete the webhook registered at<br />
        <code>{deleteTarget.url}</code>?
      </p>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
      <div class="modal-actions">
        <button
          class="btn-danger"
          id="confirm-delete-btn"
          name="confirm-delete-btn"
          disabled={deleteLoading}
          onclick={handleDelete}
        >
          {deleteLoading ? 'Deleting...' : 'Delete'}
        </button>
        <button
          class="btn-secondary"
          id="cancel-delete-btn"
          name="cancel-delete-btn"
          disabled={deleteLoading}
          onclick={() => { deleteTarget = null; error = null; }}
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    max-width: 860px;
    margin: 0 auto;
    animation: page-enter 0.25s ease-out;
  }

  @keyframes page-enter {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .page-header {
    margin-bottom: 1.5rem;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  h1 {
    font-size: 1.25rem;
    font-weight: 600;
    margin: 0;
    color: #e0e0e0;
  }

  .subtitle {
    margin: 0.25rem 0 0;
    color: #888;
    font-size: 0.875rem;
  }

  .no-project-badge {
    background: #2a2a2a;
    color: #888;
    font-size: 0.75rem;
    padding: 0.2em 0.5em;
    border-radius: 4px;
  }

  .state-block {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 2rem;
    text-align: center;
    color: #888;
    margin-bottom: 1rem;
  }

  .state-block.error {
    border-color: #7f1d1d;
    color: #f87171;
  }

  .state-block p {
    margin: 0 0 0.75rem;
  }

  .action-bar {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .form-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1.25rem;
    margin-bottom: 1.25rem;
  }

  .form-card h2 {
    font-size: 1rem;
    font-weight: 600;
    margin: 0 0 1rem;
    color: #e0e0e0;
  }

  .field {
    margin-bottom: 1rem;
  }

  label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: #a0a0a0;
    margin-bottom: 0.35rem;
  }

  .required {
    color: #f87171;
  }

  .hint {
    color: #666;
    font-size: 0.75rem;
  }

  input[type='url'],
  input[type='text'],
  input[type='password'] {
    width: 100%;
    background: #0f0f0f;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 0.5rem 0.75rem;
    color: #e0e0e0;
    font-size: 0.875rem;
    box-sizing: border-box;
  }

  input:focus {
    outline: none;
    border-color: #4a5568;
  }

  .event-suggestions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-top: 0.5rem;
  }

  .event-chip {
    background: #1e1e1e;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 0.2em 0.5em;
    font-size: 0.7rem;
    color: #7dd3fc;
    cursor: pointer;
    transition: background 0.15s;
    font-family: monospace;
  }

  .event-chip:hover {
    background: #2a2a2a;
  }

  .x-sig-badge {
    background: #1a2a1a;
    border: 1px solid #22543d;
    border-radius: 4px;
    padding: 0.1em 0.4em;
    font-size: 0.65rem;
    color: #4ade80;
    font-family: monospace;
  }

  .field-hint {
    margin: 0.3rem 0 0;
    font-size: 0.75rem;
    color: #666;
  }

  .field-hint code {
    background: #1e1e1e;
    padding: 0.1em 0.3em;
    border-radius: 3px;
    font-size: 0.9em;
    color: #86efac;
  }

  .form-error {
    color: #f87171;
    font-size: 0.8rem;
    margin: 0 0 0.75rem;
  }

  .form-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 1rem;
  }

  .btn-primary {
    background: #2a2a2a;
    border: 1px solid #444;
    border-radius: 4px;
    padding: 0.4rem 0.9rem;
    color: #e0e0e0;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-primary:hover:not(:disabled) {
    background: #383838;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 0.4rem 0.9rem;
    color: #a0a0a0;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-secondary:hover {
    background: #252525;
  }

  .btn-danger {
    background: #7f1d1d;
    border: 1px solid #991b1b;
    border-radius: 4px;
    padding: 0.4rem 0.9rem;
    color: #fca5a5;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-danger:hover:not(:disabled) {
    background: #991b1b;
  }

  .btn-danger:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-danger-sm {
    background: none;
    border: 1px solid #7f1d1d;
    border-radius: 4px;
    padding: 0.2rem 0.5rem;
    color: #f87171;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.15s;
    flex-shrink: 0;
  }

  .btn-danger-sm:hover {
    background: #7f1d1d;
  }

  .webhook-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .webhook-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem;
    transition: border-color 0.15s, box-shadow 0.15s;
    animation: card-enter 0.2s ease-out both;
  }

  .webhook-card:hover {
    border-color: #3a3a3a;
    box-shadow: 0 2px 12px rgba(0,0,0,0.3);
  }

  @keyframes card-enter {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .webhook-card.inactive {
    opacity: 0.6;
  }

  .wh-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.5rem;
    margin-bottom: 0.6rem;
  }

  .wh-url-row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.4rem;
    min-width: 0;
  }

  .wh-url {
    font-family: monospace;
    font-size: 0.8rem;
    color: #e0e0e0;
    word-break: break-all;
  }

  .wh-active-badge {
    font-size: 0.65rem;
    padding: 0.15em 0.4em;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .wh-active-badge.active {
    background: #14532d;
    color: #4ade80;
    border: 1px solid #22543d;
  }

  .wh-active-badge.inactive {
    background: #1f1f1f;
    color: #666;
    border: 1px solid #333;
  }

  .x-sig-indicator {
    background: #1a2a1a;
    border: 1px solid #22543d;
    border-radius: 4px;
    padding: 0.15em 0.4em;
    font-size: 0.6rem;
    color: #4ade80;
    font-family: monospace;
    flex-shrink: 0;
  }

  .wh-events {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-bottom: 0.75rem;
  }

  .event-tag {
    background: #0f0f0f;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 0.2em 0.5em;
    font-size: 0.7rem;
    color: #7dd3fc;
    font-family: monospace;
  }

  .wh-footer {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .wh-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .meta-item {
    font-size: 0.7rem;
    color: #666;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .meta-label {
    color: #4a4a4a;
  }

  .wh-stats {
    display: flex;
    gap: 0.5rem;
    font-size: 0.75rem;
  }

  .stat {
    font-family: monospace;
  }

  .stat.success {
    color: #4ade80;
  }

  .stat.failure {
    color: #f87171;
  }

  .empty-state {
    text-align: center;
    padding: 2.5rem;
    color: #666;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
  }

  .empty-state p {
    margin: 0 0 0.4rem;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 500;
  }

  .modal {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 400px;
    width: 90%;
  }

  .modal h3 {
    margin: 0 0 0.75rem;
    font-size: 1rem;
    color: #e0e0e0;
  }

  .modal p {
    margin: 0 0 1rem;
    color: #a0a0a0;
    font-size: 0.875rem;
    line-height: 1.5;
  }

  .modal p code {
    background: #0f0f0f;
    padding: 0.1em 0.3em;
    border-radius: 3px;
    color: #e0e0e0;
    font-size: 0.9em;
  }

  .modal-actions {
    display: flex;
    gap: 0.5rem;
  }

  code {
    background: #1e1e1e;
    padding: 0.15em 0.4em;
    border-radius: 4px;
    font-size: 0.875em;
    color: #e0e0e0;
  }
</style>
