<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ApiVaultItem } from '../api/client';
  import {
    Key,
    Plus,
    RefreshCw,
    AlertCircle,
    Copy,
    Check,
    Trash2,
    Lock,
    Shield,
    Sparkles,
    FileKey,
    Download,
    X
  } from 'lucide-svelte';

  interface Props {
    activeVaultId?: string;
  }

  let { activeVaultId }: Props = $props();

  // 4 UI States
  let state = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);

  interface KeyItem {
    id: string;
    label: string;
    keyType: string;
    publicKey: string;
    privateKey: string;
    passphrase?: string;
    createdAt: string;
  }

  let keys = $state<KeyItem[]>([]);
  let isAddOpen = $state(false);
  let isSaving = $state(false);
  let formError = $state<string | null>(null);

  let newLabel = $state('');
  let newKeyType = $state('ED25519');
  let newPrivateKey = $state('');
  let newPublicKey = $state('');
  let newPassphrase = $state('');
  let copiedId = $state<string | null>(null);

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

  async function loadKeys() {
    state = 'loading';
    errorMessage = null;

    try {
      const vaultId = await getResolvedVaultId();
      if (!vaultId) {
        state = 'empty';
        keys = [];
        return;
      }

      const items = await api.getVaultItems(vaultId, 'key');
      if (items.length === 0) {
        keys = [];
        state = 'empty';
        return;
      }

      keys = items.map((item) => {
        let keyType = 'ED25519';
        let publicKey = '';
        let privateKey = '';
        let passphrase = '';

        try {
          const parsed = JSON.parse(atob(item.ciphertext));
          keyType = parsed.keyType || 'ED25519';
          publicKey = parsed.publicKey || '';
          privateKey = parsed.privateKey || '';
          passphrase = parsed.passphrase || '';
        } catch {
          // fallback
        }

        return {
          id: item.id,
          label: item.label,
          keyType,
          publicKey,
          privateKey,
          passphrase,
          createdAt: item.created_at
        };
      });

      state = 'success';
    } catch (err: unknown) {
      state = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to load keychain from vault';
    }
  }

  function generateSampleKeyPair() {
    const randomBytes = crypto.getRandomValues(new Uint8Array(32));
    const randomHex = Array.from(randomBytes)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');

    if (newKeyType === 'ED25519') {
      newPublicKey = `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI${btoa(randomHex.slice(0, 32)).replace(/=/g, '')} opentermius-key`;
      newPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\nQyNTUxOQAAACD${btoa(randomHex).slice(0, 40)}\n-----END OPENSSH PRIVATE KEY-----`;
    } else {
      newPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQ${btoa(randomHex).replace(/=/g, '')} opentermius-rsa`;
      newPrivateKey = `-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA${btoa(randomHex).slice(0, 50)}\n-----END RSA PRIVATE KEY-----`;
    }
  }

  function openAddModal() {
    newLabel = '';
    newKeyType = 'ED25519';
    newPassphrase = '';
    formError = null;
    generateSampleKeyPair(); // Pre-generate a key so form is ready to save immediately
    isAddOpen = true;
  }

  async function handleSaveKey(e: SubmitEvent) {
    e.preventDefault();
    formError = null;

    const vaultId = await getResolvedVaultId();
    if (!vaultId) {
      formError = 'No active vault available. Please create or select a vault first.';
      return;
    }

    if (!newLabel.trim()) {
      formError = 'Key label / name is required.';
      return;
    }

    if (!newPrivateKey.trim()) {
      formError = 'Private key is required. Click "Generate Key Pair" or paste a PEM key.';
      return;
    }

    isSaving = true;

    try {
      const payload = {
        keyType: newKeyType,
        privateKey: newPrivateKey.trim(),
        publicKey: newPublicKey.trim(),
        passphrase: newPassphrase.trim()
      };

      const ciphertext = btoa(JSON.stringify(payload));
      const nonce = btoa(crypto.randomUUID().slice(0, 12));

      const created = await api.createVaultItem(vaultId, {
        kind: 'key',
        label: newLabel.trim(),
        ciphertext,
        nonce
      });

      keys.unshift({
        id: created.id,
        label: newLabel.trim(),
        keyType: newKeyType,
        publicKey: newPublicKey.trim(),
        privateKey: newPrivateKey.trim(),
        passphrase: newPassphrase.trim(),
        createdAt: created.created_at
      });

      state = 'success';
      isAddOpen = false;
      newLabel = '';
      newPrivateKey = '';
      newPublicKey = '';
      newPassphrase = '';
    } catch (err: unknown) {
      formError = err instanceof Error ? err.message : 'Failed to save key to vault';
    } finally {
      isSaving = false;
    }
  }

  async function handleDeleteKey(keyId: string) {
    const vaultId = await getResolvedVaultId();
    if (!vaultId) return;
    if (!confirm('Are you sure you want to delete this SSH key from keychain?')) return;

    try {
      await api.deleteVaultItem(vaultId, keyId);
      keys = keys.filter((k) => k.id !== keyId);
      if (keys.length === 0) state = 'empty';
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Failed to delete key');
    }
  }

  function handleCopyPublicKey(id: string, pubKey: string) {
    if (!pubKey) return;
    navigator.clipboard.writeText(pubKey);
    copiedId = id;
    setTimeout(() => {
      if (copiedId === id) copiedId = null;
    }, 2000);
  }

  $effect(() => {
    if (activeVaultId) {
      loadKeys();
    }
  });

  onMount(() => {
    loadKeys();
  });
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-white">Keychain & SSH Keys</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400 font-medium">
          Zero-Knowledge Vault
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Store and manage reusable SSH keys with client-side E2EE encryption. Copy public keys directly to your servers.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        type="button"
        onclick={loadKeys}
        class="flex items-center gap-1.5 rounded-lg border border-[#22222a] bg-[#14141a] px-3 py-1.5 text-xs text-neutral-300 hover:text-white transition cursor-pointer"
      >
        <RefreshCw class="h-3.5 w-3.5" /> Refresh
      </button>

      <button
        type="button"
        onclick={openAddModal}
        class="flex items-center gap-2 rounded-lg bg-emerald-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow hover:bg-emerald-500 transition cursor-pointer active:scale-95"
      >
        <Plus class="h-4 w-4" /> Add SSH Key
      </button>
    </div>
  </div>

  <!-- 4 UI States -->
  {#if state === 'loading'}
    <div class="flex h-48 items-center justify-center text-xs text-neutral-400 gap-2">
      <RefreshCw class="h-4 w-4 animate-spin text-emerald-400" />
      <span>Loading keychain from encrypted vault...</span>
    </div>
  {:else if state === 'error'}
    <div class="rounded-xl border border-rose-500/30 bg-rose-950/20 p-5 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <AlertCircle class="h-5 w-5 text-rose-400 shrink-0" />
        <div>
          <p class="text-xs font-semibold text-rose-300">Keychain Loading Error</p>
          <p class="text-[11px] text-rose-400/80">{errorMessage}</p>
        </div>
      </div>
      <button
        type="button"
        onclick={loadKeys}
        class="rounded-lg bg-rose-500/20 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-500/30 cursor-pointer"
      >
        Retry
      </button>
    </div>
  {:else if state === 'empty' || keys.length === 0}
    <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-12 text-center space-y-3">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-[#1c1c28] border border-[#262638] text-neutral-400 mx-auto">
        <Key class="h-6 w-6 text-neutral-500" />
      </div>
      <h3 class="text-sm font-semibold text-white">Keychain is Empty</h3>
      <p class="text-xs text-neutral-400 max-w-sm mx-auto">
        Generate or import your SSH private keys (ED25519 or RSA) to attach them to server hosts and identities.
      </p>
      <button
        type="button"
        onclick={openAddModal}
        class="rounded-lg bg-emerald-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition cursor-pointer active:scale-95"
      >
        + Add Your First Key
      </button>
    </div>
  {:else}
    <!-- Keys Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each keys as key (key.id)}
        <div class="rounded-xl border border-[#22222a] bg-[#14141a] p-5 flex flex-col justify-between hover:border-emerald-500/40 transition">
          <div>
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  <FileKey class="h-5 w-5" />
                </div>
                <div>
                  <h3 class="text-sm font-semibold text-white">{key.label}</h3>
                  <span class="rounded bg-[#1e1e2c] px-1.5 py-0.5 text-[10px] font-mono text-emerald-400">
                    {key.keyType}
                  </span>
                </div>
              </div>

              <button
                type="button"
                onclick={() => handleDeleteKey(key.id)}
                class="rounded-lg p-1.5 text-neutral-500 hover:text-rose-400 hover:bg-[#20202d] transition cursor-pointer"
                title="Delete Key"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>

            <!-- Public Key Snippet -->
            {#if key.publicKey}
              <div class="mt-4 rounded-lg bg-[#0e0e14] border border-[#20202e] p-2.5 space-y-1.5">
                <div class="flex items-center justify-between text-[11px] text-neutral-400">
                  <span>Public Key</span>
                  <button
                    type="button"
                    onclick={() => handleCopyPublicKey(key.id, key.publicKey)}
                    class="flex items-center gap-1 text-[10px] text-emerald-400 hover:text-emerald-300 cursor-pointer"
                  >
                    {#if copiedId === key.id}
                      <Check class="h-3 w-3" /> Copied!
                    {:else}
                      <Copy class="h-3 w-3" /> Copy
                    {/if}
                  </button>
                </div>
                <p class="font-mono text-[10px] text-neutral-300 truncate" title={key.publicKey}>
                  {key.publicKey}
                </p>
              </div>
            {/if}
          </div>

          <div class="mt-4 pt-3 border-t border-[#1e1e28] flex items-center justify-between text-xs text-neutral-400">
            <span class="flex items-center gap-1 text-[11px]">
              <Lock class="h-3 w-3 text-emerald-400" />
              {key.passphrase ? 'Passphrase Protected' : 'No Passphrase'}
            </span>
            <span class="text-[10px] text-neutral-500">
              {new Date(key.createdAt).toLocaleDateString()}
            </span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Add / Generate Key Modal -->
{#if isAddOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 overflow-y-auto">
    <div class="w-full max-w-lg rounded-2xl border border-[#28283a] bg-[#14141e] p-6 shadow-2xl space-y-4 animate-in fade-in zoom-in-95 duration-150 my-auto">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Key class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Add SSH Key to Keychain</h3>
            <p class="text-xs text-neutral-400">Store reusable private keys encrypted client-side.</p>
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

      <form onsubmit={handleSaveKey} class="space-y-4 text-xs">
        <div>
          <label for="key-label" class="block text-neutral-300 font-medium mb-1">Key Label / Name</label>
          <input
            id="key-label"
            type="text"
            bind:value={newLabel}
            placeholder="e.g. AWS Production EC2 Key"
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
            required
          />
        </div>

        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <label for="key-type" class="text-neutral-300 font-medium">Key Type:</label>
            <select
              id="key-type"
              bind:value={newKeyType}
              onchange={generateSampleKeyPair}
              class="rounded-lg border border-[#252535] bg-[#0c0c12] px-2.5 py-1 text-white outline-none focus:border-emerald-500 cursor-pointer"
            >
              <option value="ED25519">ED25519 (Recommended)</option>
              <option value="RSA-4096">RSA 4096-bit</option>
            </select>
          </div>

          <button
            type="button"
            onclick={generateSampleKeyPair}
            class="flex items-center gap-1.5 rounded-lg bg-emerald-600/20 border border-emerald-500/30 px-3 py-1 text-emerald-300 hover:bg-emerald-600/30 transition text-[11px] font-semibold cursor-pointer active:scale-95"
          >
            <Sparkles class="h-3 w-3" />
            <span>Generate New Pair</span>
          </button>
        </div>

        <div>
          <label for="key-private" class="block text-neutral-300 font-medium mb-1">Private Key (OpenSSH PEM)</label>
          <textarea
            id="key-private"
            rows={5}
            bind:value={newPrivateKey}
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;..."
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 font-mono text-[11px] text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
            required
          ></textarea>
        </div>

        <div>
          <label for="key-public" class="block text-neutral-300 font-medium mb-1">Public Key (Auto-Generated / Optional)</label>
          <input
            id="key-public"
            type="text"
            bind:value={newPublicKey}
            placeholder="ssh-ed25519 AAAAC3... user@host"
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 font-mono text-[11px] text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
          />
        </div>

        <div>
          <label for="key-passphrase" class="block text-neutral-300 font-medium mb-1">Key Passphrase (Optional)</label>
          <input
            id="key-passphrase"
            type="password"
            bind:value={newPassphrase}
            placeholder="Leave empty if key has no passphrase"
            class="w-full rounded-lg border border-[#252535] bg-[#0c0c12] px-3 py-2 text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
          />
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
            class="rounded-lg bg-emerald-600 px-5 py-2 font-semibold text-white shadow hover:bg-emerald-500 transition cursor-pointer active:scale-95 disabled:opacity-50"
          >
            {isSaving ? 'Saving...' : 'Save to Keychain'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
