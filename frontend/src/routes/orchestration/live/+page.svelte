<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { AGENTS, EDGES, VIEWBOX } from '$lib/magnetic/topology';
  import { restNodes } from '$lib/magnetic/homes';
  import { initLivingGraph } from '$lib/magnetic/engine';
  import type { EngineHandle, NodeSelection } from '$lib/magnetic/engine';
  import { startEventFeed, startMockFeed } from '$lib/magnetic/event-feed';
  import type { FeedHandle, ReducerSnapshot } from '$lib/magnetic/event-feed';
  import {
    listProjectPhaseGates,
    listProjectTasks,
    updateProjectPhaseGate,
    updateTaskGate,
  } from '$lib/api/client';
  import type {
    ProjectPhaseGate,
    OrchestrationTask,
    GateState,
  } from '$lib/api/orchestration';

  // Rest homes are a pure function of the roster — stable per project.
  const HOMES = restNodes(AGENTS);
  const NODES = Object.entries(AGENTS);
  const EDGES_LIST = Object.entries(EDGES);

  let projectId = $derived(page.url.searchParams.get('project') ?? '');
  let root: HTMLElement;
  let engine: EngineHandle | null = null;
  let feed: FeedHandle | null = null;

  // data source: scripted fixture (no backend) vs live SSE
  let mode = $state<'mock' | 'live'>('mock');
  // presentation: animated spring motion vs static knowledge-map (first-class
  // user toggle, scout §5 mitigation — not only prefers-reduced-motion)
  let viewMode = $state<'live' | 'static'>(
    typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches ? 'static' : 'live'
  );
  let connected = $state(false);
  let feedError = $state<string | null>(null);
  let liveSupported = $derived(projectId.length > 0);

  // click → gate drawer
  let selection = $state<NodeSelection | null>(null);
  let snapshot = $state<ReducerSnapshot | null>(null);   // observed gates (mock drawer)

  // live gate data (REST — the EXISTING gate/approval UI endpoints)
  let phaseGates = $state<ProjectPhaseGate[]>([]);
  let selTask = $state<OrchestrationTask | null>(null);
  let gateLoading = $state(false);
  let gateError = $state<string | null>(null);

  // one inline approve/reject action on the drawer
  let action = $state<{ scope: 'phase' | 'task'; gateId: string; taskId?: string; phase: string; act: 'approve' | 'reject' } | null>(null);
  let actionNote = $state('');
  let actionLoading = $state(false);
  let reloadNonce = $state(0);   // bump to refetch gates after a successful action

  const STATE_COLOR: Record<GateState, string> = { open: '#fbbf24', passed: '#4ade80', blocked: '#f87171' };

  function teardown() {
    feed?.close(); feed = null;
    engine?.destroy(); engine = null;
  }

  function start() {
    if (!engine) return;
    feed?.close(); feed = null;
    connected = false; feedError = null; snapshot = null;
    const onSnap = (s: ReducerSnapshot) => { snapshot = s; };
    if (mode === 'live' && liveSupported) {
      feed = startEventFeed(projectId, engine, {
        onConnect: () => { connected = true; },
        onError: (e) => { feedError = e.message; },
        onSnapshot: onSnap,
      });
    } else {
      mode = 'live' === mode && !liveSupported ? 'mock' : mode;
      feed = startMockFeed(engine, { onSnapshot: onSnap });
      connected = true;
    }
  }

  onMount(() => {
    if (!root) return;
    mode = liveSupported ? 'live' : 'mock';
    engine = initLivingGraph(root, {
      onSelect: (s) => { selection = s; },
      initialReduced: viewMode === 'static',
    });
    start();
    return teardown;
  });

  function switchMode(next: 'mock' | 'live') {
    if (next === mode) return;
    mode = next;
    start();
  }

  function setView(v: 'live' | 'static') {
    if (v === viewMode) return;
    viewMode = v;
    engine?.setReducedMotion(v === 'static');
  }

  // Live gate fetch: only when a node/cluster is selected in live mode with a
  // project. Reuses the exact gate endpoints the /orchestration/gates page uses.
  $effect(() => {
    // deps: selection, projectId, mode, and reloadNonce (refetch after an action)
    const sel = selection; const pid = projectId; const m = mode; void reloadNonce;
    if (!sel || m !== 'live' || !pid) { phaseGates = []; selTask = null; gateError = null; return; }
    let cancelled = false;
    gateLoading = true; gateError = null;
    (async () => {
      try {
        const [pg, taskRes] = await Promise.all([listProjectPhaseGates(pid), listProjectTasks(pid)]);
        if (cancelled) return;
        phaseGates = pg.gates;
        selTask = sel.taskId ? taskRes.tasks.find((t) => t.id === sel.taskId) ?? null : null;
      } catch (e) {
        if (!cancelled) gateError = e instanceof Error ? e.message : String(e);
      } finally {
        if (!cancelled) gateLoading = false;
      }
    })();
    return () => { cancelled = true; };
  });

  function startAction(scope: 'phase' | 'task', gateId: string, phase: string, act: 'approve' | 'reject', taskId?: string) {
    action = { scope, gateId, taskId, phase, act };
    actionNote = '';
  }

  async function submitAction(e: SubmitEvent) {
    e.preventDefault();
    if (!action) return;
    actionLoading = true;
    try {
      if (action.scope === 'phase') {
        await updateProjectPhaseGate(projectId, action.gateId,
          action.act === 'approve' ? { state: 'passed' } : { state: 'blocked', criteria: actionNote.trim() ? [actionNote.trim()] : undefined });
      } else if (action.taskId) {
        await updateTaskGate(projectId, action.taskId, action.gateId,
          action.act === 'approve' ? { state: 'passed' } : { state: 'blocked', overrideNote: actionNote.trim() || undefined });
      }
      action = null;
      reloadNonce++;   // refetch gates to reflect the new state
    } catch (e) {
      alert(`Gate action failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      actionLoading = false;
    }
  }

  // Mock-mode drawer rows: observed gate state folded from the live stream
  // (faithful fixture when the backend can't be run).
  function mockGateRows() {
    if (!snapshot || !selection) return [];
    return snapshot.gates.filter((g) => !selection!.taskId || g.taskId === selection!.taskId);
  }
</script>

<svelte:head><title>Live Magnetic Graph</title></svelte:head>

<section class="mag-page">
  <header class="mag-bar">
    <div class="mag-title">
      <h2>Magnetic Clusters</h2>
      <span class="mag-sub">live agent activity → emergent graph motion</span>
    </div>
    <div class="mag-controls">
      <span class="dot" class:on={connected} title={feedError ?? (connected ? 'feed live' : 'connecting…')}></span>
      <div class="seg view" role="group" aria-label="View mode">
        <button class:on={viewMode === 'live'} onclick={() => setView('live')} title="Animated spring motion">Motion</button>
        <button class:on={viewMode === 'static'} onclick={() => setView('static')} title="Static knowledge-map (no motion)">Map</button>
      </div>
      <div class="seg" role="group" aria-label="Data source">
        <button class:on={mode === 'mock'} onclick={() => switchMode('mock')}>Demo</button>
        <button class:on={mode === 'live'} disabled={!liveSupported}
          title={liveSupported ? 'Drive from live SSE' : 'select a project (?project=)'} onclick={() => switchMode('live')}>Live</button>
      </div>
      {#if viewMode === 'live'}
        <button class="play" data-action="play">Pause</button>
      {/if}
    </div>
  </header>

  {#if mode === 'mock'}
    <p class="banner">Demo mode — driven by a scripted SSE fixture. Click a node to open its gate/approval drawer (read-only fixture). Select a project and switch to <b>Live</b> to act on real gates via the backend endpoints.</p>
  {:else if feedError}
    <p class="banner warn">Live feed error: {feedError}</p>
  {/if}

  <div class="mag-graph" bind:this={root}>
    <svg class="map" viewBox={`0 0 ${VIEWBOX.w} ${VIEWBOX.h}`} preserveAspectRatio="xMidYMid meet" role="img" aria-label="Live magnetic agent graph">
      <defs>
        <marker id="arrow-stub" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" class="mk-dim" /></marker>
        <marker id="arrow-now-stub" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" class="mk-on" /></marker>
      </defs>

      {#each EDGES_LIST as [id, e] (id)}
        <path class="edge" class:rw={e.kind === 'rw'} data-edge={id} data-kind={e.kind} />
        <text class="elabel" data-elabel={id}>{e.label}</text>
      {/each}

      {#each NODES as [id, a] (id)}
        <g class="node" data-node={id} transform={`translate(${HOMES[id].x},${HOMES[id].y})`} style={`--role: var(--h-${a.hue})`} role="img" aria-label={`${a.role}: ${a.sub}`}>
          <circle class="ring" r="34" />
          <text class="role" y="-2">{a.role}</text>
          <text class="sub" y="14">{a.sub}</text>
        </g>
      {/each}
    </svg>

    <aside class="detail empty" data-detail><span>Click a node to inspect its gates/approvals.</span></aside>
    <div class="caption" data-caption><span class="k">Settled.</span> <span class="c">Agents rest at home until work activates.</span></div>
    <div class="counter" data-count>0 ACTIVE</div>
    <div class="progress"><div class="fill" data-fill style="width:0"></div></div>
    <div class="summary" data-summary hidden>
      <h3>Knowledge map</h3>
      <p>Static view. Cards accumulate as evidence of completed work; edges show the handoff graph that produced them. Switch to <b>Motion</b> for the live spring simulation.</p>
    </div>

    <!-- click → gate/approval drawer (reuses the existing gate endpoints) -->
    {#if selection}
      <aside class="drawer" role="dialog" aria-label="Gates and approvals">
        <header class="drawer-h">
          <div>
            <span class="d-role" style={`--role: var(--h-${AGENTS[selection.agentId]?.hue ?? 'user'})`}>{AGENTS[selection.agentId]?.role ?? selection.agentId}</span>
            <span class="d-task">{selection.taskId ? `task ${selection.taskId}` : 'no active task'}</span>
          </div>
          <button class="d-close" onclick={() => { selection = null; }} aria-label="Close drawer">×</button>
        </header>

        {#if mode === 'live'}
          {#if gateLoading}
            <p class="d-muted">Loading gates…</p>
          {:else if gateError}
            <p class="d-err">{gateError}</p>
          {:else}
            {#if selTask && selTask.gates.length}
              <h4>Task gates — {selTask.title}</h4>
              {#each selTask.gates as g (g.id)}
                <div class="grow">
                  <div class="grow-h">
                    <span class="grow-phase">{g.phase.replace(/_/g, ' ')}</span>
                    <span class="badge" style={`--c:${STATE_COLOR[g.state]}`}>{g.state}</span>
                  </div>
                  {#if g.state === 'open'}
                    <div class="grow-actions">
                      <button class="mini pass" onclick={() => startAction('task', g.id, g.phase, 'approve', selTask!.id)}>Approve</button>
                      <button class="mini block" onclick={() => startAction('task', g.id, g.phase, 'reject', selTask!.id)}>Reject</button>
                    </div>
                  {/if}
                </div>
              {/each}
            {/if}
            <h4>Phase gates</h4>
            {#if phaseGates.length}
              {#each phaseGates as g (g.id)}
                <div class="grow">
                  <div class="grow-h">
                    <span class="grow-phase">G{g.phaseIndex} · {g.phase}</span>
                    <span class="badge" style={`--c:${STATE_COLOR[g.state]}`}>{g.state}</span>
                  </div>
                  {#if g.state === 'open'}
                    <div class="grow-actions">
                      <button class="mini pass" onclick={() => startAction('phase', g.id, g.phase, 'approve')}>Approve</button>
                      <button class="mini block" onclick={() => startAction('phase', g.id, g.phase, 'reject')}>Reject</button>
                    </div>
                  {/if}
                </div>
              {/each}
            {:else}
              <p class="d-muted">No phase gates defined.</p>
            {/if}
          {/if}
        {:else}
          <!-- mock: faithful fixture folded from the live gate.* stream -->
          <p class="d-muted fixture">Demo fixture — gate state observed in the scripted stream. Switch to Live to act on real gates.</p>
          {#if mockGateRows().length}
            {#each mockGateRows() as g (g.gateType)}
              <div class="grow">
                <div class="grow-h">
                  <span class="grow-phase">{g.gateType.replace(/_/g, ' ')}</span>
                  <span class="badge" style={`--c:${STATE_COLOR[g.state]}`}>{g.state}</span>
                </div>
              </div>
            {/each}
          {:else if selection.taskId}
            <p class="d-muted">No gate events observed for this task yet.</p>
          {:else}
            <p class="d-muted">Select an agent in an active cluster to see its task's gates.</p>
          {/if}
        {/if}
      </aside>
    {/if}
  </div>
</section>

<!-- gate approve/reject action (reuses updateProjectPhaseGate / updateTaskGate) -->
{#if action}
  <div class="modal-overlay" onclick={() => { action = null; }} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h2>{action.act === 'approve' ? 'Approve' : 'Reject'} {action.scope} gate — {action.phase.replace(/_/g, ' ')}</h2>
      {#if action.act === 'reject'}
        <form onsubmit={submitAction}>
          <div class="form-field">
            <label for="ga-note">Rejection note <span class="req">*</span></label>
            <textarea id="ga-note" bind:value={actionNote} required rows="3" placeholder="Why is this gate rejected?"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" onclick={() => { action = null; }}>Cancel</button>
            <button type="submit" class="btn-danger" disabled={actionLoading || !actionNote.trim()}>{actionLoading ? 'Submitting…' : 'Reject gate'}</button>
          </div>
        </form>
      {:else}
        <form onsubmit={submitAction}>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" onclick={() => { action = null; }}>Cancel</button>
            <button type="submit" class="btn-pass" disabled={actionLoading}>{actionLoading ? 'Submitting…' : 'Approve gate'}</button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

<style>
  :global(:root) {
    --h-user: #93c5fd; --h-intake: #60a5fa; --h-architect: #a78bfa; --h-builder: #4ade80;
    --h-security: #f87171; --h-qa: #fbbf24; --h-devops: #fb923c; --h-human: #e879f9; --h-prod: #34d399;
  }

  .mag-page { max-width: 1280px; margin: 0 auto; }

  .mag-bar {
    display: flex; align-items: center; justify-content: space-between; gap: 1rem;
    margin-bottom: 0.75rem; flex-wrap: wrap;
  }
  .mag-title h2 { margin: 0; font-size: 1.1rem; }
  .mag-sub { color: #888; font-size: 0.78rem; }

  .mag-controls { display: flex; align-items: center; gap: 0.6rem; font-size: 0.8rem; flex-wrap: wrap; }
  .dot { width: 9px; height: 9px; border-radius: 50%; background: #555; }
  .dot.on { background: var(--color-success); box-shadow: 0 0 8px var(--color-success); }
  .seg { display: inline-flex; border: 1px solid #333; border-radius: 6px; overflow: hidden; }
  .seg button { background: #1a1a1a; color: #aaa; border: 0; padding: 0.3rem 0.7rem; cursor: pointer; font-size: 0.78rem; }
  .seg button.on { background: #2a4a2a; color: #fff; }
  .seg button:disabled { opacity: 0.4; cursor: not-allowed; }
  .play { background: #1a1a1a; color: #ddd; border: 1px solid #333; border-radius: 6px; padding: 0.3rem 0.8rem; cursor: pointer; font-size: 0.78rem; }

  .banner { background: #1a2a1a; border: 1px solid #2a4a2a; color: #bfe; padding: 0.5rem 0.8rem; border-radius: 6px; font-size: 0.8rem; margin: 0 0 0.75rem; }
  .banner.warn { background: #2a1a1a; border-color: #4a2a2a; color: #fcc; }

  .mag-graph {
    position: relative; background: #0b0d10; border: 1px solid #1d2024; border-radius: 10px; overflow: hidden;
  }
  .mag-graph svg.map { width: 100%; height: clamp(360px, 62vh, 760px); display: block; }

  /* The engine toggles element classes (.on/.now/.lit/.wait/.fresh/.sel) and
     mounts cards from JS, so all svg-internal element styles are :global under
     .mag-graph (Svelte would otherwise strip them as "unused" in the build). */
  .mag-graph :global(.edge) { fill: none; stroke: #2a2f36; stroke-width: 1.5; transition: stroke 0.3s, stroke-width 0.3s; }
  .mag-graph :global(.edge.lit) { stroke: #3a4250; }
  .mag-graph :global(.edge.now) { stroke: #7dd3fc; stroke-width: 2.4; filter: drop-shadow(0 0 4px rgba(125,211,252,0.5)); }
  .mag-graph :global(.edge.rw.now) { stroke-dasharray: 6 5; }
  .mag-graph :global(.mk-dim) { fill: #3a4250; }
  .mag-graph :global(.mk-on) { fill: #7dd3fc; }
  .mag-graph :global(.elabel) { fill: #6b7280; font-size: 12px; opacity: 0; transition: opacity 0.3s; pointer-events: none; }
  .mag-graph :global(.elabel.on) { opacity: 1; }

  .mag-graph :global(.node) { cursor: pointer; }
  .mag-graph :global(.node:focus-visible) { outline: none; }
  .mag-graph :global(.node .ring) { fill: #131721; stroke: var(--role, #2c3340); stroke-width: 2; transition: stroke 0.25s, filter 0.25s, stroke-width 0.25s; }
  .mag-graph :global(.node .role) { fill: #cbd5e1; font-size: 11px; font-weight: 600; text-anchor: middle; pointer-events: none; }
  .mag-graph :global(.node .sub) { fill: #6b7280; font-size: 9px; text-anchor: middle; pointer-events: none; }
  .mag-graph :global(.node.on .ring) { fill: #0f1a14; stroke: var(--role); filter: drop-shadow(0 0 10px var(--role)); }
  .mag-graph :global(.node.on .role) { fill: #fff; }
  .mag-graph :global(.node.sel .ring) { stroke-width: 3.5; stroke: #7dd3fc; filter: drop-shadow(0 0 12px rgba(125,211,252,0.85)); }
  .mag-graph :global(.node.wait .ring) { stroke: var(--color-warn); }

  .mag-graph :global(.card rect) { fill: rgba(20,24,30,0.92); stroke: #555; stroke-width: 1.5; }
  .mag-graph :global(.card .cat) { fill: #93a1b5; font-size: 9px; font-weight: 700; letter-spacing: 0.5px; }
  .mag-graph :global(.card .ln) { fill: #d6dde6; font-size: 11px; }
  .mag-graph :global(.card.fresh rect) { stroke-width: 2.5; filter: drop-shadow(0 0 8px rgba(125,211,252,0.6)); }
  .mag-graph :global(.card[data-hue="builder"] rect) { stroke: var(--h-builder); }
  .mag-graph :global(.card[data-hue="intake"] rect) { stroke: var(--h-intake); }
  .mag-graph :global(.card[data-hue="architect"] rect) { stroke: var(--h-architect); }
  .mag-graph :global(.card[data-hue="security"] rect) { stroke: var(--h-security); }
  .mag-graph :global(.card[data-hue="qa"] rect) { stroke: var(--h-qa); }
  .mag-graph :global(.card[data-hue="devops"] rect) { stroke: var(--h-devops); }
  .mag-graph :global(.card[data-hue="prod"] rect) { stroke: var(--h-prod); }

  .detail {
    position: absolute; top: 0.75rem; left: 0.75rem; width: 230px; max-height: 55%; overflow-y: auto;
    background: rgba(15,17,21,0.92); border: 1px solid #222; border-radius: 8px; padding: 0.6rem 0.7rem;
    font-size: 0.78rem; color: #aab; backdrop-filter: blur(4px);
  }
  .detail.empty { color: #667; font-style: italic; }
  .detail :global(.who) { display: flex; flex-direction: column; margin-bottom: 0.4rem; border-left: 3px solid var(--role, #444); padding-left: 0.5rem; }
  .detail :global(.name) { color: #e6edf3; font-weight: 600; }
  .detail :global(.role-sub) { color: #778; font-size: 0.72rem; }
  .detail :global(.obj) { color: #bcc; }
  .detail :global(.chips) { display: flex; flex-wrap: wrap; gap: 0.25rem; margin-top: 0.35rem; }
  .detail :global(.chip) { background: #1d2024; color: #9ab; padding: 0.05rem 0.4rem; border-radius: 4px; font-size: 0.68rem; }
  .detail :global(.chip.state) { background: #243; color: #7e7; }

  /* click → gate/approval drawer */
  .drawer {
    position: absolute; top: 0.75rem; right: 0.75rem; bottom: 0.75rem; width: 300px;
    background: rgba(15,17,21,0.95); border: 1px solid #2a4a4a; border-radius: 8px; padding: 0.7rem 0.8rem;
    font-size: 0.8rem; color: #cdd; overflow-y: auto; box-shadow: 0 4px 24px rgba(0,0,0,0.4); backdrop-filter: blur(4px);
  }
  .drawer-h { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.6rem; padding-bottom: 0.5rem; border-bottom: 1px solid #2a2f36; }
  .d-role { display: block; font-weight: 700; color: #e6edf3; border-left: 3px solid var(--role, #444); padding-left: 0.45rem; }
  .d-task { display: block; color: #778; font-size: 0.7rem; padding-left: 0.45rem; margin-top: 0.15rem; font-family: var(--font-mono, monospace); }
  .d-close { background: none; border: 0; color: #888; font-size: 1.2rem; cursor: pointer; line-height: 1; }
  .d-close:hover { color: #fff; }
  .d-muted { color: #778; font-size: 0.75rem; margin: 0.3rem 0; }
  .d-muted.fixture { background: #1a2a1a; border: 1px solid #2a4a2a; border-radius: 4px; padding: 0.4rem 0.5rem; color: #9c9; }
  .d-err { color: #f87171; font-size: 0.75rem; }
  .drawer h4 { font-size: 0.72rem; color: #889; text-transform: uppercase; letter-spacing: 0.5px; margin: 0.7rem 0 0.35rem; }
  .grow { background: #131721; border: 1px solid #1d2024; border-radius: 6px; padding: 0.45rem 0.55rem; margin-bottom: 0.4rem; }
  .grow-h { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; }
  .grow-phase { font-size: 0.78rem; color: #cdd; text-transform: capitalize; }
  .badge { font-size: 0.66rem; font-weight: 600; padding: 0.05em 0.45em; border-radius: 3px; background: color-mix(in srgb, var(--c) 20%, transparent); color: var(--c); text-transform: capitalize; }
  .grow-actions { display: flex; gap: 0.3rem; margin-top: 0.4rem; }
  .mini { background: #1a1a1a; border: 1px solid #333; color: #9ab; padding: 0.2rem 0.55rem; border-radius: 4px; cursor: pointer; font-size: 0.72rem; }
  .mini.pass { color: #4ade80; border-color: #4ade8040; }
  .mini.pass:hover { background: #4ade8020; }
  .mini.block { color: #f87171; border-color: #f8717140; }
  .mini.block:hover { background: #f8717120; }

  .caption {
    position: absolute; bottom: 0.75rem; left: 0.75rem; right: 0.75rem;
    background: rgba(15,17,21,0.9); border: 1px solid #222; border-radius: 8px; padding: 0.5rem 0.8rem;
    font-size: 0.82rem; color: #ccd; backdrop-filter: blur(4px);
  }
  .caption :global(.k) { color: #7dd3fc; font-weight: 600; }
  .counter { position: absolute; top: 0.75rem; right: 0.75rem; background: rgba(15,17,21,0.9); border: 1px solid #222; border-radius: 6px; padding: 0.25rem 0.6rem; font-size: 0.72rem; color: #7e7; font-weight: 600; }
  .progress { position: absolute; top: 0; left: 0; right: 0; height: 2px; background: #1a1d22; }
  .progress .fill { height: 100%; background: linear-gradient(90deg, var(--h-architect), var(--h-builder)); transition: width 0.4s; }
  .summary { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 80%; max-width: 520px; background: rgba(15,17,21,0.95); border: 1px solid #222; border-radius: 8px; padding: 1rem; color: #aab; text-align: center; }

  /* drawer open → shrink the caption/counter to make room on the right */
  .mag-graph:has(.drawer) .caption { right: 330px; }
  .mag-graph:has(.drawer) .counter { right: 330px; }

  .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 1rem; }
  .modal { background: #1a1a1a; border: 1px solid #333; border-radius: 8px; padding: 1.5rem; width: 100%; max-width: 440px; }
  .modal h2 { font-size: 1rem; font-weight: 600; margin: 0 0 1rem; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
  .form-field { display: flex; flex-direction: column; gap: 0.3rem; margin-bottom: 0.5rem; }
  .form-field label { font-size: 0.8rem; color: #888; }
  .req { color: #f87171; }
  textarea { background: #0f0f0f; border: 1px solid #2a2a2a; border-radius: 4px; color: #e0e0e0; padding: 0.4rem 0.6rem; font-size: 0.85rem; font-family: inherit; resize: vertical; }
  textarea:focus { outline: none; border-color: #555; }
  .btn-secondary { background: transparent; color: #888; border: 1px solid #333; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-danger { background: transparent; color: #f87171; border: 1px solid #f8717140; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-pass { background: transparent; color: #4ade80; border: 1px solid #4ade8040; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem; }
  .btn-pass:disabled { opacity: 0.5; cursor: not-allowed; }

  @media (max-width: 720px) {
    .drawer { width: calc(100% - 1.5rem); bottom: auto; max-height: 45%; }
    .caption { right: 0.75rem; }
    .counter { right: 0.75rem; }
  }
</style>
