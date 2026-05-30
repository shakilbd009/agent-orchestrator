# ADR-02-003: Webhook Signing Requirement

**ADR:** ADR-02-003  
**Subject:** Mandatory webhook signature for outbound event delivery  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

FR-02-031 says webhook signing "should" be supported "if implementation cost is low." Open Question 3 asks whether signing should be mandatory. Unsigned webhooks are a vector for injection attacks: an attacker who can send traffic to the webhook consumer URL can inject arbitrary payloads without detection.

The current "should-have" posture is underspecified and leaves the security posture ambiguous. An internal service-to-service webhook delivering task events should be protected just as an external one would be.

---

## Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | Mandatory for all registered webhooks | Strongest security; one policy | Key distribution needed; friction for first internal consumers |
| B | Mandatory for external URLs; localhost exempt | Proportionate; internal dev friction managed | Exemption handling in code; need `X-Forwarded-Host` validation |
| C | Should-have (current BRD text) | Zero immediate friction | Security posture ambiguous; OQ-3 remains unresolved |

---

## Decision

**Option B** is selected with extensions: signing is mandatory for all webhook registrations EXCEPT those where the consumer URL resolves to `localhost` or `127.0.0.1`.

**Specific rules:**
1. All webhook registrations must provide a `X-Webhook-Secret` header at registration time (sent once plaintext; stored bcrypt-hashed).
2. Every outbound webhook delivery includes `X-Webhook-Signature: HMAC-SHA256(<raw_body>, <secret_hash>)` where `<secret_hash>` is the bcrypt-hashed secret associated with that registration.
3. The signature is computed over the raw request body bytes, NOT a JSON-serialized string (prevents ambiguity from JSON encoding differences).
4. Consumer URL validation: URLs must have a valid scheme (`http` or `https`). `http` is permitted only for `localhost` and `127.0.0.1`. All non-localhost URLs must use `https`.
5. On signature mismatch or missing signature, the webhook delivery is treated as a failed delivery, retried with backoff, and ultimately exhausted after 3 attempts.
6. Secrets are stored hashed; the plaintext is returned once at registration time and never stored.

**Localhost exemption rationale:** Localhost-delivered webhooks cannot be received by external parties in normal network topology. The exemption is for development convenience only. Production deployments on Docker Compose with `http://host.docker.internal:PORT` still require signing unless explicitly allow-listed.

---

## Consequences

**Positive:**
- Webhook delivery is authenticated; injection attacks are detectable
- Security posture is explicit and testable
- One clear policy, no ambiguous "should" for consumers

**Negative:**
- Webhook registration API must accept and store secrets
- Secret rotation requires re-registration
- First-look development setup friction slightly higher

**Neutral:**
- `X-Webhook-Signature` header is a standard pattern (GitHub, Stripe, etc.)
- Signature verification should be a library function, not custom code
