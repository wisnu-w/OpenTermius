<script lang="ts">
  import {
    Server,
    Terminal,
    FileCode2,
    ShieldCheck,
    FolderLock,
    ScrollText,
    Settings,
    Users,
    Key,
    UserCheck,
    Radio,
    ChevronDown,
    Check,
    Network,
    PanelLeftClose,
    Folder
  } from 'lucide-svelte';
  import { api, type ApiVault } from '../api/client';

  interface Props {
    activeNav: string;
    onNavChange: (nav: string) => void;
    activeVault?: ApiVault | null;
    vaults?: ApiVault[];
    onSelectVault?: (vault: ApiVault) => void;
    isCollapsed?: boolean;
    onToggleCollapse?: () => void;
    isMobileOpen?: boolean;
    onCloseMobile?: () => void;
  }

  let {
    activeNav = 'hosts',
    onNavChange,
    activeVault,
    vaults = [],
    onSelectVault,
    isCollapsed = false,
    onToggleCollapse,
    isMobileOpen = false,
    onCloseMobile
  }: Props = $props();

  function handleNavClick(navId: string) {
    onNavChange(navId);
    if (onCloseMobile) {
      onCloseMobile();
    }
  }

  let isVaultDropdownOpen = $state(false);

  const navItems = [
    { id: 'hosts', label: 'Hosts', icon: Server },
    { id: 'terminal', label: 'Terminal', icon: Terminal },
    { id: 'sftp', label: 'SFTP (Files)', icon: Folder },
    { id: 'tunnels', label: 'Port Forwarding', icon: Network },
    { id: 'keychain', label: 'Keychain', icon: Key },
    { id: 'identities', label: 'Identities', icon: UserCheck },
    { id: 'snippets', label: 'Snippets', icon: FileCode2 },
    { id: 'vaults', label: 'Vaults (E2EE)', icon: FolderLock },
    { id: 'logs', label: 'Session Logs', icon: ScrollText },
    { id: 'settings', label: 'Settings', icon: Settings }
  ];
</script>

