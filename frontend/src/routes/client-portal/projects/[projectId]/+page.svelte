<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { getClientProjectDetail, decideClientApproval } from '$lib/api/client';
  import type {
    ClientProjectDetail,
    ClientTaskColumn,
    ClientRiskItem,
    ClientMilestoneItem,
    ClientApprovalItem,
    ApprovalOutcome,
  } from '$lib/api/orchestration';
  import { mapOwnerLabel } from '$lib/client-portal/owner-labels';
  import { createSSEConnection } from '$lib/api/client';

  let projectId = $derived(page.params.projectId ?? '');
  let detail = $state<ClientProjectDetail | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  let sseConnected = $state(false);
  let sseReconnecting = $state(false);
  let sseCleanup: { close: () => void } | null = null;

  let showCancelled = $state(false);

  let activeApprovalId = $state<string | null>(null);
  let approvalComment = $state('');
  let approvalAction = $state<ApprovalOutcome | null>(null);
  let approvalSubmitting = $state(false);
  let approvalError = $state<string | null>(null);

  onMount(() => {
    loadProject();
    return () => { sseCleanup?.close(); };
  });

  async function loadProject() {
    if (!projectId) return;
    loading = true;
    error = null;
    try {
      detail = await getClientProjectDetail(projectId);
      error = null;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
    startSSE();
  }

  function startSSE() {
    sseCleanup?.close();
    const close = createSSEConnection({
      projectId,
      onEvent: (envelope) => {
        const relevant = ['task.updated','task.completed','task.blocked','approval.updated','risk.updated','milestone.updated','comment.added','comment.updated'];
        if (relevant.includes(envelope.topic)) { loadProject(); }
      },
      onConnect: () => { sseConnected = true; sseReconnecting = false; },
      onDisconnect: () => { sseConnected = false; },
      onError: () => { sseConnected = false; sseReconnecting = true; },
    });
    sseCleanup = close;
  }

  function backToPortfolio() { goto('/client-portal'); }

  function isDueOverdue(dueDate: string): boolean {
    return new Date(dueDate) < new Date();
  }

  function formatDate(iso: string | null): string {
    if (!iso) return '—';
    try {
      return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch { return iso; }
  }

  function ownerLabel(phase: string): string { return mapOwnerLabel(phase); }

  /**
   * Phase 1 limitation: backend portfolio detail contract does not surface
   * phase/owner data at the detail level. Placeholder until Phase 2.
   */
  const OWNER_LABEL_PHASE1_LIMITATION = '—';

  function healthColor(health: 'on_track' | 'at_risk' | 'blocked'): string {
    switch (health) { case 'on_track': return '#4ade80'; case 'at_risk': return '#fb923c'; case 'blocked': return '#f87171'; }
  }

  function healthLabel(health: 'on_track' | 'at_risk' | 'blocked'): string {
    switch (health) { case 'on_track': return 'On Track'; case 'at_risk': return 'At Risk'; case 'blocked': return 'Blocked'; }
  }

  function taskStatusColor(status: string): string {
    const m: Record<string, string> = { todo: '#60a5fa', in_progress: '#a78bfa', blocked: '#f87171', done: '#4ade80', cancelled: '#555' };
    return m[status] ?? '#888';
  }

  function milestoneStatusColor(status: string): string {
    const m: Record<string, string> = { upcoming: '#60a5fa', at_risk: '#fb923c', completed: '#4ade80', missed: '#f87171' };
    return m[status] ?? '#888';
  }

  let visibleTaskColumns = $derived.by(() => {
    if (!detail) return [];
    return detail.taskColumns.map((col: ClientTaskColumn) => ({
      ...col,
      taskCards: col.status === 'cancelled' && !showCancelled ? [] : col.taskCards,
    }));
  });

  let visibleTaskColumnsList = $derived(visibleTaskColumns);

  async function submitApprovalDecision(approvalId: string, outcome: ApprovalOutcome) {
    if (['reject','request_changes','need_more_information'].includes(outcome) && !approvalComment.trim()) {
      approvalError = 'Comment is required for this action';
      return;
    }
    approvalSubmitting = true;
    approvalError = null;
    try {
      await decideClientApproval(approvalId, { approvalId, outcome, comments: approvalComment.trim() || undefined });
      approvalComment = '';
      activeApprovalId = null;
      approvalAction = null;
      if (projectId) detail = await getClientProjectDetail(projectId);
    } catch (e) {
      approvalError = e instanceof Error ? e.message : String(e);
    } finally {
      approvalSubmitting = false;
    }
  }

  function startApprovalAction(approvalId: string, outcome: ApprovalOutcome) {
    activeApprovalId = approvalId;
    approvalAction = outcome;
    approvalComment = '';
    approvalError = null;
  }

  function cancelApprovalAction() {
    activeApprovalId = null;
    approvalAction = null;
    approvalComment = '';
    approvalError = null;
  }
</script>

<svelte:head>
  <title>{detail?.project?.name ?? 'Project'} — Client Portal</title>
</svelte:head>

<div class="project-detail-page">
  <button class="back-btn" onclick={backToPortfolio} type="button">← Back to Portfolio</button>

  {#if loading}
    <div class="loading-state"><div class="spinner"></div><p>Loading project...</p></div>
  {:else if error}
    <div class="error-state">
      <p class="error-title">Failed to load project</p>
      <p class="error-msg">{error}</p>
      <button class="btn-retry" onclick={loadProject}>Retry</button>
    </div>
  {:else if !detail}
    <div class="empty-state"><p>Project not found.</p></div>
  {:else}
    <!-- SSE indicator -->
    <div class="sse-indicator" class:connected={sseConnected} class:reconnecting={sseReconnecting} class:offline={!sseConnected && !sseReconnecting}>
      {#if sseConnected}
        <span class="sse-dot live"></span>
        <span class="sse-label">Live updates</span>
      {:else if sseReconnecting}
        <span class="sse-dot reconnecting"></span>
        <span class="sse-label">Reconnecting...</span>
      {:else}
        <span class="sse-dot offline"></span>
        <span class="sse-label">Live updates paused</span>
        <button class="btn-refresh" onclick={loadProject}>Refresh</button>
      {/if}
    </div>

    <!-- Project header -->
    <div class="project-header">
      <div class="header-left">
        <h1>{detail.project.name}</h1>
        {#if detail.project.description}<p class="project-desc">{detail.project.description}</p>{/if}
      </div>
      <div class="header-right">
        <span class="health-badge" style="--c: {healthColor(detail.project.health)}">
          {healthLabel(detail.project.health)}
        </span>
        <span class="owner-badge">{OWNER_LABEL_PHASE1_LIMITATION}</span>
      </div>
    </div>

    <!-- Health strip -->
    <div class="health-strip">
      <div class="hs-item"><span class="hs-value" style="color:#4ade80">{detail.project.taskCounts.done}</span><span class="hs-label">Done</span></div>
      <div class="hs-item"><span class="hs-value" style="color:#a78bfa">{detail.project.taskCounts.inProgress}</span><span class="hs-label">In Progress</span></div>
      <div class="hs-item"><span class="hs-value" style="color:#60a5fa">{detail.project.taskCounts.todo}</span><span class="hs-label">To Do</span></div>
      <div class="hs-item"><span class="hs-value" style="color:#f87171">{detail.project.taskCounts.blocked}</span><span class="hs-label">Blocked</span></div>
      <div class="hs-item hs-completion">
        <span class="hs-value">{Math.round(detail.project.progressPercent)}%</span>
        <span class="hs-label">Complete</span>
        <div class="hs-bar"><div class="hs-fill" style="width:{detail.project.progressPercent}%"></div></div>
      </div>
    </div>

    <!-- Task Board -->
    <section class="board-section">
      <div class="section-header">
        <h2>Task Board</h2>
        <label class="cancelled-toggle">
          <input type="checkbox" bind:checked={showCancelled} />
          Show cancelled ({detail.taskColumns.find(c => c.status === 'cancelled')?.taskCards.length ?? 0})
        </label>
      </div>

      {#if visibleTaskColumnsList.every(col => col.taskCards.length === 0)}
        <div class="empty-section">
          <div class="empty-icon">◇</div>
          <h3>No active work</h3>
          <p>There are no active tasks in this project right now.</p>
        </div>
      {:else}
        <div class="board-columns">
          {#each visibleTaskColumnsList as column (column.id)}
            <div class="board-column" data-status={column.status}>
              <div class="column-header">
                <span class="column-title">{column.title}</span>
                <span class="column-count" style="color:{taskStatusColor(column.status)}">{column.taskCards.length}</span>
              </div>
              <div class="column-cards">
                {#each column.taskCards as card (card.id)}
                  <div class="task-card" data-status={card.status}>
                    <div class="task-title">{card.title}</div>
                    {#if card.description}<div class="task-desc">{card.description}</div>{/if}
                    <div class="task-meta">
                      <span class="task-owner">{ownerLabel(card.assignee ?? '')}</span>
                      {#if card.dueDate}<span class="task-due">Due {formatDate(card.dueDate)}</span>{/if}
                    </div>
                    {#if card.status === 'cancelled'}
                      <div class="cancelled-reason">Cancelled</div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <!-- Approvals -->
    {#if detail.approvalsPending.length > 0}
      <section class="approvals-section">
        <h2>Pending Approvals <span class="count-badge">{detail.approvalsPending.length}</span></h2>
        <div class="approval-list">
          {#each detail.approvalsPending as approval (approval.id)}
            <div class="approval-card" data-priority={approval.priority} class:overdue={approval.dueDate && isDueOverdue(approval.dueDate)}>
              <div class="approval-header">
                <span class="approval-type">{approval.decisionType.replace(/_/g, ' ')}</span>
                <span class="approval-priority priority-{approval.priority}">{approval.priority}</span>
                {#if approval.dueDate}
                  <span class="approval-due" class:overdue={isDueOverdue(approval.dueDate)} >
                    {isDueOverdue(approval.dueDate) ? 'Overdue — ' : 'Due '}{formatDate(approval.dueDate)}
                  </span>
                {/if}
              </div>
              <div class="approval-title">{approval.decisionTitle}</div>
              <div class="approval-summary">{approval.summary}</div>

              {#if activeApprovalId === approval.id}
                <div class="approval-action-form">
                  <p class="action-label">
                    {#if approvalAction === 'approve'}Confirm: Approve this item?{/if}
                    {#if approvalAction === 'reject'}Reject this item (comment required){/if}
                    {#if approvalAction === 'request_changes'}Request changes (comment required){/if}
                    {#if approvalAction === 'need_more_information'}Need more information (comment required){/if}
                  </p>
                  {#if approvalAction !== 'approve'}
                    <textarea bind:value={approvalComment} placeholder="Enter your comment (required for this action)" class="comment-input" rows="3"></textarea>
                    {#if approvalError}<p class="action-error">{approvalError}</p>{/if}
                  {/if}
                  <div class="action-btns">
                    <button class="btn-confirm btn-{approvalAction}" onclick={() => approvalAction && submitApprovalDecision(approval.id, approvalAction)} disabled={approvalSubmitting}>
                      {approvalSubmitting ? 'Submitting...' : 'Confirm'}
                    </button>
                    <button class="btn-cancel" onclick={cancelApprovalAction}>Cancel</button>
                  </div>
                </div>
              {:else}
                <div class="approval-actions">
                  <button class="btn-approve" onclick={() => startApprovalAction(approval.id, 'approve')}>Approve</button>
                  <button class="btn-reject" onclick={() => startApprovalAction(approval.id, 'reject')}>Reject</button>
                  <button class="btn-changes" onclick={() => startApprovalAction(approval.id, 'request_changes')}>Request Changes</button>
                  <button class="btn-info" onclick={() => startApprovalAction(approval.id, 'need_more_information')}>Need More Info</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </section>
    {/if}

    <!-- Risks -->
    {#if detail.activeRisks.length > 0}
      <section class="risks-section">
        <h2>Active Risks</h2>
        <div class="risk-list">
          {#each detail.activeRisks as risk (risk.id)}
            <div class="risk-card" data-severity={risk.severity}>
              <div class="risk-header">
                <span class="risk-title">{risk.title}</span>
                <span class="risk-severity severity-{risk.severity}">{risk.severity}</span>
              </div>
              <div class="risk-desc">{risk.description}</div>
              {#if risk.mitigations.length > 0}
                <div class="risk-mitigations">
                  <span class="mitigations-label">Mitigations:</span>
                  {#each risk.mitigations as m}<span class="mitigation-item">{m}</span>{/each}
                </div>
              {/if}
              <div class="risk-meta">
                <span>Raised {formatDate(risk.raisedAt)}</span>
                {#if risk.dueDate}<span>Target: {formatDate(risk.dueDate)}</span>{/if}
              </div>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    <!-- Milestones -->
    {#if detail.upcomingMilestones.length > 0}
      <section class="milestones-section">
        <h2>Milestones</h2>
        <div class="milestone-list">
          {#each detail.upcomingMilestones as milestone (milestone.id)}
            <div class="milestone-card" data-status={milestone.status}>
              <div class="milestone-header">
                <span class="milestone-title">{milestone.title}</span>
                <span class="milestone-status" style="color:{milestoneStatusColor(milestone.status)}">{milestone.status.replace('_', ' ')}</span>
              </div>
              {#if milestone.description}<div class="milestone-desc">{milestone.description}</div>{/if}
              <div class="milestone-progress">
                <div class="mp-bar"><div class="mp-fill" style="width:{milestone.completionPercent}%"></div></div>
                <span class="mp-pct">{milestone.completionPercent}%</span>
              </div>
              {#if milestone.dueDate}<div class="milestone-due">Due {formatDate(milestone.dueDate)}</div>{/if}
            </div>
          {/each}
        </div>
      </section>
    {/if}

    <!-- Recent Comments -->
    {#if detail.recentComments.length > 0}
      <section class="comments-section">
        <h2>Recent Activity</h2>
        <div class="comment-list">
          {#each detail.recentComments as comment (comment.id)}
            <div class="comment-card">
              <div class="comment-author">{comment.authorName}</div>
              <div class="comment-content">{comment.content}</div>
              <div class="comment-time">{formatDate(comment.createdAt)}</div>
            </div>
          {/each}
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .project-detail-page { max-width: 1100px; margin: 0 auto; animation: page-enter 0.25s ease-out; }
  @keyframes page-enter { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }

  .back-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 0.875rem; padding: 0; margin-bottom: 1rem; display: inline-flex; align-items: center; gap: 0.25rem; }
  .back-btn:hover { color: #e0e0e0; }

  .sse-indicator { display: flex; align-items: center; gap: 0.5rem; font-size: 0.75rem; color: var(--color-text-muted); margin-bottom: 1rem; padding: 0.4rem 0.75rem; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 6px; width: fit-content; transition: all 0.3s; }
  .sse-indicator.connected { border-color: color-mix(in srgb, var(--color-success) 30%, transparent); color: var(--color-success); }
  .sse-indicator.reconnecting { border-color: color-mix(in srgb, var(--color-warn) 30%, transparent); color: var(--color-warn); }
  .sse-indicator.offline { border-color: color-mix(in srgb, var(--color-warn) 20%, transparent); color: var(--color-text-muted); }
  .sse-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .sse-dot.live { background: var(--color-success); box-shadow: 0 0 6px var(--color-success); }
  .sse-dot.reconnecting { background: var(--color-warn); animation: pulse 1s infinite; }
  .sse-dot.offline { background: var(--color-muted); }
  @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
  .sse-label { font-weight: 500; }
  .btn-refresh { background: var(--color-surface); border: 1px solid var(--color-border); color: var(--color-text); padding: 0.2rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.75rem; margin-left: 0.25rem; transition: all 0.15s; }
  .btn-refresh:hover { border-color: var(--color-border-hover); }

  .loading-state, .error-state, .empty-state { text-align: center; padding: 3rem 2rem; color: var(--color-text-muted); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; margin: 2rem 0; transition: all 0.3s; }
  .spinner { width: 32px; height: 32px; border: 2px solid var(--color-border); border-top-color: var(--color-muted); border-radius: 50%; margin: 0 auto 1rem; animation: spin 0.7s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .error-state { color: var(--color-error); }
  .error-state .spinner { border-color: var(--color-border); border-top-color: var(--color-error); }
  .error-title { font-weight: 600; margin-bottom: 0.5rem; color: var(--color-text); }
  .error-msg { font-size: 0.875rem; color: var(--color-text-muted); }
  .btn-retry { margin-top: 1rem; background: var(--color-surface); border: 1px solid var(--color-border); color: var(--color-text); padding: 0.5rem 1.5rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; transition: all 0.15s; }
  .btn-retry:hover { border-color: var(--color-border-hover); }

  .project-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.25rem; gap: 1rem; }
  .project-header h1 { font-size: 1.5rem; font-weight: 600; margin: 0; color: #e0e0e0; }
  .project-desc { color: #888; font-size: 0.875rem; margin: 0.35rem 0 0; }
  .header-right { display: flex; align-items: center; gap: 0.5rem; flex-shrink: 0; }
  .health-badge { font-size: 0.7rem; font-weight: 500; padding: 0.15em 0.6em; border-radius: 4px; background: color-mix(in srgb, var(--c) 20%, transparent); color: var(--c); border: 1px solid var(--c); }
  .owner-badge { font-size: 0.7rem; font-weight: 500; padding: 0.15em 0.5em; border-radius: 4px; background: #2a2a2a; color: #a0a0a0; }

  .health-strip { display: flex; gap: 1.5rem; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; padding: 1rem 1.25rem; margin-bottom: 1.5rem; flex-wrap: wrap; transition: all 0.3s; }
  .hs-item { display: flex; flex-direction: column; gap: 0.2rem; }
  .hs-value { font-size: 1.5rem; font-weight: 700; line-height: 1; }
  .hs-label { font-size: 0.7rem; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
  .hs-completion { min-width: 100px; }
  .hs-bar { height: 4px; background: var(--color-border); border-radius: 2px; overflow: hidden; margin-top: 0.5rem; width: 100%; }
  .hs-fill { height: 100%; background: var(--color-success); border-radius: 2px; transition: width 0.3s; }

  .section-header { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
  h2 { font-size: 1rem; font-weight: 600; color: #e0e0e0; margin: 0; }
  .count-badge { font-size: 0.75rem; background: #2a2a2a; color: #888; padding: 0.1em 0.5em; border-radius: 4px; }
  .cancelled-toggle { font-size: 0.8rem; color: #888; cursor: pointer; display: flex; align-items: center; gap: 0.35rem; }
  .cancelled-toggle input { cursor: pointer; }

  .board-section { margin-bottom: 2rem; }
  .board-columns { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 1rem; align-items: start; }
  .board-column { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; overflow: hidden; }
  .column-header { display: flex; justify-content: space-between; align-items: center; padding: 0.6rem 0.75rem; border-bottom: 1px solid #2a2a2a; background: #151515; }
  .column-title { font-size: 0.8rem; font-weight: 600; color: #a0a0a0; }
  .column-count { font-size: 0.75rem; font-weight: 700; }
  .column-cards { padding: 0.5rem; display: flex; flex-direction: column; gap: 0.5rem; max-height: 500px; overflow-y: auto; }

  .task-card { background: #0f0f0f; border: 1px solid #2a2a2a; border-radius: 6px; padding: 0.6rem 0.75rem; display: flex; flex-direction: column; gap: 0.35rem; }
  .task-title { font-size: 0.875rem; font-weight: 500; color: #e0e0e0; line-height: 1.3; }
  .task-desc { font-size: 0.75rem; color: #888; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .task-meta { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
  .task-owner { font-size: 0.65rem; background: #2a2a2a; color: #a0a0a0; padding: 0.1em 0.4em; border-radius: 3px; }
  .task-due { font-size: 0.65rem; color: #666; }
  .cancelled-reason, .blocked-reason { font-size: 0.7rem; color: #f87171; font-style: italic; padding: 0.25rem 0.4rem; background: color-mix(in srgb, #f87171 10%, transparent); border-radius: 4px; margin-top: 0.25rem; }

  .approvals-section { margin-bottom: 2rem; }
  .approval-list { display: flex; flex-direction: column; gap: 0.75rem; }
  .approval-card { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; transition: all 0.2s; }
  .approval-card.overdue { border-color: var(--color-warn); background: color-mix(in srgb, var(--color-warn) 5%, var(--color-surface)); }
  .approval-header { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
  .approval-type { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.05em; color: #60a5fa; background: color-mix(in srgb, #60a5fa 10%, transparent); padding: 0.15em 0.4em; border-radius: 3px; }
  .approval-priority { font-size: 0.65rem; text-transform: uppercase; padding: 0.15em 0.4em; border-radius: 3px; }
  .priority-urgent { background: color-mix(in srgb, #dc2626 20%, transparent); color: #dc2626; }
  .priority-high { background: color-mix(in srgb, #f87171 20%, transparent); color: #f87171; }
  .priority-medium { background: color-mix(in srgb, #fb923c 20%, transparent); color: #fb923c; }
  .priority-low { background: color-mix(in srgb, #4ade80 15%, transparent); color: #4ade80; }
  .approval-due { font-size: 0.75rem; color: var(--color-text-muted); }
  .approval-due.overdue { color: var(--color-warn); font-weight: 600; }
  .approval-title { font-size: 0.95rem; font-weight: 600; color: #e0e0e0; }
  .approval-summary { font-size: 0.8rem; color: #888; line-height: 1.4; }
  .approval-actions, .approval-action-form { display: flex; gap: 0.5rem; flex-wrap: wrap; margin-top: 0.5rem; }
  .approval-action-form { flex-direction: column; background: #151515; padding: 0.75rem; border-radius: 6px; border: 1px solid #2a2a2a; }
  .action-label { font-size: 0.875rem; color: #e0e0e0; margin: 0 0 0.5rem; }
  .comment-input { width: 100%; background: #0f0f0f; border: 1px solid #333; border-radius: 6px; color: #e0e0e0; padding: 0.5rem 0.75rem; font-size: 0.875rem; font-family: inherit; resize: vertical; min-height: 80px; }
  .comment-input:focus { outline: none; border-color: #555; }
  .action-error { font-size: 0.8rem; color: #f87171; margin: 0.25rem 0; }
  .action-btns { display: flex; gap: 0.5rem; }

  .btn-approve, .btn-confirm.btn-approve { background: color-mix(in srgb, #4ade80 15%, transparent); color: #4ade80; border: 1px solid #4ade80; padding: 0.35rem 0.75rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem; font-weight: 500; }
  .btn-reject, .btn-confirm.btn-reject { background: color-mix(in srgb, #f87171 15%, transparent); color: #f87171; border: 1px solid #f87171; padding: 0.35rem 0.75rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem; font-weight: 500; }
  .btn-changes, .btn-confirm.btn-changes { background: color-mix(in srgb, #fb923c 15%, transparent); color: #fb923c; border: 1px solid #fb923c; padding: 0.35rem 0.75rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem; font-weight: 500; }
  .btn-info, .btn-confirm.btn-info { background: color-mix(in srgb, #60a5fa 15%, transparent); color: #60a5fa; border: 1px solid #60a5fa; padding: 0.35rem 0.75rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem; font-weight: 500; }
  .btn-cancel { background: #2a2a2a; color: #888; border: 1px solid #3a3a3a; padding: 0.35rem 0.75rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem; }

  .risks-section { margin-bottom: 2rem; }
  .risk-list { display: flex; flex-direction: column; gap: 0.75rem; }
  .risk-card { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 0.875rem 1rem; }
  .risk-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
  .risk-title { font-size: 0.95rem; font-weight: 600; color: #e0e0e0; }
  .risk-severity { font-size: 0.65rem; text-transform: uppercase; padding: 0.15em 0.4em; border-radius: 3px; }
  .severity-critical { background: color-mix(in srgb, #dc2626 20%, transparent); color: #dc2626; }
  .severity-high { background: color-mix(in srgb, #f87171 20%, transparent); color: #f87171; }
  .severity-medium { background: color-mix(in srgb, #fb923c 20%, transparent); color: #fb923c; }
  .severity-low { background: color-mix(in srgb, #4ade80 15%, transparent); color: #4ade80; }
  .risk-desc { font-size: 0.8rem; color: #888; line-height: 1.4; margin-bottom: 0.5rem; }
  .risk-mitigations { display: flex; flex-wrap: wrap; gap: 0.4rem; align-items: center; margin-bottom: 0.5rem; }
  .mitigations-label { font-size: 0.7rem; color: #666; }
  .mitigation-item { font-size: 0.7rem; background: #2a2a2a; color: #a0a0a0; padding: 0.1em 0.4em; border-radius: 3px; }
  .risk-meta { font-size: 0.7rem; color: #666; display: flex; gap: 1rem; }

  .milestones-section { margin-bottom: 2rem; }
  .milestone-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 0.75rem; }
  .milestone-card { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 0.875rem 1rem; }
  .milestone-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
  .milestone-title { font-size: 0.9rem; font-weight: 600; color: #e0e0e0; }
  .milestone-status { font-size: 0.65rem; text-transform: uppercase; font-weight: 500; }
  .milestone-desc { font-size: 0.8rem; color: #888; margin-bottom: 0.5rem; }
  .milestone-progress { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.4rem; }
  .mp-bar { flex: 1; height: 4px; background: #2a2a2a; border-radius: 2px; overflow: hidden; }
  .mp-fill { height: 100%; background: #60a5fa; border-radius: 2px; transition: width 0.3s; }
  .mp-pct { font-size: 0.75rem; color: #60a5fa; font-weight: 500; }
  .milestone-due { font-size: 0.7rem; color: #666; }

  .comments-section { margin-bottom: 2rem; }
  .comment-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .comment-card { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 0.75rem 1rem; }
  .comment-author { font-size: 0.8rem; font-weight: 600; color: #a0a0a0; margin-bottom: 0.25rem; }
  .comment-content { font-size: 0.875rem; color: #e0e0e0; line-height: 1.4; margin-bottom: 0.35rem; }
  .comment-time { font-size: 0.7rem; color: #555; }

  .empty-section { text-align: center; padding: 2rem; color: #888; background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; margin: 1rem 0; }
  .empty-icon { font-size: 1.5rem; margin-bottom: 0.5rem; }
  .empty-section h3 { font-size: 1rem; font-weight: 600; color: #e0e0e0; margin: 0 0 0.5rem; }
  .empty-section p { font-size: 0.875rem; }
</style>