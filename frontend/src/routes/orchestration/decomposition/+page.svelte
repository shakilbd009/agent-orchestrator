<script lang="ts">
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { listProjectTasks, proposeDecomposition } from '$lib/api/client';
  import type {
    OrchestrationTask,
    DecompositionProposal,
    DecompositionProposalSubmitRequest,
    DecomposedTaskSpec,
    Layer,
  } from '$lib/api/orchestration';

  let projectId = $derived(page.url.searchParams.get('project') ?? '');

  let tasks = $state<OrchestrationTask[]>([]);
  let proposals = $state<Record<string, DecompositionProposal>>({});
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Status filter tabs
  type FilterTab = 'all' | 'has-proposal' | 'needs-proposal';
  let activeFilter = $state<FilterTab>('all');

  // Proposal form state — keyed by task id so each task can have an independent form
  let proposalForms = $state<Record<string, {
    open: boolean;
    rows: DecomposedTaskSpec[];
    submitter: string;
    submitting: boolean;
    err: string | null;
  }>>({});

  // Approve/reject state
  let processing = $state<string | null>(null);

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    if (!projectId) {
      error = 'No project selected. Go to Projects and select one.';
      loading = false;
      return;
    }
    loading = true;
    error = null;
    try {
      const res = await listProjectTasks(projectId);
      tasks = res.tasks;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function refresh() {
    await loadData();
  }

  // Filtered tasks by tab
  let filteredTasks = $derived(() => {
    switch (activeFilter) {
      case 'has-proposal':
        return tasks.filter((t) => t.metadata?.decomposition_proposal_id != null);
      case 'needs-proposal':
        return tasks.filter((t) =>
          t.metadata?.decomposition_proposal_id == null && t.layer === 'A'
        );
      default:
        return tasks;
    }
  });

  // Open proposal form for a task
  function openForm(taskId: string) {
    proposalForms = {
      ...proposalForms,
      [taskId]: {
        open: true,
        rows: [
          { title: '', layer: 'B', assignee: null, order: 0, dependencies: [], tags: [] },
        ],
        submitter: '',
        submitting: false,
        err: null,
      },
    };
  }

  function closeForm(taskId: string) {
    const next = { ...proposalForms };
    delete next[taskId];
    proposalForms = next;
  }

  function addRow(taskId: string) {
    const form = proposalForms[taskId];
    if (!form) return;
    const newRows = [
      ...form.rows,
      {
        title: '',
        layer: 'B' as Layer,
        assignee: null,
        order: form.rows.length,
        dependencies: [],
        tags: [],
      },
    ];
    proposalForms = { ...proposalForms, [taskId]: { ...form, rows: newRows } };
  }

  function removeRow(taskId: string, idx: number) {
    const form = proposalForms[taskId];
    if (!form || form.rows.length <= 1) return;
    const newRows = form.rows.filter((_, i) => i !== idx);
    proposalForms = { ...proposalForms, [taskId]: { ...form, rows: newRows } };
  }

  function updateRow(taskId: string, idx: number, field: keyof DecomposedTaskSpec, value: any) {
    const form = proposalForms[taskId];
    if (!form) return;
    const newRows = form.rows.map((r, i) =>
      i === idx ? { ...r, [field]: value } : r
    );
    proposalForms = { ...proposalForms, [taskId]: { ...form, rows: newRows } };
  }

  async function submitProposal(taskId: string, e: SubmitEvent) {
    e.preventDefault();
    const form = proposalForms[taskId];
    if (!form || !form.submitter.trim()) return;

    const validRows = form.rows.filter((r) => r.title.trim());
    if (validRows.length === 0) {
      proposalForms = {
        ...proposalForms,
        [taskId]: { ...form, err: 'At least one child task must have a title.' },
      };
      return;
    }

    proposalForms = {
      ...proposalForms,
      [taskId]: { ...form, submitting: true, err: null },
    };

    try {
      const req: DecompositionProposalSubmitRequest = {
        submitter: form.submitter.trim(),
        proposedTasks: validRows.map((r, i) => ({ ...r, order: i })),
      };
      const proposal = await proposeDecomposition(projectId, taskId, req);
      proposals = { ...proposals, [taskId]: proposal };
      closeForm(taskId);
      await refresh();
    } catch (e) {
      proposalForms = {
        ...proposalForms,
        [taskId]: {
          ...form,
          submitting: false,
          err: e instanceof Error ? e.message : String(e),
        },
      };
    }
  }

  async function handleApprove(taskId: string) {
    processing = taskId;
    try {
      // Placeholder: approve via API — call update decomposition state endpoint
      // Currently no approve endpoint in client.ts; the server handles state transitions
      await new Promise((r) => setTimeout(r, 300));
      await refresh();
    } catch (e) {
      alert(`Approve failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      processing = null;
    }
  }

  async function handleReject(taskId: string) {
    processing = taskId;
    try {
      await new Promise((r) => setTimeout(r, 300));
      await refresh();
    } catch (e) {
      alert(`Reject failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      processing = null;
    }
  }

  function getProposal(taskId: string): DecompositionProposal | undefined {
    return proposals[taskId];
  }

  function stateLabel(state: string): string {
    const m: Record<string, string> = {
      draft: 'Draft',
      submitted: 'Pending Approval',
      accepted: 'Approved',
      rejected: 'Rejected',
    };
    return m[state] ?? state;
  }

  function stateColor(state: string): string {
    const m: Record<string, string> = {
      draft: '#888',
      submitted: '#fbbf24',
      accepted: '#4ade80',
      rejected: '#f87171',
    };
    return m[state] ?? '#888';
  }

  function layerBadge(layer: string): string {
    return layer === 'A' ? 'A' : 'B';
  }
</script>

<svelte:head>
  <title>Decomposition Proposals — Orchestration</title>
</svelte:head>

<div class="decomp-page">
  {#if loading}
    <div class="loading-state"><p>Loading tasks...</p></div>
  {:else if error}
    <div class="error-state">
      <p>{error}</p>
      <button onclick={refresh}>Retry</button>
    </div>
  {:else}
    <div class="page-header">
      <h1>Decomposition Proposals</h1>
      <span class="project-id">Project: {projectId}</span>
    </div>

    <div class="filter-tabs">
      <button
        class="tab-btn"
        class:active={activeFilter === 'all'}
        onclick={() => { activeFilter = 'all'; }}
      >All Tasks</button>
      <button
        class="tab-btn"
        class:active={activeFilter === 'has-proposal'}
        onclick={() => { activeFilter = 'has-proposal'; }}
      >Has Proposal</button>
      <button
        class="tab-btn"
        class:active={activeFilter === 'needs-proposal'}
        onclick={() => { activeFilter = 'needs-proposal'; }}
      >Needs Proposal</button>
      <span class="tab-count">{filteredTasks().length} tasks</span>
    </div>

    {#if filteredTasks().length === 0}
      <div class="empty-state">
        <p>No tasks match this filter.</p>
      </div>
    {:else}
      <div class="task-list">
        {#each filteredTasks() as task}
          {@const proposal = getProposal(task.id)}
          {@const form = proposalForms[task.id]}
          {@const proposalState = task.metadata?.decomposition_state as string | undefined}
          <div class="task-card">
            <div class="card-header">
              <div class="task-meta">
                <span class="layer-badge" class:layer-a={task.layer === 'A'} class:layer-b={task.layer === 'B'}>
                  {layerBadge(task.layer)}
                </span>
                <span class="task-title">{task.title}</span>
                <span class="task-id">{task.id}</span>
              </div>
              <div class="card-actions">
                {#if proposal || proposalState}
                  <span class="proposal-state" style="color: {stateColor(proposalState ?? proposal?.state ?? 'draft')}">
                    {stateLabel(proposalState ?? proposal?.state ?? 'draft')}
                  </span>
                {/if}
                {#if !form && !proposal}
                  <button class="btn-propose" onclick={() => openForm(task.id)}>
                    + Propose Decomposition
                  </button>
                {/if}
              </div>
            </div>

            {#if form?.open}
              <!-- Proposal form for this task -->
              <form class="proposal-form" onsubmit={(e) => submitProposal(task.id, e)}>
                <div class="form-header">
                  <h2>Propose Children for: {task.title}</h2>
                </div>

                <div class="form-field">
                  <label for="prop-submitter-{task.id}">Your Name / Agent ID <span class="required">*</span></label>
                  <input
                    id="prop-submitter-{task.id}"
                    name="prop-submitter-{task.id}"
                    type="text"
                    bind:value={form.submitter}
                    required
                    autocomplete="off"
                    placeholder="e.g. architect-agent or human@example.com"
                  />
                </div>

                <div class="child-tasks-section">
                  <h3>Proposed Child Tasks</h3>
                  {#each form.rows as row, idx}
                    <div class="child-row">
                      <div class="child-order">{idx + 1}</div>
                      <div class="child-fields">
                        <div class="form-field">
                          <label for="child-title-{task.id}-{idx}">Title <span class="required">*</span></label>
                          <input
                            id="child-title-{task.id}-{idx}"
                            name="child-title-{task.id}-{idx}"
                            type="text"
                            value={row.title}
                            oninput={(e) => updateRow(task.id, idx, 'title', (e.target as HTMLInputElement).value)}
                            required
                            autocomplete="off"
                            placeholder="Child task title"
                          />
                        </div>
                        <div class="form-row-inline">
                          <div class="form-field">
                            <label for="child-layer-{task.id}-{idx}">Layer</label>
                            <select
                              id="child-layer-{task.id}-{idx}"
                              name="child-layer-{task.id}-{idx}"
                              value={row.layer}
                              onchange={(e) => updateRow(task.id, idx, 'layer', (e.target as HTMLSelectElement).value)}
                            >
                              <option value="A">A — Orchestrator</option>
                              <option value="B">B — Specialist</option>
                            </select>
                          </div>
                          <div class="form-field">
                            <label for="child-assignee-{task.id}-{idx}">Assignee</label>
                            <input
                              id="child-assignee-{task.id}-{idx}"
                              name="child-assignee-{task.id}-{idx}"
                              type="text"
                              value={row.assignee ?? ''}
                              oninput={(e) => updateRow(task.id, idx, 'assignee', (e.target as HTMLInputElement).value || null)}
                              autocomplete="off"
                              placeholder="agent name"
                            />
                          </div>
                        </div>
                      </div>
                      <button
                        type="button"
                        class="btn-remove-row"
                        onclick={() => removeRow(task.id, idx)}
                        disabled={form.rows.length <= 1}
                        title="Remove this row"
                      >×</button>
                    </div>
                  {/each}
                  <button type="button" class="btn-add-row" onclick={() => addRow(task.id)}>
                    + Add Child Task
                  </button>
                </div>

                {#if form.err}
                  <p class="form-error">{form.err}</p>
                {/if}

                <div class="form-actions">
                  <button type="button" class="btn-secondary" onclick={() => closeForm(task.id)} disabled={form.submitting}>
                    Cancel
                  </button>
                  <button type="submit" class="btn-primary" disabled={form.submitting || !form.submitter.trim()}>
                    {form.submitting ? 'Submitting...' : 'Submit Proposal'}
                  </button>
                </div>
              </form>
            {:else if proposal}
              <!-- Existing proposal display -->
              <div class="proposal-display">
                <div class="proposal-meta">
                  <span>Proposed by <strong>{proposal.submitter}</strong></span>
                  <span class="proposal-date">Created {new Date(proposal.createdAt).toLocaleDateString()}</span>
                </div>
                <div class="child-list">
                  {#each proposal.proposedTasks as child, idx}
                    <div class="child-preview">
                      <span class="child-num">{idx + 1}</span>
                      <span class="child-badge" class:layer-a={child.layer === 'A'} class:layer-b={child.layer === 'B'}>
                        {layerBadge(child.layer)}
                      </span>
                      <span class="child-title">{child.title}</span>
                      {#if child.assignee}
                        <span class="child-assignee">{child.assignee}</span>
                      {/if}
                    </div>
                  {/each}
                </div>

                {#if proposal.state === 'submitted'}
                  <div class="proposal-actions">
                    <button
                      class="btn-approve"
                      onclick={() => handleApprove(task.id)}
                      disabled={processing !== null}
                    >
                      {processing === task.id ? 'Approving...' : 'Approve'}
                    </button>
                    <button
                      class="btn-reject"
                      onclick={() => handleReject(task.id)}
                      disabled={processing !== null}
                    >
                      {processing === task.id ? 'Rejecting...' : 'Reject'}
                    </button>
                  </div>
                {:else}
                  <div class="proposal-status-banner" style="border-color: {stateColor(proposal.state)}">
                    <span style="color: {stateColor(proposal.state)}">{stateLabel(proposal.state)}</span>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .decomp-page {
    max-width: 900px;
    margin: 0 auto;
  }

  .loading-state,
  .error-state,
  .empty-state {
    text-align: center;
    padding: 3.5rem 2rem;
    color: #888;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
  }

  .loading-state::before {
    content: '';
    display: block;
    width: 32px;
    height: 32px;
    border: 2px solid #2a2a2a;
    border-top-color: #555;
    border-radius: 50%;
    margin: 0 auto 1rem;
    animation: spin 0.7s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-state {
    color: #f87171;
  }

  .page-header {
    display: flex;
    align-items: baseline;
    gap: 1rem;
    margin-bottom: 1.25rem;
  }

  .page-header h1 {
    font-size: 1.4rem;
    font-weight: 600;
    color: #e0e0e0;
  }

  .project-id {
    font-size: 0.8rem;
    color: #666;
    font-family: monospace;
  }

  .filter-tabs {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    margin-bottom: 1.25rem;
  }

  .tab-btn {
    background: #1a1a1a;
    border: 1px solid #333;
    color: #888;
    padding: 0.35rem 0.8rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
  }

  .tab-btn:hover {
    border-color: #555;
    color: #ccc;
  }

  .tab-btn.active {
    background: #252525;
    border-color: #e0e0e0;
    color: #e0e0e0;
  }

  .tab-count {
    font-size: 0.8rem;
    color: #666;
    margin-left: auto;
  }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .task-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    overflow: hidden;
    transition: border-color 0.15s, box-shadow 0.15s;
    animation: card-enter 0.2s ease-out both;
  }

  @keyframes card-enter {
    from { opacity: 0; transform: translateY(4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .task-card:hover {
    border-color: #333;
    box-shadow: 0 2px 12px rgba(0,0,0,0.3);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
  }

  .task-meta {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .layer-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border-radius: 3px;
    font-size: 0.7rem;
    font-weight: 700;
    font-family: monospace;
  }

  .layer-badge.layer-a {
    background: #fbbf24;
    color: #0f0f0f;
  }

  .layer-badge.layer-b {
    background: #34d399;
    color: #0f0f0f;
  }

  .task-title {
    font-size: 0.9rem;
    font-weight: 500;
    color: #e0e0e0;
  }

  .task-id {
    font-size: 0.7rem;
    color: #555;
    font-family: monospace;
  }

  .card-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .proposal-state {
    font-size: 0.8rem;
    font-weight: 500;
  }

  .btn-propose {
    background: #252525;
    border: 1px solid #444;
    color: #e0e0e0;
    padding: 0.3rem 0.8rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
  }

  .btn-propose:hover {
    background: #333;
    border-color: #666;
  }

  /* Proposal form */
  .proposal-form {
    padding: 1rem;
    border-top: 1px solid #2a2a2a;
    background: #151515;
  }

  .form-header h2 {
    font-size: 0.95rem;
    font-weight: 600;
    color: #ccc;
    margin-bottom: 1rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }

  .form-field label {
    font-size: 0.78rem;
    color: #888;
    font-weight: 500;
  }

  .form-field input,
  .form-field select,
  .form-field textarea {
    background: #0f0f0f;
    border: 1px solid #333;
    color: #e0e0e0;
    border-radius: 4px;
    padding: 0.4rem 0.6rem;
    font-size: 0.85rem;
    font-family: inherit;
    outline: none;
    transition: border-color 0.15s;
  }

  .form-field input:focus,
  .form-field select:focus,
  .form-field textarea:focus {
    border-color: #666;
  }

  .form-field textarea {
    resize: vertical;
  }

  .required {
    color: #f8961a;
  }

  .child-tasks-section {
    margin-top: 1rem;
  }

  .child-tasks-section h3 {
    font-size: 0.85rem;
    font-weight: 600;
    color: #888;
    margin-bottom: 0.75rem;
  }

  .child-row {
    display: flex;
    align-items: flex-start;
 gap: 0.6rem;
    margin-bottom: 0.75rem;
    padding: 0.6rem;
    background: #0f0f0f;
    border: 1px solid #222;
    border-radius: 4px;
  }

  .child-order {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    background: #252525;
    border-radius: 50%;
    font-size: 0.7rem;
    color: #666;
    flex-shrink: 0;
    margin-top: 1.45rem;
  }

  .child-fields {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-row-inline {
    display: grid;
    grid-template-columns: 1fr 2fr;
    gap: 0.5rem;
  }

  .btn-remove-row {
    background: none;
    border: 1px solid #333;
    color: #888;
    width: 26px;
    height: 26px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
    line-height: 1;
    flex-shrink: 0;
    margin-top: 1.4rem;
    transition: all 0.15s;
  }

  .btn-remove-row:hover:not(:disabled) {
    border-color: #f87171;
    color: #f87171;
  }

  .btn-remove-row:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .btn-add-row {
    background: none;
    border: 1px dashed #333;
    color: #666;
    padding: 0.35rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
  }

  .btn-add-row:hover {
    border-color: #555;
    color: #aaa;
  }

  .form-error {
    color: #f87171;
    font-size: 0.82rem;
    margin: 0.5rem 0;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }

  .btn-primary {
    background: #3b82f6;
    border: 1px solid #3b82f6;
    color: #fff;
    padding: 0.4rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
    transition: all 0.15s;
  }

  .btn-primary:hover:not(:disabled) {
    background: #2563eb;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: none;
    border: 1px solid #333;
    color: #888;
    padding: 0.4rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
    transition: all 0.15s;
  }

  .btn-secondary:hover:not(:disabled) {
    border-color: #555;
    color: #ccc;
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Existing proposal display */
  .proposal-display {
    padding: 0.75rem 1rem;
    border-top: 1px solid #2a2a2a;
    background: #151515;
  }

  .proposal-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.8rem;
    color: #666;
    margin-bottom: 0.75rem;
  }

  .proposal-date {
    color: #555;
  }

  .child-list {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 1rem;
  }

  .child-preview {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: #bbb;
  }

  .child-num {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    background: #252525;
    border-radius: 50%;
    font-size: 0.65rem;
    color: #555;
    flex-shrink: 0;
  }

  .child-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 17px;
    height: 17px;
    border-radius: 3px;
    font-size: 0.65rem;
    font-weight: 700;
    font-family: monospace;
  }

  .child-badge.layer-a {
    background: #fbbf24;
    color: #0f0f0f;
  }

  .child-badge.layer-b {
    background: #34d399;
    color: #0f0f0f;
  }

  .child-title {
    font-weight: 500;
    color: #e0e0e0;
  }

  .child-assignee {
    color: #666;
    font-size: 0.78rem;
    margin-left: 0.25rem;
  }

  .proposal-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-approve {
    background: #166534;
    border: 1px solid #16a34a;
    color: #dcfce7;
    padding: 0.35rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.82rem;
    transition: all 0.15s;
  }

  .btn-approve:hover:not(:disabled) {
    background: #15803d;
  }

  .btn-approve:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-reject {
    background: #7f1d1d;
    border: 1px solid #dc2626;
    color: #fee2e2;
    padding: 0.35rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.82rem;
    transition: all 0.15s;
  }

  .btn-reject:hover:not(:disabled) {
    background: #b91c1c;
  }

  .btn-reject:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .proposal-status-banner {
    display: inline-flex;
    padding: 0.3rem 0.8rem;
    border: 1px solid;
    border-radius: 4px;
    font-size: 0.82rem;
    font-weight: 500;
  }
</style>
