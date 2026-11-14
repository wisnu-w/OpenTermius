<script lang="ts">
  import { api, type ApiSftpFile, type SftpHostConfig } from '../api/client';
  import { platform } from '../stores/platform.svelte';
  import { toast } from '../stores/toast.svelte';
  import SftpPane, { type SftpHostItem } from './SftpPane.svelte';
  import { ArrowRight, ArrowLeft, Copy, Scissors } from 'lucide-svelte';

  interface Props {
    hosts: SftpHostItem[];
    initialHostId?: string;
  }

  let { hosts = [], initialHostId = '' }: Props = $props();

  let leftTargetId = $state<string>(platform.isNativeTauri ? 'local' : '');
  let rightTargetId = $state<string>('');

  let leftPath = $state('');
  let rightPath = $state('');
  let leftSelection = $state<string[]>([]);
  let rightSelection = $state<string[]>([]);
  let leftItems = $state<ApiSftpFile[]>([]);
  let rightItems = $state<ApiSftpFile[]>([]);
  let refreshCounter = $state(0);

  let isTransferring = $state(false);
  let transferStatus = $state<string | null>(null);
  let mobileActivePane = $state<'left' | 'right'>('right');

  $effect(() => {
    if (!leftTargetId) {
      if (platform.isNativeTauri) {
        leftTargetId = 'local';
      } else if (hosts.length > 0) {
        leftTargetId = hosts[0].id;
      }
    }
    if (initialHostId && initialHostId !== rightTargetId) {
      rightTargetId = initialHostId;
    } else if (!rightTargetId && hosts.length > 0) {
      rightTargetId = hosts[0].id;
    }
  });

  function hostConfigFor(targetId: string): SftpHostConfig | null {
    const h = hosts.find((item) => item.id === targetId);
    if (!h) return null;
    return {
      address: h.address,
      port: h.port || 22,
      username: h.username,
      authType: h.authType || 'password',
      password: h.password,
      privateKey: h.privateKey,
      passphrase: h.passphrase
    };
  }

  async function readBytesFromTarget(targetId: string, filePath: string): Promise<Uint8Array> {
    if (targetId === 'local') {
      const { invoke } = await import('@tauri-apps/api/core');
      const arr = await invoke<number[]>('local_fs_read_bytes', { path: filePath });
      return new Uint8Array(arr);
    }
    const cfg = hostConfigFor(targetId);
    if (!cfg) throw new Error('Source host not found');
    const blob = await api.sftpDownloadBlob(cfg, filePath);
    return new Uint8Array(await blob.arrayBuffer());
  }

  async function writeBytesToTarget(
    targetId: string,
    destDir: string,
    filename: string,
    bytes: Uint8Array
  ): Promise<void> {
    if (targetId === 'local') {
      const { invoke } = await import('@tauri-apps/api/core');
      const base = destDir.endsWith('/') ? destDir.slice(0, -1) : destDir;
      await invoke('local_fs_write_bytes', {
        path: `${base}/${filename}`,
        data: Array.from(bytes)
      });
      return;
    }
    const cfg = hostConfigFor(targetId);
    if (!cfg) throw new Error('Destination host not found');
    const copyBuffer = new Uint8Array(bytes).buffer;
    const file = new File([copyBuffer], filename);
    await api.sftpUpload(cfg, destDir, file);
  }

  async function deleteFromTarget(targetId: string, filePath: string): Promise<void> {
    if (targetId === 'local') {
      const { invoke } = await import('@tauri-apps/api/core');
      await invoke('local_fs_delete', { path: filePath });
      return;
    }
    const cfg = hostConfigFor(targetId);
    if (!cfg) throw new Error('Source host not found');
    await api.sftpDelete(cfg, filePath);
  }

  async function transferFiles(direction: 'left-to-right' | 'right-to-left', move: boolean) {
    const srcTarget = direction === 'left-to-right' ? leftTargetId : rightTargetId;
    const dstTarget = direction === 'left-to-right' ? rightTargetId : leftTargetId;
    const dstPath = direction === 'left-to-right' ? rightPath : leftPath;
    const selected = direction === 'left-to-right' ? leftSelection : rightSelection;
    const srcItems = direction === 'left-to-right' ? leftItems : rightItems;

    if (!srcTarget || !dstTarget || !dstPath || selected.length === 0) return;

    const filesToTransfer = srcItems.filter((item) => selected.includes(item.path) && !item.isDir);
    if (filesToTransfer.length === 0) return;

    isTransferring = true;
    try {
      for (let i = 0; i < filesToTransfer.length; i++) {
        const item = filesToTransfer[i];
        transferStatus = `${move ? 'Moving' : 'Copying'} ${item.name} (${i + 1}/${filesToTransfer.length})...`;
        const bytes = await readBytesFromTarget(srcTarget, item.path);
        await writeBytesToTarget(dstTarget, dstPath, item.name, bytes);
        if (move) {
          await deleteFromTarget(srcTarget, item.path);
        }
      }
      leftSelection = [];
      rightSelection = [];
      refreshCounter += 1;
    } catch (err: unknown) {
      toast.error(err, 'Transfer failed');
    } finally {
      isTransferring = false;
      transferStatus = null;
    }
  }
</script>

