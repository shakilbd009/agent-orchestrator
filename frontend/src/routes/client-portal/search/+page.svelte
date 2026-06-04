<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { searchClientPortal } from '$lib/api/client';
  import type { ClientSearchResults } from '$lib/api/orchestration';

  let query = $state('');
  let results = $state<ClientSearchResults | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let hasSearched = $state(false);

  // Filters
  type TypeFilter = 'all' | 'task' | 'project' | 'decision' | 'milestone' | 'risk' | 'comment';
  let typeFilter = $state<TypeFilter>('all');

  const typeFilters: { value: TypeFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'task', label: 'Tasks' },
    { value: 'project', label: 'Projects' },
    { value: 'decision', label: 'Decisions' },
    { value: 'milestone', label: 'Milestones' },
    { value: 'risk', label: 'Risks' },
    { value: 'comment', label: 'Comments' },
  ];

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    const q = params.get('q');
    if (q) {
      query = q;
      handleSearchInternal(q);
    }
  });

  async function handleSearch(e: SubmitEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    await handleSearchInternal(query.trim());
  }

  async function handleSearchInternal(q: string) {
    loading = true;
    error = null;
    hasSearched = true;
    try {
      const typeFilterValue = typeFilter !== 'all' ? [typeFilter] : undefined;
      results = await searchClientPortal(q, {
        type: typeFilterValue,
        status: undefined,
        projectId: undefined,
        health: undefined,
      });
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function clearSearch() {
    query = '';
    results = null;
    hasSearched = false;
    error = null;
  }

  let filteredResults = $derived.by(() => {
    if (!results) return [];
    return results.items;
  });

  function typeIcon(t: string): string {
    const m: Record<string, string> = {
      task: '◈', project: '◇', decision: '◆', milestone: '◉', risk: '⚠', comment: '💬',
    };
    return m[t] ?? '○';
  }

  function typeColor(t: string): string {
    const m: Record<string, string> = {
      task: '#60a5fa', project: '#a78bfa', decision: '#fbbf24', milestone: '#4ade80', risk: '#f87171', comment: '#888',
    };
    return m[t] ?? '#888';
  }

  function navigateToResult(item: { type: string; projectId: string | null }) {
    if (!item.projectId) return;
    if (item.type === 'task') {
      goto(`/client-portal/projects/${item.projectId}?tab=board`);
    } else if (item.type === 'decision' || item.type === 'approval') {
      goto(`/client-portal/projects/${item.projectId}?tab=approvals`);
    } else if (item.type === 'milestone') {
      goto(`/client-portal/projects/${item.projectId}?tab=milestones`);
    } else if (item.type === 'risk') {
      goto(`/client-portal/projects/${item.projectId}?tab=risks`);
    } else {
      goto(`/client-portal/projects/${item.projectId}`);
    }
  }
</script>

<svelte:head>
  <title>Search — Client Portal</title>
</svelte:head>

<div class="search-page">
  <div class="page-header">
    <h1>Search</h1>
    <p class="header-subtitle">Find tasks, decisions, risks, milestones across your projects</p>
  </div>

  <form class="search-form" onsubmit={handleSearch}>
    <div class="search-row">
      <input
        id="search-q"
        name="search-q"
        type="text"
        bind:value={query}
        placeholder="Search by keyword..."
        autocomplete="off"
        class="search-input"
      />
      <button type="submit" class="btn-search" disabled={loading}>
        {loading ? '...' : 'Search'}
      </button>
      {#if results || query}
        <button type="button" class="btn-clear" onclick={clearSearch}>Clear</button>
      {/if}
    </div>
    <div class="filter-row">
      <div class="type-filter-chips">
        {#each typeFilters as f}
          <button
            class="filter-chip"
            class:active={typeFilter === f.value}
            onclick={() => { typeFilter = f.value as TypeFilter; }}
          >
            {f.label}
          </button>
        {/each}
      </div>
    </div>
  </form>

  {#if error}
    <div class="err-banner">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Searching...</p>
    </div>
  {:else if hasSearched && !results}
    <div class="empty-state">
      <p class="empty-title">Search failed</p>
      <p class="empty-hint">{error ?? 'An unknown error occurred.'}</p>
    </div>
  {:else if hasSearched && results && results.items.length === 0}
    <div class="empty-state">
      <div class="empty-icon">🔍</div>
      <p class="empty-title">No results for "{query}"</p>
      <p class="empty-hint">Try different keywords or adjust your filters.</p>
    </div>
  {:else if results && results.items.length > 0}
    <div class="results-header">
      <span class="results-count">
        {results.totalCount} result{results.totalCount !== 1 ? 's' : ''}
        {#if results.searchDurationMs}
          · {Math.round(results.searchDurationMs)}ms
        {/if}
      </span>
    </div>

    <div class="results-list">
      {#each filteredResults as item}
        <button
          class="result-card"
          onclick={() => navigateToResult(item)}
          disabled={!item.projectId}
        >
          <div class="result-type-icon" style="color: {typeColor(item.type)}">{typeIcon(item.type)}</div>
          <div class="result-body">
            <div class="result-title">{item.title}</div>
            {#if item.projectName}
              <div class="result-project">{item.projectName}</div>
            {/if}
            {#if item.highlightedText}
              <div class="result-snippet">{item.highlightedText}</div>
            {/if}
            <div class="result-meta">
              <span class="result-type-label" style="color: {typeColor(item.type)}">{item.type}</span>
            </div>
          </div>
        </button>
      {/each}
    </div>
  {:else}
    <div class="idle-state">
      <p>Enter a search term to find tasks, decisions, risks, and milestones across your accessible projects.</p>
    </div>
  {/if}
</div>

<style>
  .search-page {
    max-width: 800px;
    margin: 0 auto;
    animation: page-enter 0.25s ease-out;
  }

  @keyframes page-enter { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }

  .page-header { margin-bottom: 1.5rem; }

  h1 {
    font-size: 1.5rem;
    font-weight: 600;
    margin: 0 0 0.25rem 0;
  }

  .header-subtitle {
    color: #888;
    font-size: 0.875rem;
    margin: 0;
  }

  .search-form { margin-bottom: 1.5rem; }

  .search-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .search-input {
    flex: 1;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 6px;
    color: #e0e0e0;
    padding: 0.75rem 1rem;
    font-size: 0.95rem;
    font-family: inherit;
  }

  .search-input:focus {
    outline: none;
    border-color: #555;
  }

  .btn-search {
    background: #2a2a2a;
    color: #e0e0e0;
    border: 1px solid #3a3a3a;
    padding: 0.75rem 1.25rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    white-space: nowrap;
  }

  .btn-search:disabled { opacity: 0.5; }

  .btn-clear {
    background: none;
    border: 1px solid var(--color-border);
    color: var(--color-text-muted);
    padding: 0.75rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.15s;
  }

  .filter-row {
    margin-top: 0.75rem;
  }

  .type-filter-chips {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .filter-chip {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    color: var(--color-text-muted);
    padding: 0.3rem 0.75rem;
    border-radius: 20px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
  }

  .filter-chip:hover { border-color: var(--color-border-hover); color: var(--color-text); }

  .filter-chip.active {
    background: var(--color-border);
    color: var(--color-text);
    border-color: var(--color-border-hover);
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

  .loading-state,
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

  .empty-icon {
    font-size: 2.5rem;
    margin-bottom: 1rem;
  }

  .empty-title { font-weight: 600; color: #e0e0e0; margin-bottom: 0.5rem; }
  .empty-hint { font-size: 0.875rem; }

  .idle-state {
    text-align: center;
    padding: 3rem 2rem;
    color: #666;
    font-size: 0.9rem;
  }

  .results-header {
    margin-bottom: 1rem;
  }

  .results-count {
    font-size: 0.875rem;
    color: #888;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .result-card {
    display: flex;
    gap: 0.875rem;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem;
    cursor: pointer;
    text-align: left;
    width: 100%;
    transition: border-color 0.15s;
    font-family: inherit;
    color: inherit;
  }

  .result-card:hover { border-color: #3a3a3a; }
  .result-card:disabled { opacity: 0.5; cursor: not-allowed; }

  .result-type-icon {
    font-size: 1.25rem;
    flex-shrink: 0;
    padding-top: 0.1rem;
  }

  .result-body {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    flex: 1;
    min-width: 0;
  }

  .result-title {
    font-size: 0.95rem;
    font-weight: 500;
    color: #e0e0e0;
  }

  .result-project {
    font-size: 0.8rem;
    color: #888;
  }

  .result-snippet {
    font-size: 0.8rem;
    color: #666;
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-meta {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .result-type-label {
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
</style>