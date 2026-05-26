# BRD-06: Agent Memory and State

**Project:** agent-orchestrator  
**Domain:** Backend  
**Phase:** 1  
**Owner:** developer  
**Status:** placeholder  

---

## Goal

Enable agents to retain context across sessions so that task handoffs, decisions, and learned conventions persist without requiring full re-explanation.

---

## Problem Statement

Agents that cannot remember prior context force every new session to re-establish facts, conventions, and decisions. This increases latency, cost, and error rates.

---

## Scope

### In Scope
- Persistent memory store for agent profile context
- Task-level memory for handoff summaries
- Project-level memory for conventions and decisions

### Out of Scope
- Long-term memory for user personal data beyond project context
- Vector store or semantic search capabilities

---

## Dependencies

- BRD-01 (App Shell)
- Feature flag: `agent-memory`

---

*Placeholder. Full BRD to be authored in Phase 1.*
