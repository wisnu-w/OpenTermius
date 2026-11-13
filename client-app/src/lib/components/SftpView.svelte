<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ApiSftpFile, type SftpHostConfig } from '../api/client';
  import {
    Folder,
    FolderPlus,
    FileText,
    FileCode,
    FileArchive,
    Upload,
    Download,
    Trash2,
    RefreshCw,
    CornerLeftUp,
    Server,
    Search,
    AlertCircle,
    X,
    HardDrive,
    Check
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
  }

  interface Props {
    hosts: HostItem[];
    initialHostId?: string;
  }

  let { hosts = [], initialHostId = '' }: Props = $props();

  let selectedHostId = $state<string>('');
  let currentHost = $derived(hosts.find((h) => h.id === selectedHostId) || null);

  let currentPath = $state<string>('');
  let pathInputValue = $state<string>('');
  let items = $state<ApiSftpFile[]>([]);
  let searchQuery = $state<string>('');

  let loadState = $state<'idle' | 'loading' | 'success' | 'empty' | 'error'>('idle');
  let errorMessage = $state<string | null>(null);

  let isUploading = $state(false);
  let isNewFolderOpen = $state(false);
  let newFolderName = $state('');
  let fileInputRef: HTMLInputElement | null = null;

  // Reactively sync initialHostId or default host if changed
  $effect(() => {
    if (initialHostId && initialHostId !== selectedHostId) {
      selectedHostId = initialHostId;
      currentPath = '';
      loadDirectory('');
    } else if (!selectedHostId && hosts.length > 0) {
      selectedHostId = hosts[0].id;
      currentPath = '';
      loadDirectory('');
    }
  });

  function getHostConfig(): SftpHostConfig | null {
    if (!currentHost) return null;
    return {
      address: currentHost.address,
      port: currentHost.port || 22,
      username: currentHost.username,
      authType: currentHost.authType || 'password',
      password: currentHost.password,
      privateKey: currentHost.privateKey,
      passphrase: currentHost.passphrase
    };
  }

  async function loadDirectory(targetPath: string = '') {
    const config = getHostConfig();
    if (!config) {
      loadState = 'idle';
      return;
    }

    loadState = 'loading';
    errorMessage = null;

    try {
      const res = await api.sftpList(config, targetPath);
      currentPath = res.path;
      pathInputValue = res.path;
      items = res.items || [];
      loadState = items.length === 0 ? 'empty' : 'success';
    } catch (err: unknown) {
      loadState = 'error';
      errorMessage = err instanceof Error ? err.message : 'SFTP connection failed';
    }
  }

  function handleNavigateUp() {
    if (!currentPath || currentPath === '/' || currentPath === '.') return;
    const parts = currentPath.split('/').filter(Boolean);
    parts.pop();
    const upPath = '/' + parts.join('/');
    loadDirectory(upPath);
  }

  function handlePathSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (pathInputValue.trim()) {
      loadDirectory(pathInputValue.trim());
    }
  }

  async function handleDownload(item: ApiSftpFile) {
    const config = getHostConfig();
    if (!config) return;
    try {
      await api.sftpDownload(config, item.path, item.name);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Download failed');
    }
  }

  async function handleDelete(item: ApiSftpFile) {
    const config = getHostConfig();
    if (!config) return;
    const confirmed = confirm(`Delete ${item.isDir ? 'directory' : 'file'} "${item.name}"?`);
    if (!confirmed) return;

    try {
      await api.sftpDelete(config, item.path);
      await loadDirectory(currentPath);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Delete failed');
    }
  }

  async function handleCreateFolder() {
    const config = getHostConfig();
    if (!config || !newFolderName.trim()) return;

    const folderPath = currentPath.replace(/\/+$/, '') + '/' + newFolderName.trim();
    try {
      await api.sftpMkdir(config, folderPath);
      newFolderName = '';
      isNewFolderOpen = false;
      await loadDirectory(currentPath);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to create directory');
    }
  }

  async function handleFileUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    const config = getHostConfig();
    if (!config) return;

    isUploading = true;
    try {
      await api.sftpUpload(config, currentPath, file);
      input.value = '';
      await loadDirectory(currentPath);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      isUploading = false;
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function getFileIcon(filename: string, isDir: boolean) {
    if (isDir) return Folder;
    const ext = filename.split('.').pop()?.toLowerCase();
    if (['zip', 'tar', 'gz', 'bz2', '7z', 'rar'].includes(ext || '')) return FileArchive;
    if (['go', 'js', 'ts', 'svelte', 'py', 'sh', 'json', 'yml', 'yaml', 'html', 'css', 'sql'].includes(ext || '')) return FileCode;
    return FileText;
  }

  let filteredItems = $derived(
    items.filter((item) => item.name.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  onMount(() => {
    if (selectedHostId) {
      loadDirectory('');
    }
  });
</script>

<div class="flex h-full w-full flex-col bg-[#0d0d12] text-neutral-100 select-none overflow-hidden">
  <!-- Top Navigation & Host Selector Bar -->
  <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2.5 border-b border-[#20202c] bg-[#111118] px-3 sm:px-4 py-2.5 shrink-0">
    <!-- Left: Host Selector Dropdown -->
    <div class="flex items-center gap-2">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shrink-0">
        <Server class="h-4 w-4" />
      </div>

      <div class="min-w-0">
        {#if hosts.length > 0}
          <select
            bind:value={selectedHostId}
            onchange={() => {
              currentPath = '';
              loadDirectory('');
            }}
            class="rounded-lg border border-[#262638] bg-[#161622] px-2.5 py-1 text-xs font-semibold text-white outline-none focus:border-emerald-500 transition cursor-pointer"
          >
            {#each hosts as host}
              <option value={host.id}>
                {host.label} ({host.username}@{host.address})
              </option>
            {/each}
          </select>
        {:else}
          <span class="text-xs text-neutral-400">No hosts configured</span>
        {/if}
      </div>
    </div>

    <!-- Right: Quick Actions -->
    <div class="flex items-center gap-2">
      <!-- Hidden File Input for Upload -->
      <input
        bind:this={fileInputRef}
        type="file"
        onchange={handleFileUpload}
        class="hidden"
      />

      <button
        type="button"
        onclick={() => fileInputRef?.click()}
        disabled={loadState === 'loading' || isUploading || !currentHost}
        class="flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition shadow disabled:opacity-50"
      >
        <Upload class="h-3.5 w-3.5" />
        <span>{isUploading ? 'Uploading...' : 'Upload File'}</span>
      </button>

      <button
        type="button"
        onclick={() => (isNewFolderOpen = true)}
        disabled={loadState === 'loading' || !currentHost}
        class="flex items-center gap-1.5 rounded-lg border border-[#262638] bg-[#161622] px-2.5 py-1.5 text-xs font-semibold text-neutral-200 hover:text-white hover:border-[#38384d] transition disabled:opacity-50"
      >
        <FolderPlus class="h-3.5 w-3.5 text-emerald-400" />
        <span class="hidden sm:inline">New Folder</span>
      </button>

      <button
        type="button"
        onclick={() => loadDirectory(currentPath)}
        disabled={loadState === 'loading' || !currentHost}
        class="flex items-center justify-center h-8 w-8 rounded-lg border border-[#262638] bg-[#161622] text-neutral-400 hover:text-white hover:border-[#38384d] transition disabled:opacity-50"
        title="Refresh directory"
      >
        <RefreshCw class="h-3.5 w-3.5 {loadState === 'loading' ? 'animate-spin text-emerald-400' : ''}" />
      </button>
    </div>
  </div>

  <!-- Path & Search Bar -->
  <div class="flex items-center gap-2 border-b border-[#1c1c28] bg-[#0f0f15] px-3 sm:px-4 py-2 shrink-0">
    <!-- Navigate Up Button -->
    <button
      type="button"
      onclick={handleNavigateUp}
      disabled={!currentPath || currentPath === '/' || currentPath === '.'}
      class="flex h-8 w-8 items-center justify-center rounded-lg border border-[#252538] bg-[#14141e] text-neutral-300 hover:text-white hover:border-[#38384d] transition disabled:opacity-30 disabled:pointer-events-none shrink-0"
      title="Go up to parent directory"
    >
      <CornerLeftUp class="h-4 w-4" />
    </button>

    <!-- Path Form -->
    <form onsubmit={handlePathSubmit} class="flex-1 min-w-0">
      <input
        type="text"
        bind:value={pathInputValue}
        placeholder="/var/www/html"
        class="w-full rounded-lg border border-[#222232] bg-[#13131c] px-3 py-1.5 text-xs font-mono text-neutral-200 placeholder-neutral-500 outline-none focus:border-emerald-500 transition"
      />
    </form>

    <!-- Search in current folder -->
    <div class="relative w-36 sm:w-48 shrink-0">
      <Search class="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-neutral-500" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Filter..."
        class="w-full rounded-lg border border-[#222232] bg-[#13131c] pl-8 pr-3 py-1.5 text-xs text-white placeholder-neutral-500 outline-none focus:border-emerald-500 transition"
      />
    </div>
  </div>

  <!-- Main File Explorer View Area -->
  <div class="flex-1 overflow-y-auto">
    {#if loadState === 'loading'}
      <div class="flex h-64 flex-col items-center justify-center gap-3 text-xs text-neutral-400">
        <RefreshCw class="h-6 w-6 animate-spin text-emerald-400" />
        <span>Connecting to SFTP server and reading files...</span>
      </div>
    {:else if loadState === 'error'}
      <div class="p-6">
        <div class="max-w-md mx-auto rounded-2xl border border-rose-500/30 bg-rose-950/20 p-5 text-center space-y-3">
          <AlertCircle class="h-8 w-8 text-rose-400 mx-auto" />
          <h4 class="text-sm font-bold text-white">SFTP Connection Error</h4>
          <p class="text-xs text-rose-300">{errorMessage}</p>
          <button
            type="button"
            onclick={() => loadDirectory(currentPath)}
            class="rounded-lg bg-rose-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-rose-500 transition"
          >
            Retry
          </button>
        </div>
      </div>
    {:else if loadState === 'empty' || filteredItems.length === 0}
      <div class="flex h-64 flex-col items-center justify-center text-center p-6 space-y-3">
        <Folder class="h-10 w-10 text-neutral-600" />
        <p class="text-sm font-semibold text-white">
          {searchQuery ? 'No matching files found' : 'Directory is empty'}
        </p>
        <p class="text-xs text-neutral-400">
          {searchQuery ? 'Try another search term.' : 'Upload a file or create a subfolder.'}
        </p>
        {#if !searchQuery}
          <button
            type="button"
            onclick={() => fileInputRef?.click()}
            class="rounded-lg bg-emerald-600 px-3.5 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition"
          >
            Upload File
          </button>
        {/if}
      </div>
    {:else}
      <!-- Files Table -->
      <table class="w-full text-left text-xs border-collapse">
        <thead class="sticky top-0 bg-[#12121a] border-b border-[#20202c] text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">
          <tr>
            <th class="py-2.5 px-4">Name</th>
            <th class="py-2.5 px-3 w-28 hidden sm:table-cell">Size</th>
            <th class="py-2.5 px-3 w-32 hidden md:table-cell">Permissions</th>
            <th class="py-2.5 px-3 w-40 hidden lg:table-cell">Modified</th>
            <th class="py-2.5 px-4 w-24 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[#181824]">
          {#each filteredItems as item (item.path)}
            {@const IconComponent = getFileIcon(item.name, item.isDir)}
            <tr
              class="group hover:bg-[#161622] transition cursor-pointer"
              onclick={() => item.isDir && loadDirectory(item.path)}
            >
              <!-- Name & Icon -->
              <td class="py-2.5 px-4 flex items-center gap-2.5 font-medium truncate">
                <IconComponent class="h-4 w-4 shrink-0 {item.isDir ? 'text-amber-400' : 'text-neutral-400'}" />
                <span class="truncate {item.isDir ? 'text-white font-semibold group-hover:text-amber-300' : 'text-neutral-200 group-hover:text-white'}">
                  {item.name}
                </span>
              </td>

              <!-- Size -->
              <td class="py-2.5 px-3 text-neutral-400 font-mono text-[11px] hidden sm:table-cell">
                {item.isDir ? '-' : formatBytes(item.size)}
              </td>

              <!-- Permissions -->
              <td class="py-2.5 px-3 text-neutral-400 font-mono text-[11px] hidden md:table-cell">
                {item.permissions}
              </td>

              <!-- Modified Date -->
              <td class="py-2.5 px-3 text-neutral-400 text-[11px] hidden lg:table-cell whitespace-nowrap">
                {item.modTime}
              </td>

              <!-- Actions -->
              <td class="py-2.5 px-4 text-right" onclick={(e) => e.stopPropagation()}>
                <div class="flex items-center justify-end gap-1">
                  {#if !item.isDir}
                    <button
                      type="button"
                      onclick={() => handleDownload(item)}
                      class="p-1 rounded text-neutral-400 hover:text-emerald-400 hover:bg-[#202030] transition"
                      title="Download file"
                    >
                      <Download class="h-3.5 w-3.5" />
                    </button>
                  {/if}

                  <button
                    type="button"
                    onclick={() => handleDelete(item)}
                    class="p-1 rounded text-neutral-400 hover:text-rose-400 hover:bg-[#202030] transition"
                    title="Delete item"
                  >
                    <Trash2 class="h-3.5 w-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <!-- Status Footer -->
  <div class="flex items-center justify-between border-t border-[#1c1c28] bg-[#0f0f15] px-4 py-2 text-[11px] text-neutral-400 shrink-0">
    <div class="flex items-center gap-2 truncate">
      <HardDrive class="h-3.5 w-3.5 text-neutral-500" />
      <span class="truncate font-mono">{currentPath || '/'}</span>
    </div>
    <div class="shrink-0 text-neutral-500">
      <span>{filteredItems.length} items</span>
    </div>
  </div>
</div>

<!-- New Folder Modal -->
{#if isNewFolderOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4 animate-in fade-in duration-100">
    <div class="w-full max-w-sm rounded-2xl border border-[#252538] bg-[#14141e] p-5 shadow-2xl space-y-4 animate-in zoom-in-95 duration-100">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <FolderPlus class="h-4 w-4 text-emerald-400" />
          <h4 class="text-sm font-bold text-white">Create New Folder</h4>
        </div>
        <button
          type="button"
          onclick={() => (isNewFolderOpen = false)}
          class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#202030]"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleCreateFolder(); }} class="space-y-3">
        <div>
          <label for="new-folder-name" class="block text-xs text-neutral-300 font-medium mb-1">Folder Name</label>
          <input
            id="new-folder-name"
            type="text"
            bind:value={newFolderName}
            placeholder="e.g. logs"
            class="w-full rounded-lg border border-[#252538] bg-[#0e0e16] px-3 py-2 text-xs text-white placeholder-neutral-500 outline-none focus:border-emerald-500 transition"
            required
          />
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onclick={() => (isNewFolderOpen = false)}
            class="rounded-lg bg-[#20202d] px-3 py-1.5 text-xs text-neutral-300 hover:bg-[#2a2a3c] transition"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition shadow"
          >
            Create
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
