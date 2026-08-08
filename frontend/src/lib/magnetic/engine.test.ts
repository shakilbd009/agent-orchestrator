import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { initLivingGraph } from './engine';
import { createReducer } from './event-feed';
import { clusterCentroid, separateCentroids, restNodes } from './homes';
import { AGENTS, EDGES } from './topology';
import type { EventEnvelope } from '$lib/api/orchestration';
import type { EngineHandle as EngineH, CardSpec } from './engine';

// jsdom lacks matchMedia; the engine checks prefers-reduced-motion at init.
function stubMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (q: string) => ({ matches: false, media: q, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {}, dispatchEvent: () => false }),
  });
}

// Minimal SVG scaffold mirroring what /orchestration/live renders.
function scaffold(): HTMLElement {
  const root = document.createElement('div');
  const homes: Record<string, { x: number; y: number }> = {
    pm: { x: 120, y: 200 }, architect: { x: 120, y: 360 }, developer: { x: 430, y: 280 },
    reviewer: { x: 680, y: 200 }, qa: { x: 680, y: 400 }, devops: { x: 980, y: 260 },
    captain: { x: 980, y: 420 }, release: { x: 1200, y: 340 },
  };
  let svg = '<svg class="map" viewBox="0 0 1320 820">';
  for (const [id, e] of Object.entries(EDGES)) {
    svg += `<path data-edge="${id}" data-kind="${e.kind}"></path><text data-elabel="${id}">${e.label}</text>`;
  }
  for (const [id, a] of Object.entries(AGENTS)) {
    const h = homes[id];
    svg += `<g data-node="${id}" transform="translate(${h.x},${h.y})" style="--role:var(--h-${a.hue})"><circle class="ring"></circle><text class="role">${a.role}</text><text class="sub">${a.sub}</text></g>`;
  }
  svg += '</svg>';
  root.innerHTML = svg + '<div data-detail></div><div data-caption></div>';
  return root;
}

function env(topic: string, payload: Record<string, unknown>, taskId: string | null): EventEnvelope {
  return { eventId: 'e', schemaVersion: 'v1alpha', projectId: 'p', topic, actorId: 'x', actorRole: 'layer_b', taskId, parentTaskId: null, gateId: null, timestamp: new Date().toISOString(), payload };
}

// Replicates event-feed wireReducer's per-tick action on a live engine.
function drive(reducer: ReturnType<typeof createReducer>, engine: EngineH, e: EventEnvelope) {
  for (const c of reducer.reduce(e)) engine.mountCard(c);
  engine.pushBeat(reducer.snapshot());
}

describe('magnetic engine + feed integration (jsdom)', () => {
  beforeEach(() => { stubMatchMedia(); });
  afterEach(() => { vi.restoreAllMocks(); });

  it('an agent.activated beat lights the node; handoff.submitted materializes a card', () => {
    const root = scaffold();
    document.body.appendChild(root);
    const engine = initLivingGraph(root);
    const reducer = createReducer({ agents: AGENTS, edges: EDGES });
    try {
      const pmNode = root.querySelector('[data-node="pm"]')!;
      expect(pmNode.classList.contains('on')).toBe(false);

      drive(reducer, engine, env('agent.activated', { agentName: 'pm', layer: 'A', taskId: 't1' }, 't1'));
      expect(pmNode.classList.contains('on')).toBe(true);   // active → pulled into cluster

      drive(reducer, engine, env('agent.idle', { agentName: 'pm', layer: 'A' }, null));
      expect(pmNode.classList.contains('on')).toBe(false);   // released home

      drive(reducer, engine, env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 't2' }, 't2'));
      drive(reducer, engine, env('handoff.submitted', { taskId: 't2', submittedBy: 'developer', summary: 'done', artifacts: ['diff.patch'], validationPerformed: 'unit' }, 't2'));
      const card = root.querySelector('[data-card="diff.patch"]') as SVGGElement | null;
      expect(card).not.toBeNull();                            // card materialized
      expect(card!.style.opacity).toBe('1');                  // and made visible
    } finally {
      engine.destroy();
      root.remove();
    }
  });

  // Phase E — node/cluster click opens the gate drawer (scout §5: repurpose
  // hover/focus renderDetail to a CLICK that resolves the agent's active task
  // from the current beat's clusters).
  it('clicking a node fires onSelect with the agent and its active task; canvas click clears', () => {
    const root = scaffold();
    document.body.appendChild(root);
    let sel: { agentId: string; taskId: string | null } | null | undefined;
    const engine = initLivingGraph(root, { onSelect: (s) => { sel = s; } });
    const reducer = createReducer({ agents: AGENTS, edges: EDGES });
    try {
      drive(reducer, engine, env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 't1' }, 't1'));
      const devNode = root.querySelector('[data-node="developer"]')!;
      devNode.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      expect(sel).toEqual({ agentId: 'developer', taskId: 't1' });   // resolved from cluster t1
      expect(devNode.classList.contains('sel')).toBe(true);

      // an idle agent (no active cluster) → taskId null
      root.querySelector('[data-node="pm"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      expect(sel).toEqual({ agentId: 'pm', taskId: null });

      // clicking the empty canvas clears the selection
      const svg = root.querySelector('svg.map')!;
      svg.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      expect(sel).toBeNull();
    } finally {
      engine.destroy();
      root.remove();
    }
  });
});

