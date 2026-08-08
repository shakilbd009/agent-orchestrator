import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { initLivingGraph } from './engine';
import { createReducer } from './event-feed';
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
});
