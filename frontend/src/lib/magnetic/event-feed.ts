// Magnetic Clusters — event-feed (the live "mic" replacing the cassette).
// Subscribes to the repaired SSE client (PR #9: createSSEConnection, path
// /projects/:projectId/events/stream, per-frame parsed named events) and reduces
// agent/handoff/gate/task events into a live active set, then pushes a
// Beat-shaped delta {active, lit, rework, add} into the engine via pushBeat.
//
// Reduction rules (scout report §3.3, active-set §4.3):
//   agent.activated{taskId}        → add agent to activeByTask[taskId]
//   agent.idle / agent.blocked     → remove agent from active; blocked → tension
//   handoff.submitted{artifacts[]} → remove submitter; each artifact → a Card
//   task.status.changed{done/cancelled} → clear that task's active set
//   gate.rejected{rejectedBy}      → reviewer tension (rework edges light)
//   gate.approved                  → clear tension
//   task.decomposition.approved    → informational (children activate on their
//                                     own agent.activated events)
//
// `lit[]`: edges whose both endpoints are currently active (handoff in flight).
// `rework[]`: rw-kind edges whose reviewer (f) is in tension and target is active.
// `add[]`: card ids materialized since the last tick (handoff artifacts).
//
// The reducer is a pure fold over the ordered event log, so any client can
// reconstruct the live graph from cold start (the backend persists audit_events
// and supports Last-Event-ID replay — scout report §1.3).

import { createSSEConnection } from '$lib/api/client';
import type { SSEEventHandler } from '$lib/api/client';
import type { EventEnvelope } from '$lib/api/orchestration';
import { AGENTS as DEFAULT_AGENTS, EDGES as DEFAULT_EDGES } from './topology';
import type { Agent, Edge, Hue } from './topology';
import type { EngineHandle, CardSpec } from './engine';

export interface ReducerRoster { agents: Record<string, Agent>; edges: Record<string, Edge>; }

export interface BeatDelta {
  active: string[];
  lit: string[];
  rework: string[];
  add: string[];            // card ids materialized since last snapshot
}

export interface FeedReducer {
  reduce(env: EventEnvelope): CardSpec[];   // fold one event; returns new cards to mount
  snapshot(): BeatDelta;                     // derive active/lit/rework/add; drains pending cards
}

export function createReducer(roster: ReducerRoster = { agents: DEFAULT_AGENTS, edges: DEFAULT_EDGES }): FeedReducer {
  const activeByTask = new Map<string, Set<string>>();
  const tension = new Set<string>();        // reviewers that recently rejected/blocked
  const mounted = new Set<string>();        // card ids already materialized
  let pendingCards: CardSpec[] = [];

  function addActive(taskId: string | null, agent: string) {
    const t = taskId || '_';
    let s = activeByTask.get(t);
    if (!s) { s = new Set(); activeByTask.set(t, s); }
    s.add(agent);
  }
  function removeAgent(agent: string) {
    for (const s of activeByTask.values()) s.delete(agent);
    // drop empty task sets so done tasks release cleanly
    for (const [t, s] of activeByTask) if (s.size === 0 && t !== '_') activeByTask.delete(t);
  }
  function clearTask(taskId: string | null) {
    if (taskId) activeByTask.delete(taskId);
  }

  function reduce(env: EventEnvelope): CardSpec[] {
    const p = env.payload || {};
    const newCards: CardSpec[] = [];
    switch (env.topic) {
      case 'agent.activated': {
        const agent = String(p.agentName ?? '');
        if (agent) { addActive((p.taskId as string) ?? env.taskId, agent); tension.delete(agent); }
        break;
      }
      case 'agent.idle': {
        const agent = String(p.agentName ?? '');
        if (agent) { removeAgent(agent); tension.delete(agent); }
        break;
      }
      case 'agent.blocked': {
        const agent = String(p.agentName ?? '');
        if (agent) { removeAgent(agent); tension.add(agent); }
        break;
      }
      case 'handoff.submitted': {
        const agent = String(p.submittedBy ?? '');
        if (agent) removeAgent(agent);
        const artifacts = Array.isArray(p.artifacts) ? (p.artifacts as string[]) : [];
        artifacts.forEach((a, i) => {
          const id = mounted.has(a) ? `${a}#${i}` : a;
          if (mounted.has(id)) return;
          mounted.add(id);
          const hue: Hue = (DEFAULT_AGENTS[agent]?.hue ?? 'builder') as Hue;
          const spec: CardSpec = { id, agent, cat: 'ARTIFACT', hue, lines: [a] };
          pendingCards.push(spec);
          newCards.push(spec);
        });
        break;
      }
      case 'task.status.changed': {
        if (p.toStatus === 'done' || p.toStatus === 'cancelled') clearTask(env.taskId);
        break;
      }
      case 'gate.rejected': {
        const by = String(p.rejectedBy ?? '');
        if (by) tension.add(by);
        break;
      }
      case 'gate.approved': {
        const by = String(p.approvedBy ?? '');
        if (by) tension.delete(by);
        break;
      }
      // task.decomposition.approved: children activate via their own agent.activated.
      default: break;
    }
    return newCards;
  }

  function snapshot(): BeatDelta {
    const active = new Set<string>();
    for (const s of activeByTask.values()) for (const a of s) active.add(a);

    const lit: string[] = [];
    const rework: string[] = [];
    for (const [id, e] of Object.entries(roster.edges)) {
      if (active.has(e.f) && active.has(e.t)) {
        if (e.kind === 'rw' && tension.has(e.f)) rework.push(id);
        else lit.push(id);
      }
    }
    const add = pendingCards.map((c) => c.id);
    pendingCards = [];
    return { active: [...active], lit, rework, add };
  }

  return { reduce, snapshot };
}