{#if isMobileOpen}
  <div
    class="fixed inset-0 z-40 bg-black/70 backdrop-blur-xs md:hidden animate-in fade-in duration-150"
    onclick={onCloseMobile}
    role="button"
    tabindex="0"
    onkeydown={(e) => e.key === 'Escape' && onCloseMobile?.()}
    aria-label="Close sidebar backdrop"
  ></div>
{/if}

<aside
  class="flex flex-col justify-between border-r border-[#22222a] bg-[#101014] select-none transition-all duration-200 ease-in-out shrink-0
    fixed inset-y-0 left-0 z-50 w-72 p-4 shadow-2xl md:relative md:z-auto md:shadow-none
    {isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
    {isCollapsed ? 'md:w-0 md:p-0 md:border-r-0 md:overflow-hidden md:opacity-0 md:pointer-events-none' : 'md:w-64 md:p-4 md:opacity-100'}
  "
>
  <!-- Brand Header -->
  <div>
    <div class="flex items-center justify-between px-1 py-2 mb-4">
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 shadow-inner shrink-0">
          <Terminal class="h-4 w-4" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <span class="text-xs font-bold tracking-tight text-white truncate">OpenTermius</span>
            <span class="rounded bg-emerald-500/15 px-1 py-0.2 text-[9px] font-semibold text-emerald-400">OSS</span>
          </div>
          <p class="text-[10px] text-neutral-400 truncate">Zero Paywall</p>
        </div>
      </div>
      {#if isMobileOpen && onCloseMobile}
        <button
          type="button"
          onclick={onCloseMobile}
          class="md:hidden text-neutral-500 hover:text-white p-1 rounded-lg hover:bg-[#1a1a24] transition shrink-0"
          aria-label="Close mobile sidebar"
        >
          <PanelLeftClose class="h-4 w-4" />
        </button>
      {:else if onToggleCollapse}
        <button
          type="button"
          onclick={onToggleCollapse}
          class="hidden md:block text-neutral-500 hover:text-white p-1 rounded-lg hover:bg-[#1a1a24] transition shrink-0"
          title="Hide Sidebar (⌘B)"
        >
          <PanelLeftClose class="h-4 w-4" />
        </button>
      {/if}
    </div>

    <!-- Active Vault Selector Dropdown -->
    <div class="relative mb-5">
      <button
        onclick={() => (isVaultDropdownOpen = !isVaultDropdownOpen)}
        class="w-full text-left rounded-lg border border-[#22222a] bg-[#15151d] p-2.5 hover:border-emerald-500/40 transition"
      >
        <div class="flex items-center justify-between text-[11px] text-neutral-400 mb-1">
          <span>Active Vault</span>
          <span class="flex items-center gap-1 text-[10px] text-emerald-400">
            <ShieldCheck class="h-3 w-3" /> E2EE
          </span>
        </div>
        <div class="flex items-center justify-between font-medium text-xs text-white">
          <div class="flex items-center gap-2 truncate">
            {#if activeVault?.type === 'team'}
              <Users class="h-3.5 w-3.5 text-indigo-400 shrink-0" />
            {:else}
              <FolderLock class="h-3.5 w-3.5 text-emerald-400 shrink-0" />
            {/if}
            <span class="truncate">{activeVault?.name || 'Connecting to Vault...'}</span>
          </div>
          <ChevronDown class="h-3.5 w-3.5 text-neutral-500 transition-transform {isVaultDropdownOpen ? 'rotate-180' : ''}" />
        </div>
      </button>

      {#if isVaultDropdownOpen && vaults.length > 0}
        <div class="absolute left-0 right-0 top-full mt-1.5 z-30 rounded-xl border border-[#28283a] bg-[#14141f] p-1.5 shadow-2xl space-y-1 animate-in fade-in zoom-in-95 duration-100">
          <div class="px-2 py-1 text-[10px] font-semibold text-neutral-400 uppercase tracking-wider">
            Switch Vault
          </div>
          {#each vaults as vault (vault.id)}
            <button
              onclick={() => {
                onSelectVault?.(vault);
                isVaultDropdownOpen = false;
              }}
              class="flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-xs text-left transition {activeVault?.id === vault.id ? 'bg-emerald-600/20 text-emerald-300 font-medium' : 'text-neutral-300 hover:bg-[#1f1f2c] hover:text-white'}"
            >
              <div class="flex items-center gap-2 truncate">
                {#if vault.type === 'team'}
                  <Users class="h-3.5 w-3.5 text-indigo-400 shrink-0" />
                {:else}
                  <FolderLock class="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                {/if}
                <span class="truncate">{vault.name}</span>
                <span class="rounded bg-[#20202d] px-1 py-0.2 text-[9px] text-neutral-400 capitalize">
                  {vault.type}
                </span>
              </div>
              {#if activeVault?.id === vault.id}
                <Check class="h-3.5 w-3.5 text-emerald-400 shrink-0" />
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Navigation Items -->
    <nav class="space-y-1">
      {#each navItems as item}
        {@const Icon = item.icon}
        <button
          onclick={() => handleNavClick(item.id)}
          class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-xs font-medium transition-all {activeNav === item.id ? 'bg-emerald-500/10 text-emerald-400 font-semibold shadow-sm border border-emerald-500/20' : 'text-neutral-400 hover:bg-[#181822] hover:text-neutral-200'}"
        >
          <div class="flex items-center gap-3">
            <Icon class="h-4 w-4 {activeNav === item.id ? 'text-emerald-400' : 'text-neutral-400'}" />
            <span>{item.label}</span>
          </div>
        </button>
      {/each}
    </nav>
  </div>

  <!-- Bottom Server & User Status -->
  <div class="border-t border-[#22222a] pt-3">
    <div class="flex items-center justify-between rounded-lg bg-[#15151d] px-3 py-2 border border-[#22222a]">
      <div class="flex items-center gap-2 truncate">
        <span class="relative flex h-2 w-2 shrink-0">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full {api.isStandalone() ? 'bg-indigo-400' : 'bg-emerald-400'} opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 {api.isStandalone() ? 'bg-indigo-500' : 'bg-emerald-500'}"></span>
        </span>
        <span class="text-[11px] font-medium text-neutral-300 truncate">
          {api.isStandalone() ? 'Standalone' : 'Gateway Sync'}
        </span>
      </div>
      <span class="text-[10px] text-neutral-400 shrink-0 font-mono">
        {api.isStandalone() ? 'local' : api.getApiBase().replace(/^https?:\/\//, '').split('/')[0]}
      </span>
    </div>
  </div>
</aside>
