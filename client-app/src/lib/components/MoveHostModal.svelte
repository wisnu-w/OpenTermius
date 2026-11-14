<script lang="ts">
  import { X, FolderInput, Folder, Users, FolderLock, ArrowRight, Check } from 'lucide-svelte';
  import type { ApiVault } from '../api/client';

  interface Props {
    isOpen: boolean;
    host: any;
    currentVault?: ApiVault | null;
    vaults: ApiVault[];
    availableFolders: string[];
    onClose: () => void;
    onMoveToFolder: (folder: string) => void;
    onMoveToVault: (targetVaultId: string) => void;
  }

  let {
    isOpen,
    host,
    currentVault,
    vaults,
    availableFolders,
    onClose,
    onMoveToFolder,
    onMoveToVault
  }: Props = $props();

  let activeTab = $state<'folder' | 'vault'>('folder');
  let selectedFolder = $state('');
  let customFolder = $state('');
  let selectedVaultId = $state('');

  $effect(() => {
    if (isOpen && host) {
      selectedFolder = host.group || 'Default';
      customFolder = '';
      const otherVaults = vaults.filter((v) => v.id !== currentVault?.id);
      selectedVaultId = otherVaults.length > 0 ? otherVaults[0].id : '';
    }
  });

  function handleFolderSubmit(e: SubmitEvent) {
    e.preventDefault();
    const target = customFolder.trim() || selectedFolder.trim() || 'Default';
    onMoveToFolder(target);
    onClose();
  }

  function handleVaultSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!selectedVaultId) return;
    onMoveToVault(selectedVaultId);
    onClose();
  }
</script>

{#if isOpen && host}
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/75 backdrop-blur-sm p-0 sm:p-4 overflow-y-auto">
    <div class="w-full max-w-md max-h-[92dvh] sm:max-h-[90vh] overflow-y-auto rounded-t-3xl sm:rounded-2xl border border-[#28283a] bg-[#14141e] p-4 pb-[calc(1.25rem+var(--sab))] sm:p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-150 sm:my-auto">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            <FolderInput class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Move Host</h3>
            <p class="text-xs text-neutral-400">Host: <span class="text-white font-medium">{host.label}</span></p>
          </div>
        </div>
        <button onclick={onClose} class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#20202c]">
          <X class="h-4 w-4" />
        </button>
      </div>

      <!-- Mode Selector Tabs -->
      <div class="grid grid-cols-2 gap-1 rounded-xl bg-[#0d0d14] p-1 border border-[#222230] text-xs font-medium">
        <button
          type="button"
          onclick={() => (activeTab = 'folder')}
          class="flex items-center justify-center gap-2 rounded-lg py-2 transition {activeTab === 'folder' ? 'bg-[#1e1e2c] text-white shadow' : 'text-neutral-400 hover:text-white'}"
        >
          <Folder class="h-3.5 w-3.5" />
          <span>Move to Folder</span>
        </button>
        <button
          type="button"
          onclick={() => (activeTab = 'vault')}
          class="flex items-center justify-center gap-2 rounded-lg py-2 transition {activeTab === 'vault' ? 'bg-[#1e1e2c] text-white shadow' : 'text-neutral-400 hover:text-white'}"
        >
          <FolderLock class="h-3.5 w-3.5" />
          <span>Move to Vault</span>
        </button>
      </div>

      {#if activeTab === 'folder'}
        <form onsubmit={handleFolderSubmit} class="space-y-4 text-xs">
          <div>
            <label for="folder-select" class="block text-neutral-300 font-medium mb-1.5">Choose Existing Folder</label>
            <div class="space-y-1.5 max-h-36 overflow-y-auto pr-1">
              {#each availableFolders as folder}
                <button
                  type="button"
                  onclick={() => {
                    selectedFolder = folder;
                    customFolder = '';
                  }}
                  class="flex w-full items-center justify-between rounded-lg border border-[#20202e] bg-[#0d0d14] px-3 py-2 text-left transition {selectedFolder === folder && !customFolder ? 'border-emerald-500 bg-emerald-500/10 text-emerald-300' : 'text-neutral-300 hover:bg-[#151520]'}"
                >
                  <div class="flex items-center gap-2">
                    <Folder class="h-3.5 w-3.5 text-neutral-400" />
                    <span>{folder}</span>
                  </div>
                  {#if selectedFolder === folder && !customFolder}
                    <Check class="h-3.5 w-3.5 text-emerald-400" />
                  {/if}
                </button>
              {/each}
            </div>
          </div>

          <div>
            <label for="custom-folder" class="block text-neutral-300 font-medium mb-1.5">Or Create New Folder</label>
            <input
              id="custom-folder"
              type="text"
              bind:value={customFolder}
              placeholder="e.g. Asia-Pacific-Nodes"
              class="w-full rounded-lg border border-[#262635] bg-[#0d0d14] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
            />
          </div>

          <div class="flex justify-end gap-2 pt-3 border-t border-[#20202c]">
            <button
              type="button"
              onclick={onClose}
              class="rounded-lg bg-[#20202c] px-4 py-2 font-semibold text-neutral-300 hover:bg-[#282838]"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="rounded-lg bg-emerald-600 px-4 py-2 font-semibold text-white shadow hover:bg-emerald-500 transition"
            >
              Update Folder
            </button>
          </div>
        </form>
      {:else}
        <form onsubmit={handleVaultSubmit} class="space-y-4 text-xs">
          <div>
            <label for="vault-select" class="block text-neutral-300 font-medium mb-1.5">Target Destination Vault</label>
            {#if vaults.filter((v) => v.id !== currentVault?.id).length === 0}
              <p class="text-neutral-400 p-3 rounded-lg bg-[#0d0d14] border border-[#20202e]">
                No other vaults available. Create a Team Vault in the "Vaults" menu first to transfer hosts.
              </p>
            {:else}
              <div class="space-y-2">
                {#each vaults.filter((v) => v.id !== currentVault?.id) as v (v.id)}
                  <button
                    type="button"
                    onclick={() => (selectedVaultId = v.id)}
                    class="flex w-full items-center justify-between rounded-lg border border-[#20202e] bg-[#0d0d14] px-3 py-2.5 text-left transition {selectedVaultId === v.id ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300' : 'text-neutral-300 hover:bg-[#151520]'}"
                  >
                    <div class="flex items-center gap-2.5">
                      {#if v.type === 'team'}
                        <Users class="h-4 w-4 text-indigo-400 shrink-0" />
                      {:else}
                        <FolderLock class="h-4 w-4 text-emerald-400 shrink-0" />
                      {/if}
                      <div>
                        <p class="font-semibold text-white">{v.name}</p>
                        <p class="text-[10px] text-neutral-400 capitalize">{v.type} Vault</p>
                      </div>
                    </div>
                    {#if selectedVaultId === v.id}
                      <Check class="h-4 w-4 text-indigo-400" />
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}
          </div>

          <div class="rounded-lg bg-indigo-950/30 border border-indigo-500/20 p-3 text-[11px] text-indigo-300">
            Moving to a Team Vault will make this host accessible to all invited team members in that vault.
          </div>

          <div class="flex justify-end gap-2 pt-3 border-t border-[#20202c]">
            <button
              type="button"
              onclick={onClose}
              class="rounded-lg bg-[#20202c] px-4 py-2 font-semibold text-neutral-300 hover:bg-[#282838]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!selectedVaultId}
              class="rounded-lg bg-indigo-600 px-4 py-2 font-semibold text-white shadow hover:bg-indigo-500 transition disabled:opacity-50"
            >
              Transfer to Vault
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}