// Drain the reducer into the engine on a rAF tick (debounced per scout §3.3 #5).
function wireReducer(
  reducer: FeedReducer,
  engine: EngineHandle,
  onCard?: (c: CardSpec) => void
): (env: EventEnvelope) => void {
  let scheduled = false;
  let queued: EventEnvelope[] = [];
  return (env: EventEnvelope) => {
    queued.push(env);
    if (scheduled) return;
    scheduled = true;
    requestAnimationFrame(() => {
      scheduled = false;
      const batch = queued; queued = [];
      for (const e of batch) for (const c of reducer.reduce(e)) { engine.mountCard(c); onCard?.(c); }
      engine.pushBeat(reducer.snapshot());
    });
  };
}

export interface FeedHandle { close: () => void; }

// Live SSE feed. Reuses the repaired createSSEConnection (PR #9).
export function startEventFeed(
  projectId: string,
  engine: EngineHandle,
  opts: { roster?: ReducerRoster; onError?: (e: Error) => void; onConnect?: () => void } = {}
): FeedHandle {
  const reducer = createReducer(opts.roster);
  const handler: SSEEventHandler = wireReducer(reducer, engine);
  const conn = createSSEConnection({ projectId, onEvent: handler, onError: opts.onError, onConnect: opts.onConnect });
  return { close: conn.close };
}

// Mock feed: replays a scripted event sequence through the SAME reducer+wiring,
// proving the graph reacts to event-shaped data without a running backend.
// (Scout report §9 / task milestone: drive from a fixture if backend unreachable.)
export interface MockTimelineStep { delay: number; env: EventEnvelope; }

export function startMockFeed(
  engine: EngineHandle,
  opts: { roster?: ReducerRoster; speed?: number; onStep?: (env: EventEnvelope) => void } = {}
): FeedHandle {
  const reducer = createReducer(opts.roster);
  const handler = wireReducer(reducer, engine);
  const speed = opts.speed ?? 1;
  const timers: ReturnType<typeof setTimeout>[] = [];
  let stopped = false;   // the handle owns this; close() is the only thing that flips it

  // Schedule one MOCK_TIMELINE pass, then re-arm in place. A single long-lived
  // handle (no recursion) so close() can reach every timer. Every scheduled
  // callback AND the reschedule gate on `stopped` — after close(), no further
  // pushBeat/mountCard ever fires (PR #13 review M1).
  const schedulePass = () => {
    let t = 0;
    for (const step of MOCK_TIMELINE) {
      t += step.delay;
      timers.push(setTimeout(() => {
        if (stopped) return;
        opts.onStep?.(step.env);
        handler(step.env);
      }, t / speed));
    }
    timers.push(setTimeout(() => {
      if (stopped) return;
      schedulePass();
    }, (t + 2500) / speed));
  };
  schedulePass();
  return {
    close() { stopped = true; for (const id of timers) clearTimeout(id); },
  };
}

