<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ApiVaultItem, type ApiVault } from '../api/client';
  import { encryptVaultItem, decryptVaultItem } from '../crypto/vault';
  import { FileCode2, Play, Plus, RefreshCw, AlertCircle, Sparkles } from 'lucide-svelte';

  interface Props {
    activeVaultId?: string;
  }

  let { activeVaultId }: Props = $props();

  let state = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);
  let snippets = $state<Array<{ id: string; title: string; script: string; category: string }>>([]);
  let isCreating = $state(false);

  let newTitle = $state('');
  let newScript = $state('');
  let newCategory = $state('General');

  async function loadSnippets() {
    if (!activeVaultId) {
      state = 'empty';
      return;
    }

    state = 'loading';
    errorMessage = null;

    try {
      const items = await api.getVaultItems(activeVaultId, 'snippet');
      if (items.length === 0) {
        snippets = [];
        state = 'empty';
        return;
      }

      // Parse payload
      snippets = items.map((item) => {
        let script = '';
        let category = 'General';
        try {
          const parsed = JSON.parse(atob(item.ciphertext));
          script = parsed.script || '';
          category = parsed.category || 'General';
        } catch {
          script = item.ciphertext;
        }
        return {
          id: item.id,
          title: item.label,
          script,
          category
        };
      });

      state = 'success';
    } catch (err: unknown) {
      state = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to fetch snippets from backend';
    }
  }

  async function handleCreateSnippet(e: SubmitEvent) {
    e.preventDefault();
    if (!activeVaultId || !newTitle.trim() || !newScript.trim()) return;

    try {
      const payload = { script: newScript, category: newCategory };
      const ciphertext = btoa(JSON.stringify(payload));
      const nonce = btoa(crypto.randomUUID().slice(0, 12));

      const created = await api.createVaultItem(activeVaultId, {
        kind: 'snippet',
        label: newTitle.trim(),
        ciphertext,
        nonce
      });

      snippets.unshift({
        id: created.id,
        title: created.label,
        script: newScript,
        category: newCategory
      });

      state = 'success';
      isCreating = false;
      newTitle = '';
      newScript = '';
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to save snippet');
    }
  }

  onMount(() => {
    loadSnippets();
  });
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-white">Startup Snippets & Automation</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400 font-medium">
          Free Automation
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Execute shell commands and automated diagnostics automatically on SSH connection start.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={loadSnippets}
        class="flex items-center gap-1.5 rounded-lg border border-[#22222a] bg-[#14141a] px-3 py-1.5 text-xs text-neutral-300 hover:text-white transition"
      >
        <RefreshCw class="h-3.5 w-3.5" /> Refresh
      </button>

      <button
        onclick={() => (isCreating = !isCreating)}
        class="flex items-center gap-2 rounded-lg bg-emerald-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow hover:bg-emerald-500 transition"
      >
        <Plus class="h-4 w-4" /> Create Snippet
      </button>
    </div>
  </div>

  {#if isCreating}
    <form onsubmit={handleCreateSnippet} class="rounded-xl border border-emerald-500/30 bg-[#14141d] p-5 space-y-3 animate-in fade-in text-xs">
      <div>
        <label for="snip-title" class="block font-medium text-neutral-300 mb-1">Snippet Title</label>
        <input
          id="snip-title"
          type="text"
          bind:value={newTitle}
          placeholder="e.g. Inspect Docker Container Health"
          class="w-full rounded-lg border border-[#262635] bg-[#0c0c12] px-3 py-1.5 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
          required
        />
      </div>
      <div>
        <label for="snip-cat" class="block font-medium text-neutral-300 mb-1">Category</label>
        <input
          id="snip-cat"
          type="text"
          bind:value={newCategory}
          placeholder="e.g. Diagnostics / DevOps"
          class="w-full rounded-lg border border-[#262635] bg-[#0c0c12] px-3 py-1.5 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
        />
      </div>
      <div>
        <label for="snip-code" class="block font-medium text-neutral-300 mb-1">Shell Script / Command</label>
        <textarea
          id="snip-code"
          bind:value={newScript}
          rows="3"
          placeholder="uptime && free -m && df -h"
          class="w-full rounded-lg border border-[#262635] bg-[#0c0c12] p-3 font-mono text-emerald-400 placeholder-neutral-600 outline-none focus:border-emerald-500"
          required
        ></textarea>
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={() => (isCreating = false)}
          class="rounded-lg bg-[#20202c] px-3 py-1.5 text-neutral-400 hover:text-white"
        >
          Cancel
        </button>
        <button
          type="submit"
          class="rounded-lg bg-emerald-600 px-4 py-1.5 font-semibold text-white hover:bg-emerald-500"
        >
          Save to Vault
        </button>
      </div>
    </form>
  {/if}

  <!-- 4 UI States -->
  {#if state === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="h-4 w-4 animate-spin text-emerald-400" />
      <span>Loading snippets from backend database...</span>
    </div>
  {:else if state === 'error'}
    <div class="rounded-xl border border-rose-500/30 bg-rose-950/20 p-5 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <AlertCircle class="h-5 w-5 text-rose-400 shrink-0" />
        <div>
          <p class="text-xs font-semibold text-rose-300">Backend Communication Error</p>
          <p class="text-[11px] text-rose-400/80">{errorMessage}</p>
        </div>
      </div>
      <button
        onclick={loadSnippets}
        class="rounded-lg bg-rose-500/20 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/30"
      >
        Retry
      </button>
    </div>
  {:else if state === 'empty'}
    <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-12 text-center space-y-3">
      <FileCode2 class="h-10 w-10 text-neutral-600 mx-auto" />
      <h3 class="text-sm font-semibold text-white">No Snippets in Vault</h3>
      <p class="text-xs text-neutral-400 max-w-sm mx-auto">
        Create shell snippets to automate server updates, container checks, and repetitive commands.
      </p>
      <button
        onclick={() => (isCreating = true)}
        class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500"
      >
        + Create Your First Snippet
      </button>
    </div>
  {:else}
    <div class="space-y-3">
      {#each snippets as snippet (snippet.id)}
        <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-4 hover:border-emerald-500/30 transition">
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
              <h3 class="text-sm font-semibold text-white">{snippet.title}</h3>
              <span class="rounded bg-[#1f1f2b] px-2 py-0.5 text-[10px] text-neutral-300 font-medium">
                {snippet.category}
              </span>
            </div>

            <button class="flex items-center gap-1.5 rounded bg-[#1e1e28] px-2.5 py-1 text-xs font-medium text-emerald-400 hover:bg-[#252533] transition">
              <Play class="h-3 w-3" /> Run in Terminal
            </button>
          </div>

          <pre class="rounded-lg bg-[#0d0d12] border border-[#1a1a24] p-3 text-xs font-mono text-neutral-300 overflow-x-auto"><code>{snippet.script}</code></pre>
        </div>
      {/each}
    </div>
  {/if}
</div>
