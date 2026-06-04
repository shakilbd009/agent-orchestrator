<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { getClientPortfolio } from '$lib/api/client';
  import type { ClientPortfolio } from '$lib/api/orchestration';

  let portfolio = $state<ClientPortfolio | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(() => {
    loadPortfolio();
  });

  async function loadPortfolio(silent = false) {
    try {
      if (!silent) loading = true;
      error = null;
      portfolio = await getClientPortfolio();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      if (!silent) loading = false;
    }
  }

  function openProject(projectId: string) {
    goto(`/client-portal/projects/${projectId}`);
  }

  // Health counts from projectsSummary (backend-computed, AC-03-003)
  let onTrackCount = $derived(portfolio?.projectsSummary.onTrack ?? 0);
  let atRiskCount = $derived(portfolio?.projectsSummary.atRisk ?? 0);
  let blockedCount = $derived(portfolio?.projectsSummary.blocked ?? 0);

  // Decision counts from decisionSummary (BRD FR-03-004 / AC-03-004)
  let totalPendingDecisions = $derived(portfolio?.decisionSummary.totalPending ?? 0);
  let overdueDecisions = $derived(portfolio?.decisionSummary.overdue ?? 0);
  let waitingOnClient = $derived(portfolio?.decisionSummary.waitingOnClient ?? 0);
  let projectsBlockedOrAtRisk = $derived(
    (portfolio?.decisionSummary.blockedCount ?? 0) + (portfolio?.decisionSummary.atRiskCount ?? 0)
  );

  function healthColor(health: 'on_track' | 'at_risk' | 'blocked'): string {
    switch (health) {
      case 'on_track': return 'var(--color-success)';
      case 'at_risk': return 'var(--color-warn)';
      case 'blocked': return 'var(--color-error)';
    }
  }

  function healthLabel(health: 'on_track' | 'at_risk' | 'blocked'): string {
    switch (health) {
      case 'on_track': return 'On Track';
      case 'at_risk': return 'At Risk';
      case 'blocked': return 'Blocked';
    }
  }

  /**
   * Per-project completion display.
   * Backend sends -1 when no active tasks exist (ADR-03-001 semantics).
   * Does NOT use portfolio-wide total===0 heuristic — each project is independent.
   */
  function completionDisplay(completionPercent: number): { label: string; className: string; pct: number } {
    if (completionPercent < 0) {
      return { label: 'No active work yet', className: 'empty', pct: 0 };
    }
    return { label: `${Math.round(completionPercent)}%`, className: 'percent', pct: completionPercent };
  }

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return '—';
    }
  }

  function confidenceLabel(confidence: 'high' | 'medium' | 'low'): string {
    switch (confidence) {
      case 'high': return 'High confidence';
      case 'medium': return 'Medium confidence';
      case 'low': return 'Low confidence';
    }
  }

  /**
   * Owner label mapping — Phase 1 limitation:
   * Backend does not yet surface owner/phase data on the portfolio list contract.
   * mapOwnerLabel(phase, override) will be wired once backend provides phase+override per project.
   * This limitation is explicit here to avoid silent no-ops in the data flow.
   */
  const OWNER_LABEL_PHASE1_LIMITATION = '—'; // placeholder until Phase 2 override contract is available
</script>

<svelte:head>
  <title>Portfolio — Client Portal</title>
</svelte:head>

