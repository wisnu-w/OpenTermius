<script lang="ts">
  import { onMount } from 'svelte';
  import { ScrollText, Clock, Server, RefreshCw, Trash2, Copy, Check, AlertCircle } from 'lucide-svelte';
  import { toast, formatErrorMessage } from '../stores/toast.svelte';

  interface SessionLogItem {
    id: string;
    hostLabel: string;
    target: string;
    mode: string;
    startedAt: string;
    durationSec: number;
  }

  let state = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);
  let logs = $state<SessionLogItem[]>([]);
  let copiedId = $state<string | null>(null);

  function formatDuration(sec: number): string {
    if (!sec || sec < 1) return 'Active / <1s';
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return m > 0 ? `${m}m ${s}s` : `${s}s`;
  }

  function formatTimestamp(iso: string): string {
    try {
      return new Date(iso).toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return iso;
    }
  }

  function loadLogs() {
    state = 'loading';
    errorMessage = null;
    try {
      const raw = localStorage.getItem('opentermius_session_logs');
      const parsed: SessionLogItem[] = raw ? JSON.parse(raw) : [];
      logs = parsed;
      state = parsed.length === 0 ? 'empty' : 'success';
    } catch (err: unknown) {
      state = 'error';
      errorMessage = formatErrorMessage(err, 'Failed to load session logs');
    }
  }

  function clearLogs() {
    localStorage.removeItem('opentermius_session_logs');
    logs = [];
    state = 'empty';
    toast.info('Session logs cleared.');
  }

  function copyTarget(log: SessionLogItem) {
    navigator.clipboard?.writeText(log.target);
    copiedId = log.id;
    setTimeout(() => {
      if (copiedId === log.id) copiedId = null;
    }, 1800);
  }

  onMount(() => {
    loadLogs();
  });
</script>

<div class="h-full overflow-y-auto p-4 sm:p-6 space-y-4 sm:space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#22222a] pb-4">
    <div class="min-w-0">
      <div class="flex items-center gap-2 flex-wrap">
        <h2 class="text-base font-bold text-white tracking-tight">Session Logs</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-[10px] font-semibold text-emerald-400">
          Audit Trail
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Connection history, target endpoints, and session duration recorded on this device.
      </p>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      <button
        type="button"
        onclick={loadLogs}
        class="flex flex-1 sm:flex-none items-center justify-center gap-1.5 rounded-lg border border-[#22222a] bg-[#14141d] px-3 py-2 sm:py-1.5 text-xs text-neutral-300 hover:text-white transition"
      >
        <RefreshCw class="h-3.5 w-3.5" /> Refresh
      </button>
      {#if logs.length > 0}
        <button
          type="button"
          onclick={clearLogs}
          class="flex flex-1 sm:flex-none items-center justify-center gap-1.5 rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2 sm:py-1.5 text-xs font-medium text-rose-300 hover:bg-rose-500/20 transition"
        >
          <Trash2 class="h-3.5 w-3.5" /> Clear
        </button>
      {/if}
    </div>
  </div>

  {#if state === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="h-4 w-4 animate-spin text-emerald-400" />
      <span>Loading session logs...</span>
    </div>
  {:else if state === 'error'}
    <div class="rounded-xl border border-rose-500/30 bg-rose-950/20 p-4 flex items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <AlertCircle class="h-5 w-5 text-rose-400 shrink-0" />
        <p class="text-xs text-rose-300">{errorMessage}</p>
      </div>
      <button
        type="button"
        onclick={loadLogs}
        class="rounded-lg bg-rose-500/20 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/30"
      >
        Retry
      </button>
    </div>
  {:else if state === 'empty'}
    <div class="rounded-2xl border border-dashed border-[#22222f] bg-[#121218] p-8 text-center space-y-2.5">
      <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 mx-auto">
        <ScrollText class="h-6 w-6" />
      </div>
      <h3 class="text-sm font-semibold text-white">No sessions recorded yet</h3>
      <p class="text-xs text-neutral-400 max-w-sm mx-auto">
        Open a terminal or connect to an SSH host to automatically log session activity.
      </p>
    </div>
  {:else}
    <!-- Mobile Card List -->
    <div class="space-y-2.5 md:hidden">
      {#each logs as log (log.id)}
        <div class="rounded-2xl border border-[#22222c] bg-[#14141c] p-3.5 space-y-2.5">
          <div class="flex items-start justify-between gap-2">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                <Server class="h-4 w-4" />
              </div>
              <div class="min-w-0">
                <div class="text-xs font-semibold text-white truncate">{log.hostLabel}</div>
                <div class="text-[11px] font-mono text-neutral-400 truncate">{log.target}</div>
              </div>
            </div>
            <span class="shrink-0 rounded-full bg-[#1e1e2a] px-2 py-0.5 text-[10px] font-medium text-emerald-400">
              {log.mode}
            </span>
          </div>

          <div class="flex items-center justify-between pt-2 border-t border-[#1e1e28] text-[11px] text-neutral-400">
            <div class="flex items-center gap-3">
              <span class="flex items-center gap-1 font-mono">
                <Clock class="h-3 w-3 text-neutral-500" />
                {formatDuration(log.durationSec)}
              </span>
              <span>•</span>
              <span>{formatTimestamp(log.startedAt)}</span>
            </div>
            <button
              type="button"
              onclick={() => copyTarget(log)}
              class="flex items-center gap-1 rounded-lg bg-[#1e1e2a] px-2.5 py-1 text-[11px] text-neutral-300 active:bg-[#282838]"
            >
              {#if copiedId === log.id}
                <Check class="h-3 w-3 text-emerald-400" />
                <span class="text-emerald-400">Copied</span>
              {:else}
                <Copy class="h-3 w-3" />
                <span>Copy</span>
              {/if}
            </button>
          </div>
        </div>
      {/each}
    </div>

    <!-- Desktop Table -->
    <div class="hidden md:block rounded-xl border border-[#22222a] bg-[#14141a] overflow-hidden">
      <table class="w-full text-left text-xs">
        <thead class="border-b border-[#22222a] bg-[#181822] text-neutral-400 font-semibold">
          <tr>
            <th class="py-3 px-4">Session</th>
            <th class="py-3 px-4">Target</th>
            <th class="py-3 px-4">Transport</th>
            <th class="py-3 px-4">Duration</th>
            <th class="py-3 px-4">Started</th>
            <th class="py-3 px-4 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[#1e1e28] text-neutral-200">
          {#each logs as log (log.id)}
            <tr class="hover:bg-[#181824] transition-colors">
              <td class="py-3 px-4 font-medium text-white flex items-center gap-2">
                <Server class="h-3.5 w-3.5 text-emerald-400" />
                {log.hostLabel}
              </td>
              <td class="py-3 px-4 font-mono text-neutral-300">{log.target}</td>
              <td class="py-3 px-4 text-neutral-400">{log.mode}</td>
              <td class="py-3 px-4 font-mono text-neutral-400">{formatDuration(log.durationSec)}</td>
              <td class="py-3 px-4 text-neutral-400">{formatTimestamp(log.startedAt)}</td>
              <td class="py-3 px-4 text-right">
                <button
                  type="button"
                  onclick={() => copyTarget(log)}
                  class="inline-flex items-center gap-1 rounded bg-[#20202c] px-2.5 py-1 text-xs text-emerald-400 hover:bg-emerald-600 hover:text-white transition"
                >
                  {#if copiedId === log.id}
                    <Check class="h-3 w-3" /> Copied
                  {:else}
                    <Copy class="h-3 w-3" /> Copy Target
                  {/if}
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
