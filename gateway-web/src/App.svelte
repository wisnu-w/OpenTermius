<script lang="ts">
  import { onMount } from 'svelte';
  import {
    gatewayApi,
    type GatewayUser,
    type GatewayVault,
    type GatewayMember,
    type GatewaySessionLog
  } from './lib/api/gateway_client';
  import {
    Server,
    Shield,
    Users,
    Activity,
    ScrollText,
    Settings,
    LogOut,
    Plus,
    RefreshCw,
    CheckCircle2,
    AlertCircle,
    UserPlus,
    Radio,
    HardDrive,
    Lock,
    KeyRound,
    Network,
    Menu,
    X
  } from 'lucide-svelte';

  // Navigation & Screen State
  let activeTab = $state<'overview' | 'teams' | 'audit' | 'settings'>('overview');
  let isAuthenticated = $state(false);
  let currentUser = $state<GatewayUser | null>(null);
  let isMobileOpen = $state(false);

  // Auth Form State
  let authEmail = $state('admin@opentermius.local');
  let authPassword = $state('OpenTermius2026!');
  let authName = $state('Gateway Admin');
  let authError = $state<string | null>(null);
  let isAuthSubmitting = $state(false);
  let authMode = $state<'login' | 'register'>('login');

  // Gateway Data State
  let backendUrl = $state(gatewayApi.getApiBase());
  let wsUrl = $state('ws://localhost:8080/ws');
  let vaults = $state<GatewayVault[]>([]);
  let teamMembers = $state<GatewayMember[]>([]);
  let sessionLogs = $state<GatewaySessionLog[]>([]);
  let loadState = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);

  // Invite Modal
  let isInviteOpen = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state<'admin' | 'member'>('member');
  let inviteVaultId = $state('');
  let isInviting = $state(false);
  let inviteError = $state<string | null>(null);

  async function checkAuthAndLoad() {
    loadState = 'loading';
    errorMessage = null;

    const token = gatewayApi.getToken();
    if (!token) {
      isAuthenticated = false;
      loadState = 'success';
      return;
    }

    try {
      currentUser = await gatewayApi.getMe();
      isAuthenticated = true;
      await refreshData();
    } catch (err: unknown) {
      gatewayApi.clearToken();
      isAuthenticated = false;
      loadState = 'success';
    }
  }

  async function refreshData() {
    loadState = 'loading';
    errorMessage = null;

    try {
      vaults = await gatewayApi.getVaults();
      const teamVaults = vaults.filter((v) => v.type === 'team');

      if (teamVaults.length > 0) {
        inviteVaultId = teamVaults[0].id;
        try {
          teamMembers = await gatewayApi.getVaultMembers(teamVaults[0].id);
        } catch {
          teamMembers = [];
        }
      } else {
        teamMembers = [];
      }

      try {
        sessionLogs = await gatewayApi.getSessionLogs();
      } catch {
        sessionLogs = [];
      }

      loadState = 'success';
    } catch (err: unknown) {
      loadState = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to reach OpenTermius Gateway API';
    }
  }

  async function handleAuthSubmit() {
    isAuthSubmitting = true;
    authError = null;

    try {
      gatewayApi.setApiBase(backendUrl);
      if (authMode === 'register') {
        const res = await gatewayApi.register(authEmail, authPassword, authName);
        currentUser = res.user;
      } else {
        const res = await gatewayApi.login(authEmail, authPassword);
        currentUser = res.user;
      }
      isAuthenticated = true;
      await refreshData();
    } catch (err: unknown) {
      authError = err instanceof Error ? err.message : 'Gateway authentication failed';
    } finally {
      isAuthSubmitting = false;
    }
  }

  async function handleInviteMember() {
    if (!inviteEmail.trim() || !inviteVaultId) {
      inviteError = 'Email and target team vault are required.';
      return;
    }

    isInviting = true;
    inviteError = null;

    try {
      const added = await gatewayApi.inviteMember(inviteVaultId, inviteEmail.trim(), inviteRole);
      teamMembers = [added, ...teamMembers];
      isInviteOpen = false;
      inviteEmail = '';
    } catch (err: unknown) {
      inviteError = err instanceof Error ? err.message : 'Failed to invite member';
    } finally {
      isInviting = false;
    }
  }

  function handleLogout() {
    gatewayApi.clearToken();
    isAuthenticated = false;
    currentUser = null;
  }

  onMount(() => {
    checkAuthAndLoad();
  });
