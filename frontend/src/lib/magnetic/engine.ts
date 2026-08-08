// Magnetic Clusters — living-graph motion engine (faithful port).
// Ported verbatim from AgentArks/src/scripts/living-graph.ts: the bounded
// spring / repulsion / focal-attraction loop where motion emerges from changing
// graph relationships (active work pulls into a temporary cluster at the active
// centroid, then releases home). Motion is NOT driven by a scripted cassette.
//
// API change vs the reference (per scout report §3.4):
//   - `physics` reads a mutable `currentBeat` instead of `BEATS[cur]`.
//   - ONE beat entrypoint `pushBeat(b)` replaces the scripted `advance()` +
//     `setTimeout` autoplay. The `BEATS` import and the `start()/advance()`
//     timer path are deleted.
//   - `applyBeat`'s highlight/card logic is reused (lit/rework/now edge classes,
//     node on/wait, card fresh/opacity); `litEver` is now an accumulating set
//     instead of a BEATS scan.
//   - `renderStaticFinal()` renders the accumulated card map (the end-state
//     knowledge map) — kept for reduced-motion.
//   - Cards are mounted dynamically (`mountCard`) since live artifacts are not
//     known ahead of time (canon pre-rendered CARDS from a cassette).
//
// Physics constants and the per-frame physics(now)+render() are byte-faithful
// to the AgentArks source. Cite: living-graph.ts DAMP/VMAX/REST_*/REPEL_K/
// SPRING_K/FOCAL_K/BREATH_* and `function physics` / `function render`.

import { AGENTS, EDGES } from './topology';
import type { Agent, Beat, Hue } from './topology';
import { cardHome } from './homes';
import type { Pt } from './homes';

// physics constants (single tunable block) — verbatim from living-graph.ts
const DAMP = 0.86, VMAX = 6.8;
const REST_RK = 0.0050, REST_KA = 0.0018;
const REPEL_K = 2800, REPEL_MAX = 195;
const SPRING_K = 0.015;
const FOCAL_K = 0.0026;          // active-centroid attraction (magnetic cluster)
const BREATH_AMP = 3.6, BREATH_SPD = 0.0021;
const BOUND_PAD = 70;
const CW = 156, CH = 64;
const SVGNS = 'http://www.w3.org/2000/svg';

export interface CardSpec {
  id: string;
  agent: string;        // responsible agent id (anchor home)
  cat: string;          // category badge
  hue: Hue;
  lines: string[];      // content lines
  meta?: string;
}

export interface EngineHandle {
  pushBeat: (b: Partial<Beat>) => void;
  mountCard: (spec: CardSpec) => void;
  setReducedMotion: (on: boolean) => void;
  destroy: () => void;
}

const HUE_VAR: Record<Hue, string> = {
  user: '--h-user', intake: '--h-intake', architect: '--h-architect', builder: '--h-builder',
  security: '--h-security', qa: '--h-qa', devops: '--h-devops', human: '--h-human', prod: '--h-prod',
};

