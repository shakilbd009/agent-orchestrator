// Feature flag check — BRD-02
// VITE_FF_ENABLE_PLATFORM_ORCHESTRATION default false
// Also check VITE_FF_ENABLE_APP_SHELL for base app shell

export function isOrchestrationEnabled(): boolean {
  return import.meta.env.VITE_FF_ENABLE_PLATFORM_ORCHESTRATION === 'true';
}

export function isAppShellEnabled(): boolean {
  return import.meta.env.VITE_FF_ENABLE_APP_SHELL === 'true';
}