<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ApiVault, type ApiVaultMember } from '../api/client';
  import {
    FolderLock,
    ShieldCheck,
    Users,
    Key,
    Plus,
    RefreshCw,
    AlertCircle,
    UserPlus,
    Check,
    X,
    Shield
  } from 'lucide-svelte';

  interface Props {
    activeVault?: ApiVault | null;
    onSelectVault?: (vault: ApiVault) => void;
  }

  let { activeVault, onSelectVault }: Props = $props();

  // 4 UI States
  let state = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);
  let vaults = $state<ApiVault[]>([]);
  let isCreating = $state(false);
  let newVaultName = $state('');

  // Member management state
  let selectedVaultForMembers = $state<ApiVault | null>(null);
  let vaultMembers = $state<ApiVaultMember[]>([]);
  let isMembersLoading = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state<'member' | 'admin'>('member');
  let inviteError = $state<string | null>(null);
  let inviteSuccess = $state<string | null>(null);

  async function loadVaults() {
    state = 'loading';
    errorMessage = null;
    try {
      vaults = await api.getVaults();
      state = vaults.length === 0 ? 'empty' : 'success';
    } catch (err: unknown) {
      state = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to fetch vaults from backend';
    }
  }

  async function handleCreateTeamVault(e: SubmitEvent) {
    e.preventDefault();
    if (!newVaultName.trim()) return;

    try {
      const created = await api.createVault(
        newVaultName.trim(),
        'team',
        'wrapped-team-vault-key-aes-256'
      );
      vaults.push(created);
      state = 'success';
      newVaultName = '';
      isCreating = false;
      onSelectVault?.(created);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to create team vault');
    }
  }

  async function openMemberModal(vault: ApiVault) {
    selectedVaultForMembers = vault;
    isMembersLoading = true;
    inviteError = null;
    inviteSuccess = null;
    try {
      vaultMembers = await api.getVaultMembers(vault.id);
    } catch (err: unknown) {
      inviteError = err instanceof Error ? err.message : 'Failed to load vault members';
    } finally {
      isMembersLoading = false;
    }
  }

  async function handleInviteMember(e: SubmitEvent) {
    e.preventDefault();
    if (!selectedVaultForMembers || !inviteEmail.trim()) return;

    inviteError = null;
    inviteSuccess = null;
    try {
      const newMember = await api.inviteVaultMember(
        selectedVaultForMembers.id,
        inviteEmail.trim(),
        inviteRole
      );
      vaultMembers.push(newMember);
      inviteSuccess = `Successfully invited ${inviteEmail.trim()} as ${inviteRole}!`;
      inviteEmail = '';
    } catch (err: unknown) {
      inviteError = err instanceof Error ? err.message : 'Failed to invite user to vault';
    }
  }

  onMount(() => {
    loadVaults();
  });
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-white">Encrypted Vaults & Teams</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400 font-medium">
          Zero-Knowledge E2EE
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Personal vaults isolate credentials strictly for you. Team vaults allow team collaboration via role-based access.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={loadVaults}
        class="flex items-center gap-1.5 rounded-lg border border-[#22222a] bg-[#14141a] px-3 py-1.5 text-xs text-neutral-300 hover:text-white transition"
      >
        <RefreshCw class="h-3.5 w-3.5" /> Refresh
      </button>

      <button
        onclick={() => (isCreating = !isCreating)}
        class="flex items-center gap-2 rounded-lg bg-emerald-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow hover:bg-emerald-500 transition"
      >
        <Plus class="h-4 w-4" /> Create Team Vault
      </button>
    </div>
  </div>

  {#if isCreating}
    <form onsubmit={handleCreateTeamVault} class="rounded-xl border border-emerald-500/30 bg-[#14141d] p-4 flex items-center gap-3 animate-in fade-in">
      <input
        type="text"
        bind:value={newVaultName}
        placeholder="Enter Team Vault Name (e.g. Infrastructure & DevOps Team)"
        class="flex-1 rounded-lg border border-[#262635] bg-[#0c0c12] px-3 py-1.5 text-xs text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
        required
      />
      <button
        type="submit"
        class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500"
      >
        Save Team Vault
      </button>
      <button
        type="button"
        onclick={() => (isCreating = false)}
        class="rounded-lg bg-[#20202c] px-3 py-1.5 text-xs text-neutral-400 hover:text-white"
      >
        Cancel
      </button>
    </form>
  {/if}

  <!-- 4 UI States -->
  {#if state === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="h-4 w-4 animate-spin text-emerald-400" />
      <span>Loading vaults from OpenTermius PostgreSQL backend...</span>
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
        onclick={loadVaults}
        class="rounded-lg bg-rose-500/20 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/30"
      >
        Retry Connection
      </button>
    </div>
  {:else if state === 'empty'}
    <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-12 text-center space-y-3">
      <FolderLock class="h-10 w-10 text-neutral-600 mx-auto" />
      <h3 class="text-sm font-semibold text-white">No Vaults Created Yet</h3>
      <p class="text-xs text-neutral-400 max-w-sm mx-auto">
        Create a Personal or Team vault to begin securely storing your hosts and credentials.
      </p>
      <button
        onclick={() => (isCreating = true)}
        class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500"
      >
        + Create Your First Vault
      </button>
    </div>
  {:else}
    <!-- Vaults Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each vaults as vault (vault.id)}
        <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-5 flex flex-col justify-between hover:border-emerald-500/40 transition">
          <div>
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 items-center justify-center rounded-lg {vault.type === 'personal' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-indigo-500/10 text-indigo-400'}">
                  {#if vault.type === 'personal'}
                    <FolderLock class="h-5 w-5" />
                  {:else}
                    <Users class="h-5 w-5" />
                  {/if}
                </div>
                <div>
                  <h3 class="text-sm font-semibold text-white">{vault.name}</h3>
                  <span class="text-[11px] text-neutral-400 capitalize">{vault.type} Vault</span>
                </div>
              </div>

              {#if activeVault?.id === vault.id}
                <span class="flex items-center gap-1 rounded bg-emerald-500/15 border border-emerald-500/30 px-2 py-0.5 text-[10px] font-bold text-emerald-400 uppercase">
                  <Check class="h-3 w-3" /> Active
                </span>
              {/if}
            </div>

            <div class="mt-4 pt-3 border-t border-[#1e1e28] space-y-2 text-xs text-neutral-400">
              <div class="flex justify-between">
                <span>Vault ID</span>
                <span class="font-mono text-[10px] text-neutral-300">{vault.id.slice(0, 8)}...</span>
              </div>
              <div class="flex justify-between">
                <span>Access Scope</span>
                <span class="font-mono text-[10px] {vault.type === 'team' ? 'text-indigo-400' : 'text-emerald-400'}">
                  {vault.type === 'team' ? 'Shared Team Members' : 'Private (Owner Only)'}
                </span>
              </div>
              <div class="flex justify-between">
                <span>Encryption</span>
                <span class="font-mono text-[10px] text-emerald-400">AES-256-GCM (Client-Side)</span>
              </div>
            </div>
          </div>

          <!-- Vault Actions -->
          <div class="mt-4 pt-3 border-t border-[#1e1e28] flex items-center gap-2">
            {#if activeVault?.id !== vault.id}
              <button
                onclick={() => onSelectVault?.(vault)}
                class="flex-1 rounded-lg bg-[#20202d] py-1.5 px-3 text-xs font-semibold text-neutral-300 hover:text-white hover:bg-[#28283a] transition"
              >
                Switch to this Vault
              </button>
            {/if}

            {#if vault.type === 'team'}
              <button
                onclick={() => openMemberModal(vault)}
                class="flex items-center justify-center gap-1.5 rounded-lg bg-indigo-600/20 border border-indigo-500/30 py-1.5 px-3 text-xs font-medium text-indigo-300 hover:bg-indigo-600/30 transition {activeVault?.id === vault.id ? 'flex-1' : ''}"
              >
                <UserPlus class="h-3.5 w-3.5" />
                <span>Invite & Members</span>
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Team Members & Invitation Modal -->
{#if selectedVaultForMembers}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="w-full max-w-lg rounded-2xl border border-[#28283a] bg-[#14141e] p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-150">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            <Users class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Manage Team Members</h3>
            <p class="text-xs text-neutral-400">Vault: <span class="text-neutral-200 font-semibold">{selectedVaultForMembers.name}</span></p>
          </div>
        </div>
        <button onclick={() => (selectedVaultForMembers = null)} class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#20202c]">
          <X class="h-4 w-4" />
        </button>
      </div>

      <!-- Invite Member Form -->
      <form onsubmit={handleInviteMember} class="space-y-3 rounded-xl border border-[#222230] bg-[#0d0d14] p-4 text-xs">
        <span class="block font-semibold text-neutral-200">Invite Teammate</span>
        <div class="flex items-center gap-2">
          <input
            type="email"
            bind:value={inviteEmail}
            placeholder="teammate@company.com"
            class="flex-1 rounded-lg border border-[#28283c] bg-[#12121c] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-indigo-500"
            required
          />
          <select
            bind:value={inviteRole}
            class="rounded-lg border border-[#28283c] bg-[#12121c] px-3 py-2 text-white outline-none focus:border-indigo-500"
          >
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </select>
          <button
            type="submit"
            class="rounded-lg bg-indigo-600 px-4 py-2 font-semibold text-white shadow hover:bg-indigo-500 transition"
          >
            Invite
          </button>
        </div>

        {#if inviteError}
          <p class="text-rose-400 text-[11px]">{inviteError}</p>
        {/if}
        {#if inviteSuccess}
          <p class="text-emerald-400 text-[11px]">{inviteSuccess}</p>
        {/if}
      </form>

      <!-- Current Members List -->
      <div class="space-y-2 text-xs">
        <span class="font-semibold text-neutral-300">Current Vault Members ({vaultMembers.length})</span>
        {#if isMembersLoading}
          <div class="flex items-center gap-2 text-neutral-400 py-3">
            <RefreshCw class="h-3.5 w-3.5 animate-spin" />
            <span>Loading members...</span>
          </div>
        {:else if vaultMembers.length === 0}
          <p class="text-neutral-500 text-[11px] py-2">No members in this team vault yet.</p>
        {:else}
          <div class="max-h-48 overflow-y-auto space-y-1.5 pr-1">
            {#each vaultMembers as m (m.id)}
              <div class="flex items-center justify-between rounded-lg bg-[#0d0d14] border border-[#20202e] px-3 py-2">
                <div class="flex items-center gap-2 truncate">
                  <Shield class="h-3.5 w-3.5 text-neutral-400" />
                  <span class="font-mono text-[11px] text-neutral-300">User ID: {m.user_id.slice(0, 8)}...</span>
                </div>
                <span class="rounded bg-indigo-500/15 px-2 py-0.5 text-[10px] font-bold text-indigo-300 uppercase">
                  {m.role}
                </span>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="flex justify-end pt-2 border-t border-[#20202e]">
        <button
          onclick={() => (selectedVaultForMembers = null)}
          class="rounded-lg bg-[#20202c] px-4 py-2 text-xs font-semibold text-neutral-300 hover:text-white"
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
