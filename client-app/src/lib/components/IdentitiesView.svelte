<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ApiVaultItem } from '../api/client';
  import {
    UserCheck,
    Plus,
    RefreshCw,
    AlertCircle,
    Trash2,
    Lock,
    Key,
    Shield,
    X,
    User,
    Check
  } from 'lucide-svelte';

  interface Props {
    activeVaultId?: string;
  }

  let { activeVaultId }: Props = $props();

  // 4 UI States
  let state = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);

  interface IdentityItem {
    id: string;
    label: string;
    username: string;
    authType: 'password' | 'key';
    password?: string;
    keyId?: string;
    keyLabel?: string;
    createdAt: string;
  }

  let identities = $state<IdentityItem[]>([]);
  let availableKeys = $state<Array<{ id: string; label: string; privateKey: string; passphrase?: string }>>([]);

  let isAddOpen = $state(false);
  let isSaving = $state(false);
  let formError = $state<string | null>(null);

  let newLabel = $state('');
  let newUsername = $state('root');
  let newAuthType = $state<'password' | 'key'>('password');
  let newPassword = $state('');
  let selectedKeyId = $state('');

  async function getResolvedVaultId(): Promise<string | null> {
    if (activeVaultId) return activeVaultId;
    try {
      const vList = await api.getVaults();
      if (vList.length > 0) return vList[0].id;
    } catch {
      // ignore
    }
    return null;
  }

  async function loadData() {
    state = 'loading';
    errorMessage = null;

    try {
      const vaultId = await getResolvedVaultId();
      if (!vaultId) {
        state = 'empty';
        identities = [];
        return;
      }

      // 1. Fetch available keys in keychain for linking
      const keyItems = await api.getVaultItems(vaultId, 'key');
      availableKeys = keyItems.map((k) => {
        let privateKey = '';
        let passphrase = '';
        try {
          const parsed = JSON.parse(atob(k.ciphertext));
          privateKey = parsed.privateKey || '';
          passphrase = parsed.passphrase || '';
        } catch {}
        return {
          id: k.id,
          label: k.label,
          privateKey,
          passphrase
        };
      });

      // 2. Fetch identities
      const items = await api.getVaultItems(vaultId, 'identity');
      if (items.length === 0) {
        identities = [];
        state = 'empty';
        return;
      }

      identities = items.map((item) => {
        let username = 'root';
        let authType: 'password' | 'key' = 'password';
        let password = '';
        let keyId = '';
        let keyLabel = '';

        try {
          const parsed = JSON.parse(atob(item.ciphertext));
          username = parsed.username || 'root';
          authType = parsed.authType || 'password';
          password = parsed.password || '';
          keyId = parsed.keyId || '';
          const foundKey = availableKeys.find((k) => k.id === keyId);
          if (foundKey) keyLabel = foundKey.label;
        } catch {
          // fallback
        }

        return {
          id: item.id,
          label: item.label,
          username,
          authType,
          password,
          keyId,
          keyLabel,
          createdAt: item.created_at
        };
      });

      state = 'success';
    } catch (err: unknown) {
      state = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to load identities from vault';
    }
  }

  function openAddModal() {
    newLabel = '';
    newUsername = 'root';
    newAuthType = 'password';
    newPassword = '';
    selectedKeyId = availableKeys.length > 0 ? availableKeys[0].id : '';
    formError = null;
    isAddOpen = true;
  }

  async function handleSaveIdentity(e: SubmitEvent) {
    e.preventDefault();
    formError = null;

    const vaultId = await getResolvedVaultId();
    if (!vaultId) {
      formError = 'No active vault available. Please select or create a vault first.';
      return;
    }

    if (!newLabel.trim()) {
      formError = 'Identity label is required.';
      return;
    }

    if (!newUsername.trim()) {
      formError = 'Default username is required.';
      return;
    }

    isSaving = true;

    try {
      const selectedKey = availableKeys.find((k) => k.id === selectedKeyId);

      const payload = {
        username: newUsername.trim(),
        authType: newAuthType,
        password: newPassword,
        keyId: selectedKeyId,
        privateKey: selectedKey?.privateKey || '',
        passphrase: selectedKey?.passphrase || ''
      };

      const ciphertext = btoa(JSON.stringify(payload));
      const nonce = btoa(crypto.randomUUID().slice(0, 12));

      const created = await api.createVaultItem(vaultId, {
        kind: 'identity',
        label: newLabel.trim(),
        ciphertext,
        nonce
      });

      identities.unshift({
        id: created.id,
        label: newLabel.trim(),
        username: newUsername.trim(),
        authType: newAuthType,
        password: newPassword,
        keyId: selectedKeyId,
        keyLabel: selectedKey?.label,
        createdAt: created.created_at
      });

      state = 'success';
      isAddOpen = false;
      newLabel = '';
      newUsername = 'root';
      newPassword = '';
      selectedKeyId = '';
    } catch (err: unknown) {
      formError = err instanceof Error ? err.message : 'Failed to save identity to vault';
    } finally {
      isSaving = false;
    }
  }

  async function handleDeleteIdentity(id: string) {
    const vaultId = await getResolvedVaultId();
    if (!vaultId) return;
    if (!confirm('Are you sure you want to delete this identity?')) return;

    try {
      await api.deleteVaultItem(vaultId, id);
      identities = identities.filter((i) => i.id !== id);
      if (identities.length === 0) state = 'empty';
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to delete identity');
    }
  }

  $effect(() => {
    if (activeVaultId) {
      loadData();
    }
  });

  onMount(() => {
    loadData();
  });
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-white">Identities</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400 font-medium">
          Zero-Knowledge Vault
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Reusable credential profiles combining usernames and keys/passwords. Attach an identity to multiple hosts.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        type="button"
        onclick={loadData}
        class="flex items-center gap-1.5 rounded-lg border border-[#22222a] bg-[#14141a] px-3 py-1.5 text-xs text-neutral-300 hover:text-white transition cursor-pointer"
      >
        <RefreshCw class="h-3.5 w-3.5" /> Refresh
      </button>

      <button
        type="button"
        onclick={openAddModal}
        class="flex items-center gap-2 rounded-lg bg-emerald-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow hover:bg-emerald-500 transition cursor-pointer active:scale-95"
      >
        <Plus class="h-4 w-4" /> Add Identity
      </button>
    </div>
  </div>

  <!-- 4 UI States -->
  {#if state === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="h-4 w-4 animate-spin text-emerald-400" />
      <span>Loading identities from encrypted vault...</span>
    </div>
  {:else if state === 'error'}
    <div class="rounded-xl border border-rose-500/30 bg-rose-950/20 p-5 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <AlertCircle class="h-5 w-5 text-rose-400 shrink-0" />
        <div>
          <p class="text-xs font-semibold text-rose-300">Identity Loading Error</p>
          <p class="text-[11px] text-rose-400/80">{errorMessage}</p>
        </div>
      </div>
      <button
        type="button"
        onclick={loadData}
        class="rounded-lg bg-rose-500/20 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/30 cursor-pointer"
      >
        Retry
      </button>
    </div>
  {:else if state === 'empty' || identities.length === 0}
    <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-12 text-center space-y-3">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-[#1c1c28] border border-[#262638] text-neutral-400 mx-auto">
        <UserCheck class="h-6 w-6 text-neutral-500" />
      </div>
      <h3 class="text-sm font-semibold text-white">No Identities Saved Yet</h3>
      <p class="text-xs text-neutral-400 max-w-sm mx-auto">
        Create a reusable identity profile (e.g. "Ubuntu EC2 Admin" with username and attached Keychain key) to use across hosts.
      </p>
      <button
        type="button"
        onclick={openAddModal}
        class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition cursor-pointer active:scale-95"
      >
        + Add Your First Identity
      </button>
    </div>
  {:else}
    <!-- Identities Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each identities as identity (identity.id)}
        <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-5 flex flex-col justify-between hover:border-emerald-500/40 transition">
          <div>
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                  <UserCheck class="h-5 w-5" />
                </div>
                <div>
                  <h3 class="text-sm font-semibold text-white">{identity.label}</h3>
                  <p class="text-xs text-neutral-400 font-mono">user: <span class="text-neutral-200">{identity.username}</span></p>
                </div>
              </div>

              <button
                type="button"
                onclick={() => handleDeleteIdentity(identity.id)}
                class="rounded-lg p-1.5 text-neutral-500 hover:text-rose-400 hover:bg-[#20202d] transition cursor-pointer"
                title="Delete Identity"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>

            <!-- Auth Details -->
            <div class="mt-4 rounded-lg bg-[#0e0e14] border border-[#20202e] p-2.5 space-y-1 text-xs">
              <div class="flex justify-between items-center">
                <span class="text-neutral-400">Auth Method</span>
                <span class="rounded bg-[#1a1a26] px-2 py-0.5 text-[10px] font-semibold text-neutral-300 uppercase">
                  {identity.authType === 'key' ? 'SSH Key' : 'Password'}
                </span>
              </div>
              {#if identity.authType === 'key'}
                <div class="flex justify-between items-center pt-1 border-t border-[#1e1e28]">
                  <span class="text-neutral-400">Attached Key</span>
                  <span class="font-mono text-[10px] text-emerald-400">{identity.keyLabel || 'Linked Key'}</span>
                </div>
              {/if}
            </div>
          </div>

          <div class="mt-4 pt-3 border-t border-[#1e1e28] flex items-center justify-between text-[10px] text-neutral-500">
            <span>Created</span>
            <span>{new Date(identity.createdAt).toLocaleDateString()}</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Add Identity Modal -->
{#if isAddOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 overflow-y-auto">
    <div class="w-full max-w-lg rounded-2xl border border-[#28283a] bg-[#14141e] p-6 shadow-2xl space-y-4 animate-in fade-in zoom-in-95 duration-150 my-auto">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            <UserCheck class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Add Reusable Identity</h3>
            <p class="text-xs text-neutral-400">Create an identity profile to auto-fill credentials on hosts.</p>
          </div>
        </div>
        <button type="button" onclick={() => (isAddOpen = false)} class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#20202c] cursor-pointer">
          <X class="h-4 w-4" />
        </button>
      </div>

      {#if formError}
        <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
          {formError}
        </div>
      {/if}

      <form onsubmit={handleSaveIdentity} class="space-y-4 text-xs">
        <div>
          <label for="identity-label" class="block text-neutral-300 font-medium mb-1">Identity Label</label>
          <input
            id="identity-label"
            type="text"
            bind:value={newLabel}
            placeholder="e.g. AWS Production Ubuntu Admin"
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-indigo-500"
            required
          />
        </div>

        <div>
          <label for="identity-user" class="block text-neutral-300 font-medium mb-1">Default Username</label>
          <input
            id="identity-user"
            type="text"
            bind:value={newUsername}
            placeholder="ubuntu / root / ec2-user"
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 font-mono text-white placeholder-neutral-500 outline-none focus:border-indigo-500"
            required
          />
        </div>

        <!-- Auth Method Selector -->
        <div class="rounded-xl border border-[#252535] bg-[#0d0d16] p-3 space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-neutral-200 font-semibold">Credential Type</span>
            <div class="flex items-center gap-1 bg-[#1a1a24] p-0.5 rounded-lg border border-[#2a2a3a]">
              <button
                type="button"
                onclick={() => (newAuthType = 'password')}
                class="px-2.5 py-1 rounded-md text-[11px] font-medium transition cursor-pointer {newAuthType === 'password' ? 'bg-indigo-600 text-white' : 'text-neutral-400 hover:text-white'}"
              >
                Password
              </button>
              <button
                type="button"
                onclick={() => (newAuthType = 'key')}
                class="px-2.5 py-1 rounded-md text-[11px] font-medium transition cursor-pointer {newAuthType === 'key' ? 'bg-indigo-600 text-white' : 'text-neutral-400 hover:text-white'}"
              >
                Keychain Key
              </button>
            </div>
          </div>

          {#if newAuthType === 'password'}
            <div>
              <label for="identity-pass" class="block text-neutral-400 text-[11px] mb-1">Default Password</label>
              <input
                id="identity-pass"
                type="password"
                bind:value={newPassword}
                placeholder="Password for this identity"
                class="w-full rounded-lg border border-[#28283c] bg-[#090910] px-3 py-2 text-white placeholder-neutral-600 outline-none focus:border-indigo-500"
              />
            </div>
          {:else}
            <div>
              <label for="identity-key-select" class="block text-neutral-400 text-[11px] mb-1">Select from Keychain</label>
              {#if availableKeys.length === 0}
                <p class="text-neutral-500 text-[11px] py-2">
                  No SSH keys in keychain yet. Add a key in the Keychain tab first.
                </p>
              {:else}
                <select
                  id="identity-key-select"
                  bind:value={selectedKeyId}
                  class="w-full rounded-lg border border-[#28283c] bg-[#090910] px-3 py-2 text-white outline-none focus:border-indigo-500 cursor-pointer"
                  required
                >
                  <option value="" disabled>-- Choose a Key --</option>
                  {#each availableKeys as k}
                    <option value={k.id}>{k.label}</option>
                  {/each}
                </select>
              {/if}
            </div>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-3 border-t border-[#20202c]">
          <button
            type="button"
            onclick={() => (isAddOpen = false)}
            class="rounded-lg bg-[#20202c] px-4 py-2 font-semibold text-neutral-300 hover:bg-[#282838] cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSaving}
            class="rounded-lg bg-indigo-600 px-5 py-2 font-semibold text-white shadow hover:bg-indigo-500 transition cursor-pointer active:scale-95 disabled:opacity-50"
          >
            {isSaving ? 'Saving...' : 'Save Identity'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
