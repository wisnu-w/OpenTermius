<script lang="ts">
  import {
    Terminal,
    Radio,
    FolderKanban,
    Zap,
    ExternalLink,
    Clock,
    Pencil,
    Trash2,
    FolderInput,
    Folder
  } from 'lucide-svelte';

  interface HostItem {
    id: string;
    label: string;
    address: string;
    port: number;
    username: string;
    authType?: 'password' | 'key';
    password?: string;
    privateKey?: string;
    passphrase?: string;
    tags: string[];
    useMosh?: boolean;
    pingMs: number;
    group: string;
  }

  interface Props {
    host: HostItem;
    onConnect: (host: HostItem) => void;
    onEdit?: (host: HostItem) => void;
    onDelete?: (host: HostItem) => void;
    onMove?: (host: HostItem) => void;
    onOpenSftp?: (host: HostItem) => void;
  }

  let { host, onConnect, onEdit, onDelete, onMove, onOpenSftp }: Props = $props();
</script>

<div class="group relative rounded-xl border border-[#22222a] bg-[#14141a] p-4 transition-all duration-200 hover:border-emerald-500/40 hover:bg-[#171720] hover:shadow-lg hover:shadow-emerald-950/20 overflow-hidden">
  <div class="flex items-start justify-between gap-3">
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <!-- Host Icon / Type -->
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[#1f1f2a] text-neutral-300 group-hover:bg-emerald-500/10 group-hover:text-emerald-400 transition-colors">
        {#if host.useMosh}
          <Radio class="h-5 w-5" />
        {:else}
          <Terminal class="h-5 w-5" />
        {/if}
      </div>

      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold text-white group-hover:text-emerald-300 transition-colors truncate">
            {host.label}
          </h3>
          {#if host.useMosh}
            <span class="shrink-0 rounded bg-indigo-500/15 px-1.5 py-0.5 text-[9px] font-bold tracking-wider text-indigo-400 uppercase">
              MOSH
            </span>
          {/if}
        </div>
        <p
          class="text-xs text-neutral-400 font-mono truncate"
          title={`${host.username}@${host.address}:${host.port}`}
        >
          {host.username}@{host.address}:{host.port}
        </p>
      </div>
    </div>

    <!-- Ping / Latency Badge -->
    <div class="shrink-0 flex items-center gap-1.5 text-[11px] text-neutral-400 bg-[#1c1c25] px-2 py-0.5 rounded-full border border-[#262633]">
      <span class="h-1.5 w-1.5 rounded-full {host.pingMs < 30 ? 'bg-emerald-400' : 'bg-amber-400'}"></span>
      <span class="whitespace-nowrap">{host.pingMs}ms</span>
    </div>
  </div>

  <!-- Tags & Meta -->
  <div class="mt-4 flex flex-wrap items-center gap-1.5">
    {#each host.tags as tag}
      <span class="rounded bg-[#1e1e28] px-2 py-0.5 text-[10px] font-medium text-neutral-300">
        #{tag}
      </span>
    {/each}
    <span class="text-[10px] text-neutral-400 ml-auto flex items-center gap-1">
      <FolderKanban class="h-3 w-3" />
      {host.group}
    </span>
  </div>

  <!-- Quick Action Buttons -->
  <div class="mt-4 pt-3 border-t border-[#1e1e28] flex items-center gap-1.5">
    <button
      onclick={() => onConnect(host)}
      class="flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-emerald-600/90 py-1.5 px-2.5 text-xs font-semibold text-white shadow hover:bg-emerald-500 transition-colors active:scale-[0.98]"
    >
      <Zap class="h-3.5 w-3.5" /> Connect SSH
    </button>
    {#if onOpenSftp}
      <button
        onclick={() => onOpenSftp(host)}
        class="flex items-center justify-center gap-1 rounded-lg bg-[#20202d] px-2.5 py-1.5 text-xs font-medium text-neutral-300 hover:text-emerald-400 hover:bg-[#28283a] transition-colors"
        title="Open SFTP File Browser"
      >
        <Folder class="h-3.5 w-3.5 text-amber-400" />
        <span>SFTP</span>
      </button>
    {/if}
    {#if onEdit}
      <button
        onclick={() => onEdit(host)}
        class="flex items-center justify-center rounded-lg bg-[#20202d] p-2 text-neutral-400 hover:text-white hover:bg-[#28283a] transition-colors"
        title="Edit Host"
      >
        <Pencil class="h-3.5 w-3.5" />
      </button>
    {/if}
    {#if onMove}
      <button
        onclick={() => onMove(host)}
        class="flex items-center justify-center rounded-lg bg-[#20202d] p-2 text-neutral-400 hover:text-indigo-400 hover:bg-[#28283a] transition-colors"
        title="Move to Folder or Vault"
      >
        <FolderInput class="h-3.5 w-3.5" />
      </button>
    {/if}
    {#if onDelete}
      <button
        onclick={() => onDelete(host)}
        class="flex items-center justify-center rounded-lg bg-[#20202d] p-2 text-neutral-400 hover:text-rose-400 hover:bg-[#28283a] transition-colors"
        title="Delete Host"
      >
        <Trash2 class="h-3.5 w-3.5" />
      </button>
    {/if}
  </div>
</div>
