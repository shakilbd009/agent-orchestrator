// BRD-03 Client Portal — feature flag helpers
// VITE_FF_ENABLE_CLIENT_PORTAL defaults false

export function isClientPortalEnabled(): boolean {
  return import.meta.env.VITE_FF_ENABLE_CLIENT_PORTAL === 'true';
}