<div class="flex h-full flex-col bg-[#0a0a0c] p-3 gap-2.5 overflow-hidden">
  <!-- Dual-Pane Transfer Bar -->
  <div class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[#20202b] bg-[#121218] px-3 py-2 shrink-0">
    <div class="flex items-center gap-2">
      <span class="text-xs font-semibold text-white">SFTP Dual-Pane</span>
      {#if transferStatus}
        <span class="rounded-full bg-emerald-500/15 border border-emerald-500/30 px-2.5 py-0.5 text-[11px] text-emerald-300 animate-pulse">
          {transferStatus}
        </span>
      {:else}
        <span class="text-[11px] text-neutral-400 hidden sm:inline">
          Select files in either pane to Copy or Move across Local & Remote hosts
        </span>
      {/if}
    </div>

    <div class="flex items-center gap-1.5">
      {#if platform.isMobile}
        <div class="flex rounded-lg border border-[#262635] bg-[#0c0c10] p-0.5 text-[11px]">
          <button
            type="button"
            onclick={() => (mobileActivePane = 'left')}
            class="rounded-md px-2.5 py-1 transition {mobileActivePane === 'left' ? 'bg-emerald-500/20 text-emerald-300 font-medium' : 'text-neutral-400'}"
          >
            Left
          </button>
          <button
            type="button"
            onclick={() => (mobileActivePane = 'right')}
            class="rounded-md px-2.5 py-1 transition {mobileActivePane === 'right' ? 'bg-emerald-500/20 text-emerald-300 font-medium' : 'text-neutral-400'}"
          >
            Right
          </button>
        </div>
      {/if}

      {#if leftSelection.length > 0}
        <button
          type="button"
          disabled={isTransferring || !rightTargetId}
          onclick={() => transferFiles('left-to-right', false)}
          class="flex items-center gap-1 rounded-lg bg-emerald-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-emerald-500 disabled:opacity-40 transition"
        >
          <Copy class="h-3 w-3" /> Copy ({leftSelection.length}) <ArrowRight class="h-3 w-3" />
        </button>
        <button
          type="button"
          disabled={isTransferring || !rightTargetId}
          onclick={() => transferFiles('left-to-right', true)}
          class="flex items-center gap-1 rounded-lg border border-amber-500/40 bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-300 hover:bg-amber-500/25 disabled:opacity-40 transition"
        >
          <Scissors class="h-3 w-3" /> Move <ArrowRight class="h-3 w-3" />
        </button>
      {/if}

      {#if rightSelection.length > 0}
        <button
          type="button"
          disabled={isTransferring || !leftTargetId}
          onclick={() => transferFiles('right-to-left', false)}
          class="flex items-center gap-1 rounded-lg bg-emerald-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-emerald-500 disabled:opacity-40 transition"
        >
          <ArrowLeft class="h-3 w-3" /> Copy ({rightSelection.length})
        </button>
        <button
          type="button"
          disabled={isTransferring || !leftTargetId}
          onclick={() => transferFiles('right-to-left', true)}
          class="flex items-center gap-1 rounded-lg border border-amber-500/40 bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-300 hover:bg-amber-500/25 disabled:opacity-40 transition"
        >
          <ArrowLeft class="h-3 w-3" /> Move
        </button>
      {/if}
    </div>
  </div>

  <!-- Panes Container -->
  {#if platform.isMobile}
    <div class="flex-1 min-h-0">
      {#if mobileActivePane === 'left'}
        <SftpPane
          paneTitle="Left"
          {hosts}
          targetId={leftTargetId}
          onTargetChange={(id) => (leftTargetId = id)}
          selectedPaths={leftSelection}
          onSelectionChange={(paths) => (leftSelection = paths)}
          currentPath={leftPath}
          onPathChange={(p) => (leftPath = p)}
          onItemsLoaded={(items) => (leftItems = items)}
          refreshSignal={refreshCounter}
        />
      {:else}
        <SftpPane
          paneTitle="Right"
          {hosts}
          targetId={rightTargetId}
          onTargetChange={(id) => (rightTargetId = id)}
          selectedPaths={rightSelection}
          onSelectionChange={(paths) => (rightSelection = paths)}
          currentPath={rightPath}
          onPathChange={(p) => (rightPath = p)}
          onItemsLoaded={(items) => (rightItems = items)}
          refreshSignal={refreshCounter}
        />
      {/if}
    </div>
  {:else}
    <div class="grid flex-1 grid-cols-2 gap-2.5 min-h-0">
      <SftpPane
        paneTitle="Left"
        {hosts}
        targetId={leftTargetId}
        onTargetChange={(id) => (leftTargetId = id)}
        selectedPaths={leftSelection}
        onSelectionChange={(paths) => (leftSelection = paths)}
        currentPath={leftPath}
        onPathChange={(p) => (leftPath = p)}
        onItemsLoaded={(items) => (leftItems = items)}
        refreshSignal={refreshCounter}
      />
      <SftpPane
        paneTitle="Right"
        {hosts}
        targetId={rightTargetId}
        onTargetChange={(id) => (rightTargetId = id)}
        selectedPaths={rightSelection}
        onSelectionChange={(paths) => (rightSelection = paths)}
        currentPath={rightPath}
        onPathChange={(p) => (rightPath = p)}
        onItemsLoaded={(items) => (rightItems = items)}
        refreshSignal={refreshCounter}
      />
    </div>
  {/if}
</div>
