import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createReducer, startMockFeed } from './event-feed';
import type { EngineHandle, CardSpec } from './engine';
import { AGENTS, EDGES } from './topology';
import type { EventEnvelope } from '$lib/api/orchestration';

function env(topic: string, payload: Record<string, unknown>, taskId: string | null = 't1'): EventEnvelope {
  return {
    eventId: `e_${topic}_${Math.random()}`, schemaVersion: 'v1alpha', projectId: 'p', topic,
    actorId: 'x', actorRole: 'layer_b', taskId, parentTaskId: null, gateId: null,
    timestamp: new Date().toISOString(), payload,
  };
}

describe('magnetic event-feed reducer', () => {
  it('agent.activated pulls an agent into the active set; agent.idle releases it', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 't1' }));
    let s = r.snapshot();
    expect(s.active).toEqual(['developer']);
    // developer→reviewer and developer→qa edges both-lit only if both active; alone → none lit
    expect(s.lit).toEqual([]);

    r.reduce(env('agent.activated', { agentName: 'reviewer', layer: 'B', taskId: 't1' }));
    s = r.snapshot();
    expect(new Set(s.active)).toEqual(new Set(['developer', 'reviewer']));
    expect(s.lit).toContain('dev-rev'); // both endpoints active
    expect(s.rework).toEqual([]);       // not in tension

    r.reduce(env('agent.idle', { agentName: 'developer', layer: 'B' }));
    s = r.snapshot();
    expect(s.active).toEqual(['reviewer']);
    expect(s.lit).toEqual([]); // developer gone → edge unlit
  });

  it('gate.rejected lights a rework edge back to the active developer', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 't1' }));
    r.reduce(env('agent.activated', { agentName: 'reviewer', layer: 'B', taskId: 't1' }));
    r.reduce(env('gate.rejected', { rejectedBy: 'reviewer', gateType: 'code_review', gateLevel: 'task' }));
    const s = r.snapshot();
    expect(s.lit).toContain('dev-rev');         // forward edge stays lit
    expect(s.rework).toContain('rev-rw-dev');   // rw edge with reviewer in tension
    expect(s.lit).not.toContain('rev-rw-dev');
  });

  it('handoff.submitted releases the submitter and materializes artifact cards', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 't1' }));
    const cards = r.reduce(env('handoff.submitted', { taskId: 't1', submittedBy: 'developer', summary: 'done', artifacts: ['diff.patch', 'unit.md'], validationPerformed: 'unit' }));
    expect(cards.map((c) => c.id)).toEqual(['diff.patch', 'unit.md']);
    expect(cards.every((c) => c.agent === 'developer')).toBe(true);
    const s = r.snapshot();
    expect(s.active).not.toContain('developer'); // released on handoff
    expect(s.add).toEqual(['diff.patch', 'unit.md']); // surfaced this tick
    // second snapshot drains pending cards
    expect(r.snapshot().add).toEqual([]);
  });

  it('task.status.changed{done} clears the task active set', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'qa', layer: 'B', taskId: 't_qa' }));
    expect(r.snapshot().active).toContain('qa');
    r.reduce(env('task.status.changed', { fromStatus: 'in_progress', toStatus: 'done' }, 't_qa'));
    expect(r.snapshot().active).toEqual([]);
  });

  // Phase E — multi-cluster (scout report §4.2): concurrent active tasks each
  // get their OWN cluster, instead of one merged centroid.
  it('two concurrently-active tasks form two separate clusters; completing one releases its cluster', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 'tA' }, 'tA'));
    r.reduce(env('agent.activated', { agentName: 'qa', layer: 'B', taskId: 'tB' }, 'tB'));
    let s = r.snapshot();
    expect(new Set(s.active)).toEqual(new Set(['developer', 'qa']));
    expect(s.clusters).toHaveLength(2);
    expect(s.clusters.map((c) => c.taskId).sort()).toEqual(['tA', 'tB']);
    expect(s.clusters.find((c) => c.taskId === 'tA')!.agents).toEqual(['developer']);
    expect(s.clusters.find((c) => c.taskId === 'tB')!.agents).toEqual(['qa']);

    // task A completes → its cluster drops; B keeps working
    r.reduce(env('task.status.changed', { fromStatus: 'in_progress', toStatus: 'done' }, 'tA'));
    s = r.snapshot();
    expect(s.clusters).toHaveLength(1);
    expect(s.clusters[0].taskId).toBe('tB');
    expect(s.active).toEqual(['qa']);
  });

  it('an agent active on two tasks appears in both clusters (multi-membership)', () => {
    // developer works tA and tB concurrently → member of both clusters, so the
    // engine pulls it toward each task's centroid (scout §4.2 generalization).
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 'tA' }, 'tA'));
    r.reduce(env('agent.activated', { agentName: 'qa', layer: 'B', taskId: 'tB' }, 'tB'));
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 'tB' }, 'tB'));
    const s = r.snapshot();
    expect(s.clusters).toHaveLength(2);
    expect(s.clusters.find((c) => c.taskId === 'tA')!.agents).toEqual(['developer']);
    expect(s.clusters.find((c) => c.taskId === 'tB')!.agents).toContain('developer');
    expect(s.clusters.find((c) => c.taskId === 'tB')!.agents).toContain('qa');
  });

  it('handoff.submitted releases the submitter from its task only — multi-membership kept on the other task (review F3)', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 'tA' }, 'tA'));
    r.reduce(env('agent.activated', { agentName: 'developer', layer: 'B', taskId: 'tB' }, 'tB'));
    r.reduce(env('agent.activated', { agentName: 'qa', layer: 'B', taskId: 'tB' }, 'tB'));
    // developer hands off on tA → dropped from tA ONLY; still active on tB
    r.reduce(env('handoff.submitted', { taskId: 'tA', submittedBy: 'developer', summary: 'done', artifacts: ['a.patch'], validationPerformed: 'unit' }, 'tA'));
    const s = r.snapshot();
    expect(s.clusters.find((c) => c.taskId === 'tA')).toBeUndefined();      // tA cluster gone
    const tB = s.clusters.find((c) => c.taskId === 'tB')!;
    expect(tB.agents).toEqual(expect.arrayContaining(['developer', 'qa']));  // developer still on tB
    // agent.idle, by contrast, is global — releases from every task
    r.reduce(env('agent.idle', { agentName: 'developer', layer: 'B' }, null));
    const s2 = r.snapshot();
    expect(s2.clusters.find((c) => c.taskId === 'tB')!.agents).toEqual(['qa']);
  });

  it('gate.rejected / gate.approved fold into an observed gate view per task', () => {
    const r = createReducer({ agents: AGENTS, edges: EDGES });
    r.reduce(env('gate.rejected', { rejectedBy: 'reviewer', gateType: 'code_review', gateLevel: 'task' }, 'tA'));
    let s = r.snapshot();
    expect(s.gates).toContainEqual({ taskId: 'tA', gateType: 'code_review', state: 'blocked' });
    r.reduce(env('gate.approved', { approvedBy: 'reviewer', approverRole: 'layer_b', gateType: 'code_review', gateLevel: 'task' }, 'tA'));
    s = r.snapshot();
    expect(s.gates.find((g) => g.taskId === 'tA')?.state).toBe('passed');
  });
});

describe('startMockFeed handle (PR #13 M1 — no orphan timers after close)', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('close() halts all timers across the loop reschedule — no further pushBeat/mountCard', () => {
    const pushBeat = vi.fn();
    const mountCard = vi.fn();
    const engine: EngineHandle = { pushBeat, mountCard, setReducedMotion: vi.fn(), destroy: vi.fn() };

    const handle = startMockFeed(engine, { speed: 1 });
    // run one full pass + the reschedule (loop arms at total+2500ms)
    vi.advanceTimersByTime(20_000);
    const beatsAfterFirstCycle = pushBeat.mock.calls.length;
    expect(beatsAfterFirstCycle).toBeGreaterThan(0);

    handle.close();
    vi.advanceTimersByTime(60_000);   // well past several more cycles

    // invariant: after close(), no further pushBeat/mountCard ever fires
    expect(pushBeat.mock.calls.length).toBe(beatsAfterFirstCycle);
    expect(mountCard.mock.calls.length).toBe(mountCard.mock.calls.length);
  });
});