export function initLivingGraph(root: HTMLElement): EngineHandle {
  const svgEl = root.querySelector<SVGSVGElement>('svg.map');
  const detail = root.querySelector<HTMLElement>('[data-detail]');
  const capEl = root.querySelector<HTMLElement>('[data-caption]');
  const fill = root.querySelector<HTMLElement>('[data-fill]');
  const countEl = root.querySelector<HTMLElement>('[data-count]');
  const summaryEl = root.querySelector<HTMLElement>('[data-summary]');
  const playBtn = root.querySelector<HTMLButtonElement>('[data-action="play"]');
  const rmToggle = root.querySelector<HTMLInputElement>('[data-rm]');

  if (!svgEl) throw new Error('magnetic engine: no svg.map found');
  const svg: SVGSVGElement = svgEl;

  // gather live node/edge/label elements from the server-rendered scaffold.
  // (cards are mounted dynamically — none pre-rendered in the live model.)
  const nodeEls: Record<string, HTMLElement> = {};
  const cardEls: Record<string, SVGGElement> = {};
  const edgeEls: Record<string, SVGPathElement> = {};
  const edgeLab: Record<string, SVGTextElement> = {};
  root.querySelectorAll<HTMLElement>('[data-node]').forEach((n) => { nodeEls[n.getAttribute('data-node')!] = n; });
  root.querySelectorAll<SVGPathElement>('[data-edge]').forEach((p) => { edgeEls[p.getAttribute('data-edge')!] = p; });
  root.querySelectorAll<SVGTextElement>('[data-elabel]').forEach((t) => { edgeLab[t.getAttribute('data-elabel')!] = t; });

  // read rest homes from the server-rendered transforms (source of truth) —
  // verbatim from living-graph.ts node rest read.
  const rest: Record<string, Pt> = {};
  for (const id in nodeEls) {
    const tr = nodeEls[id].getAttribute('transform') || '';
    const m = tr.match(/translate\(([-\d.]+),([-\d.]+)\)/);
    rest[id] = m ? { x: +m[1], y: +m[2] } : { x: 0, y: 0 };
  }
  const restCard: Record<string, Pt> = {};

  const vbW = +(svgEl.getAttribute('viewBox')!.split(' ')[2] || 1320);
  const vbH = +(svgEl.getAttribute('viewBox')!.split(' ')[3] || 820);
  const center: Pt = { x: vbW / 2, y: vbH / 2 };

  // live node state — verbatim shape from living-graph.ts
  const nodes: Record<string, Pt & { vx: number; vy: number; fx: number; fy: number }> = {};
  for (const id in nodeEls) nodes[id] = { x: rest[id].x, y: rest[id].y, vx: 0, vy: 0, fx: 0, fy: 0 };
  const cards: Record<string, Pt & { vx: number; vy: number; agent: string; ox: number; oy: number; hx: number; hy: number }> = {};

  // mutable beat source (replaces BEATS[cur]) + accumulating maps
  let currentBeat: Beat = { active: [], lit: [], rework: [], add: [], win: 'closed' };
  const seen = new Set<string>();        // cards ever materialized (persist as knowledge map)
  const litEver = new Set<string>();     // edges ever lit (faded-but-present)
  const perAgentCards: Record<string, number> = {};
  let hoverLock: string | null = null;
  let playing = true;
  let reduced = false;
  let raf = 0;

  // ---- physics (verbatim from living-graph.ts `function physics`) ----
  function centroid(ids: string[]): Pt {
    let x = 0, y = 0, n = 0;
    for (const i of ids) { const h = rest[i]; if (h) { x += h.x; y += h.y; n++; } }
    return n ? { x: x / n, y: y / n } : { x: center.x, y: center.y };
  }
  function physics(now: number) {
    const beat = currentBeat;                                  // <-- was BEATS[cur]
    const active = new Set(beat.active || []);
    const litSet = new Set<string>([...(beat.lit || []), ...(beat.rework || [])]);
    const springSet = [...litSet];
    const cen = centroid(beat.active || []);
    const breath = Math.sin(now * BREATH_SPD) * BREATH_AMP;
    const ids = Object.keys(nodes);
    for (const id of ids) { nodes[id].fx = 0; nodes[id].fy = 0; }
    for (const id of ids) {
      const n = nodes[id], h = rest[id];
      const dirx = h.x - center.x, diry = h.y - center.y, dl = Math.hypot(dirx, diry) || 1;
      const hx = h.x + dirx / dl * breath, hy = h.y + diry / dl * breath;
      const rk = active.has(id) ? REST_KA : REST_RK;
      n.fx += (hx - n.x) * rk; n.fy += (hy - n.y) * rk;
      if (active.has(id)) { n.fx += (cen.x - n.x) * FOCAL_K; n.fy += (cen.y - n.y) * FOCAL_K; }
    }
    for (let i = 0; i < ids.length; i++) {
      for (let j = i + 1; j < ids.length; j++) {
        const a = nodes[ids[i]], b = nodes[ids[j]];
        let dx = a.x - b.x, dy = a.y - b.y; let d = Math.hypot(dx, dy);
        if (d < REPEL_MAX) { d = Math.max(d, 12); const f = REPEL_K / (d * d); dx /= d; dy /= d;
          a.fx += dx * f; a.fy += dy * f; b.fx -= dx * f; b.fy -= dy * f; }
      }
    }
    for (const id of springSet) {
      const e = EDGES[id]; const a = nodes[e.f], b = nodes[e.t]; if (!a || !b) continue;
      const ha = rest[e.f], hb = rest[e.t]; const on = litSet.has(id);
      const restLen = (Math.hypot(ha.x - hb.x, ha.y - hb.y) || 150) * (on ? 0.80 : 1.0);
      let dx = b.x - a.x, dy = b.y - a.y; const d = Math.max(Math.hypot(dx, dy), 1);
      const f = (d - restLen) * SPRING_K; dx /= d; dy /= d;
      a.fx += dx * f; a.fy += dy * f; b.fx -= dx * f; b.fy -= dy * f;
    }
    for (const id of ids) {
      const n = nodes[id];
      n.vx = (n.vx + n.fx) * DAMP; n.vy = (n.vy + n.fy) * DAMP;
      const sp = Math.hypot(n.vx, n.vy); if (sp > VMAX) { n.vx *= VMAX / sp; n.vy *= VMAX / sp; }
      n.x = Math.max(BOUND_PAD, Math.min(vbW - BOUND_PAD, n.x + n.vx));
      n.y = Math.max(BOUND_PAD, Math.min(vbH - BOUND_PAD, n.y + n.vy));
    }
    // cards ease toward agent+offset, then resolve overlaps so they spread
    const cardIds = Object.keys(cards);
    for (const id of cardIds) {
      if (!seen.has(id)) continue;
      const c = cards[id], a = nodes[c.agent]; if (!a) continue;
      const tx = a.x + c.ox, ty = a.y + c.oy;
      c.vx = (c.vx + (tx - c.x) * 0.035) * 0.5; c.vy = (c.vy + (ty - c.y) * 0.035) * 0.5;
      c.x += c.vx; c.y += c.vy;
    }
    for (let pass = 0; pass < 6; pass++) {
      for (let i = 0; i < cardIds.length; i++) {
        if (!seen.has(cardIds[i])) continue;
        for (let j = i + 1; j < cardIds.length; j++) {
          if (!seen.has(cardIds[j])) continue;
          const A = cards[cardIds[i]], B = cards[cardIds[j]];
          let dx = (A.x + 78) - (B.x + 78), dy = (A.y + 32) - (B.y + 32);
          let ox = 162 - Math.abs(dx), oy = 70 - Math.abs(dy);
          if (ox > 0 && oy > 0) {
            if (ox < oy) { const s = dx >= 0 ? ox : -ox; A.x += s; B.x -= s; }
            else { const s = dy >= 0 ? oy : -oy; A.y += s; B.y -= s; }
          }
        }
      }
    }
    for (const id of cardIds) {
      if (!seen.has(id)) continue;
      const c = cards[id];
      c.x = Math.max(8, Math.min(vbW - CW - 8, c.x)); c.y = Math.max(8, Math.min(vbH - CH - 8, c.y));
    }
  }

  // ---- render (verbatim from living-graph.ts `function render`) ----
  function render() {
    for (const id in nodeEls) nodeEls[id].setAttribute('transform', `translate(${nodes[id].x.toFixed(1)},${nodes[id].y.toFixed(1)})`);
    for (const id in EDGES) {
      const e = EDGES[id]; const a = nodes[e.f], b = nodes[e.t]; if (!a || !b) continue;
      const dx = b.x - a.x, dy = b.y - a.y, d = Math.hypot(dx, dy) || 1;
      const nx = -dy / d, ny = dx / d; const bow = 14;
      const mx = (a.x + b.x) / 2 + nx * bow, my = (a.y + b.y) / 2 + ny * bow;
      edgeEls[id]?.setAttribute('d', `M${a.x.toFixed(1)},${a.y.toFixed(1)} Q${mx.toFixed(1)},${my.toFixed(1)} ${b.x.toFixed(1)},${b.y.toFixed(1)}`);
      const lb = edgeLab[id];
      if (lb) { lb.setAttribute('x', mx.toFixed(1)); lb.setAttribute('y', (my - 4).toFixed(1)); }
    }
    for (const id in cardEls) cardEls[id].setAttribute('transform', `translate(${cards[id].x.toFixed(1)},${cards[id].y.toFixed(1)})`);
  }

  // ---- detail panel (adapted from living-graph.ts renderDetail) ----
  function renderDetail(active: string[]) {
    if (!detail) return;
    if (hoverLock) active = [hoverLock];
    const show = active.length > 3 ? [] : active;
    if (!show.length) {
      detail.classList.add('empty');
      detail.innerHTML = '<span>Agents settle at home; cards persist as a readable knowledge map of the work.</span>';
      return;
    }
    detail.classList.remove('empty');
    detail.innerHTML = show.map((id) => {
      const node = nodeEls[id];
      const role = node?.querySelector('.role')?.textContent || AGENTS[id]?.role || id;
      const sub = node?.querySelector('.sub')?.textContent || AGENTS[id]?.sub || '';
      const instr = AGENTS[id]?.instr || '';
      const tools = AGENTS[id]?.tools || [];
      const state = (currentBeat.win === 'open' && id === 'captain') ? 'open'
        : (currentBeat.active || []).includes(id) ? 'active' : 'idle';
      const roleVar = getComputedStyle(node!).getPropertyValue('--role') || `var(${HUE_VAR[AGENTS[id]?.hue || 'user']})`;
      return `<div class="who" style="--role:${roleVar}"><span class="name">${role}</span><span class="role-sub">${sub}</span></div>
        <div class="body"><span class="obj">${instr}</span><div class="chips">${tools.map((t) => `<span class="chip">${t}</span>`).join('')}<span class="chip state">${state}</span></div></div>`;
    }).join('');
  }

  // ---- beat application: highlights, cards, caption (from living-graph.ts applyBeat) ----
  function applyBeatClasses(b: Beat) {
    (b.add || []).forEach((c) => seen.add(c));
    (b.lit || []).forEach((e) => litEver.add(e));
    (b.rework || []).forEach((e) => litEver.add(e));
    const litNow = new Set(b.lit || []), rwNow = new Set(b.rework || []);
    for (const id in EDGES) {
      const p = edgeEls[id]; if (!p) continue;
      p.classList.remove('lit', 'now');
      p.removeAttribute('marker-end');
      if (rwNow.has(id) || litNow.has(id)) { p.classList.add('now'); p.setAttribute('marker-end', 'url(#arrow-now-stub)'); }
      else if (litEver.has(id)) { p.classList.add('lit'); p.setAttribute('marker-end', 'url(#arrow-stub)'); }
      edgeLab[id]?.classList.toggle('on', litNow.has(id) || rwNow.has(id));
    }
    for (const id in nodeEls) {
      const on = (b.active || []).includes(id);
      nodeEls[id].classList.toggle('on', on);
      nodeEls[id].classList.toggle('wait', id === 'devops' && b.win === 'closed' && on);
    }
    for (const id in cardEls) {
      const g = cardEls[id];
      g.style.opacity = seen.has(id) ? '1' : '0';
      g.classList.toggle('fresh', (b.add || []).includes(id));
    }
  }

  function renderCaption(b: Beat) {
    if (capEl) {
      const n = (b.active || []).length;
      const names = (b.active || []).slice(0, 4).map((id) => AGENTS[id]?.role || id).join(' · ');
      const kicker = n === 0 ? 'Settled' : n === 1 ? 'One active' : `${n} active`;
      const body = n === 0
        ? 'No active work; agents rest at home. Cards persist as a knowledge map.'
        : `Active cluster: ${names || '…'}.`;
      capEl.innerHTML = `<span class="k">${kicker}.</span> <span class="c">${body}</span>`;
    }
    if (countEl) countEl.textContent = `${(b.active || []).length} ACTIVE`;
    if (fill) {
      const total = Math.max(Object.keys(nodeEls).length, 1);
      fill.style.width = ((b.active || []).length / total * 100) + '%';
    }
  }

  // ---- dynamic card mounting (live artifacts, not a pre-rendered cassette) ----
  function mountCard(spec: CardSpec) {
    if (cardEls[spec.id]) return; // idempotent
    const ah = rest[spec.agent] || center;
    const idx = perAgentCards[spec.agent] = (perAgentCards[spec.agent] || 0) + 1;
    const home = cardHome(ah, idx - 1);
    restCard[spec.id] = home;

    const g = document.createElementNS(SVGNS, 'g');
    g.setAttribute('class', 'card');
    g.setAttribute('data-card', spec.id);
    g.setAttribute('data-hue', spec.hue);
    g.style.opacity = '0';
    const r = document.createElementNS(SVGNS, 'rect');
    r.setAttribute('width', String(CW)); r.setAttribute('height', String(CH)); r.setAttribute('rx', '10');
    const cat = document.createElementNS(SVGNS, 'text');
    cat.setAttribute('class', 'cat'); cat.setAttribute('x', '10'); cat.setAttribute('y', '18');
    cat.textContent = spec.cat;
    g.appendChild(r); g.appendChild(cat);
    spec.lines.slice(0, 3).forEach((ln, i) => {
      const t = document.createElementNS(SVGNS, 'text');
      t.setAttribute('class', 'ln'); t.setAttribute('x', '10'); t.setAttribute('y', String(36 + i * 15));
      t.textContent = ln.length > 26 ? ln.slice(0, 25) + '…' : ln;
      g.appendChild(t);
    });
    svg.appendChild(g);
    cardEls[spec.id] = g;
    cards[spec.id] = {
      x: home.x, y: home.y, vx: 0, vy: 0, agent: spec.agent,
      ox: home.x - ah.x, oy: home.y - ah.y, hx: home.x, hy: home.y,
    };
  }

  // ---- the beat entrypoint (replaces scripted advance()/setTimeout) ----
  function pushBeat(b: Partial<Beat>) {
    currentBeat = {
      active: b.active ?? currentBeat.active,
      lit: b.lit ?? currentBeat.lit,
      rework: b.rework ?? currentBeat.rework,
      add: b.add ?? currentBeat.add,
      win: b.win ?? currentBeat.win,
    };
    applyBeatClasses(currentBeat);
    if (!hoverLock) renderDetail(currentBeat.active || []);
    renderCaption(currentBeat);
    if (reduced) renderStatic();
  }

  // ---- loop + reduced-motion static map ----
  function loop(now: number) { physics(now); render(); if (playing && !reduced) raf = requestAnimationFrame(loop); }
  function renderStatic() {
    for (const id in nodes) { nodes[id].x = rest[id].x; nodes[id].y = rest[id].y; nodes[id].vx = 0; nodes[id].vy = 0; }
    for (const id in cards) { cards[id].x = cards[id].hx; cards[id].y = cards[id].hy; cards[id].vx = 0; cards[id].vy = 0; }
    render();
  }
  function renderStaticFinal() {
    // reduced-motion knowledge map: every materialized card at its home.
    for (const id in nodes) { nodes[id].x = rest[id].x; nodes[id].y = rest[id].y; nodes[id].vx = 0; nodes[id].vy = 0; }
    for (const id in cards) { cards[id].x = cards[id].hx; cards[id].y = cards[id].hy; cards[id].vx = 0; cards[id].vy = 0; }
    render();
    applyBeatClasses(currentBeat);
    if (summaryEl) summaryEl.hidden = false;
  }

  function setReducedMotion(on: boolean) {
    reduced = on;
    if (on) { if (raf) { cancelAnimationFrame(raf); raf = 0; } root.classList.add('reduced'); renderStaticFinal(); }
    else { root.classList.remove('reduced'); if (summaryEl) summaryEl.hidden = true; if (!raf) raf = requestAnimationFrame(loop); }
    if (playBtn) playBtn.textContent = on ? 'Play' : 'Pause';
  }

  // ---- controls (optional elements; guarded so a minimal scaffold still runs) ----
  playBtn?.addEventListener('click', () => {
    if (reduced) return;
    playing = !playing;
    if (playing && !raf) raf = requestAnimationFrame(loop);
    playBtn.textContent = playing ? 'Pause' : 'Play';
  });
  rmToggle?.addEventListener('change', (e) => setReducedMotion((e.target as HTMLInputElement).checked));
  root.querySelectorAll<HTMLElement>('[data-node]').forEach((n) => {
    const id = n.getAttribute('data-node')!;
    n.addEventListener('mouseenter', () => { hoverLock = id; renderDetail(currentBeat.active || []); });
    n.addEventListener('mouseleave', () => { hoverLock = null; renderDetail(currentBeat.active || []); });
    n.addEventListener('focus', () => { hoverLock = id; renderDetail(currentBeat.active || []); });
    n.addEventListener('blur', () => { hoverLock = null; renderDetail(currentBeat.active || []); });
  });

  const onVis = () => {
    if (document.hidden) { if (raf) { cancelAnimationFrame(raf); raf = 0; } }
    else if (playing && !reduced && !raf) raf = requestAnimationFrame(loop);
  };
  document.addEventListener('visibilitychange', onVis);

  function destroy() {
    if (raf) { cancelAnimationFrame(raf); raf = 0; }
    document.removeEventListener('visibilitychange', onVis);
  }

  // reduced-motion by OS preference; else start the live loop immediately
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
    setReducedMotion(true);
    if (rmToggle) rmToggle.checked = true;
  } else {
    raf = requestAnimationFrame(loop);
  }

  return { pushBeat, mountCard, setReducedMotion, destroy };
}
