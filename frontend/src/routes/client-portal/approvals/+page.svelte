<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { getClientApprovalInbox, decideClientApproval } from '$lib/api/client';
  import type { ClientApprovalInbox, ClientApprovalItem, ApprovalDecisionRequest, ApprovalOutcome } from '$lib/api/orchestration';

  let inbox = $state<ClientApprovalInbox | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Filter
  type PriorityFilter = 'all' | 'urgent' | 'high' | 'medium' | 'low';
  let priorityFilter = $state<PriorityFilter>('all');

  // Per-card approval state (replaces single global comment)
  let activeComment = $state<Record<string, string>>({});
  let processingId = $state<string | null>(null);
  let approvalErr = $state<string | null>(null);

  onMount(() => {
    loadInbox();
  });

  async function loadInbox() {
    loading = true;
    error = null;
    try {
      inbox = await getClientApprovalInbox();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function getComment(itemId: string): string {
    return activeComment[itemId] ?? '';
  }

  function setComment(itemId: string, value: string) {
    activeComment = { ...activeComment, [itemId]: value };
  }

  async function handleDecision(item: ClientApprovalItem, outcome: ApprovalOutcome) {
    const comment = getComment(item.id);
    if (outcome !== 'approve' && !comment.trim()) {
      approvalErr = 'A comment is required for this decision.';
      return;
    }
    processingId = item.id;
    approvalErr = null;
    try {
      const req: ApprovalDecisionRequest = {
        approvalId: item.id,
        outcome,
        comments: comment.trim() || undefined,
      };
      await decideClientApproval(item.id, req);
      // Clear this card's comment
      const updated = { ...activeComment };
      delete updated[item.id];
      activeComment = updated;
      await loadInbox();
    } catch (e) {
      approvalErr = e instanceof Error ? e.message : String(e);
    } finally {
      processingId = null;
    }
  }

  let filteredItems = $derived(
    inbox
      ? priorityFilter === 'all'
        ? [...inbox.items].sort(byPriority)
        : inbox.items.filter(i => i.priority === priorityFilter)
      : []
  );

  function byPriority(a: ClientApprovalItem, b: ClientApprovalItem): number {
    const priorityOrder = { urgent: 0, high: 1, medium: 2, low: 3 };
    const aOverdue = isOverdue(a.requestedAt);
    const bOverdue = isOverdue(b.requestedAt);
    // Overdue items first, then by priority
    if (aOverdue !== bOverdue) return aOverdue ? -1 : 1;
    return (priorityOrder[a.priority] ?? 4) - (priorityOrder[b.priority] ?? 4);
  }

  function formatAge(iso: string): string {
    if (!iso) return '';
    const ms = Date.now() - new Date(iso).getTime();
    const h = Math.floor(ms / 3600000);
    if (h < 1) return '< 1h ago';
    if (h < 24) return `${h}h ago`;
    const d = Math.floor(h / 24);
    return `${d}d ago`;
  }

  function isOverdue(iso: string): boolean {
    return Date.now() - new Date(iso).getTime() > 24 * 3600000;
  }

  function priorityColor(p: string): string {
    const m: Record<string, string> = {
      urgent: '#f87171', high: '#fb923c', medium: '#60a5fa', low: '#4ade80',
    };
    return m[p] ?? '#888';
  }

  function decisionTypeLabel(t: string): string {
    return t.replace(/_/g, ' ');
  }
</script>

<svelte:head>
  <title>Approvals — Client Portal</title>
</svelte:head>

{#if loading}
  <div class="loading-state">
    <div class="spinner"></div>
    <p>Loading approvals...</p>
  </div>
{:else if error}
  <div class="error-state">
    <p class="error-title">Failed to load approvals</p>
    <p class="error-msg">{error}</p>
    <button class="btn-retry" onclick={loadInbox}>Retry</button>
  </div>
{:else if !inbox || inbox.items.length === 0}
  <div class="empty-state">
    <div class="empty-icon">✓</div>
    <p class="empty-title">No pending approvals</p>
    <p class="empty-hint">You're all caught up. New approval requests will appear here when decisions are needed.</p>
    <button class="btn-action" onclick={() => goto('/client-portal')}>Back to Portfolio</button>
  </div>
{:else}
  <div class="page-header">
    <div class="header-top">
      <div>
        <h1>Approval Inbox</h1>
        <p class="header-subtitle">{inbox.totalCount} pending · {inbox.urgentCount} urgent</p>
      </div>
    </div>

    {#if inbox.oldestPending}
      {@const overdue = isOverdue(inbox.oldestPending)}
      <div class="oldest-pending" class:overdue={overdue}>
        Oldest pending: {formatAge(inbox.oldestPending)}
        {#if overdue}<span class="overdue-tag">Overdue</span>{/if}
      </div>
    {/if}

    <!-- Priority filter -->
    <div class="filter-row">
      <span class="filter-label">Priority:</span>
      {#each ['all', 'urgent', 'high', 'medium', 'low'] as p}
        <button
          class="filter-btn"
          class:active={priorityFilter === p}
          onclick={() => { priorityFilter = p as PriorityFilter; }}
        >
          {p === 'all' ? 'All' : p.charAt(0).toUpperCase() + p.slice(1)}
          {#if p !== 'all' && inbox.byPriority[p as keyof typeof inbox.byPriority] > 0}
            <span class="filter-count">{inbox.byPriority[p as keyof typeof inbox.byPriority]}</span>
          {/if}
        </button>
      {/each}
    </div>
  </div><!-- end .page-header -->

  {#if approvalErr}
    <div class="err-banner">{approvalErr}</div>
  {/if}

  <div class="approval-list">
    {#each filteredItems as item}
      {@const overdue = isOverdue(item.requestedAt)}
      <div class="approval-card" class:overdue={isOverdue(item.requestedAt)}>
        <div class="approval-header">
          <span class="approval-type">{decisionTypeLabel(item.decisionType)}</span>
          <span class="priority-badge" style="--c: {priorityColor(item.priority)}">
            {item.priority}
          </span>
        </div>

        <h3 class="approval-title">{item.decisionTitle}</h3>
        <p class="approval-summary">{item.summary}</p>

        <div class="approval-meta">
          <span class="meta-chip">{item.projectName}</span>
          <span class="meta-age" class:overdue={overdue}>
            Requested {formatAge(item.requestedAt)}
          </span>
          {#if item.dueDate}
            <span class="meta-due" class:overdue={isOverdue(item.dueDate)}>
              Due {new Date(item.dueDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
            </span>
          {/if}
        </div>

        {#if item.riskImpact}
          <p class="risk-impact">⚠ {item.riskImpact}</p>
        {/if}

        {#if item.affectedMilestones.length > 0}
          <div class="affected-items">
            {#each item.affectedMilestones as m}
              <span class="item-tag">◆ {m}</span>
            {/each}
          </div>
        {/if}

        <div class="approval-actions">
          <button
            class="btn-approve"
            disabled={processingId !== null}
            onclick={() => handleDecision(item, 'approve')}
          >Approve</button>
          <button
            class="btn-reject"
            disabled={processingId !== null}
            onclick={() => handleDecision(item, 'reject')}
          >Reject</button>
          <button
            class="btn-changes"
            disabled={processingId !== null}
            onclick={() => handleDecision(item, 'request_changes')}
          >Request Changes</button>
          <button
            class="btn-info"
            disabled={processingId !== null}
            onclick={() => handleDecision(item, 'need_more_information')}
          >Need More Info</button>
        </div>

        <div class="comment-row">
          <input
            type="text"
            class="comment-input"
            placeholder="Add a comment (required for reject, request changes, or need more info)..."
            value={getComment(item.id)}
            oninput={(e) => setComment(item.id, (e.target as HTMLInputElement).value)}
            disabled={processingId !== null}
          />
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .loading-state,
  .error-state,
  .empty-state {
    text-align: center;
    padding: 3.5rem 2rem;
    color: var(--color-text-muted);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    margin-top: 1.5rem;
    transition: all 0.3s;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 2px solid #2a2a2a;
    border-top-color: #555;
    border-radius: 50%;
    margin: 0 auto 1rem;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-state { color: #f87171; }
  .error-title { font-weight: 600; margin-bottom: 0.5rem; }
  .error-msg { font-size: 0.875rem; color: #a0a0a0; }

  .btn-retry {
    margin-top: 1rem;
    background: #2a2a2a;
    color: #e0e0e0;
    border: 1px solid #3a3a3a;
    padding: 0.5rem 1.5rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .empty-icon {
    font-size: 2.5rem;
    color: #4ade80;
    margin-bottom: 1rem;
  }

  .empty-title { font-weight: 600; color: #e0e0e0; margin-bottom: 0.5rem; }
  .empty-hint { font-size: 0.875rem; margin-bottom: 1.5rem; }

  .btn-action {
    background: #2a2a2a;
    color: #e0e0e0;
    border: 1px solid #3a3a3a;
    padding: 0.5rem 1.25rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .page-header { margin-bottom: 1.5rem; }

  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 0.75rem;
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 600;
    margin: 0;
  }

  .header-subtitle {
    color: #888;
    font-size: 0.875rem;
    margin: 0.25rem 0 0;
  }

  .oldest-pending {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: #888;
    margin-bottom: 1rem;
    padding: 0.5rem 0.875rem;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    width: fit-content;
  }

  .oldest-pending.overdue { color: #fb923c; border-color: #fb923c; }

  .overdue-tag {
    background: #fb923c;
    color: #000;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.15em 0.5em;
    border-radius: 4px;
  }

  .filter-row {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    flex-wrap: wrap;
  }

  .filter-label {
    font-size: 0.8rem;
    color: #888;
    margin-right: 0.25rem;
  }

  .filter-btn {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    color: #888;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    display: flex;
    align-items: center;
    gap: 0.35rem;
    transition: all 0.15s;
  }

  .filter-btn:hover { border-color: #3a3a3a; color: #e0e0e0; }

  .filter-btn.active {
    background: #2a2a2a;
    color: #fff;
    border-color: #555;
  }

  .filter-count {
    background: #3a3a3a;
    font-size: 0.65rem;
    padding: 0.1em 0.4em;
    border-radius: 8px;
  }

  .err-banner {
    background: color-mix(in srgb, #f87171 10%, transparent);
    border: 1px solid #f87171;
    color: #f87171;
    padding: 0.75rem 1rem;
    border-radius: 6px;
    font-size: 0.875rem;
    margin-bottom: 1rem;
  }

  .approval-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .approval-card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    transition: all 0.2s;
  }

  .approval-card.overdue {
    border-color: var(--color-warn);
    background: color-mix(in srgb, var(--color-warn) 5%, var(--color-surface));
  }

  .approval-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .approval-type {
    font-size: 0.7rem;
    color: #60a5fa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .priority-badge {
    font-size: 0.7rem;
    font-weight: 500;
    padding: 0.15em 0.6em;
    border-radius: 4px;
    background: color-mix(in srgb, var(--c) 20%, transparent);
    color: var(--c);
    border: 1px solid var(--c);
    text-transform: capitalize;
  }

  .approval-title {
    font-size: 1rem;
    font-weight: 600;
    margin: 0;
    color: #e0e0e0;
  }

  .approval-summary {
    font-size: 0.875rem;
    color: #a0a0a0;
    margin: 0;
    line-height: 1.5;
  }

  .approval-meta {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .meta-chip {
    font-size: 0.75rem;
    background: #2a2a2a;
    color: #888;
    padding: 0.15em 0.5em;
    border-radius: 4px;
  }

  .meta-age, .meta-due {
    font-size: 0.75rem;
    color: #666;
  }

  .meta-age.overdue, .meta-due.overdue { color: #fb923c; }

  .risk-impact {
    font-size: 0.8rem;
    color: #fb923c;
    margin: 0;
  }

  .affected-items {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .item-tag {
    font-size: 0.7rem;
    background: #2a2a2a;
    color: #888;
    padding: 0.15em 0.5em;
    border-radius: 4px;
  }

  .approval-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-approve, .btn-reject, .btn-changes, .btn-info {
    padding: 0.4rem 1rem;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    border: none;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .btn-approve { background: #4ade80; color: #000; }
  .btn-approve:hover { opacity: 0.85; }
  .btn-reject { background: #f87171; color: #000; }
  .btn-reject:hover { opacity: 0.85; }
  .btn-changes { background: #fbbf24; color: #000; }
  .btn-changes:hover { opacity: 0.85; }
  .btn-info { background: #60a5fa; color: #000; }
  .btn-info:hover { opacity: 0.85; }
  .btn-approve:disabled, .btn-reject:disabled, .btn-changes:disabled, .btn-info:disabled { opacity: 0.4; cursor: not-allowed; }

  .comment-row { margin-top: 0.25rem; }

  .comment-input {
    width: 100%;
    background: #0f0f0f;
    border: 1px solid #333;
    border-radius: 6px;
    color: #e0e0e0;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    box-sizing: border-box;
  }

  .comment-input:focus {
    outline: none;
    border-color: #555;
  }
</style>