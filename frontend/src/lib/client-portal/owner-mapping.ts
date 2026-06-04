// BRD-03 Client Portal — hardcoded owner label mapping
// Phase 1 uses hardcoded mapping; Phase 2 uses API-provided mapping from GET /config/owner-mapping

export type OwnerRole = string;

export const OWNER_LABEL_MAP: Record<string, string> = {
  engineering: 'Engineering',
  backend: 'Engineering',
  frontend: 'Engineering',
  developer: 'Engineering',
  product_manager: 'Product',
  product_owner: 'Product',
  pm: 'Product',
  reviewer: 'Review',
  qa: 'Review',
  quality_assurance: 'Review',
  quality: 'Quality',
  architect: 'Quality',
  design: 'Quality',
  client: 'Client',
  client_stakeholder: 'Client',
  external: 'Client',
};

export function getOwnerLabel(role: string | undefined | null, override?: string | null): string {
  if (override) return override;
  if (!role) return 'Product';
  return OWNER_LABEL_MAP[role.toLowerCase()] ?? 'Product';
}