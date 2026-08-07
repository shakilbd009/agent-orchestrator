# Curated BRDs

**Project:** agent-orchestrator  
**Phase:** 2 — Core Delivery (Phase 0 + Phase 1 complete; BRD-02/03 shipped)  

---

This directory contains the canonical BRD artifacts for the Agent Orchestrator Platform, organized by domain. Each BRD is a living document that progresses from `draft` to `approved` through the governance pipeline defined in AGENTS.md.

---

## BRD Index

| BRD | Domain | Phase | Status | Title |
|-----|--------|-------|--------|-------|
| BRD-01 | App Shell | Phase 1 | implemented | Application Shell and Minimal Scaffold |
| BRD-02 | Orchestration | Phase 2 | implemented | Platform-Native Orchestration Pipeline |
| BRD-03 | UI | Phase 2 | implemented | Client Portal and Business Project Board |
| BRD-04 | UI | Phase 1 | placeholder | Agent Workstream Dashboard |
| BRD-05 | Backend | Phase 1 | placeholder | LLM Provider Integration |
| BRD-06 | Backend | Phase 1 | placeholder | Agent Memory and State |
| BRD-07 | Workflow | Phase 1 | placeholder | BRD Authoring Workflow |
| BRD-08 | Quality | Phase 1 | placeholder | Quality Gates |
| BRD-09 | Quality | Phase 1 | placeholder | Code Review |
| BRD-10 | Quality | Phase 1 | placeholder | Security Review |
| BRD-11 | Quality | Phase 1 | placeholder | QA Automation |
| BRD-12 | DevOps | Phase 1 | placeholder | Deployment Pipeline |
| BRD-13 | DevOps | Phase 1 | placeholder | Docker Development Environment |
| BRD-14 | QA | Phase 1 | placeholder | Playwright Acceptance Testing |
| BRD-15 | Governance | Phase 1 | placeholder | Scope Control |
| BRD-16 | Governance | Phase 1 | placeholder | Risk Tracking |
| BRD-17 | Backend | Phase 2 | placeholder | Multi-Tenant Isolation |
| BRD-18 | Workflow | Phase 2 | placeholder | Collaboration and Team Management |
| BRD-19 | Workflow | Phase 2 | placeholder | Decision and Artifact History |
| BRD-20 | Operations | Phase 2 | placeholder | Post-Deployment Operations |
| BRD-21 | UX | Phase 2 | placeholder | Notifications and Alerts |

---

## Domain Mapping

| Domain | BRDs |
|--------|------|
| Orchestration | BRD-01, BRD-02 |
| UI | BRD-03, BRD-04 |
| Backend | BRD-05, BRD-06, BRD-17 |
| Workflow | BRD-07, BRD-18, BRD-19 |
| Quality | BRD-08, BRD-09, BRD-10, BRD-11 |
| DevOps | BRD-12, BRD-13, BRD-14 |
| Governance | BRD-15, BRD-16 |
| Operations | BRD-20 |
| UX | BRD-21 |

---

## Phase Status

Phase 0 (governance) and Phase 1 (app shell, BRD-01) are complete. Phase 2 is in progress: BRD-02 (Platform-Native Orchestration Pipeline) and BRD-03 (Client Portal) are implemented behind feature flags. BRD-04+ remain placeholders — not yet built (agent-execution harness, LLM provider, agent memory still unbuilt).

Authored BRDs: `BRD-01-app-shell.md`, `BRD-02-orchestration-pipeline.md`, `BRD-03-client-portal.md`. The remaining entries below are placeholders.

The template for all BRDs is at `specs/_template.md`. The feature flag registry is at `specs/feature-flags.md`.