// A representative request → release flow expressed as canonical envelopes.
// Mirrors the orchestrator pipeline (pm → architect → developer → reviewer∥qa →
// devops → captain → release), with a reviewer rework loop.
export const MOCK_TIMELINE: MockTimelineStep[] = [
  { delay: 600, env: mock('agent.activated', 'layer_a', 'pm', 't_req', { agentName: 'pm', layer: 'A', taskId: 't_req' }) },
  { delay: 900, env: mock('agent.activated', 'layer_a', 'architect', 't_req', { agentName: 'architect', layer: 'A', taskId: 't_req' }) },
  { delay: 800, env: mock('agent.idle', 'layer_a', 'pm', null, { agentName: 'pm', layer: 'A', idleSince: null }) },
  { delay: 700, env: mock('agent.activated', 'layer_b', 'developer', 't_req', { agentName: 'developer', layer: 'B', taskId: 't_req' }) },
  { delay: 900, env: mock('agent.idle', 'layer_a', 'architect', null, { agentName: 'architect', layer: 'A', idleSince: null }) },
  { delay: 600, env: mock('agent.activated', 'layer_b', 'reviewer', 't_rev', { agentName: 'reviewer', layer: 'B', taskId: 't_rev' }) },
  { delay: 200, env: mock('agent.activated', 'layer_b', 'qa', 't_qa', { agentName: 'qa', layer: 'B', taskId: 't_qa' }) },
  { delay: 1100, env: mock('gate.rejected', 'layer_b', 'reviewer', 't_rev', { rejectedBy: 'reviewer', gateType: 'code_review', gateLevel: 'task' }) },
  { delay: 300, env: mock('agent.activated', 'layer_b', 'developer', 't_req', { agentName: 'developer', layer: 'B', taskId: 't_req' }) },
  { delay: 900, env: mock('handoff.submitted', 'layer_b', 'developer', 't_req', { taskId: 't_req', submittedBy: 'developer', summary: 'revised diff', artifacts: ['diff.patch', 'unit results'], validationPerformed: 'unit suite' }) },
  { delay: 400, env: mock('agent.activated', 'layer_b', 'reviewer', 't_rev2', { agentName: 'reviewer', layer: 'B', taskId: 't_rev2' }) },
  { delay: 900, env: mock('gate.approved', 'human', 'captain', 't_rev2', { approvedBy: 'reviewer', approverRole: 'layer_b', gateType: 'code_review', gateLevel: 'task' }) },
  { delay: 200, env: mock('agent.idle', 'layer_b', 'reviewer', null, { agentName: 'reviewer', layer: 'B', idleSince: null }) },
  { delay: 400, env: mock('handoff.submitted', 'layer_b', 'qa', 't_qa', { taskId: 't_qa', submittedBy: 'qa', summary: 'qa green', artifacts: ['qa-report.md'], validationPerformed: 'acceptance suite' }) },
  { delay: 600, env: mock('agent.activated', 'layer_b', 'devops', 't_rel', { agentName: 'devops', layer: 'B', taskId: 't_rel' }) },
  { delay: 800, env: mock('agent.activated', 'human', 'captain', 't_rel', { agentName: 'captain', layer: 'human', taskId: 't_rel' }) },
  { delay: 1000, env: mock('handoff.submitted', 'layer_b', 'devops', 't_rel', { taskId: 't_rel', submittedBy: 'devops', summary: 'release staged', artifacts: ['release-plan.md'], validationPerformed: 'checklist' }) },
  { delay: 300, env: mock('agent.activated', 'system', 'release', 't_rel', { agentName: 'release', layer: 'system', taskId: 't_rel' }) },
  { delay: 900, env: mock('agent.idle', 'human', 'captain', null, { agentName: 'captain', layer: 'human', idleSince: null }) },
  { delay: 600, env: mock('agent.idle', 'system', 'release', null, { agentName: 'release', layer: 'system', idleSince: null }) },
];

function mock(
  topic: string,
  actorRole: EventEnvelope['actorRole'],
  actorId: string,
  taskId: string | null,
  payload: Record<string, unknown>,
  i = 0
): EventEnvelope {
  return {
    eventId: `mock_${topic}_${actorId}_${i}`,
    schemaVersion: 'v1alpha',
    projectId: 'mock',
    topic,
    actorId,
    actorRole,
    taskId,
    parentTaskId: null,
    gateId: null,
    timestamp: new Date().toISOString(),
    payload,
  };
}
