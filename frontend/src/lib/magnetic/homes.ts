// Magnetic Clusters — deterministic agent rest homes.
// Mirrors AgentArks canon REST_NODES (viewBox 1320×820): Layer A orchestrators
// on a left spine, Layer B specialists in the work field, human gate on the
// right, system/release at the far right — left-to-right request→production.
// Pure function of the roster so the graph is stable per project (an agent
// always rests at the same home). Reference: canon.ts REST_NODES.

import type { Agent } from './topology';
import { VIEWBOX } from './topology';

export interface Pt { x: number; y: number; }

type Layer = Agent['layer'];

const LAYER_ORDER: Layer[] = ['A', 'B', 'human', 'system'];

function byLayer(roster: Record<string, Agent>): Record<Layer, string[]> {
  const out: Record<Layer, string[]> = { A: [], B: [], human: [], system: [] };
  for (const id of Object.keys(roster)) out[roster[id].layer].push(id);
  for (const l of LAYER_ORDER) out[l].sort((a, b) => a.localeCompare(b));
  return out;
}

function spread(n: number, lo: number, hi: number): number[] {
  if (n <= 1) return [(lo + hi) / 2];
  const step = (hi - lo) / (n - 1);
  return Array.from({ length: n }, (_, i) => lo + step * i);
}

// Rest node positions for a roster. Deterministic: same roster → same layout.
export function restNodes(
  roster: Record<string, Agent>,
  vb: { w: number; h: number } = VIEWBOX
): Record<string, Pt> {
  const g = byLayer(roster);
  const out: Record<string, Pt> = {};

  // Layer A: left spine.
  const aYs = spread(g.A.length, vb.h * 0.20, vb.h * 0.80);
  g.A.forEach((id, i) => { out[id] = { x: vb.w * 0.09, y: aYs[i] }; });

  // Layer B: work field. ≤3 → one column at ~0.34w; else two columns (0.30 / 0.55).
  if (g.B.length <= 3) {
    const ys = spread(g.B.length, vb.h * 0.26, vb.h * 0.74);
    g.B.forEach((id, i) => { out[id] = { x: vb.w * 0.34, y: ys[i] }; });
  } else {
    const half = Math.ceil(g.B.length / 2);
    const left = g.B.slice(0, half);
    const right = g.B.slice(half);
    const lY = spread(left.length, vb.h * 0.26, vb.h * 0.74);
    const rY = spread(right.length, vb.h * 0.26, vb.h * 0.74);
    left.forEach((id, i) => { out[id] = { x: vb.w * 0.30, y: lY[i] }; });
    right.forEach((id, i) => { out[id] = { x: vb.w * 0.52, y: rY[i] }; });
  }

  // Human gate: right spine.
  g.human.forEach((id, i) => {
    out[id] = { x: vb.w * 0.74, y: vb.h * (0.50 + i * 0.10) };
  });

  // System / release: far right.
  g.system.forEach((id, i) => {
    out[id] = { x: vb.w * 0.91, y: vb.h * (0.45 + i * 0.10) };
  });

  return out;
}

// Card anchor offset from its responsible agent's home. Canon anchors each card
// near its agent (REST_CARDS ≈ agent + offset). Live cards stack per-agent so
// multiples don't pile on one point. Deterministic by per-agent card index.
const CARD_OFFSET_X = 55;
const CARD_OFFSET_Y = -32;
const CARD_STACK = 12; // px nudge per additional card on the same agent

export function cardHome(
  agentHome: Pt,
  agentCardCount: number
): Pt {
  const stack = agentCardCount * CARD_STACK;
  return {
    x: agentHome.x + CARD_OFFSET_X + stack,
    y: agentHome.y + CARD_OFFSET_Y + stack,
  };
}

// Mean position of a cluster's active agents' homes = that task's focal center
// (scout report §4.2: each active task gets its OWN centroid, so concurrent
// tasks form separate temporary clusters). Pure so the per-task centroid split
// is unit-testable without driving the physics loop. Returns null for an empty
// / all-unknown cluster (engine then falls back to the global centroid).
export function clusterCentroid(
  cluster: { agents: string[] },
  homes: Record<string, Pt>
): Pt | null {
  let x = 0, y = 0, n = 0;
  for (const a of cluster.agents) { const h = homes[a]; if (h) { x += h.x; y += h.y; n++; } }
  return n ? { x: x / n, y: y / n } : null;
}

// Deterministic 0..2π angle seeded from a string (task id). Never random — the
// graph must stay stable per project (same tasks → same offsets every frame).
function hashAngle(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = Math.imul(h, 16777619); }
  return ((h >>> 0) % 3600) / 3600 * Math.PI * 2;
}

// Breaks exact / near centroid coincidences so concurrent clusters with
// complementary agent splits don't collapse to one focal point (review M1:
// the 2×2 Layer-B layout makes {developer,reviewer} and {qa,devops} share the
// rectangle's center → separation 0 → one merged cluster). Each coinciding task
// is nudged by `radius` at a deterministic angle (hash of its id + even index
// spacing within the coincidence group), so two tasks always end up apart.
// Tasks that don't coincide with any other are returned unchanged.
export function separateCentroids(
  centroids: Record<string, Pt>,
  epsilon = 6,
  radius = 75
): Record<string, Pt> {
  const ids = Object.keys(centroids);
  if (ids.length < 2) return centroids;
  const out: Record<string, Pt> = {};
  const visited = new Set<string>();
  for (const id of ids) {
    const base = centroids[id];
    if (visited.has(id)) { out[id] = base; continue; }
    // coincidence group: every task within epsilon of this one
    const group = ids.filter((o) => Math.hypot(base.x - centroids[o].x, base.y - centroids[o].y) <= epsilon);
    group.forEach((g) => visited.add(g));
    if (group.length <= 1) { out[id] = base; continue; }
    group.forEach((taskId, i) => {
      const a = hashAngle(taskId) + (i / group.length) * Math.PI * 2;
      out[taskId] = { x: base.x + Math.cos(a) * radius, y: base.y + Math.sin(a) * radius };
    });
  }
  return out;
}
