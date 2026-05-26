# check-brd-open-items — Architecture Fitness Function

**Project:** agent-orchestrator
**Owner:** qa

---

## What It Checks

This fitness function enforces that every open item, risk, decision, or
deferred feature listed in the BRD (`agent-orchestrator.md`) has a corresponding
tracking artifact (an ADR, a kanban task, or an explicit deferral status in the
BRD itself).

**Rule:** Every open item in the BRD must have one of:
1. A tracking Kanban task (visible in the project board)
2. A corresponding ADR in `docs/adr/`
3. An explicit `Status: deferred` (or `Status: open`) in the BRD line item

If an open item has none of the above, the script reports it as untracked.

**Rationale:** A BRD that lists open items with no follow-up is a zombie
document — it creates the appearance of governance without the substance.
This check ensures that every stated risk and deferred decision has an
owner and a path to resolution.

---

## Phase 0 Scope

Phase 0 is the planning phase. Many items are intentionally deferred to
future phases. This script:

1. Reads `agent-orchestrator.md` and extracts all items marked as deferred,
   open, or risk-flagged.
2. Checks for a corresponding tracking artifact (ADR or kanban task).
3. If no artifact is found, reports the item as untracked.
4. Items explicitly marked `Status: deferred` in the BRD with a future-phase
   reference are considered tracked.

---

## BRD Items to Track

The following are known open items from the BRD. Items marked `// BRD-BYPASS:`
in the BRD itself are skipped by this check.

From `agent-orchestrator.md`:

| Item | Deferral | Tracking Needed |
|------|----------|----------------|
| LLM provider abstraction | BRD-05 | Kanban task or ADR |
| Agent memory store | BRD-06 | Kanban task or ADR |
| Audit trail implementation | BRD-19 | Kanban task or ADR |
| Multi-tenant isolation | BRD-20 | Kanban task or ADR |
| Notification system | BRD-21 | Kanban task or ADR |
| (all other BRD-08 through BRD-18) | respective BRD | Kanban task |

---

## Failure Output

```
agent-orchestrator.md:142: RULE: brd-open-item-untracked
  ITEM: "LLM inference event schemas"
  DEFERRED TO: BRD-05
  REMEDIATION: Create a kanban task or ADR for BRD-05 before proceeding to Phase 2
```

---

## Inline Bypass

The `// BRD-BYPASS:` comment suppresses the open-item flag for that specific
item only. It must appear on the same line as the item in the BRD, or on the
line immediately following it.

Example in BRD:
```
| LLM inference event schemas | BRD-05 (LLM Provider) | LLM provider abstraction not finalised | // BRD-BYPASS: blocked on vendor selection
```

---

## References

- `agent-orchestrator.md` — Business Requirement Document
- `docs/adr/0001-record-architecture-decisions.md` — Decision record
- `contracts/events.md` — Event contract placeholders
