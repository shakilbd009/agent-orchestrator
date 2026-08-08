import { describe, it, expect } from 'vitest';
import { createReducer } from './event-feed';
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
});
