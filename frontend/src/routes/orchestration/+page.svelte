<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { listProjects, createProject, deleteProject } from '$lib/api/client';
  import type { Project, ProjectCreateRequest } from '$lib/api/orchestration';

  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Create form state
  let showCreate = $state(false);
  let createName = $state('');
  let createDesc = $state('');
  let createLoading = $state(false);
  let createErr = $state<string | null>(null);

  onMount(async () => {
    try {
      const res = await listProjects();
      projects = res.projects;
      error = null;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  });

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    if (!createName.trim()) return;
    createLoading = true;
    createErr = null;
    try {
      const req: ProjectCreateRequest = {
        name: createName.trim(),
        description: createDesc.trim() || undefined,
      };
      const proj = await createProject(req);
      projects = [proj, ...projects];
      showCreate = false;
      createName = '';
      createDesc = '';
      goto(`/orchestration/board?project=${proj.id}`);
    } catch (e) {
      createErr = e instanceof Error ? e.message : String(e);
    } finally {
      createLoading = false;
    }
  }

  async function handleDelete(projectId: string) {
    if (!confirm('Archive this project? All tasks will be archived.')) return;
    try {
      await deleteProject(projectId);
      projects = projects.filter((p) => p.id !== projectId);
    } catch (e) {
      alert(`Failed: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  function openProject(projectId: string) {
    goto(`/orchestration/board?project=${projectId}`);
  }

  function statusColor(phase: string): string {
    const map: Record<string, string> = {
      planning: '#60a5fa',
      decomposition: '#a78bfa',
      execution: '#34d399',
      validation: '#fbbf24',
      acceptance: '#f87171',
      closed: '#6b7280',
    };
    return map[phase] ?? '#888';
  }
</script>

<svelte:head>
  <title>Projects — Orchestration</title>
</svelte:head>

<div class="projects-page">
  <div class="page-header">
    <h1>Projects</h1>
    <button class="btn-primary" onclick={() => { showCreate = !showCreate; createErr = null; }}>
      {showCreate ? 'Cancel' : '+ New Project'}
    </button>
  </div>

  {#if showCreate}
    <form class="create-form" onsubmit={handleCreate}>
      <h2>Create Project</h2>
      <div class="form-field">
        <label for="proj-name">Project Name <span class="required">*</span></label>
        <input
          id="proj-name"
          name="proj-name"
          type="text"
          bind:value={createName}
          placeholder="BRD Authoring System"
          required
          autocomplete="off"
        />
      </div>
      <div class="form-field">
        <label for="proj-desc">Description</label>
        <textarea
          id="proj-desc"
          name="proj-desc"
          bind:value={createDesc}
          placeholder="What does this project own?"
          rows="3"
        ></textarea>
      </div>
      {#if createErr}
        <p class="form-error">{createErr}</p>
      {/if}
      <div class="form-actions">
        <button type="submit" class="btn-primary" disabled={createLoading || !createName.trim()}>
          {createLoading ? 'Creating...' : 'Create Project'}
        </button>
      </div>
    </form>
  {/if}

  {#if loading}
    <div class="loading-state">
      <p>Loading projects...</p>
    </div>
  {:else if error}
    <div class="error-state">
      <p>Failed to load projects: {error}</p>
      <button onclick={() => location.reload()}>Retry</button>
    </div>
  {:else if projects.length === 0}
    <div class="empty-state">
      <p>No projects yet. Create one to get started.</p>
    </div>
  {:else}
    <div class="projects-grid">
      {#each projects as project}
        <div class="project-card" role="button" tabindex="0"
          onclick={() => openProject(project.id)}
          onkeydown={(e) => e.key === 'Enter' && openProject(project.id)}
        >
          <div class="card-header">
            <h3 class="project-name">{project.name}</h3>
            <button
              class="btn-icon btn-delete"
              title="Archive project"
              onclick={(e) => { e.stopPropagation(); handleDelete(project.id); }}
            >×</button>
          </div>

          {#if project.description}
            <p class="project-desc">{project.description}</p>
          {/if}

          <div class="project-meta">
            <span class="phase-badge" style="--phase-color: {statusColor(project.phase)}">
              {project.phase}
            </span>
            <span class="stat">{project.statistics.totalTasks} tasks</span>
            {#if project.statistics.activeGateCount > 0}
              <span class="stat stat-gates">{project.statistics.activeGateCount} gates</span>
            {/if}
          </div>

          <div class="task-bar">
            {#if project.statistics.totalTasks > 0}
              {@const done = project.statistics.doneTasks}
              {@const blocked = project.statistics.blockedTasks}
              {@const total = project.statistics.totalTasks}
              <div class="bar-done" style="width: {(done / total) * 100}%"></div>
              <div class="bar-blocked" style="width: {(blocked / total) * 100}%"></div>
            {/if}
          </div>

          <div class="task-stats">
            <span class="ts-todo">{project.statistics.todoTasks} todo</span>
            <span class="ts-progress">{project.statistics.inProgressTasks} in progress</span>
            <span class="ts-blocked">{project.statistics.blockedTasks} blocked</span>
            <span class="ts-done">{project.statistics.doneTasks} done</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .projects-page {
    max-width: 960px;
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

  h1 {
    font-size: 1.5rem;
    font-weight: 600;
  }

  h2 {
    font-size: 1.1rem;
    font-weight: 600;
    margin-bottom: 1rem;
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

  .btn-primary:hover:not(:disabled) {
    background: #3a3a3a;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .create-form {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.25rem;
    margin-bottom: 2rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 1rem;
  }

  label {
    font-size: 0.875rem;
    color: #a0a0a0;
    font-weight: 500;
  }

  .required {
    color: #f87171;
  }

  input, textarea {
    background: #0f0f0f;
    border: 1px solid #333;
    border-radius: 4px;
    color: #e0e0e0;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    resize: vertical;
  }

  input:focus, textarea:focus {
    outline: none;
    border-color: #555;
  }

  .form-error {
    color: #f87171;
    font-size: 0.85rem;
    margin-bottom: 0.75rem;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
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

  .btn-icon {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    color: #666;
    font-size: 1rem;
    line-height: 1;
  }

  .btn-icon:hover {
    background: #2a2a2a;
    color: #e0e0e0;
  }

  .btn-delete:hover {
    color: #f87171;
  }

  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1rem;
  }

  .project-card {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1rem;
    cursor: pointer;
    transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
  }

  .project-card:hover {
    border-color: #3a3a3a;
    box-shadow: 0 2px 12px rgba(0,0,0,0.4);
    transform: translateY(-1px);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 0.5rem;
  }

  .project-name {
    font-size: 1rem;
    font-weight: 600;
    margin: 0;
    line-height: 1.3;
  }

  .project-desc {
    font-size: 0.8rem;
    color: #888;
    margin: 0 0 0.75rem 0;
    line-height: 1.4;
  }

  .project-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    flex-wrap: wrap;
  }

  .phase-badge {
    font-size: 0.7rem;
    font-weight: 500;
    padding: 0.15em 0.6em;
    border-radius: 4px;
    background: color-mix(in srgb, var(--phase-color) 20%, transparent);
    color: var(--phase-color);
    text-transform: capitalize;
  }

  .stat {
    font-size: 0.75rem;
    color: #666;
  }

  .stat-gates {
    color: #fbbf24;
  }

  .task-bar {
    display: flex;
    height: 4px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: 0.5rem;
  }

  .bar-done {
    background: #4ade80;
    height: 100%;
    transition: width 0.3s;
  }

  .bar-blocked {
    background: #f87171;
    height: 100%;
    transition: width 0.3s;
  }

  .task-stats {
    display: flex;
    gap: 0.5rem;
    font-size: 0.7rem;
    flex-wrap: wrap;
  }

  .ts-todo { color: #60a5fa; }
  .ts-progress { color: #a78bfa; }
  .ts-blocked { color: #f87171; }
  .ts-done { color: #4ade80; }
</style>