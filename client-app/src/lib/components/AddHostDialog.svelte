<script lang="ts">
  import { X, Server, Key, Lock, Radio, Shield, Zap, UserCheck, Network, Globe } from 'lucide-svelte';
  import { api } from '../api/client';

  interface Props {
    isOpen: boolean;
    initialHost?: any;
    activeVaultId?: string;
    onClose: () => void;
    onSave: (host: any) => void;
  }

  let { isOpen, initialHost = null, activeVaultId, onClose, onSave }: Props = $props();

  let label = $state('');
  let address = $state('');
  let port = $state(22);
  let username = $state('root');
  let credMode = $state<'manual' | 'identity' | 'keychain'>('manual');
  let authType = $state<'password' | 'key'>('password');
  let password = $state('');
  let privateKey = $state('');
  let passphrase = $state('');
  let useMosh = $state(false);
  let group = $state('Production');
  let tags = $state('aws, web');

  let vaultIdentities = $state<any[]>([]);
  let vaultKeys = $state<any[]>([]);
  let vaultHosts = $state<any[]>([]);
  let selectedIdentityId = $state('');
  let selectedKeyId = $state('');

  // Proxy / Jump Host Configuration
  let proxyMode = $state<'none' | 'jump_host' | 'socks5' | 'http'>('none');
  let proxyJumpHostId = $state('');
  let proxyHost = $state('127.0.0.1');
  let proxyPort = $state(1080);

  async function loadVaultCredentials() {
    if (!activeVaultId) return;
    try {
      const [identities, keys, rawHosts] = await Promise.all([
        api.getVaultItems(activeVaultId, 'identity'),
        api.getVaultItems(activeVaultId, 'key'),
        api.getVaultItems(activeVaultId, 'host')
      ]);

      vaultHosts = rawHosts.filter((h) => !initialHost || h.id !== initialHost.id);

      vaultIdentities = identities.map((item) => {
        let parsed: any = {};
        try {
          parsed = JSON.parse(atob(item.ciphertext));
        } catch {}
        return {
          id: item.id,
          label: item.label,
          username: parsed.username || 'root',
          authType: parsed.authType || 'password',
          password: parsed.password || '',
          privateKey: parsed.privateKey || '',
          passphrase: parsed.passphrase || ''
        };
      });

      vaultKeys = keys.map((item) => {
        let parsed: any = {};
        try {
          parsed = JSON.parse(atob(item.ciphertext));
        } catch {}
        return {
          id: item.id,
          label: item.label,
          privateKey: parsed.privateKey || '',
          passphrase: parsed.passphrase || ''
        };
      });
    } catch {
      // ignore
    }
  }

  function handleSelectIdentity(id: string) {
    selectedIdentityId = id;
    const found = vaultIdentities.find((i) => i.id === id);
    if (found) {
      username = found.username;
      authType = found.authType;
      password = found.password;
      privateKey = found.privateKey;
      passphrase = found.passphrase;
    }
  }

  function handleSelectKey(id: string) {
    selectedKeyId = id;
    const found = vaultKeys.find((k) => k.id === id);
    if (found) {
      authType = 'key';
      privateKey = found.privateKey;
      passphrase = found.passphrase;
    }
  }

  $effect(() => {
    if (isOpen) {
      loadVaultCredentials();
      if (initialHost) {
        label = initialHost.label || '';
        address = initialHost.address || '';
        port = initialHost.port || 22;
        username = initialHost.username || 'root';
        authType = initialHost.authType || 'password';
        password = initialHost.password || '';
        privateKey = initialHost.privateKey || '';
        passphrase = initialHost.passphrase || '';
        useMosh = !!initialHost.useMosh;
        group = initialHost.group || 'Default';
        tags = Array.isArray(initialHost.tags)
          ? initialHost.tags.join(', ')
          : initialHost.tags || '';
        credMode = 'manual';
        if (initialHost.proxy) {
          proxyMode = initialHost.proxy.mode || 'none';
          proxyJumpHostId = initialHost.proxy.jumpHostId || '';
          proxyHost = initialHost.proxy.host || '127.0.0.1';
          proxyPort = initialHost.proxy.port || 1080;
        } else {
          proxyMode = 'none';
          proxyJumpHostId = '';
          proxyHost = '127.0.0.1';
          proxyPort = 1080;
        }
      } else {
        label = '';
        address = '';
        port = 22;
        username = 'root';
        authType = 'password';
        password = '';
        privateKey = '';
        passphrase = '';
        useMosh = false;
        group = 'Production';
        tags = 'aws, web';
        credMode = 'manual';
        proxyMode = 'none';
        proxyJumpHostId = '';
        proxyHost = '127.0.0.1';
        proxyPort = 1080;
      }
    }
  });

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    onSave({
      id: initialHost?.id || crypto.randomUUID(),
      label: label || 'SSH Host',
      address: address || '127.0.0.1',
      port,
      username: username || 'root',
      authType,
      password,
      privateKey,
      passphrase,
      useMosh,
      group: group || 'Default',
      tags: tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
      pingMs: initialHost?.pingMs || Math.floor(Math.random() * 25) + 10,
      proxy:
        proxyMode === 'none'
          ? undefined
          : {
              mode: proxyMode,
              jumpHostId: proxyJumpHostId,
              host: proxyHost,
              port: proxyPort
            }
    });
    onClose();
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/75 backdrop-blur-sm p-0 sm:p-4 overflow-y-auto">
    <div class="w-full max-w-lg max-h-[92dvh] sm:max-h-[90vh] overflow-y-auto rounded-t-3xl sm:rounded-2xl border border-[#262635] bg-[#14141d] p-4 pb-[calc(1.25rem+var(--sab))] sm:p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-150 sm:my-auto">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Server class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">
              {initialHost ? 'Edit SSH Host' : 'Add New SSH Host'}
            </h3>
            <p class="text-xs text-neutral-400">
              {initialHost ? 'Update connection details and credentials.' : 'Configure connection details, identities, and keys.'}
            </p>
          </div>
        </div>
        <button onclick={onClose} class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#20202c]">
          <X class="h-4 w-4" />
        </button>
      </div>

      <form onsubmit={handleSubmit} class="space-y-4 text-xs">
        <div>
          <label for="host-label" class="block text-neutral-300 font-medium mb-1">Host Label</label>
          <input
            id="host-label"
            type="text"
            bind:value={label}
            placeholder="e.g. AWS Production Web 01"
            class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
            required
          />
        </div>

        <div class="grid grid-cols-3 gap-3">
          <div class="col-span-2">
            <label for="host-addr" class="block text-neutral-300 font-medium mb-1">Hostname or IP Address</label>
            <input
              id="host-addr"
              type="text"
              bind:value={address}
              placeholder="e.g. 54.210.88.14 or 192.168.1.10"
              class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 font-mono text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
              required
            />
          </div>
          <div>
            <label for="host-port" class="block text-neutral-300 font-medium mb-1">Port</label>
            <input
              id="host-port"
              type="number"
              bind:value={port}
              class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 font-mono text-white outline-none focus:border-emerald-500"
              required
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="host-user" class="block text-neutral-300 font-medium mb-1">Username</label>
            <input
              id="host-user"
              type="text"
              bind:value={username}
              placeholder="ubuntu / root / ec2-user"
              class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 font-mono text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
              required
            />
          </div>
          <div>
            <label for="host-group" class="block text-neutral-300 font-medium mb-1">Group / Folder</label>
            <input
              id="host-group"
              type="text"
              bind:value={group}
              placeholder="Production / Staging"
              class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
            />
          </div>
        </div>

        <!-- Credential Source Selector -->
        <div class="rounded-xl border border-[#252535] bg-[#0f0f18] p-3 space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold text-neutral-200">Credential Source</span>
            <div class="flex items-center gap-1 bg-[#1a1a24] p-0.5 rounded-lg border border-[#2a2a3a]">
              <button
                type="button"
                onclick={() => (credMode = 'manual')}
                class="px-2 py-0.5 rounded-md text-[10px] font-medium transition {credMode === 'manual' ? 'bg-emerald-600 text-white' : 'text-neutral-400 hover:text-white'}"
              >
                Custom
              </button>
              {#if vaultIdentities.length > 0}
                <button
                  type="button"
                  onclick={() => (credMode = 'identity')}
                  class="px-2 py-0.5 rounded-md text-[10px] font-medium transition {credMode === 'identity' ? 'bg-indigo-600 text-white' : 'text-neutral-400 hover:text-white'}"
                >
                  Identity
                </button>
              {/if}
              {#if vaultKeys.length > 0}
                <button
                  type="button"
                  onclick={() => (credMode = 'keychain')}
                  class="px-2 py-0.5 rounded-md text-[10px] font-medium transition {credMode === 'keychain' ? 'bg-emerald-600 text-white' : 'text-neutral-400 hover:text-white'}"
                >
                  Keychain
                </button>
              {/if}
            </div>
          </div>

          {#if credMode === 'identity'}
            <div>
              <label for="id-select" class="block text-neutral-400 text-[11px] mb-1">Select Identity Profile</label>
              <select
                id="id-select"
                bind:value={selectedIdentityId}
                onchange={(e) => handleSelectIdentity(e.currentTarget.value)}
                class="w-full rounded-lg border border-[#28283c] bg-[#0a0a10] px-3 py-2 text-white outline-none focus:border-indigo-500"
              >
                <option value="" disabled>-- Choose an Identity --</option>
                {#each vaultIdentities as id}
                  <option value={id.id}>{id.label} ({id.username})</option>
                {/each}
              </select>
            </div>
          {:else if credMode === 'keychain'}
            <div>
              <label for="keychain-select" class="block text-neutral-400 text-[11px] mb-1">Select SSH Key from Keychain</label>
              <select
                id="keychain-select"
                bind:value={selectedKeyId}
                onchange={(e) => handleSelectKey(e.currentTarget.value)}
                class="w-full rounded-lg border border-[#28283c] bg-[#0a0a10] px-3 py-2 text-white outline-none focus:border-emerald-500"
              >
                <option value="" disabled>-- Choose a Key --</option>
                {#each vaultKeys as k}
                  <option value={k.id}>{k.label}</option>
                {/each}
              </select>
            </div>
          {:else}
            <!-- Manual Credential Inputs -->
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-neutral-400 text-[11px]">Authentication Method</span>
                <div class="flex items-center gap-1 bg-[#1a1a24] p-0.5 rounded-lg border border-[#2a2a3a]">
                  <button
                    type="button"
                    onclick={() => (authType = 'password')}
                    class="px-2.5 py-1 rounded-md text-[11px] font-medium transition {authType === 'password' ? 'bg-emerald-600 text-white' : 'text-neutral-400 hover:text-white'}"
                  >
                    Password
                  </button>
                  <button
                    type="button"
                    onclick={() => (authType = 'key')}
                    class="px-2.5 py-1 rounded-md text-[11px] font-medium transition {authType === 'key' ? 'bg-emerald-600 text-white' : 'text-neutral-400 hover:text-white'}"
                  >
                    SSH Key
                  </button>
                </div>
              </div>

              {#if authType === 'password'}
                <div>
                  <label for="host-pass" class="block text-neutral-400 text-[11px] mb-1">SSH Password</label>
                  <input
                    id="host-pass"
                    type="password"
                    bind:value={password}
                    placeholder="Enter SSH password"
                    class="w-full rounded-lg border border-[#28283c] bg-[#0a0a10] px-3 py-2 text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
                  />
                </div>
              {:else}
                <div class="space-y-2">
                  <div>
                    <label for="host-privkey" class="block text-neutral-400 text-[11px] mb-1">Private Key (OpenSSH PEM)</label>
                    <textarea
                      id="host-privkey"
                      rows={4}
                      bind:value={privateKey}
                      placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;..."
                      class="w-full rounded-lg border border-[#28283c] bg-[#0a0a10] px-3 py-2 font-mono text-[11px] text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
                    ></textarea>
                  </div>
                  <div>
                    <label for="host-passphrase" class="block text-neutral-400 text-[11px] mb-1">Key Passphrase (Optional)</label>
                    <input
                      id="host-passphrase"
                      type="password"
                      bind:value={passphrase}
                      placeholder="Passphrase if key is encrypted"
                      class="w-full rounded-lg border border-[#28283c] bg-[#0a0a10] px-3 py-2 text-white placeholder-neutral-600 outline-none focus:border-emerald-500"
                    />
                  </div>
                </div>
              {/if}
            </div>
          {/if}
        </div>

        <div>
          <label for="host-tags" class="block text-neutral-300 font-medium mb-1">Tags (comma separated)</label>
          <input
            id="host-tags"
            type="text"
            bind:value={tags}
            placeholder="aws, web, nginx"
            class="w-full rounded-lg border border-[#252535] bg-[#0e0e14] px-3 py-2 text-white placeholder-neutral-500 outline-none focus:border-emerald-500"
          />
        </div>

        <!-- Proxy & Jump Host Section -->
        <div class="rounded-xl border border-[#22222e] bg-[#0e0e16] p-3 space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Network class="h-4 w-4 text-emerald-400" />
              <div>
                <p class="font-medium text-white text-xs">Proxy & Jump Host</p>
                <p class="text-[10px] text-neutral-400">Route SSH traffic via Bastion host or Proxy</p>
              </div>
            </div>
            <select
              bind:value={proxyMode}
              class="rounded-lg border border-[#282838] bg-[#14141d] px-2 py-1 text-[11px] text-neutral-200 outline-none focus:border-emerald-500"
            >
              <option value="none">Direct (No Proxy)</option>
              <option value="jump_host">Jump Host (Bastion)</option>
              <option value="socks5">SOCKS5 Proxy</option>
              <option value="http">HTTP / HTTPS Proxy</option>
            </select>
          </div>

          {#if proxyMode === 'jump_host'}
            <div class="pt-2 border-t border-[#1e1e28] text-xs">
              <label for="proxy-bastion-select" class="block text-neutral-400 text-[11px] mb-1">Select Bastion Host</label>
              {#if vaultHosts.length === 0}
                <p class="text-[11px] text-amber-400/90">No other hosts in vault to act as bastion.</p>
              {:else}
                <select
                  id="proxy-bastion-select"
                  bind:value={proxyJumpHostId}
                  class="w-full rounded-lg border border-[#282838] bg-[#0a0a10] px-3 py-1.5 text-xs text-white outline-none focus:border-emerald-500"
                >
                  <option value="">-- Choose Jump Host --</option>
                  {#each vaultHosts as vh}
                    <option value={vh.id}>{vh.label}</option>
                  {/each}
                </select>
              {/if}
            </div>
          {:else if proxyMode === 'socks5' || proxyMode === 'http'}
            <div class="pt-2 border-t border-[#1e1e28] grid grid-cols-3 gap-2 text-xs">
              <div class="col-span-2">
                <label for="proxy-target-host" class="block text-neutral-400 text-[11px] mb-1">Proxy Host / IP</label>
                <input
                  id="proxy-target-host"
                  type="text"
                  bind:value={proxyHost}
                  placeholder="127.0.0.1 or proxy.corp"
                  class="w-full rounded-lg border border-[#282838] bg-[#0a0a10] px-2.5 py-1.5 text-xs text-white outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label for="proxy-target-port" class="block text-neutral-400 text-[11px] mb-1">Port</label>
                <input
                  id="proxy-target-port"
                  type="number"
                  bind:value={proxyPort}
                  placeholder="1080"
                  class="w-full rounded-lg border border-[#282838] bg-[#0a0a10] px-2.5 py-1.5 text-xs text-white outline-none focus:border-emerald-500 font-mono"
                />
              </div>
            </div>
          {/if}
        </div>

        <!-- Mosh option -->
        <div class="rounded-xl border border-[#22222e] bg-[#0e0e16] p-3 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Radio class="h-4 w-4 text-indigo-400" />
            <div>
              <p class="font-medium text-white">Enable Mosh Protocol</p>
              <p class="text-[10px] text-neutral-400">Resilient UDP roaming for unstable networks</p>
            </div>
          </div>
          <input
            type="checkbox"
            bind:checked={useMosh}
            class="h-4 w-4 accent-emerald-500 rounded cursor-pointer"
          />
        </div>

        <div class="flex justify-end gap-2 pt-3 border-t border-[#20202c]">
          <button
            type="button"
            onclick={onClose}
            class="rounded-lg bg-[#20202c] px-4 py-2 font-semibold text-neutral-300 hover:bg-[#282838] transition"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-emerald-600 px-5 py-2 font-semibold text-white shadow hover:bg-emerald-500 transition active:scale-95"
          >
            {initialHost ? 'Update Host' : 'Save Host'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
