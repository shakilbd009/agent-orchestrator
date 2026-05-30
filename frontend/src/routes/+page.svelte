<script lang="ts">
	import { onMount } from 'svelte';

	let data: { status: string; version: string; timestamp: string } | null = $state(null);
	let error: string | null = $state(null);

	onMount(async () => {
		const controller = new AbortController();
		const timeout = setTimeout(() => controller.abort(), 5000);

		try {
			const base = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:3001';
			const res = await fetch(`${base}/health`, { signal: controller.signal });
			data = await res.json();
			error = null;
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : String(e);
			data = null;
		} finally {
			clearTimeout(timeout);
		}
	});
</script>

<svelte:head>
	<title>Agent Orchestrator — Health</title>
</svelte:head>

<main>
	<h1>Agent Orchestrator</h1>

	{#if error}
		<p class="error">Error: {error}</p>
	{:else if data}
		<dl>
			<dt>Status</dt>
			<dd><code>{data.status}</code></dd>

			<dt>Version</dt>
			<dd><code>{data.version}</code></dd>

			<dt>Timestamp</dt>
			<dd><code>{data.timestamp}</code></dd>
		</dl>
	{:else}
		<p>Loading...</p>
	{/if}
</main>

<style>
	:global(body) {
		font-family: system-ui, sans-serif;
		margin: 2rem;
		background: #0f0f0f;
		color: #e0e0e0;
	}

	main {
		max-width: 480px;
		margin: 0 auto;
	}

	h1 {
		font-size: 1.5rem;
		margin-bottom: 1.5rem;
	}

	dl {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.5rem 1rem;
	}

	dt {
		font-weight: 600;
		color: #a0a0a0;
	}

	dd {
		margin: 0;
	}

	code {
		background: #1e1e1e;
		padding: 0.2em 0.4em;
		border-radius: 4px;
		font-size: 0.9em;
	}

	.error {
		color: #ff6b6b;
	}
</style>