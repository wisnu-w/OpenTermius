<script lang="ts">
  import { toast } from '../stores/toast.svelte';
  import {
    Network,
    Plus,
    RefreshCw,
    Trash2,
    Copy,
    Check,
    ArrowRight,
    Globe
  } from 'lucide-svelte';
  import { api } from '../api/client';

  interface TunnelConfig {
    id: string;
    label: string;
    type: 'local' | 'dynamic' | 'remote';
    hostId: string;
    hostLabel?: string;
    localPort: number;
    remoteHost?: string;
    remotePort?: number;
    autoConnect?: boolean;
    isActive?: boolean;
  }

  interface Props {
    activeVaultId?: string;
    hosts?: Array<{ id: string; label: string; address: string }>;
  }

  let { activeVaultId, hosts = [] }: Props = $props();

  let tunnels = $state<TunnelConfig[]>([]);
  let loadState = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);

  // Modal State
  let showModal = $state(false);
  let isSubmitting = $state(false);
  let formError = $state<string | null>(null);

  // Form Fields
  let newLabel = $state('');
  let newType = $state<'local' | 'dynamic' | 'remote'>('local');
  let newHostId = $state('');
  let newLocalPort = $state(5432);
  let newRemoteHost = $state('127.0.0.1');
  let newRemotePort = $state(5432);
  let newAutoConnect = $state(false);

  let copiedId = $state<string | null>(null);

  async function loadTunnels() {
    loadState = 'loading';
    errorMessage = null;

    let targetVaultId = activeVaultId;
    if (!targetVaultId) {
      try {
        const vaults = await api.getVaults();
        const primary = vaults.find((v) => v.type === 'personal') || vaults[0];
        if (primary) targetVaultId = primary.id;
      } catch (err: unknown) {
        errorMessage = err instanceof Error ? err.message : 'Failed to detect active vault.';
        loadState = 'error';
        return;
      }
    }

    if (!targetVaultId) {
      errorMessage = 'No vault found. Create a personal or team vault first.';
      loadState = 'error';
      return;
    }

    try {
      const items = await api.getVaultItems(targetVaultId);
      const tunnelItems = items.filter((item) => item.kind === 'tunnel');

      const loaded: TunnelConfig[] = [];
      for (const item of tunnelItems) {
        try {
          const raw = atob(item.ciphertext);
          const parsed = JSON.parse(raw);
          const matchedHost = hosts.find((h) => h.id === parsed.hostId);
          loaded.push({
            id: item.id,
            label: item.label,
            type: parsed.type || 'local',
            hostId: parsed.hostId || '',
            hostLabel: matchedHost?.label || parsed.hostLabel || 'Unknown Host',
            localPort: parsed.localPort || 5432,
            remoteHost: parsed.remoteHost || '127.0.0.1',
            remotePort: parsed.remotePort || 5432,
            autoConnect: parsed.autoConnect || false,
            isActive: false
          });
        } catch {
          loaded.push({
            id: item.id,
            label: item.label,
            type: 'local',
            hostId: '',
            hostLabel: 'SSH Host',
            localPort: 5432,
            remoteHost: '127.0.0.1',
            remotePort: 5432,
            isActive: false
          });
        }
      }

      tunnels = loaded;
      loadState = tunnels.length === 0 ? 'empty' : 'success';
    } catch (err: unknown) {
      errorMessage = err instanceof Error ? err.message : 'Failed to fetch tunnels from backend.';
      loadState = 'error';
    }
  }

  function openAddModal() {
    formError = null;
    newLabel = '';
    newType = 'local';
    newHostId = hosts.length > 0 ? hosts[0].id : '';
    newLocalPort = 5432;
    newRemoteHost = '127.0.0.1';
    newRemotePort = 5432;
    newAutoConnect = false;
    showModal = true;
  }

  async function handleSaveTunnel() {
    formError = null;
    if (!newLabel.trim()) {
      formError = 'Please provide a descriptive tunnel label.';
      return;
    }
    if (!newHostId && hosts.length > 0) {
      newHostId = hosts[0].id;
    }
    if (!newLocalPort || newLocalPort <= 0 || newLocalPort > 65535) {
      formError = 'Invalid local port (must be 1 - 65535).';
      return;
    }

    isSubmitting = true;

    let targetVaultId = activeVaultId;
    if (!targetVaultId) {
      const vaults = await api.getVaults();
      const primary = vaults.find((v) => v.type === 'personal') || vaults[0];
      targetVaultId = primary?.id;
    }

    if (!targetVaultId) {
      formError = 'Target vault not resolved. Please select a vault first.';
      isSubmitting = false;
      return;
    }

    try {
      const payload = {
        label: newLabel.trim(),
        type: newType,
        hostId: newHostId,
        localPort: newLocalPort,
        remoteHost: newRemoteHost.trim() || '127.0.0.1',
        remotePort: newRemotePort || 5432,
        autoConnect: newAutoConnect
      };

      const ciphertext = btoa(JSON.stringify(payload));
      const nonce = btoa(crypto.randomUUID().slice(0, 12));

      const created = await api.createVaultItem(targetVaultId, {
        kind: 'tunnel',
        label: newLabel.trim(),
        ciphertext,
        nonce
      });

      const matchedHost = hosts.find((h) => h.id === newHostId);
      tunnels = [
        ...tunnels,
        {
          id: created.id,
          label: newLabel.trim(),
          type: newType,
          hostId: newHostId,
          hostLabel: matchedHost?.label || 'SSH Host',
          localPort: newLocalPort,
          remoteHost: newRemoteHost.trim() || '127.0.0.1',
          remotePort: newRemotePort || 5432,
          autoConnect: newAutoConnect,
          isActive: false
        }
      ];

      loadState = 'success';
      showModal = false;
    } catch (err: unknown) {
      formError = err instanceof Error ? err.message : 'Failed to save tunnel to backend.';
    } finally {
      isSubmitting = false;
    }
  }

  async function handleDeleteTunnel(id: string) {
    if (!confirm('Are you sure you want to delete this tunnel configuration?')) return;
    let targetVaultId = activeVaultId;
    if (!targetVaultId) {
      const vaults = await api.getVaults();
      targetVaultId = vaults[0]?.id;
    }
    if (!targetVaultId) return;

    try {
      await api.deleteVaultItem(targetVaultId, id);
      tunnels = tunnels.filter((t) => t.id !== id);
      if (tunnels.length === 0) loadState = 'empty';
    } catch (err: unknown) {
      toast.error(err, 'Failed to delete tunnel');
    }
  }

  function toggleTunnelActive(id: string) {
    tunnels = tunnels.map((t) => {
      if (t.id === id) {
        return { ...t, isActive: !t.isActive };
      }
      return t;
    });
  }

  function copyConnectionString(tunnel: TunnelConfig) {
    let str = '';
    if (tunnel.type === 'dynamic') {
      str = `socks5://127.0.0.1:${tunnel.localPort}`;
    } else {
      str = `127.0.0.1:${tunnel.localPort} -> ${tunnel.remoteHost}:${tunnel.remotePort}`;
    }
    navigator.clipboard.writeText(str);
    copiedId = tunnel.id;
    setTimeout(() => {
      copiedId = null;
    }, 2000);
  }

  $effect(() => {
    if (activeVaultId !== undefined) {
      loadTunnels();
    }
  });