describe('clusterCentroid (Phase E per-task centroids, scout §4.2)', () => {
  it('two concurrent tasks get two distinct centroids at their own agents\' home means', () => {
    const homes: Record<string, { x: number; y: number }> = {
      developer: { x: 100, y: 100 }, reviewer: { x: 120, y: 140 },
      qa: { x: 900, y: 800 }, devops: { x: 920, y: 760 },
    };
    const cA = clusterCentroid({ agents: ['developer', 'reviewer'] }, homes)!;
    const cB = clusterCentroid({ agents: ['qa', 'devops'] }, homes)!;
    // cluster A centered near (110,120); cluster B centered near (910,780)
    expect(cA.x).toBeCloseTo(110); expect(cA.y).toBeCloseTo(120);
    expect(cB.x).toBeCloseTo(910); expect(cB.y).toBeCloseTo(780);
    // the two clusters are clearly separated (no merged centroid)
    expect(Math.hypot(cA.x - cB.x, cA.y - cB.y)).toBeGreaterThan(800);
    // an empty / all-unknown cluster yields no centroid (engine falls back)
    expect(clusterCentroid({ agents: [] }, homes)).toBeNull();
    expect(clusterCentroid({ agents: ['ghost'] }, homes)).toBeNull();
  });
});

describe('separateCentroids — review M1 (2×2 Layer-B diagonal collapse)', () => {
  // The REAL restNodes(AGENTS) layout: Layer B sits in a 2×2 grid, so the
  // complementary diagonal splits share the rectangle's center → raw centroids
  // coincide (separation 0) → one merged cluster. This is the exact hole the
  // synthetic-homes test above masks. Prove it, then prove separateCentroids
  // fixes it against the real layout.
  const realHomes = restNodes(AGENTS);

  it('raw centroids coincide for the {developer,reviewer} vs {qa,devops} diagonal', () => {
    const a = clusterCentroid({ agents: ['developer', 'reviewer'] }, realHomes)!;
    const b = clusterCentroid({ agents: ['qa', 'devops'] }, realHomes)!;
    expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeLessThanOrEqual(1);   // documents the gap
  });

  it('separateCentroids splits the diagonal into two distinct focal points', () => {
    const raw: Record<string, { x: number; y: number }> = {
      tA: clusterCentroid({ agents: ['developer', 'reviewer'] }, realHomes)!,
      tB: clusterCentroid({ agents: ['qa', 'devops'] }, realHomes)!,
    };
    const sep = separateCentroids(raw);
    const d = Math.hypot(sep.tA.x - sep.tB.x, sep.tA.y - sep.tB.y);
    expect(d).toBeGreaterThan(40);   // clearly separated, not one merged point
    // non-coincident clusters are left untouched
    const raw2: Record<string, { x: number; y: number }> = {
      tA: clusterCentroid({ agents: ['developer', 'qa'] }, realHomes)!,       // top row
      tB: clusterCentroid({ agents: ['reviewer', 'devops'] }, realHomes)!,    // bottom row
    };
    const sep2 = separateCentroids(raw2);
    expect(sep2.tA).toEqual(raw2.tA);
    expect(sep2.tB).toEqual(raw2.tB);
  });

  it('separateCentroids is deterministic — same ids, same offsets every call', () => {
    const raw: Record<string, { x: number; y: number }> = {
      tA: clusterCentroid({ agents: ['developer', 'reviewer'] }, realHomes)!,
      tB: clusterCentroid({ agents: ['qa', 'devops'] }, realHomes)!,
    };
    expect(separateCentroids(raw)).toEqual(separateCentroids(raw));
  });
});