</script>

<div class="flex h-screen w-screen overflow-hidden bg-[#0c0c12] text-neutral-100 font-sans select-none">
  {#if isMobileOpen}
    <div
      class="fixed inset-0 z-40 bg-black/70 backdrop-blur-xs md:hidden animate-in fade-in duration-150"
      onclick={() => (isMobileOpen = false)}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Escape' && (isMobileOpen = false)}
      aria-label="Close sidebar backdrop"
    ></div>
  {/if}

  <!-- Sidebar -->
  <aside
    class="flex flex-col justify-between border-r border-[#1e1e2c] bg-[#101018] p-4 shrink-0 transition-all duration-200
      fixed inset-y-0 left-0 z-50 w-64 shadow-2xl md:relative md:z-auto md:shadow-none
      {isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
    "
  >
    <div class="space-y-6">
      <!-- Brand -->
      <div class="flex items-center justify-between px-2 py-2">
        <div class="flex items-center gap-3">
          <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
            <Server class="h-5 w-5" />
          </div>
          <div>
            <div class="flex items-center gap-1.5">
              <span class="text-sm font-bold text-white tracking-tight">OpenTermius</span>
              <span class="rounded bg-indigo-500/20 border border-indigo-500/30 px-1.5 py-0.2 text-[9px] font-semibold text-indigo-400">GATEWAY</span>
            </div>
            <p class="text-[10px] text-neutral-400">Admin & Sync Console</p>
          </div>
        </div>
        {#if isMobileOpen}
          <button
            type="button"
            onclick={() => (isMobileOpen = false)}
            class="md:hidden text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#1a1a24] transition"
            aria-label="Close sidebar"
          >
            <X class="h-4 w-4" />
          </button>
        {/if}
      </div>

      <!-- Navigation -->
      {#if isAuthenticated}
        <nav class="space-y-1 text-xs">
          <button
            type="button"
            onclick={() => {
              activeTab = 'overview';
              isMobileOpen = false;
            }}
            class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 transition {activeTab === 'overview' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <Activity class="h-4 w-4" />
            <span>Overview & Health</span>
          </button>

          <button
            type="button"
            onclick={() => {
              activeTab = 'teams';
              isMobileOpen = false;
            }}
            class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 transition {activeTab === 'teams' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <Users class="h-4 w-4" />
            <span>Teams & Members</span>
          </button>

          <button
            type="button"
            onclick={() => {
              activeTab = 'audit';
              isMobileOpen = false;
            }}
            class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 transition {activeTab === 'audit' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <ScrollText class="h-4 w-4" />
            <span>Audit & Session Logs</span>
          </button>

          <button
            type="button"
            onclick={() => {
              activeTab = 'settings';
              isMobileOpen = false;
            }}
            class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 transition {activeTab === 'settings' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <Settings class="h-4 w-4" />
            <span>Gateway Config</span>
          </button>
        </nav>
      {/if}
    </div>

    <!-- User Card -->
    {#if isAuthenticated && currentUser}
      <div class="border-t border-[#1e1e2c] pt-3 flex items-center justify-between text-xs">
        <div class="truncate mr-2">
          <p class="font-medium text-white truncate">{currentUser.name}</p>
          <p class="text-[10px] text-neutral-400 truncate">{currentUser.email}</p>
        </div>
        <button
          type="button"
          onclick={handleLogout}
          class="text-neutral-500 hover:text-rose-400 p-1.5 rounded-lg transition"
          title="Sign Out"
        >
          <LogOut class="h-4 w-4" />
        </button>
      </div>
    {/if}
  </aside>

  <!-- Main Viewport -->
  <main class="flex flex-1 flex-col overflow-hidden">
    <!-- Top Bar -->
    <header class="flex h-12 items-center justify-between border-b border-[#1e1e2c] bg-[#0e0e16] px-4 sm:px-6">
      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (isMobileOpen = !isMobileOpen)}
          class="md:hidden text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#1a1a24] transition mr-1"
          aria-label="Toggle menu"
        >
          <Menu class="h-4 w-4 text-emerald-400" />
        </button>
        <span class="text-xs font-semibold text-neutral-300 capitalize">{activeTab}</span>
      </div>
      <div class="flex items-center gap-2.5">
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
        </span>
        <span class="text-xs font-semibold text-white">Go Backend Sync Gateway</span>
        <span class="text-[10px] text-neutral-400 font-mono">:8080 (REST + WS)</span>
      </div>

      {#if isAuthenticated}
        <button
          type="button"
          onclick={refreshData}
          class="flex items-center gap-1.5 rounded-lg border border-[#222234] bg-[#141420] px-3 py-1.5 text-xs text-neutral-300 hover:text-white transition"
        >
          <RefreshCw class="h-3 w-3" /> Refresh
        </button>
      {/if}
    </header>

    <!-- Body Area -->
    <div class="flex-1 overflow-y-auto p-6 bg-[#0a0a0f]">
      {#if !isAuthenticated}
        <!-- Gateway Admin Login -->
        <div class="flex h-full items-center justify-center">
          <div class="w-full max-w-sm rounded-2xl border border-[#252538] bg-[#12121c] p-6 shadow-2xl space-y-4 text-xs">
            <div class="text-center space-y-1">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 mx-auto mb-2">
                <Lock class="h-5 w-5" />
              </div>
              <h2 class="text-base font-bold text-white">OpenTermius Gateway Login</h2>
              <p class="text-neutral-400">Authenticate with self-hosted admin credentials</p>
            </div>

            {#if authError}
              <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
                {authError}
              </div>
            {/if}

            <div class="space-y-3">
              <div>
                <label for="adm-email" class="block font-medium text-neutral-300 mb-1">Email</label>
                <input
                  id="adm-email"
                  type="email"
                  bind:value={authEmail}
                  class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-1.5 text-white outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label for="adm-pw" class="block font-medium text-neutral-300 mb-1">Master Password</label>
                <input
                  id="adm-pw"
                  type="password"
                  bind:value={authPassword}
                  class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-1.5 text-white outline-none focus:border-emerald-500"
                />
              </div>

              {#if authMode === 'register'}
                <div>
                  <label for="adm-name" class="block font-medium text-neutral-300 mb-1">Full Name</label>
                  <input
                    id="adm-name"
                    type="text"
                    bind:value={authName}
                    class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-1.5 text-white outline-none focus:border-emerald-500"
                  />
                </div>
              {/if}

              <div class="pt-2 flex gap-2">
                <button
                  type="button"
                  onclick={handleAuthSubmit}
                  disabled={isAuthSubmitting}
                  class="flex-1 rounded-lg bg-emerald-600 py-2 font-semibold text-white hover:bg-emerald-500 transition shadow disabled:opacity-50"
                >
                  {isAuthSubmitting ? 'Authenticating...' : authMode === 'login' ? 'Sign In to Console' : 'Register Operator'}
                </button>
              </div>

              <div class="text-center pt-1">
                <button
                  type="button"
                  onclick={() => (authMode = authMode === 'login' ? 'register' : 'login')}
                  class="text-[11px] text-neutral-400 hover:text-emerald-400 transition"
                >
                  {authMode === 'login' ? 'Need to register an admin account?' : 'Already have an account? Sign in'}
                </button>
              </div>
            </div>
          </div>
        </div>
      {:else if activeTab === 'overview'}
        <!-- Overview & Server Metrics -->
        <div class="space-y-6 max-w-5xl">
          <div>
            <h2 class="text-lg font-bold text-white tracking-tight">Gateway Health & Cluster Overview</h2>
            <p class="text-xs text-neutral-400 mt-0.5">Real-time status of Go backend sync server, database, and client connections.</p>
          </div>

          <!-- Stat Cards Grid -->
          <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Total Vaults</span>
              <div class="text-2xl font-bold text-white">{vaults.length}</div>
              <p class="text-[10px] text-emerald-400 flex items-center gap-1">
                <CheckCircle2 class="h-3 w-3" /> E2EE Active
              </p>
            </div>

            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Team Members</span>
              <div class="text-2xl font-bold text-white">{teamMembers.length}</div>
              <p class="text-[10px] text-neutral-400">Role-based Access</p>
            </div>

            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Logged Sessions</span>
              <div class="text-2xl font-bold text-white">{sessionLogs.length}</div>
              <p class="text-[10px] text-indigo-400">Audited SSH Sessions</p>
            </div>

            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Zero-Knowledge</span>
              <div class="text-sm font-bold text-emerald-400 pt-1">100% Client-Side</div>
              <p class="text-[10px] text-neutral-400">AES-256-GCM Encrypted</p>
            </div>
          </div>

          <!-- Infrastructure Specs Table -->
          <div class="rounded-xl border border-[#222233] bg-[#12121c] p-5 space-y-3 text-xs">
            <h3 class="font-semibold text-white">Active Infrastructure Topology</h3>
            <div class="space-y-2 text-neutral-300">
              <div class="flex items-center justify-between py-1.5 border-b border-[#1c1c28]">
                <span class="text-neutral-400">Backend Server Runtime</span>
                <span class="font-mono text-emerald-400">Go 1.24 Clean Architecture (Hexagonal)</span>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b border-[#1c1c28]">
                <span class="text-neutral-400">Database Adapter</span>
                <span class="font-mono text-emerald-400">PostgreSQL (Supabase Remote Connection Pool)</span>
              </div>
              <div class="flex items-center justify-between py-1.5 border-b border-[#1c1c28]">
                <span class="text-neutral-400">Interactive SSH Transport</span>
                <span class="font-mono text-emerald-400">Full-duplex WebSocket PTY bridge & In-Memory Native IPC</span>
              </div>
              <div class="flex items-center justify-between py-1.5">
                <span class="text-neutral-400">Client Packaging</span>
                <span class="font-mono text-emerald-400">Tauri v2 (macOS DMG/App, Windows, Linux, Android)</span>
              </div>
            </div>
          </div>
        </div>
      {:else if activeTab === 'teams'}
        <!-- Teams & Members -->
        <div class="space-y-6 max-w-5xl">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="text-lg font-bold text-white tracking-tight">Team Vaults & User Access</h2>
              <p class="text-xs text-neutral-400 mt-0.5">Manage team members, roles, and collaborative vault sharing.</p>
            </div>
            <button
              type="button"
              onclick={() => (isInviteOpen = true)}
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs shadow transition"
            >
              <UserPlus class="h-4 w-4" /> Invite Member
            </button>
          </div>

          <!-- Members Table -->
          <div class="rounded-xl border border-[#222233] bg-[#12121c] overflow-hidden text-xs">
            <table class="w-full text-left">
              <thead class="bg-[#151524] text-neutral-400 border-b border-[#222233]">
                <tr>
                  <th class="p-3">User Email / ID</th>
                  <th class="p-3">Role</th>
                  <th class="p-3">Target Vault</th>
                  <th class="p-3">Added Date</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[#1e1e2c]">
                {#if teamMembers.length === 0}
                  <tr>
                    <td colspan="4" class="p-6 text-center text-neutral-500">
                      No members invited yet. Click "Invite Member" to add team collaborators.
                    </td>
                  </tr>
                {:else}
                  {#each teamMembers as member}
                    <tr class="hover:bg-[#161622] transition">
                      <td class="p-3 font-mono text-white">{member.email || member.user_id}</td>
                      <td class="p-3">
                        <span class="rounded px-2 py-0.5 text-[10px] font-semibold uppercase {member.role === 'owner' ? 'bg-amber-500/15 text-amber-400 border border-amber-500/30' : 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'}">
                          {member.role}
                        </span>
                      </td>
                      <td class="p-3 font-mono text-neutral-400">{member.vault_id.slice(0, 8)}...</td>
                      <td class="p-3 text-neutral-400">{new Date(member.created_at).toLocaleDateString()}</td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      {:else if activeTab === 'audit'}
        <!-- Audit Logs -->
        <div class="space-y-6 max-w-5xl">
          <div>
            <h2 class="text-lg font-bold text-white tracking-tight">Audit Trail & Session Logs</h2>
            <p class="text-xs text-neutral-400 mt-0.5">Immutable record of terminal connection sessions and administrative actions.</p>
          </div>

          <div class="rounded-xl border border-[#222233] bg-[#12121c] overflow-hidden text-xs">
            <table class="w-full text-left">
              <thead class="bg-[#151524] text-neutral-400 border-b border-[#222233]">
                <tr>
                  <th class="p-3">Timestamp</th>
                  <th class="p-3">User ID</th>
                  <th class="p-3">Host ID</th>
                  <th class="p-3">Type</th>
                  <th class="p-3">Status</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[#1e1e2c]">
                {#if sessionLogs.length === 0}
                  <tr>
                    <td colspan="5" class="p-6 text-center text-neutral-500">
                      No session logs recorded yet. Sessions opened by clients are audited here.
                    </td>
                  </tr>
                {:else}
                  {#each sessionLogs as log}
                    <tr class="hover:bg-[#161622] transition font-mono">
                      <td class="p-3 text-neutral-300">{new Date(log.started_at).toLocaleString()}</td>
                      <td class="p-3 text-white truncate max-w-[120px]">{log.user_id}</td>
                      <td class="p-3 text-emerald-400 truncate max-w-[120px]">{log.host_id}</td>
                      <td class="p-3">{log.session_type}</td>
                      <td class="p-3 text-emerald-400">Completed</td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      {:else if activeTab === 'settings'}
        <!-- Configuration -->
        <div class="space-y-6 max-w-2xl text-xs">
          <div>
            <h2 class="text-lg font-bold text-white tracking-tight">Gateway Console Settings</h2>
            <p class="text-xs text-neutral-400 mt-0.5">Configure backend connection and security policies.</p>
          </div>

          <div class="rounded-xl border border-[#222233] bg-[#12121c] p-5 space-y-4">
            <div>
              <label for="set-api" class="block font-medium text-neutral-300 mb-1">Backend REST API Endpoint</label>
              <input
                id="set-api"
                type="text"
                bind:value={backendUrl}
                class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-2 text-white font-mono outline-none focus:border-emerald-500"
              />
            </div>
            <div>
              <label for="set-ws" class="block font-medium text-neutral-300 mb-1">WebSocket Gateway Endpoint</label>
              <input
                id="set-ws"
                type="text"
                bind:value={wsUrl}
                class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-2 text-white font-mono outline-none focus:border-emerald-500"
              />
            </div>
            <button
              type="button"
              onclick={() => {
                gatewayApi.setApiBase(backendUrl);
                alert('Gateway configuration saved!');
              }}
              class="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold shadow transition"
            >
              Save Configuration
            </button>
          </div>
        </div>
      {/if}
    </div>
  </main>
</div>

<!-- Invite Member Modal -->
{#if isInviteOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="w-full max-w-md rounded-2xl border border-[#252538] bg-[#141420] p-6 shadow-2xl space-y-4 text-xs animate-in fade-in zoom-in-95 duration-150">
      <div class="flex items-center justify-between border-b border-[#222233] pb-3">
        <h3 class="text-sm font-bold text-white">Invite Team Member</h3>
        <button type="button" onclick={() => (isInviteOpen = false)} class="text-neutral-400 hover:text-white">✕</button>
      </div>

      {#if inviteError}
        <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
          {inviteError}
        </div>
      {/if}

      <div class="space-y-3">
        <div>
          <label for="inv-email" class="block text-neutral-300 font-medium mb-1">Colleague Email</label>
          <input
            id="inv-email"
            type="email"
            bind:value={inviteEmail}
            placeholder="colleague@company.com"
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          />
        </div>

        <div>
          <label for="inv-role" class="block text-neutral-300 font-medium mb-1">Role Permissions</label>
          <select
            id="inv-role"
            bind:value={inviteRole}
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          >
            <option value="member">Member (Read & Connect)</option>
            <option value="admin">Admin (Manage Hosts & Keys)</option>
          </select>
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-3 border-t border-[#222233]">
        <button
          type="button"
          onclick={() => (isInviteOpen = false)}
          class="px-3 py-1.5 rounded-lg border border-[#262638] text-neutral-400 hover:text-white transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleInviteMember}
          disabled={isInviting}
          class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold shadow transition disabled:opacity-50"
        >
          {isInviting ? 'Inviting...' : 'Send Invitation'}
        </button>
      </div>
    </div>
  </div>
{/if}
