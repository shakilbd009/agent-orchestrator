// BRD-03 Client Portal — Owner Label Mapper
// Maps internal project phases to client-facing owner labels (Phase 1 hardcoded mapping)

/**
 * Phase 1 hardcoded mapping:
 * planning       → Product
 * decomposition  → Product
 * execution      → Engineering
 * validation     → Review
 * acceptance     → Quality
 * closed         → Product
 *
 * Per BRD-03 FR-03-016/FR-03-017: default mapping uses labels like
 * Product, Engineering, Review, Quality rather than internal agent profile names.
 * Internal project owners can override via `client_owner_label_override` in Phase 2.
 */

const PHASE_OWNER_MAP: Record<string, string> = {
  planning: 'Product',
  decomposition: 'Product',
  execution: 'Engineering',
  validation: 'Review',
  acceptance: 'Quality',
  closed: 'Product',
};

const DEFAULT_LABEL = 'Product';

/**
 * Maps a project phase to a client-facing owner label.
 * Phase 1 uses hardcoded mapping; Phase 2 will support override via
 * `client_owner_label_override` on the project configuration.
 */
export function mapOwnerLabel(
  phaseOrRole: string,
  _override?: string | null
): string {
  if (_override && _override.trim().length > 0) {
    return _override.trim();
  }

  if (!phaseOrRole) return DEFAULT_LABEL;

  const normalized = phaseOrRole.toLowerCase().trim();

  if (normalized in PHASE_OWNER_MAP) {
    return PHASE_OWNER_MAP[normalized];
  }

  // Fallback for any unmapped phases
  return DEFAULT_LABEL;
}

/**
 * Returns the full map for reference/documentation purposes.
 */
export function getOwnerLabelMap(): Record<string, string> {
  return { ...PHASE_OWNER_MAP };
}