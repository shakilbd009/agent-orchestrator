// Magnetic Clusters — static topology (the backend-fed roster).
// Faithful port of AgentArks canon.ts: the AGENTS/EDGES/VIEWBOX schema, with
// BEATS deleted (replaced by the live SSE feed — see event-feed.ts).
// Reference: AgentArks/src/data/canon.ts (AGENTS, EDGES, VIEWBOX, types).
//
// The default roster is derived from this project's Layer A/B roles
// (AGENTS.md: architect/pm = Layer A; developer/reviewer/qa/devops = Layer B;
// captain = human gate; release = system). A future backend feed replaces
// DEFAULT_ROSTER at runtime; the engine + homes are roster-driven, not hardcoded.

export type Hue =
  | 'user' | 'intake' | 'architect' | 'builder'
  | 'security' | 'qa' | 'devops' | 'human' | 'prod';

export interface Agent {
  role: string;
  sub: string;
  hue: Hue;
  tools: string[];
  idle: string;
  instr?: string;
  layer: 'A' | 'B' | 'human' | 'system';
}

// A materialized knowledge-map card (artifact). Live cards are produced from
// handoff.submitted.artifacts[]; canon's CARDS table is not ported (no cassette).
export interface Card {
  cat: string;        // category badge
  hue: Hue;           // stroke hue
  agent: string;      // responsible agent id (anchor home)
  lines: string[];    // concise content
  meta: string;       // artifact metadata
}

export type EdgeKind = 'f' | 'rw' | 'w' | 'p'; // forward / rework / wait / prod

export interface Edge {
  f: string;      // from agent id
  t: string;      // to agent id
  label: string;
  kind: EdgeKind;
}

// Per-beat delta pushed by the live feed. `t`/`stage`/`cap`/`win` from canon's
// Beat are optional here (the feed supplies active/lit/rework/add; the engine
// derives caption/state from the active set). Reference: canon.ts Beat.
export interface Beat {
  active: string[];
  lit?: string[];
  rework?: string[];
  add?: string[];            // card ids materializing this beat (accumulate)
  win?: 'closed' | 'open';
}

export const VIEWBOX = { w: 1320, h: 820 };

// Default roster — the project's governed agent roles as a handoff graph.
// Flow: pm → architect → developer →(reviewer ∥ qa)→ devops → captain → release.
export const AGENTS: Record<string, Agent> = {
  pm:        { role: 'PM', sub: 'intake & planning', hue: 'intake', layer: 'A',
               tools: ['brd parser', 'kanban · read'], idle: 'idle',
               instr: 'Structure the request into reviewable requirements and gates.' },
  architect: { role: 'ARCHITECT', sub: 'solution design', hue: 'architect', layer: 'A',
               tools: ['code graph · read', 'spec writer'], idle: 'idle',
               instr: 'Design the change and the review boundary.' },
  developer: { role: 'DEVELOPER', sub: 'implementation', hue: 'builder', layer: 'B',
               tools: ['editor · src', 'unit runner · sandbox', 'repo branch'], idle: 'idle',
               instr: 'Implement the change per the design spec.' },
  reviewer:  { role: 'REVIEWER', sub: 'code review', hue: 'security', layer: 'B',
               tools: ['diff scan', 'policy gate'], idle: 'idle',
               instr: 'Review the diff for correctness and policy.' },
  qa:        { role: 'QA', sub: 'validation', hue: 'qa', layer: 'B',
               tools: ['test runner', 'coverage'], idle: 'idle',
               instr: 'Validate behavior against the acceptance criteria.' },
  devops:    { role: 'DEVOPS', sub: 'release & delivery', hue: 'devops', layer: 'B',
               tools: ['release planner · read', 'deploy · gated'], idle: 'idle',
               instr: 'Stage the release and hold for the window.' },
  captain:   { role: 'APPROVAL', sub: 'human-authorized gate', hue: 'human', layer: 'human',
               tools: [], idle: 'closed',
               instr: 'Human-authorized release gate.' },
  release:   { role: 'RELEASED', sub: 'your environment', hue: 'prod', layer: 'system',
               tools: [], idle: 'idle',
               instr: 'Change live in your environment.' },
};

export const EDGES: Record<string, Edge> = {
  'pm-arch':       { f: 'pm',        t: 'architect', label: 'scoped request',        kind: 'f' },
  'arch-dev':      { f: 'architect', t: 'developer', label: 'design spec',           kind: 'f' },
  'dev-rev':       { f: 'developer', t: 'reviewer',  label: 'diff artifact',         kind: 'f' },
  'dev-qa':        { f: 'developer', t: 'qa',        label: 'diff artifact',         kind: 'f' },
  'rev-rw-dev':    { f: 'reviewer',  t: 'developer', label: 'rework: review notes',  kind: 'rw' },
  'rev-devops':    { f: 'reviewer',  t: 'devops',    label: 'approved review',       kind: 'f' },
  'qa-devops':     { f: 'qa',        t: 'devops',    label: 'qa report',             kind: 'f' },
  'devops-cap':    { f: 'devops',    t: 'captain',   label: 'holds for approval',    kind: 'w' },
  'cap-release':   { f: 'captain',   t: 'release',   label: 'authorized release',    kind: 'p' },
};
