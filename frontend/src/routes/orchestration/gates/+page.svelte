<script lang="ts">
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { listProjectPhaseGates, updateProjectPhaseGate, getProject, listProjectTasks, createTaskGate, updateTaskGate } from '$lib/api/client';
  import type { ProjectPhaseGate, ProjectPhaseGateUpdateRequest, OrchestrationTask, TaskGate, TaskGateCreateRequest, TaskGateUpdateRequest, GateState, Project } from '$lib/api/orchestration';

  let projectId = $derived(page.url.searchParams.get('project') ?? '');

  let project = $state<Project | null>(null);
  let phaseGates = $state<ProjectPhaseGate[]>([]);
  let tasks = $state<OrchestrationTask[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let activeTab = $state<'phase' | 'task'>('phase');

  // Gate action modals
  let actionGate = $state<(ProjectPhaseGate & { action: 'approve' | 'reject' | null }) | null>(null);
  let actionNote = $state('');
  let actionLoading = $state(false);

  // Task gate action
  let actionTaskGate = $state<{ task: OrchestrationTask; gate: TaskGate; action: 'approve' | 'reject' | null } | null>(null);
  let taskGateNote = $state('');

  // Create task gate
  let createGateTask = $state<OrchestrationTask | null>(null);
  let createGatePhase = $state('implementation_review');
  let createGateCriteria = $state('');
  let createGateLoading = $state(false);

  const GATE_TYPES = ['scope_review', 'architecture_review', 'implementation_review', 'code_review', 'qa_review', 'release_review'];

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
      const [proj, gatesRes, taskRes] = await Promise.all([
        getProject(projectId),
        listProjectPhaseGates(projectId),
        listProjectTasks(projectId),
      ]);
      project = proj;
      phaseGates = gatesRes.gates;
      tasks = taskRes.tasks;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function gateStateColor(state: GateState): string {
    const m: Record<string, string> = { open: '#fbbf24', passed: '#4ade80', blocked: '#f87171' };
    return m[state] ?? '#888';
  }

  function openAction(gate: ProjectPhaseGate, action: 'approve' | 'reject') {
    actionGate = { ...gate, action };
    actionNote = '';
  }

  async function submitPhaseGateAction(e: SubmitEvent) {
    e.preventDefault();
    if (!actionGate || !actionGate.action) return;
    actionLoading = true;
    try {
      const req: ProjectPhaseGateUpdateRequest = {};
      if (actionGate.action === 'approve') {
        req.state = 'passed';
      } else {
        req.state = 'blocked';
        if (actionNote.trim()) req.criteria = [actionNote.trim()];
      }
      await updateProjectPhaseGate(projectId, actionGate.id, req);
      actionGate = null;
      await loadData();
    } catch (e) {
      alert(`Action failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      actionLoading = false;
    }
  }

  async function submitTaskGateAction(e: SubmitEvent) {
    e.preventDefault();
    if (!actionTaskGate || !actionTaskGate.action) return;
    actionLoading = true;
    try {
      const req: TaskGateUpdateRequest = {};
      if (actionTaskGate.action === 'approve') {
        req.state = 'passed';
      } else {
        req.state = 'blocked';
        if (taskGateNote.trim()) req.overrideNote = taskGateNote.trim();
      }
      await updateTaskGate(projectId, actionTaskGate.task.id, actionTaskGate.gate.id, req);
      actionTaskGate = null;
      taskGateNote = '';
      await loadData();
    } catch (e) {
      alert(`Action failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      actionLoading = false;
    }
  }

  async function handleCreateGate(e: SubmitEvent) {
    e.preventDefault();
    if (!createGateTask) return;
    createGateLoading = true;
    try {
      const req: TaskGateCreateRequest = {
        phase: createGatePhase,
        criteria: createGateCriteria ? createGateCriteria.split(',').map((c) => c.trim()).filter(Boolean) : [],
      };
      await createTaskGate(projectId, createGateTask.id, req);
      createGateTask = null;
      createGatePhase = 'implementation_review';
      createGateCriteria = '';
      await loadData();
    } catch (e) {
      alert(`Create gate failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      createGateLoading = false;
    }
  }

  function tasksWithGates(): OrchestrationTask[] {
    return tasks.filter((t) => t.gates && t.gates.length > 0);
  }
</script>

<svelte:head>
  <title>Gates — {project?.name ?? 'Orchestration'}</title>
</svelte:head>

<div class="gates-page">
  {#if loading}
    <div class="loading-state"><p>Loading gates...</p></div>
  {:else if error}
    <div class="error-state"><p>{error}</p><button onclick={loadData}>Retry</button></div>
  {:else if !project}
    <div class="error-state"><p>Project not found.</p></div>
  {:else}
    <div class="gates-header">
      <h1>Gates — {project.name}</h1>
    </div>

    <div class="tab-bar">
      <button class="tab" class:active={activeTab === 'phase'} onclick={() => { activeTab = 'phase'; }}>
        Phase Gates
      </button>
      <button class="tab" class:active={activeTab === 'task'} onclick={() => { activeTab = 'task'; }}>
        Task Gates
      </button>
    </div>

    {#if activeTab === 'phase'}
      <div class="gate-section">
        <h2>Project Phase Gates</h2>
        <p class="section-desc">Phase gates govern major project transitions. Human approval required to pass a phase gate.</p>

        {#if phaseGates.length === 0}
          <div class="empty-state"><p>No phase gates defined for this project.</p></div>
        {:else}
          <div class="gate-list">
            {#each phaseGates as gate}
              <div class="gate-card">
                <div class="gate-header">
                  <div class="gate-left">
                    <span class="gate-index">G{gate.phaseIndex}</span>
                    <span class="gate-phase">{gate.phase}</span>
                    <span class="gate-state-badge" style="--c: {gateStateColor(gate.state)}">{gate.state}</span>
                  </div>
                  <div class="gate-actions">
                    {#if gate.state === 'open'}
                      <button class="btn-sm btn-pass" onclick={() => openAction(gate, 'approve')}>Approve</button>
                      <button class="btn-sm btn-block" onclick={() => openAction(gate, 'reject')}>Reject</button>
                    {/if}
                  </div>
                </div>

                {#if gate.criteria?.length}
                  <ul class="gate-criteria">
                    {#each gate.criteria as criterion}
                      <li>{criterion}</li>
                    {/each}
                  </ul>
                {/if}

                {#if gate.passCondition}
                  <p class="pass-condition"><strong>Pass condition:</strong> {gate.passCondition}</p>
                {/if}

                {#if gate.passedAt}
                  <p class="gate-meta">Passed: {new Date(gate.passedAt).toLocaleString()} by {gate.passedBy ?? 'unknown'}</p>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>

    {:else}
      <div class="gate-section">
        <h2>Task-Level Gates</h2>
        <p class="section-desc">Quality gates attached to individual tasks. Block task advancement until gate is passed.</p>

        {#if tasksWithGates().length === 0}
          <div class="empty-state">
            <p>No task gates exist yet. Gates are created when tasks are promoted or manually attached.</p>
          </div>
        {:else}
          <div class="gate-list">
            {#each tasksWithGates() as task}
              {#each task.gates as gate}
                <div class="gate-card">
                  <div class="gate-header">
                    <div class="gate-left">
                      <span class="task-ref">{task.id}</span>
                      <span class="gate-phase">{gate.phase}</span>
                      <span class="gate-state-badge" style="--c: {gateStateColor(gate.state)}">{gate.state}</span>
                    </div>
                    <div class="gate-actions">
                      {#if gate.state === 'open'}
                        <button class="btn-sm btn-pass" onclick={() => { actionTaskGate = { task, gate, action: 'approve' }; }}>Approve</button>
                        <button class="btn-sm btn-block" onclick={() => { actionTaskGate = { task, gate, action: 'reject' }; }}>Reject</button>
                      {/if}
                    </div>
                  </div>
                  <p class="task-title-small">{task.title}</p>
                  {#if gate.criteria?.length}
                    <ul class="gate-criteria">
                      {#each gate.criteria as criterion}
                        <li>{criterion}</li>
                      {/each}
                    </ul>
                  {/if}
                  {#if gate.passedAt}
                    <p class="gate-meta">Passed: {new Date(gate.passedAt).toLocaleString()} by {gate.passedBy ?? 'unknown'}</p>
                  {/if}
                  {#if gate.overrideNote}
                    <p class="override-note">Override note: {gate.overrideNote}</p>
                  {/if}
                </div>
              {/each}
            {/each}
          </div>
        {/if}

        <!-- Create gate on a task -->
        <div class="create-gate-section">
          <h3>Attach Gate to Task</h3>
          {#if createGateTask}
            <div class="cg-task-info">
              <span>Attaching gate to: <strong>{createGateTask.title}</strong></span>
              <button class="btn-icon" onclick={() => { createGateTask = null; }}>×</button>
            </div>
            <form class="cg-form" onsubmit={handleCreateGate}>
              <div class="form-field">
                <label for="cg-phase">Gate Type</label>
                <select id="cg-phase" name="cg-phase" bind:value={createGatePhase}>
                  {#each GATE_TYPES as t}
                    <option value={t}>{t.replace(/_/g, ' ')}</option>
                  {/each}
                </select>
              </div>
              <div class="form-field">
                <label for="cg-criteria">Criteria (comma-separated)</label>
                <input id="cg-criteria" name="cg-criteria" type="text" bind:value={createGateCriteria} placeholder="tests pass, review approved" autocomplete="off" />
              </div>
              <div class="form-actions">
                <button type="submit" class="btn-primary" disabled={createGateLoading}>
                  {createGateLoading ? 'Creating...' : 'Attach Gate'}
                </button>
              </div>
            </form>
          {:else}
            <div class="task-select-list">
              {#each tasks.filter(t => t.status !== 'done' && t.status !== 'cancelled') as task}
                <button class="task-select-btn" onclick={() => { createGateTask = task; }}>
                  <span class="ts-title">{task.title}</span>
                  <span class="ts-id">{task.id}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Phase Gate Action Modal -->
{#if actionGate}
  <div class="modal-overlay" onclick={() => { actionGate = null; }} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h2>{actionGate.action === 'approve' ? 'Approve' : 'Reject'} Phase Gate — {actionGate.phase}</h2>
      {#if actionGate.action === 'reject'}
        <p class="modal-subtitle">Rejection requires a reason. The gate will transition to blocked.</p>
      {:else}
        <p class="modal-subtitle">Approve this phase gate to advance the project.</p>
      {/if}
      <form onsubmit={submitPhaseGateAction}>
        {#if actionGate.action === 'reject'}
          <div class="form-field">
            <label for="pg-reason">Rejection Reason <span class="required">*</span></label>
            <textarea id="pg-reason" name="pg-reason" bind:value={actionNote} required rows="3" placeholder="Why is this gate rejected?"></textarea>
          </div>
        {/if}
        <div class="modal-actions">
          <button type="button" class="btn-secondary" onclick={() => { actionGate = null; }}>Cancel</button>
          <button
            type="submit"
            class={actionGate.action === 'approve' ? 'btn-pass' : 'btn-danger'}
            disabled={actionLoading || (actionGate.action === 'reject' && !actionNote.trim())}
          >
            {actionLoading ? 'Submitting...' : actionGate.action === 'approve' ? 'Approve Gate' : 'Reject Gate'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Task Gate Action Modal -->
{#if actionTaskGate}
  <div class="modal-overlay" onclick={() => { actionTaskGate = null; }} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h2>{actionTaskGate.action === 'approve' ? 'Approve' : 'Reject'} Task Gate — {actionTaskGate.gate.phase}</h2>
      <p class="task-ref-inline">{actionTaskGate.task.title}</p>
      {#if actionTaskGate.action === 'reject'}
        <p class="modal-subtitle">Provide a note explaining the rejection.</p>
        <form onsubmit={submitTaskGateAction}>
          <div class="form-field">
            <label for="tg-reason">Rejection Note <span class="required">*</span></label>
            <textarea id="tg-reason" name="tg-reason" bind:value={taskGateNote} required rows="3" placeholder="Why is this gate rejected?"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" onclick={() => { actionTaskGate = null; }}>Cancel</button>
            <button type="submit" class="btn-danger" disabled={actionLoading || !taskGateNote.trim()}>
              {actionLoading ? 'Submitting...' : 'Reject Gate'}
            </button>
          </div>
        </form>
      {:else}
        <form onsubmit={submitTaskGateAction}>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" onclick={() => { actionTaskGate = null; }}>Cancel</button>
            <button type="submit" class="btn-pass" disabled={actionLoading}>
              {actionLoading ? 'Submitting...' : 'Approve Gate'}
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

<style>
  .gates-page { max-width: 800px; margin: 0 auto; animation: page-enter 0.25s ease-out; }
  @keyframes page-enter {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .loading-state, .error-state, .empty-state { text-align: center; padding: 3.5rem 2rem; color: #888; background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; }
  .loading-state::before {
    content: ''; display: block; width: 32px; height: 32px; border: 2px solid #2a2a2a;
    border-top-color: #555; border-radius: 50%; margin: 0 auto 1rem; animation: spin 0.7s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .error-state { color: #f87171; border-color: #7f1d1d; }

  .gates-header { margin-bottom: 1.5rem; }
  h1 { font-size: 1.4rem; font-weight: 600; }
  h2 { font-size: 1rem; font-weight: 600; margin-bottom: 0.5rem; }
  h3 { font-size: 0.9rem; font-weight: 600; margin-bottom: 0.5rem; }

  .section-desc { font-size: 0.8rem; color: #888; margin-bottom: 1rem; }

  .tab-bar { display: flex; gap: 0.25rem; margin-bottom: 1.5rem; border-bottom: 1px solid #2a2a2a; }
  .tab { background: none; border: none; color: #888; padding: 0.5rem 1rem; cursor: pointer; font-size: 0.875rem; border-bottom: 2px solid transparent; margin-bottom: -1px; }
  .tab:hover { color: #ccc; }
  .tab.active { color: #e0e0e0; border-bottom-color: #4ade80; }

  .gate-list { display: flex; flex-direction: column; gap: 0.75rem; }
  .gate-card { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 1rem; transition: border-color 0.15s, box-shadow 0.15s; }
  .gate-card:hover { border-color: #3a3a3a; box-shadow: 0 2px 12px rgba(0,0,0,0.3); }
  .gate-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
  .gate-left { display: flex; align-items: center; gap: 0.5rem; }
  .gate-index { font-size: 0.7rem; font-weight: 700; color: #888; background: #2a2a2a; padding: 0.15em 0.5em; border-radius: 3px; }
  .gate-phase { font-size: 0.85rem; font-weight: 500; text-transform: capitalize; }
  .gate-state-badge { font-size: 0.7rem; font-weight: 600; padding: 0.1em 0.5em; border-radius: 3px; background: color-mix(in srgb, var(--c) 20%, transparent); color: var(--c); text-transform: capitalize; }
  .gate-actions { display: flex; gap: 0.25rem; }
  .gate-criteria { margin: 0.5rem 0 0.5rem 1.25rem; font-size: 0.8rem; color: #aaa; }
  .gate-criteria li { margin-bottom: 0.2rem; }
  .pass-condition { font-size: 0.8rem; color: #888; margin: 0.5rem 0 0; }
  .gate-meta { font-size: 0.75rem; color: #666; margin: 0.4rem 0 0; }
  .override-note { font-size: 0.75rem; color: #fbbf24; margin: 0.4rem 0 0; font-style: italic; }
  .task-title-small { font-size: 0.8rem; color: #888; margin: 0.25rem 0 0.5rem; }

  .btn-sm { background: #2a2a2a; border: 1px solid #333; color: #888; padding: 0.25rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.75rem; transition: all 0.15s; }
  .btn-sm:hover { background: #3a3a3a; color: #e0e0e0; }
  .btn-pass { color: #4ade80; border-color: #4ade8040; }
  .btn-pass:hover { background: #4ade8020; }
  .btn-block { color: #f87171; border-color: #f8717140; }
  .btn-block:hover { background: #f8717120; }

  .btn-primary { background: #2a2a2a; color: #e0e0e0; border: 1px solid #3a3a3a; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-primary:hover:not(:disabled) { background: #3a3a3a; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-secondary { background: transparent; color: #888; border: 1px solid #333; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-danger { background: transparent; color: #f87171; border: 1px solid #f8717140; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-danger:hover:not(:disabled) { background: #f8717120; }
  .btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-pass { background: transparent; color: #4ade80; border: 1px solid #4ade8040; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-pass:hover:not(:disabled) { background: #4ade8020; }
  .btn-pass:disabled { opacity: 0.5; cursor: not-allowed; }

  .form-field { display: flex; flex-direction: column; gap: 0.3rem; margin-bottom: 0.75rem; }
  label { font-size: 0.8rem; color: #888; font-weight: 500; }
  .required { color: #f87171; }
  input, textarea, select { background: #0f0f0f; border: 1px solid #2a2a2a; border-radius: 4px; color: #e0e0e0; padding: 0.4rem 0.6rem; font-size: 0.85rem; font-family: inherit; resize: vertical; }
  input:focus, textarea:focus, select:focus { outline: none; border-color: #555; }
  select { cursor: pointer; }

  .create-gate-section { margin-top: 2rem; background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 1rem; }
  .task-select-list { display: flex; flex-direction: column; gap: 0.25rem; max-height: 240px; overflow-y: auto; }
  .task-select-btn { background: #0f0f0f; border: 1px solid #2a2a2a; color: #ccc; padding: 0.4rem 0.75rem; border-radius: 4px; cursor: pointer; text-align: left; font-size: 0.8rem; display: flex; justify-content: space-between; transition: border-color 0.15s; }
  .task-select-btn:hover { border-color: #555; }
  .ts-title { font-weight: 500; }
  .ts-id { color: #666; font-size: 0.7rem; font-family: monospace; }
  .cg-task-info { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; font-size: 0.85rem; color: #ccc; background: #0f0f0f; padding: 0.5rem 0.75rem; border-radius: 4px; }
  .cg-form { margin-top: 0.75rem; }
  .form-actions { display: flex; justify-content: flex-end; margin-top: 0.5rem; }

  .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 1rem; }
  .modal { background: #1a1a1a; border: 1px solid #333; border-radius: 8px; padding: 1.5rem; width: 100%; max-width: 440px; }
  .modal h2 { font-size: 1rem; font-weight: 600; margin-bottom: 0.25rem; }
  .modal-subtitle { font-size: 0.8rem; color: #888; margin-bottom: 1rem; }
  .task-ref-inline { font-size: 0.8rem; color: #666; margin-bottom: 1rem; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
  .btn-icon { background: none; border: none; cursor: pointer; padding: 0.2rem 0.4rem; color: #666; font-size: 1rem; }
  .btn-icon:hover { color: #e0e0e0; }
</style>