<div class="portfolio-page">
  <div class="page-header">
    <div class="header-left">
      <h1>My Portfolio</h1>
      <span class="refresh-hint">
        {portfolio ? `Last updated: ${formatDate(portfolio.timestamp)}` : 'Loading...'}
      </span>
    </div>
    <div class="header-right">
      <button class="refresh-btn" onclick={() => loadPortfolio()} title="Refresh portfolio" disabled={loading}>
        <span class="refresh-icon" class:spinning={loading}>↻</span>
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading-state">
      <p>Loading portfolio...</p>
    </div>
  {:else if error}
    <div class="error-state">
      <p>Failed to load portfolio: {error}</p>
      <button onclick={() => location.reload()}>Retry</button>
    </div>
  {:else if !portfolio || portfolio.projectList.length === 0}
    <div class="empty-state">
      <div class="empty-icon">◇</div>
      <h2>No projects assigned yet</h2>
      <p>You don't have access to any projects yet. Contact your administrator to request access to your portfolio.</p>
      <button class="btn-action" onclick={() => window.location.reload()}>Check Again</button>
    </div>
  {:else}
    <!-- Portfolio Health Summary (from projectsSummary, AC-03-003) -->
    <section class="health-summary">
      <div class="summary-card health-on-track">
        <span class="summary-count">{onTrackCount}</span>
        <span class="summary-label">On Track</span>
        <div class="summary-bar">
          <div
            class="bar-fill"
            style="width: {portfolio.projectList.length > 0 ? (onTrackCount / portfolio.projectList.length) * 100 : 0}%; background: var(--color-success);"
          ></div>
        </div>
      </div>
      <div class="summary-card health-at-risk">
        <span class="summary-count">{atRiskCount}</span>
        <span class="summary-label">At Risk</span>
        <div class="summary-bar">
          <div
            class="bar-fill"
            style="width: {portfolio.projectList.length > 0 ? (atRiskCount / portfolio.projectList.length) * 100 : 0}%; background: var(--color-warn);"
          ></div>
        </div>
      </div>
      <div class="summary-card health-blocked">
        <span class="summary-count">{blockedCount}</span>
        <span class="summary-label">Blocked</span>
        <div class="summary-bar">
          <div
            class="bar-fill"
            style="width: {portfolio.projectList.length > 0 ? (blockedCount / portfolio.projectList.length) * 100 : 0}%; background: var(--color-error);"
          ></div>
        </div>
      </div>
      <div class="summary-card summary-decisions">
        <span class="summary-count">{totalPendingDecisions}</span>
        <span class="summary-label">Pending Decisions</span>
      </div>
      <div class="summary-card summary-projects">
        <span class="summary-count">{portfolio.projectList.length}</span>
        <span class="summary-label">Total Projects</span>
      </div>
    </section>

    <!-- Decision Summary Bar (from decisionSummary, BRD FR-03-004 / AC-03-004) -->
    <section class="decision-summary">
      <div class="decision-item">
        <span class="decision-value">{totalPendingDecisions}</span>
        <span class="decision-label">Total pending client decisions</span>
      </div>
      <div class="decision-divider"></div>
      <div class="decision-item">
        <span class="decision-value">{overdueDecisions}</span>
        <span class="decision-label">Overdue decisions</span>
      </div>
      <div class="decision-divider"></div>
      <div class="decision-item">
        <span class="decision-value">{waitingOnClient}</span>
        <span class="decision-label">Waiting on client action</span>
      </div>
      <div class="decision-divider"></div>
      <div class="decision-item">
        <span class="decision-value">{projectsBlockedOrAtRisk}</span>
        <span class="decision-label">Projects blocked or at risk</span>
      </div>
      <div class="decision-item refresh-time">
        <span class="decision-label">Last refreshed: {formatDate(portfolio.timestamp)}</span>
      </div>
    </section>

    <!-- Project List -->
    <section class="project-list">
      <h2>My Projects</h2>
      <div class="projects-grid">
        {#each portfolio.projectList as project (project.id)}
          {@const completion = completionDisplay(project.completionPercent)}
          <button
            class="project-card"
            onclick={() => openProject(project.id)}
            type="button"
          >
            <div class="card-header">
              <span class="project-name">{project.name}</span>
              <span
                class="health-badge"
                style="--badge-color: {healthColor(project.health)}"
              >
                {healthLabel(project.health)}
              </span>
            </div>

            <!-- Confidence -->
            <div class="confidence-row">
              <span class="confidence-label">{confidenceLabel(project.confidence)}</span>
            </div>

            <!-- Decisions row (FR-03-005) -->
            <div class="decisions-row">
              {#if project.pendingDecisions > 0}
                <span class="decision-chip decision-pending">
                  {project.pendingDecisions} pending {project.pendingDecisions === 1 ? 'decision' : 'decisions'}
                </span>
              {/if}
              {#if project.overdueDecisions > 0}
                <span class="decision-chip decision-overdue">
                  {project.overdueDecisions} overdue
                </span>
              {/if}
            </div>

            <!-- Completion bar -->
            <div class="completion-section">
              <div class="completion-row">
                <span class="completion-label">Completion</span>
                <span class="completion-value {completion.className}">{completion.label}</span>
              </div>
              <div class="completion-bar">
                <div
                  class="bar-fill"
                  style="width: {completion.className === 'empty' ? 0 : completion.pct}%"
                ></div>
              </div>
            </div>

            <!-- Next milestone hint (FR-03-005) -->
            {#if project.nextMilestone}
              <div class="milestone-hint">
                <span class="milestone-icon">◇</span>
                <span class="milestone-name">{project.nextMilestone}</span>
              </div>
            {/if}

            <!-- Owner badge — Phase 1 limitation: no phase/override data on portfolio list contract -->
            <div class="project-meta">
              <span class="owner-badge">{OWNER_LABEL_PHASE1_LIMITATION}</span>
            </div>

            <div class="card-footer">
              <span class="updated-at">Updated {formatDate(project.latestUpdate)}</span>
            </div>
          </button>
        {/each}
      </div>
    </section>
  {/if}
</div>

<style>
  .portfolio-page {
    max-width: 1100px;
    margin: 0 auto;
    animation: page-enter 0.25s ease-out;
  }

  @keyframes page-enter {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .page-header h1 {
    font-size: 1.5rem;
    font-weight: 600;
  }

  .refresh-hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }

  .refresh-btn {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    color: var(--color-text-muted);
    width: 2rem;
    height: 2rem;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.15s;
  }

  .refresh-btn:hover:not(:disabled) {
    border-color: var(--color-border-hover);
    color: var(--color-text);
  }

  .refresh-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .refresh-icon {
    font-size: 1rem;
    display: inline-block;
    transition: transform 0.3s;
  }

  .refresh-icon.spinning {
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .header-left {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .header-right {
    display: flex;
    align-items: center;
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

  .error-state { color: #f87171; }

  .empty-icon {
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
    color: var(--color-text-subtle);
  }

  .empty-state h2 {
    font-size: 1.1rem;
    font-weight: 600;
    margin: 0 0 0.5rem 0;
    color: var(--color-text);
  }

  .empty-state p {
    color: var(--color-text-muted);
    font-size: 0.9rem;
    max-width: 360px;
    margin: 0 auto 1.25rem;
  }

  .btn-action {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    color: var(--color-text);
    padding: 0.5rem 1.25rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.15s;
  }

  .btn-action:hover {
    border-color: var(--color-border-hover);
    background: var(--color-border);
  }

  .error-state button {
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    color: #e0e0e0;
    padding: 0.4rem 0.8rem;
    border-radius: 4px;
    cursor: pointer;
    margin-top: 0.5rem;
    font-size: 0.85rem;
  }

  /* Health Summary */
  .health-summary {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }

  .summary-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    animation: card-enter 0.2s ease-out backwards;
  }

  .summary-card:nth-child(1) { animation-delay: 0ms; }
  .summary-card:nth-child(2) { animation-delay: 50ms; }
  .summary-card:nth-child(3) { animation-delay: 100ms; }
  .summary-card:nth-child(4) { animation-delay: 150ms; }
  .summary-card:nth-child(5) { animation-delay: 200ms; }

  @keyframes card-enter {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .summary-count {
    font-size: 2rem;
    font-weight: 700;
    line-height: 1;
  }

  .summary-label {
    font-size: 0.75rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .summary-bar {
    height: 3px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
    margin-top: 0.5rem;
  }

  .bar-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.4s ease-out;
  }

  /* Decision Summary */
  .decision-summary {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    display: flex;
    align-items: center;
    gap: 1.5rem;
    margin-bottom: 2rem;
    flex-wrap: wrap;
  }

  .decision-item {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .decision-value {
    font-size: 1.5rem;
    font-weight: 700;
    color: #e0e0e0;
  }

  .decision-label {
    font-size: 0.8rem;
    color: #888;
  }

  .decision-divider {
    width: 1px;
    height: 40px;
    background: #2a2a2a;
    flex-shrink: 0;
  }

  .refresh-time {
    margin-left: auto;
  }

  .refresh-time .decision-label {
    font-size: 0.75rem;
  }

  /* Project List */
  .project-list h2 {
    font-size: 1rem;
    font-weight: 600;
    margin-bottom: 1rem;
    color: #a0a0a0;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 1rem;
  }

  .project-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem;
    cursor: pointer;
    text-align: left;
    width: 100%;
    transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    animation: card-enter 0.25s ease-out backwards;
  }

  .project-card:hover {
    border-color: #3a3a3a;
    box-shadow: 0 2px 16px rgba(0,0,0,0.4);
    transform: translateY(-2px);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .project-name {
    font-size: 1rem;
    font-weight: 600;
    line-height: 1.3;
    color: #e0e0e0;
  }

  .health-badge {
    font-size: 0.7rem;
    font-weight: 500;
    padding: 0.15em 0.6em;
    border-radius: 4px;
    background: color-mix(in srgb, var(--badge-color) 20%, transparent);
    color: var(--badge-color);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .confidence-row {
    display: flex;
    align-items: center;
  }

  .confidence-label {
    font-size: 0.7rem;
    color: #888;
  }

  .decisions-row {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .decision-chip {
    font-size: 0.65rem;
    font-weight: 500;
    padding: 0.15em 0.5em;
    border-radius: 4px;
  }

  .decision-pending {
    background: rgba(251, 191, 36, 0.15);
    color: #fbbf24;
  }

  .decision-overdue {
    background: rgba(248, 113, 113, 0.15);
    color: #f87171;
  }

  .project-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .owner-badge {
    font-size: 0.7rem;
    font-weight: 500;
    padding: 0.15em 0.5em;
    border-radius: 4px;
    background: #2a2a2a;
    color: #a0a0a0;
  }

  /* Completion */
  .completion-section {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .completion-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .completion-label {
    font-size: 0.7rem;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .completion-value {
    font-size: 0.8rem;
    font-weight: 600;
  }

  .completion-value.percent { color: #4ade80; }
  .completion-value.empty { color: #888; font-style: italic; font-weight: 400; }

  .completion-bar {
    height: 4px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
  }

  .completion-bar .bar-fill {
    background: #4ade80;
    height: 100%;
    transition: width 0.4s ease-out;
  }

  /* Milestone hint */
  .milestone-hint {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    color: #888;
    background: #151515;
    padding: 0.4rem 0.6rem;
    border-radius: 4px;
    border: 1px solid #222;
  }

  .milestone-icon {
    color: #60a5fa;
    font-size: 0.7rem;
  }

  .milestone-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Footer */
  .card-footer {
    display: flex;
    justify-content: flex-end;
  }

  .updated-at {
    font-size: 0.7rem;
    color: #555;
  }
</style>