</script>

<div class="flex h-full flex-col p-4 sm:p-6 overflow-y-auto space-y-4 sm:space-y-6">
  <!-- Header Banner -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#22222a] pb-4">
    <div class="min-w-0">
      <div class="flex items-center gap-2 flex-wrap">
        <h2 class="text-base font-bold text-white tracking-tight">Port Forwarding</h2>
        <span class="rounded bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-400 border border-emerald-500/20">
          SSH Tunnel
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Forward local ports, remote ports, or a SOCKS5 proxy over SSH.
      </p>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      <button
        type="button"
        onclick={loadTunnels}
        class="flex flex-1 sm:flex-none items-center justify-center gap-1.5 px-3 py-2 sm:py-1.5 rounded-lg border border-[#22222a] bg-[#14141d] hover:bg-[#181824] text-neutral-300 text-xs transition"
      >
        <RefreshCw class="w-3.5 h-3.5" />
        Refresh
      </button>
      <button
        type="button"
        onclick={openAddModal}
        class="flex flex-1 sm:flex-none items-center justify-center gap-1.5 px-3.5 py-2 sm:py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold shadow-sm transition"
      >
        <Plus class="w-4 h-4" />
        New Tunnel
      </button>
    </div>
  </div>

  <!-- Content States -->
  {#if loadState === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="w-4 h-4 animate-spin text-emerald-400" />
      <span>Loading encrypted tunnels...</span>
    </div>
  {:else if loadState === 'error'}
    <div class="rounded-xl border border-rose-500/20 bg-rose-500/10 p-4 text-xs text-rose-300">
      {errorMessage}
    </div>
  {:else if loadState === 'empty'}
    <div class="flex min-h-56 flex-col items-center justify-center text-center p-6 sm:p-8 space-y-3 rounded-2xl border border-dashed border-[#22222f] bg-[#121218]">
      <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
        <Network class="w-6 h-6" />
      </div>
      <div>
        <h4 class="text-sm font-semibold text-white">No tunnels yet</h4>
        <p class="text-xs text-neutral-400 mt-1 max-w-sm">
          Forward database ports, internal apps, or a SOCKS5 proxy over SSH.
        </p>
      </div>
      <button
        type="button"
        onclick={openAddModal}
        class="mt-2 flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium transition shadow"
      >
        <Plus class="w-4 h-4" />
        Add Tunnel
      </button>
    </div>
  {:else}
    <!-- Tunnels Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each tunnels as tunnel (tunnel.id)}
        <div class="group relative rounded-xl border border-[#22222e] bg-[#13131c] hover:border-emerald-500/40 p-4 shadow-sm transition flex flex-col justify-between space-y-4">
          <!-- Top Row -->
          <div class="flex items-start justify-between gap-2">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-[#181825] border border-[#262635] text-emerald-400 shrink-0">
                {#if tunnel.type === 'dynamic'}
                  <Globe class="w-4 h-4" />
                {:else}
                  <Network class="w-4 h-4" />
                {/if}
              </div>
              <div class="min-w-0">
                <h4 class="text-xs font-semibold text-white truncate">{tunnel.label}</h4>
                <div class="flex items-center gap-1.5 text-[10px] text-neutral-400 mt-0.5">
                  <span class="uppercase tracking-wider font-mono text-emerald-400 font-semibold">{tunnel.type}</span>
                  <span>•</span>
                  <span class="truncate">{tunnel.hostLabel}</span>
                </div>
              </div>
            </div>

            <!-- Active Switch / Badge -->
            <button
              type="button"
              onclick={() => toggleTunnelActive(tunnel.id)}
              class="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium transition {tunnel.isActive ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'bg-neutral-800 text-neutral-400 hover:text-white'}"
            >
              <span class="h-1.5 w-1.5 rounded-full {tunnel.isActive ? 'bg-emerald-400 animate-pulse' : 'bg-neutral-500'}"></span>
              {tunnel.isActive ? 'Active' : 'Inactive'}
            </button>
          </div>

          <!-- Port Mapping Visual -->
          <div class="rounded-lg bg-[#0e0e14] border border-[#1e1e28] p-2.5 text-xs font-mono">
            {#if tunnel.type === 'dynamic'}
              <div class="flex items-center justify-between text-neutral-300">
                <span class="text-neutral-400">SOCKS5 Proxy:</span>
                <span class="text-emerald-400 font-semibold">127.0.0.1:{tunnel.localPort}</span>
              </div>
            {:else}
              <div class="flex items-center justify-between gap-2">
                <div class="text-neutral-300 truncate">
                  <span class="text-neutral-500 text-[10px] block font-sans">Local</span>
                  :{tunnel.localPort}
                </div>
                <ArrowRight class="w-3.5 h-3.5 text-neutral-500 shrink-0" />
                <div class="text-right text-neutral-300 truncate">
                  <span class="text-neutral-500 text-[10px] block font-sans">Remote</span>
                  {tunnel.remoteHost}:{tunnel.remotePort}
                </div>
              </div>
            {/if}
          </div>

          <!-- Bottom Actions -->
          <div class="flex items-center justify-between pt-2 border-t border-[#1e1e28] text-xs">
            <button
              type="button"
              onclick={() => copyConnectionString(tunnel)}
              class="flex items-center gap-1.5 text-neutral-400 hover:text-white transition"
              title="Copy connection string"
            >
              {#if copiedId === tunnel.id}
                <Check class="w-3.5 h-3.5 text-emerald-400" />
                <span class="text-emerald-400 text-[11px]">Copied!</span>
              {:else}
                <Copy class="w-3.5 h-3.5" />
                <span class="text-[11px]">Copy Address</span>
              {/if}
            </button>

            <button
              type="button"
              onclick={() => handleDeleteTunnel(tunnel.id)}
              class="text-neutral-500 hover:text-rose-400 p-1 transition"
              title="Delete tunnel"
            >
              <Trash2 class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Add Tunnel Modal -->
{#if showModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 overflow-y-auto">
    <div class="w-full max-w-md max-h-[85dvh] overflow-y-auto rounded-2xl border border-[#262635] bg-[#14141d] p-4 sm:p-6 shadow-2xl space-y-4 animate-in fade-in zoom-in-95 duration-150 my-auto">
      <div class="flex items-center justify-between border-b border-[#22222e] pb-3">
        <div class="flex items-center gap-2.5">
          <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Network class="w-4 h-4" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-white">New Tunnel</h3>
            <p class="text-[11px] text-neutral-400">Route TCP traffic through an SSH host.</p>
          </div>
        </div>
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="flex h-8 w-8 items-center justify-center rounded-lg text-neutral-400 hover:text-white active:bg-[#1e1e2c] text-sm transition"
          aria-label="Close"
        >
          ✕
        </button>
      </div>

      {#if formError}
        <div class="rounded-xl border border-rose-500/30 bg-rose-500/10 p-2.5 text-xs text-rose-300">
          {formError}
        </div>
      {/if}

      <div class="space-y-3.5 text-xs">
        <div>
          <label for="tunnel-label-input" class="block text-neutral-300 font-medium mb-1.5">Label</label>
          <input
            id="tunnel-label-input"
            type="text"
            bind:value={newLabel}
            placeholder="e.g. Postgres Staging"
            class="w-full rounded-xl sm:rounded-lg border border-[#22222e] bg-[#0f0f15] px-3 py-2.5 sm:py-2 text-sm sm:text-xs text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
          />
        </div>

        <div>
          <span class="block text-neutral-300 font-medium mb-1.5">Type</span>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              onclick={() => (newType = 'local')}
              class="rounded-xl sm:rounded-lg border p-2.5 text-left transition {newType === 'local' ? 'border-emerald-500 bg-emerald-500/10 text-white' : 'border-[#22222e] bg-[#0f0f15] text-neutral-400 hover:text-white'}"
            >
              <div class="font-semibold text-xs">Local</div>
              <div class="text-[10px] text-neutral-400 mt-0.5">Local → Remote</div>
            </button>
            <button
              type="button"
              onclick={() => (newType = 'dynamic')}
              class="rounded-xl sm:rounded-lg border p-2.5 text-left transition {newType === 'dynamic' ? 'border-emerald-500 bg-emerald-500/10 text-white' : 'border-[#22222e] bg-[#0f0f15] text-neutral-400 hover:text-white'}"
            >
              <div class="font-semibold text-xs">SOCKS5</div>
              <div class="text-[10px] text-neutral-400 mt-0.5">Dynamic proxy</div>
            </button>
          </div>
        </div>

        <div>
          <label for="tunnel-bastion-host" class="block text-neutral-300 font-medium mb-1.5">SSH Host</label>
          {#if hosts.length === 0}
            <div class="rounded-xl sm:rounded-lg border border-amber-500/30 bg-amber-500/10 p-2.5 text-[11px] text-amber-300">
              No hosts yet. Add one in Hosts first.
            </div>
          {:else}
            <select
              id="tunnel-bastion-host"
              bind:value={newHostId}
              class="w-full rounded-xl sm:rounded-lg border border-[#22222e] bg-[#0f0f15] px-3 py-2.5 sm:py-2 text-sm sm:text-xs text-white outline-none focus:border-emerald-500"
            >
              {#each hosts as host}
                <option value={host.id}>{host.label} ({host.address})</option>
              {/each}
            </select>
          {/if}
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="tunnel-local-port" class="block text-neutral-300 font-medium mb-1.5">Local Port</label>
            <input
              id="tunnel-local-port"
              type="number"
              bind:value={newLocalPort}
              placeholder="5432"
              class="w-full rounded-xl sm:rounded-lg border border-[#22222e] bg-[#0f0f15] px-3 py-2.5 sm:py-2 text-sm sm:text-xs text-white outline-none focus:border-emerald-500 font-mono"
            />
          </div>

          {#if newType === 'local'}
            <div>
              <label for="tunnel-remote-port" class="block text-neutral-300 font-medium mb-1.5">Remote Port</label>
              <input
                id="tunnel-remote-port"
                type="number"
                bind:value={newRemotePort}
                placeholder="5432"
                class="w-full rounded-xl sm:rounded-lg border border-[#22222e] bg-[#0f0f15] px-3 py-2.5 sm:py-2 text-sm sm:text-xs text-white outline-none focus:border-emerald-500 font-mono"
              />
            </div>
          {/if}
        </div>

        {#if newType === 'local'}
          <div>
            <label for="tunnel-remote-host" class="block text-neutral-300 font-medium mb-1.5">Remote Host</label>
            <input
              id="tunnel-remote-host"
              type="text"
              bind:value={newRemoteHost}
              placeholder="127.0.0.1"
              class="w-full rounded-xl sm:rounded-lg border border-[#22222e] bg-[#0f0f15] px-3 py-2.5 sm:py-2 text-sm sm:text-xs text-white outline-none focus:border-emerald-500 font-mono"
            />
          </div>
        {/if}
      </div>

      <div class="grid grid-cols-2 sm:flex sm:items-center sm:justify-end gap-2.5 pt-3 border-t border-[#22222e]">
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="px-3.5 py-2.5 sm:py-1.5 rounded-xl sm:rounded-lg border border-[#22222e] text-neutral-300 hover:text-white text-xs font-medium transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleSaveTunnel}
          disabled={isSubmitting}
          class="px-4 py-2.5 sm:py-1.5 rounded-xl sm:rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold shadow transition disabled:opacity-50"
        >
          {isSubmitting ? 'Saving...' : 'Save Tunnel'}
        </button>
      </div>
    </div>
  </div>
{/if}
