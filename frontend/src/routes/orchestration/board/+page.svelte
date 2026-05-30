<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import {
    listProjectTasks,
    createProjectTask,
    promoteProjectTask,
    completeProjectTask,
    blockProjectTask,
    getProject,
  } from '$lib/api/client';
  import type {
    Project,
    OrchestrationTask,
    TaskStatus,
    OrchestrationTaskCreateRequest,
    TaskCompleteRequest,
  } from '$lib/api/orchestration';

  // Status filter from URL
  let statusFilter = $derived(page.url.searchParams.get('status') ?? 'all');
  let projectId = $derived(page.url.searchParams.get('project') ?? '');

  let project = $state<Project | null>(null);
  let tasks = $state<OrchestrationTask[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Create task form
  let showCreate = $state(false);
  let createTitle = $state('');
  let createBody = $state('');
  let createLayer = $state<'A' | 'B'>('B');
  let createAssignee = $state('');
  let createPriority = $state(0);
  let createParentId = $state('');
  let createLoading = $state(false);
  let createErr = $state<string | null>(null);

  // Complete task form
  let completingTask = $state<OrchestrationTask | null>(null);
  let completeSummary = $state('');
  let completeArtifacts = $state('');
  let completeValidation = $state('');
  let completeRisks = $state('');
  let completeNextGate = $state('');
  let completeLoading = $state(false);

  // Block task form
  let blockingTask = $state<OrchestrationTask | null>(null);
  let blockReason = $state('');
  let blockLoading = $state(false);

  const STATUS_OPTIONS: Array<{ value: string; label: string; color: string }> = [
    { value: 'all', label: 'All', color: '#888' },
    { value: 'todo', label: 'To Do', color: '#60a5fa' },
    { value: 'in_progress', label: 'In Progress', color: '#a78bfa' },
    { value: 'blocked', label: 'Blocked', color: '#f87171' },
    { value: 'done', label: 'Done', color: '#4ade80' },
    { value: 'cancelled', label: 'Cancelled', color: '#6b7280' },
  ];

  const LAYER_OPTIONS = [
    { value: 'A', label: 'Layer A — Orchestrator' },
    { value: 'B', label: 'Layer B — Specialist' },
  ];

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
      const [proj, taskRes] = await Promise.all([
        getProject(projectId),
        listProjectTasks(projectId, statusFilter !== 'all' ? { status: statusFilter } : undefined),
      ]);
      project = proj;
      tasks = taskRes.tasks;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function refresh() {
    await loadData();
  }

  // Filter tasks by selected status
  let filteredTasks = $derived(
    statusFilter === 'all' ? tasks : tasks.filter((t) => t.status === statusFilter)
  );

  // Build task tree from parent_id hierarchy
  // Top-level = tasks with no parent or parent not in this project
  let taskTree = $derived(() => {
    const map = new Map<string, OrchestrationTask[]>();
    for (const task of tasks) {
      const parentId = task.parents[0] ?? '';
      if (!map.has(parentId)) map.set(parentId, []);
      map.get(parentId)!.push(task);
    }
    // Return top-level (no parent in our task list)
    return map.get('') ?? [];
  });

  function getChildren(taskId: string): OrchestrationTask[] {
    return tasks.filter((t) => t.parents.includes(taskId));
  }

  function statusColor(status: TaskStatus): string {
    const m: Record<string, string> = {
      todo: '#60a5fa', in_progress: '#a78bfa', blocked: '#f87171',
      done: '#4ade80', cancelled: '#6b7280',
    };
    return m[status] ?? '#888';
  }

  function layerColor(layer: string): string {
    return layer === 'A' ? '#fbbf24' : '#34d399';
  }

  function setStatusFilter(status: string) {
    const url = new URL(page.url);
    if (status === 'all') url.searchParams.delete('status');
    else url.searchParams.set('status', status);
    goto(url.toString(), { replaceState: true });
  }

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    if (!createTitle.trim() || !projectId) return;
    createLoading = true;
    createErr = null;
    try {
      const req: OrchestrationTaskCreateRequest = {
        title: createTitle.trim(),
        body: createBody.trim() || undefined,
        layer: createLayer,
        assignee: createAssignee.trim() || createLayer === 'B' ? 'developer' : 'architect',
        priority: createPriority,
        parents: createParentId ? [createParentId] : [],
      };
      await createProjectTask(projectId, req);
      showCreate = false;
      createTitle = ''; createBody = ''; createLayer = 'B';
      createAssignee = ''; createPriority = 0; createParentId = '';
      await refresh();
    } catch (e) {
      createErr = e instanceof Error ? e.message : String(e);
    } finally {
      createLoading = false;
    }
  }

  async function handlePromote(task: OrchestrationTask) {
    try {
      await promoteProjectTask(projectId, task.id);
      await refresh();
    } catch (e) {
      alert(`Promote failed: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  async function handleComplete(task: OrchestrationTask) {
    completingTask = task;
    completeSummary = '';
    completeArtifacts = '';
    completeValidation = '';
    completeRisks = '';
    completeNextGate = '';
  }

  async function submitComplete(e: SubmitEvent) {
    e.preventDefault();
    if (!completingTask) return;
    completeLoading = true;
    try {
      const req: TaskCompleteRequest = {
        summary: completeSummary,
        artifacts: completeArtifacts ? completeArtifacts.split(',').map((a) => a.trim()) : [],
        validationPerformed: completeValidation,
        risksOrResidualIssues: completeRisks || undefined,
        recommendedNextGate: completeNextGate || undefined,
      };
      await completeProjectTask(projectId, completingTask.id, req);
      completingTask = null;
      await refresh();
    } catch (e) {
      alert(`Complete failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      completeLoading = false;
    }
  }

  async function handleBlock(task: OrchestrationTask) {
    blockingTask = task;
    blockReason = '';
  }

  async function submitBlock(e: SubmitEvent) {
    e.preventDefault();
    if (!blockingTask) return;
    blockLoading = true;
    try {
      await blockProjectTask(projectId, blockingTask.id, { reason: blockReason });
      blockingTask = null;
      blockReason = '';
      await refresh();
    } catch (e) {
      alert(`Block failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      blockLoading = false;
    }
  }

  function closeDialogs() {
    completingTask = null;
    blockingTask = null;
    createErr = null;
  }

  function renderTaskTree(tasks: OrchestrationTask[], depth = 0): any[] {
    return tasks;
  }
</script>

<svelte:head>
  <title>{project?.name ?? 'Board'} — Orchestration</title>
</svelte:head>

<div class="board-page">
  {#if loading}
    <div class="loading-state"><p>Loading board...</p></div>
  {:else if error}
    <div class="error-state">
      <p>{error}</p>
      <button onclick={refresh}>Retry</button>
    </div>
  {:else if !project}
    <div class="error-state"><p>Project not found.</p></div>
  {:else}
    <div class="board-header">
      <div class="board-title-row">
        <h1>{project.name}</h1>
        <button class="btn-primary" onclick={() => { showCreate = !showCreate; createErr = null; }}>
          {showCreate ? 'Cancel' : '+ New Task'}
        </button>
      </div>

      <div class="status-filters">
        {#each STATUS_OPTIONS as opt}
          <button
            class="filter-btn"
            class:active={statusFilter === opt.value}
            style="--c: {opt.color}"
            onclick={() => setStatusFilter(opt.value)}
          >
            {opt.label}
          </button>
        {/each}
        <span class="task-count">{filteredTasks.length} tasks</span>
      </div>
    </div>

    {#if showCreate}
      <form class="task-create-form" onsubmit={handleCreate}>
        <h2>Create Task in {project.name}</h2>
        <div class="form-grid">
          <div class="form-field">
            <label for="t-title">Title <span class="required">*</span></label>
            <input id="t-title" name="t-title" type="text" bind:value={createTitle} required placeholder="Task title" autocomplete="off" />
          </div>
          <div class="form-field">
            <label for="t-layer">Layer</label>
            <select id="t-layer" name="t-layer" bind:value={createLayer}>
              {#each LAYER_OPTIONS as o}
                <option value={o.value}>{o.label}</option>
              {/each}
            </select>
          </div>
          <div class="form-field">
            <label for="t-assignee">Assignee</label>
            <input id="t-assignee" name="t-assignee" type="text" bind:value={createAssignee} placeholder="agent name" autocomplete="off" />
          </div>
          <div class="form-field">
            <label for="t-priority">Priority</label>
            <input id="t-priority" name="t-priority" type="number" bind:value={createPriority} min="0" max="100" />
          </div>
          <div class="form-field full-width">
            <label for="t-body">Body / Description</label>
            <textarea id="t-body" name="t-body" bind:value={createBody} rows="3" placeholder="Full task specification"></textarea>
          </div>
          <div class="form-field">
            <label for="t-parent">Parent Task ID</label>
            <input id="t-parent" name="t-parent" type="text" bind:value={createParentId} placeholder="optional parent task id" autocomplete="off" />
          </div>
        </div>
        {#if createErr}
          <p class="form-error">{createErr}</p>
        {/if}
        <div class="form-actions">
          <button type="submit" class="btn-primary" disabled={createLoading || !createTitle.trim()}>
            {createLoading ? 'Creating...' : 'Create Task'}
          </button>
        </div>
      </form>
    {/if}

    {#if filteredTasks.length === 0}
      <div class="empty-state"><p>No tasks matching this filter.</p></div>
    {:else}
      <div class="task-list">
        {#each filteredTasks as task}
          {@const children = getChildren(task.id)}
          <div class="task-row" style="--depth: {task.parents.filter(p => tasks.some(t => t.id === p)).length}">
            <div class="task-main">
              <div class="task-left">
                <span class="status-dot" style="background: {statusColor(task.status)}" title={task.status}></span>
                <div class="task-info">
                  <span class="task-title">{task.title}</span>
                  {#if task.assignee}
                    <span class="task-assignee">{task.assignee}</span>
                  {/if}
                </div>
              </div>
              <div class="task-right">
                <span class="layer-badge" style="--c: {layerColor(task.layer)}">{task.layer}</span>
                {#if task.gates.length > 0}
                  {#each task.gates as gate}
                    <span class="gate-badge" class:open={gate.state === 'open'} class:passed={gate.state === 'passed'} class:blocked={gate.state === 'blocked'}>
                      {gate.phase}
                    </span>
                  {/each}
                {/if}
                {#if children.length > 0}
                  <span class="child-count">{children.length} children</span>
                {/if}
                <div class="task-actions">
                  {#if task.status === 'todo' || task.status === 'in_progress'}
                    <button class="btn-sm" onclick={() => handlePromote(task)} title="Promote to next gate">→</button>
                    <button class="btn-sm" onclick={() => handleComplete(task)} title="Mark done">✓</button>
                    <button class="btn-sm btn-danger" onclick={() => handleBlock(task)} title="Block">▌</button>
                  {/if}
                </div>
              </div>
            </div>
            {#if task.blockedReason}
              <div class="blocked-reason">
                <span class="blocked-icon">▌</span> {task.blockedReason}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<!-- Complete Task Modal -->
{#if completingTask}
  <div class="modal-overlay" onclick={closeDialogs} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h2>Complete Task: {completingTask.title}</h2>
      <p class="modal-subtitle">Layer B structured handoff — all fields required per BRD-02 FR-02-015</p>
      <form onsubmit={submitComplete}>
        <div class="form-field">
          <label for="comp-summary">Handoff Summary <span class="required">*</span></label>
          <textarea id="comp-summary" name="comp-summary" bind:value={completeSummary} required rows="3" placeholder="What was accomplished"></textarea>
        </div>
        <div class="form-field">
          <label for="comp-artifacts">Artifacts (comma-separated paths)</label>
          <input id="comp-artifacts" name="comp-artifacts" type="text" bind:value={completeArtifacts} placeholder="src/api/auth.ts, tests/auth.spec.ts" autocomplete="off" />
        </div>
        <div class="form-field">
          <label for="comp-validation">Validation Performed <span class="required">*</span></label>
          <input id="comp-validation" name="comp-validation" type="text" bind:value={completeValidation} required placeholder="e.g. pnpm test passes, manual QA checked" autocomplete="off" />
        </div>
        <div class="form-field">
          <label for="comp-risks">Risks / Residual Issues</label>
          <textarea id="comp-risks" name="comp-risks" bind:value={completeRisks} rows="2" placeholder="Known issues, followed-up items"></textarea>
        </div>
        <div class="form-field">
          <label for="comp-next">Recommended Next Gate / Reviewer</label>
          <input id="comp-next" name="comp-next" type="text" bind:value={completeNextGate} placeholder="e.g. code_review, architect" autocomplete="off" />
        </div>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" onclick={closeDialogs}>Cancel</button>
          <button type="submit" class="btn-primary" disabled={completeLoading || !completeSummary.trim() || !completeValidation.trim()}>
            {completeLoading ? 'Submitting...' : 'Submit Handoff'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Block Task Modal -->
{#if blockingTask}
  <div class="modal-overlay" onclick={closeDialogs} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h2>Block Task: {blockingTask.title}</h2>
      <form onsubmit={submitBlock}>
        <div class="form-field">
          <label for="block-reason">Reason for blocking <span class="required">*</span></label>
          <textarea id="block-reason" name="block-reason" bind:value={blockReason} required rows="3" placeholder="What is needed to unblock this task?"></textarea>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn-secondary" onclick={closeDialogs}>Cancel</button>
          <button type="submit" class="btn-danger" disabled={blockLoading || !blockReason.trim()}>
            {blockLoading ? 'Blocking...' : 'Block Task'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .board-page {
    max-width: 960px;
    margin: 0 auto;
    animation: page-enter 0.25s ease-out;
  }

  @keyframes page-enter {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .loading-state, .error-state, .empty-state {
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

  .board-header {
    margin-bottom: 1.5rem;
  }

  .board-title-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  h1 {
    font-size: 1.4rem;
    font-weight: 600;
  }

  h2 {
    font-size: 1rem;
    font-weight: 600;
    margin-bottom: 1rem;
  }

  .status-filters {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    flex-wrap: wrap;
  }

  .filter-btn {
    background: #1a1a1a;
    border: 1px solid #333;
    color: #888;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
  }

  .filter-btn:hover {
    border-color: #555;
    color: #ccc;
  }

  .filter-btn.active {
    background: color-mix(in srgb, var(--c) 20%, transparent);
    border-color: var(--c);
    color: var(--c);
  }

  .task-count {
    font-size: 0.8rem;
    color: #666;
    margin-left: auto;
  }

  .task-create-form {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .form-field.full-width {
    grid-column: 1 / -1;
  }

  label {
    font-size: 0.8rem;
    color: #888;
    font-weight: 500;
  }

  .required {
    color: #f87171;
  }

  input, textarea, select {
    background: #0f0f0f;
    border: 1px solid #2a2a2a;
    border-radius: 4px;
    color: #e0e0e0;
    padding: 0.4rem 0.6rem;
    font-size: 0.85rem;
    font-family: inherit;
    resize: vertical;
  }

  input:focus, textarea:focus, select:focus {
    outline: none;
    border-color: #555;
  }

  select {
    cursor: pointer;
  }

  .form-error {
    color: #f87171;
    font-size: 0.85rem;
    margin: 0.5rem 0;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 0.75rem;
  }

  .btn-primary {
    background: #2a2a2a;
    color: #e0e0e0;
    border: 1px solid #3a3a3a;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: background 0.15s;
  }

  .btn-primary:hover:not(:disabled) { background: #3a3a3a; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-secondary {
    background: transparent;
    color: #888;
    border: 1px solid #333;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .btn-secondary:hover { color: #ccc; }

  .btn-danger {
    background: transparent;
    color: #f87171;
    border: 1px solid #f8717140;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .btn-danger:hover:not(:disabled) { background: #f8717120; }
  .btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .task-row {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    padding: 0.75rem 1rem;
    transition: border-color 0.15s, box-shadow 0.15s;
    margin-left: calc(var(--depth, 0) * 1.5rem);
    animation: row-enter 0.2s ease-out both;
  }

  @keyframes row-enter {
    from { opacity: 0; transform: translateX(-4px); }
    to { opacity: 1; transform: translateX(0); }
  }

  .task-row:hover {
    border-color: #3a3a3a;
    box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  }

  .task-main {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }

  .task-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .task-info {
    min-width: 0;
  }

  .task-title {
    font-size: 0.9rem;
    font-weight: 500;
    color: #e0e0e0;
    display: block;
  }

  .task-assignee {
    font-size: 0.75rem;
    color: #666;
  }

  .task-right {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  .layer-badge {
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.1em 0.5em;
    border-radius: 3px;
    background: color-mix(in srgb, var(--c) 20%, transparent);
    color: var(--c);
  }

  .gate-badge {
    font-size: 0.65rem;
    padding: 0.15em 0.5em;
    border-radius: 3px;
    background: #2a2a2a;
    color: #888;
  }

  .gate-badge.open { color: #fbbf24; }
  .gate-badge.passed { color: #4ade80; }
  .gate-badge.blocked { color: #f87171; }

  .child-count {
    font-size: 0.7rem;
    color: #666;
  }

  .task-actions {
    display: flex;
    gap: 0.25rem;
    margin-left: 0.5rem;
  }

  .btn-sm {
    background: #2a2a2a;
    border: 1px solid #333;
    color: #888;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    transition: all 0.15s;
  }

  .btn-sm:hover { background: #3a3a3a; color: #e0e0e0; }
  .btn-danger { color: #f87171; border-color: #f8717140; }
  .btn-danger:hover { background: #f8717120; }

  .blocked-reason {
    margin-top: 0.5rem;
    font-size: 0.75rem;
    color: #f87171;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.4rem 0.6rem;
    background: #f8717110;
    border-radius: 4px;
  }

  .blocked-icon { font-size: 0.6rem; }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1rem;
  }

  .modal {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.5rem;
    width: 100%;
    max-width: 480px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .modal h2 {
    font-size: 1rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
  }

  .modal-subtitle {
    font-size: 0.75rem;
    color: #888;
    margin-bottom: 1rem;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }

  .modal .form-field {
    margin-bottom: 0.75rem;
  }
</style>