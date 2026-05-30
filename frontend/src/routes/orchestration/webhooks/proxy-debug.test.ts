/**
 * Minimal debug test to isolate $derived + $app/state Proxy behavior
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte/svelte5';
import { readable } from 'svelte/store';

// pageState is mutated in-place by tests
const { pageState } = vi.hoisted(() => {
  const pageState = {
    url: new URL('http://localhost/orchestration/webhooks?project=proj-alpha'),
  };
  return { pageState };
});

vi.mock('$app/state', () => ({
  page: new Proxy(
    { __brand: 'page' },
    {
      get(_target, prop) {
        if (prop === 'url') {
          return pageState.url;
        }
        return undefined;
      },
    },
  ),
}));

// Component source: reads projectId exactly the way webhooks/+page.svelte does
const COMPONENT_SOURCE = `
<script>
  import { page } from '$app/state';
  let projectId = $derived(page.url.searchParams.get('project') ?? '');
  let loading = $state(true);
  let webhooks = $state([]);

  $effect(() => {
    console.log('effect fired, projectId:', projectId);
    if (projectId) {
      loading = false;
    }
  });
</script>

<p>projectId: {projectId}</p>
{#if loading}
  <p>Loading...</p>
{:else}
  <p>Ready: {webhooks.length}</p>
{/if}
`;

// Write temp component
import { writeFileSync } from 'fs';
writeFileSync('/tmp/TestCmp.svelte', COMPONENT_SOURCE);

it('page.url.searchParams.get works in $derived', async () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { default: TestCmp } = require('/tmp/TestCmp.svelte');
  render(TestCmp);
  await new Promise(r => setTimeout(r, 100));
  const p = screen.queryByText(/projectId:/);
  console.log('Found p:', p?.textContent ?? 'null');
  const loading = screen.queryByText(/Loading/);
  console.log('Found Loading:', loading?.textContent ?? 'null');
});
