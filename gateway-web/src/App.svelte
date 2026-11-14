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
    Trash2,
    Lock,
    FolderPlus,
    Menu,
    X
  } from 'lucide-svelte';

  // Navigation & Screen State
  let activeTab = $state<'overview' | 'users' | 'teams' | 'audit' | 'settings'>('overview');
  let isAuthenticated = $state(false);
  let currentUser = $state<GatewayUser | null>(null);
  let isMobileOpen = $state(false);

  // Auth Form State (Default Admin: admin@opentermius.local / OpenTermius2026!)
  let authEmail = $state('admin@opentermius.local');
  let authPassword = $state('OpenTermius2026!');
  let authError = $state<string | null>(null);
  let isAuthSubmitting = $state(false);

  // Gateway Data State
  let backendUrl = $state(gatewayApi.getApiBase());
  let users = $state<GatewayUser[]>([]);
  let vaults = $state<GatewayVault[]>([]);
  let selectedTeamVaultId = $state('');
  let teamMembers = $state<GatewayMember[]>([]);
  let sessionLogs = $state<GatewaySessionLog[]>([]);
  let loadState = $state<'loading' | 'success' | 'empty' | 'error'>('loading');
  let errorMessage = $state<string | null>(null);
  let statusBanner = $state<string | null>(null);

  // Create User Modal (Admin registers new user for client-app)
  let isCreateUserOpen = $state(false);
  let newUserName = $state('');
  let newUserEmail = $state('');
  let newUserPassword = $state('');
  let isCreatingUser = $state(false);
  let createUserError = $state<string | null>(null);

  // Create Team Vault Modal
  let isCreateVaultOpen = $state(false);
  let newVaultName = $state('');
  let isCreatingVault = $state(false);
  let createVaultError = $state<string | null>(null);

  // Invite Member Modal
  let isInviteOpen = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state<'admin' | 'member'>('member');
  let isInviting = $state(false);
  let inviteError = $state<string | null>(null);

  let teamVaults = $derived(vaults.filter((v) => v.type === 'team'));

  function showBanner(msg: string) {
    statusBanner = msg;
    setTimeout(() => {
      if (statusBanner === msg) statusBanner = null;
    }, 3500);
  }

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
    } catch {
      gatewayApi.clearToken();
      isAuthenticated = false;
      loadState = 'success';
    }
  }

  async function refreshData() {
    loadState = 'loading';
    errorMessage = null;

    try {
      const [loadedUsers, loadedVaults] = await Promise.all([
        gatewayApi.getUsers().catch(() => [] as GatewayUser[]),
        gatewayApi.getVaults()
      ]);

      users = loadedUsers;
      vaults = loadedVaults;

      const tVaults = loadedVaults.filter((v) => v.type === 'team');
      if (tVaults.length > 0) {
        if (!selectedTeamVaultId || !tVaults.some((v) => v.id === selectedTeamVaultId)) {
          selectedTeamVaultId = tVaults[0].id;
        }
        await loadVaultMembers(selectedTeamVaultId);
      } else {
        selectedTeamVaultId = '';
        teamMembers = [];
      }

      loadState = 'success';
    } catch (err: unknown) {
      loadState = 'error';
      errorMessage = err instanceof Error ? err.message : 'Failed to reach OpenTermius Gateway API';
    }
  }

  async function loadVaultMembers(vaultId: string) {
    if (!vaultId) {
      teamMembers = [];
      return;
    }
    try {
      teamMembers = await gatewayApi.getVaultMembers(vaultId);
    } catch {
      teamMembers = [];
    }
  }

  async function handleAuthSubmit() {
    isAuthSubmitting = true;
    authError = null;

    try {
      gatewayApi.setApiBase(backendUrl);
      try {
        const res = await gatewayApi.login(authEmail.trim(), authPassword);
        currentUser = res.user;
      } catch (loginErr: unknown) {
        // Auto-bootstrap default admin if database was just initialized
        if (authEmail.trim() === 'admin@opentermius.local') {
          const res = await gatewayApi.register(
            'admin@opentermius.local',
            authPassword,
            'Gateway Admin'
          );
          currentUser = res.user;
        } else {
          throw loginErr;
        }
      }
      isAuthenticated = true;
      await refreshData();
    } catch (err: unknown) {
      authError = err instanceof Error ? err.message : 'Gateway authentication failed';
    } finally {
      isAuthSubmitting = false;
    }
  }

  async function handleCreateUser() {
    if (!newUserName.trim() || !newUserEmail.trim() || !newUserPassword) {
      createUserError = 'Full name, email, and password are required.';
      return;
    }

    isCreatingUser = true;
    createUserError = null;

    try {
      const created = await gatewayApi.createUser(
        newUserEmail.trim(),
        newUserPassword,
        newUserName.trim()
      );
      users = [created, ...users];
      isCreateUserOpen = false;
      newUserName = '';
      newUserEmail = '';
      newUserPassword = '';
      showBanner(`User ${created.email} registered. They can now sign in from the Client App.`);
    } catch (err: unknown) {
      createUserError = err instanceof Error ? err.message : 'Failed to create user';
    } finally {
      isCreatingUser = false;
    }
  }

  async function handleDeleteUser(u: GatewayUser) {
    if (u.id === currentUser?.id) return;
    if (!confirm(`Delete user "${u.name}" (${u.email})?`)) return;

    try {
      await gatewayApi.deleteUser(u.id);
      users = users.filter((item) => item.id !== u.id);
      showBanner(`Deleted user ${u.email}.`);
    } catch (err: unknown) {
      errorMessage = err instanceof Error ? err.message : 'Failed to delete user';
    }
  }

  async function handleCreateTeamVault() {
    if (!newVaultName.trim()) {
      createVaultError = 'Team vault name is required.';
      return;
    }

    isCreatingVault = true;
    createVaultError = null;

    try {
      const created = await gatewayApi.createVault(newVaultName.trim(), 'team');
      vaults = [created, ...vaults];
      selectedTeamVaultId = created.id;
      await loadVaultMembers(created.id);
      isCreateVaultOpen = false;
      newVaultName = '';
      showBanner(`Team Vault "${created.name}" created.`);
    } catch (err: unknown) {
      createVaultError = err instanceof Error ? err.message : 'Failed to create team vault';
    } finally {
      isCreatingVault = false;
    }
  }

  async function handleInviteMember() {
    if (!inviteEmail.trim() || !selectedTeamVaultId) {
      inviteError = 'User email and target team vault are required.';
      return;
    }

    isInviting = true;
    inviteError = null;

    try {
      const added = await gatewayApi.inviteMember(
        selectedTeamVaultId,
        inviteEmail.trim(),
        inviteRole
      );
      teamMembers = [added, ...teamMembers];
      isInviteOpen = false;
      inviteEmail = '';
      showBanner(`Added ${added.email || inviteEmail} to team vault.`);
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
              activeTab = 'users';
              isMobileOpen = false;
            }}
            class="flex w-full items-center justify-between rounded-lg px-3 py-2 transition {activeTab === 'users' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <span class="flex items-center gap-2.5">
              <UserPlus class="h-4 w-4" />
              <span>Users & Accounts</span>
            </span>
            <span class="rounded bg-[#1b1b28] px-1.5 py-0.5 text-[10px] text-neutral-300">{users.length}</span>
          </button>

          <button
            type="button"
            onclick={() => {
              activeTab = 'teams';
              isMobileOpen = false;
            }}
            class="flex w-full items-center justify-between rounded-lg px-3 py-2 transition {activeTab === 'teams' ? 'bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-medium' : 'text-neutral-400 hover:bg-[#181824] hover:text-white'}"
          >
            <span class="flex items-center gap-2.5">
              <Users class="h-4 w-4" />
              <span>Teams & Vaults</span>
            </span>
            <span class="rounded bg-[#1b1b28] px-1.5 py-0.5 text-[10px] text-neutral-300">{teamVaults.length}</span>
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
        <span class="hidden sm:inline text-[10px] text-neutral-400 font-mono">:8080 (REST E2EE)</span>
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
    <div class="flex-1 overflow-y-auto p-4 sm:p-6 bg-[#0a0a0f]">
      {#if statusBanner}
        <div class="mb-4 max-w-5xl rounded-xl border border-emerald-500/30 bg-emerald-950/30 px-4 py-2.5 text-xs text-emerald-300 flex items-center justify-between">
          <span class="flex items-center gap-2">
            <CheckCircle2 class="h-4 w-4 text-emerald-400 shrink-0" />
            {statusBanner}
          </span>
          <button type="button" onclick={() => (statusBanner = null)} class="text-neutral-400 hover:text-white">✕</button>
        </div>
      {/if}

      {#if !isAuthenticated}
        <!-- Gateway Admin Login -->
        <div class="flex h-full items-center justify-center">
          <div class="w-full max-w-sm rounded-2xl border border-[#252538] bg-[#12121c] p-6 shadow-2xl space-y-4 text-xs">
            <div class="text-center space-y-1">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 mx-auto mb-2">
                <Lock class="h-5 w-5" />
              </div>
              <h2 class="text-base font-bold text-white">OpenTermius Gateway Login</h2>
              <p class="text-neutral-400">Sign in to manage users, team vaults, and sync policies</p>
            </div>

            <div class="rounded-lg border border-indigo-500/25 bg-indigo-950/25 p-3 text-[11px] text-indigo-200 space-y-1">
              <div class="font-semibold text-indigo-300">Default Admin Credentials:</div>
              <div class="font-mono text-[10px]">Email: <span class="text-white">admin@opentermius.local</span></div>
              <div class="font-mono text-[10px]">Password: <span class="text-white">OpenTermius2026!</span></div>
            </div>

            {#if authError}
              <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
                {authError}
              </div>
            {/if}

            <div class="space-y-3">
              <div>
                <label for="adm-email" class="block font-medium text-neutral-300 mb-1">Admin Email</label>
                <input
                  id="adm-email"
                  type="email"
                  bind:value={authEmail}
                  class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label for="adm-pw" class="block font-medium text-neutral-300 mb-1">Password</label>
                <input
                  id="adm-pw"
                  type="password"
                  bind:value={authPassword}
                  class="w-full rounded-lg border border-[#252538] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
                />
              </div>

              <div class="pt-2">
                <button
                  type="button"
                  onclick={handleAuthSubmit}
                  disabled={isAuthSubmitting}
                  class="w-full rounded-lg bg-emerald-600 py-2.5 font-semibold text-white hover:bg-emerald-500 transition shadow disabled:opacity-50"
                >
                  {isAuthSubmitting ? 'Signing in...' : 'Sign In to Admin Console'}
                </button>
              </div>
            </div>
          </div>
        </div>
      {:else if activeTab === 'overview'}
        <!-- Overview & Server Metrics -->
        <div class="space-y-6 max-w-5xl">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div>
              <h2 class="text-lg font-bold text-white tracking-tight">Gateway Health & Cluster Overview</h2>
              <p class="text-xs text-neutral-400 mt-0.5">Real-time status of Go backend sync server, registered users, and vaults.</p>
            </div>
            <div class="flex items-center gap-2">
              <button
                type="button"
                onclick={() => {
                  activeTab = 'users';
                  isCreateUserOpen = true;
                }}
                class="flex items-center gap-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 px-3 py-2 text-xs font-semibold text-white transition"
              >
                <UserPlus class="h-3.5 w-3.5" /> Register New User
              </button>
            </div>
          </div>

          <!-- Stat Cards Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Registered Users</span>
              <div class="text-2xl font-bold text-white">{users.length}</div>
              <p class="text-[10px] text-emerald-400 flex items-center gap-1">
                <CheckCircle2 class="h-3 w-3" /> Managed via Gateway
              </p>
            </div>

            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Total Vaults</span>
              <div class="text-2xl font-bold text-white">{vaults.length}</div>
              <p class="text-[10px] text-neutral-400">{teamVaults.length} Team · {vaults.length - teamVaults.length} Personal</p>
            </div>

            <div class="rounded-xl border border-[#222233] bg-[#12121c] p-4 space-y-1">
              <span class="text-xs text-neutral-400">Team Members</span>
              <div class="text-2xl font-bold text-white">{teamMembers.length}</div>
              <p class="text-[10px] text-indigo-400">Role-based Access</p>
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
                <span class="text-neutral-400">Interactive SSH & SFTP Transport</span>
                <span class="font-mono text-emerald-400">Direct Client-Side Rust SSH/SFTP (russh + russh-sftp)</span>
              </div>
              <div class="flex items-center justify-between py-1.5">
                <span class="text-neutral-400">Client Packaging</span>
                <span class="font-mono text-emerald-400">Tauri v2 (macOS DMG/App, Windows, Linux, Android)</span>
              </div>
            </div>
          </div>
        </div>
      {:else if activeTab === 'users'}
        <!-- Users & Accounts Management -->
        <div class="space-y-6 max-w-5xl">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div>
              <h2 class="text-lg font-bold text-white tracking-tight">Users & Accounts</h2>
              <p class="text-xs text-neutral-400 mt-0.5">
                Register user accounts here so team members can sign in from the Desktop, Mobile, or Tablet app.
              </p>
            </div>
            <button
              type="button"
              onclick={() => {
                createUserError = null;
                isCreateUserOpen = true;
              }}
              class="flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow transition"
            >
              <UserPlus class="h-4 w-4" /> Register New User
            </button>
          </div>

          <div class="rounded-xl border border-[#222233] bg-[#12121c] overflow-hidden text-xs">
            <table class="w-full text-left">
              <thead class="bg-[#151524] text-neutral-400 border-b border-[#222233]">
                <tr>
                  <th class="p-3">Full Name</th>
                  <th class="p-3">Email Address</th>
                  <th class="p-3">Account Type</th>
                  <th class="p-3">Created</th>
                  <th class="p-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[#1e1e2c]">
                {#if users.length === 0}
                  <tr>
                    <td colspan="5" class="p-6 text-center text-neutral-500">
                      No users found. Click "Register New User" to create an account.
                    </td>
                  </tr>
                {:else}
                  {#each users as u (u.id)}
                    <tr class="hover:bg-[#161622] transition">
                      <td class="p-3 font-medium text-white">{u.name}</td>
                      <td class="p-3 font-mono text-emerald-400">{u.email}</td>
                      <td class="p-3">
                        {#if u.email === 'admin@opentermius.local' || u.id === currentUser?.id}
                          <span class="rounded bg-amber-500/15 border border-amber-500/30 px-2 py-0.5 text-[10px] font-semibold text-amber-400">
                            ADMIN
                          </span>
                        {:else}
                          <span class="rounded bg-indigo-500/15 border border-indigo-500/30 px-2 py-0.5 text-[10px] font-semibold text-indigo-300">
                            USER
                          </span>
                        {/if}
                      </td>
                      <td class="p-3 text-neutral-400">{new Date(u.created_at).toLocaleDateString()}</td>
                      <td class="p-3 text-right">
                        {#if u.id !== currentUser?.id}
                          <button
                            type="button"
                            onclick={() => handleDeleteUser(u)}
                            class="inline-flex items-center gap-1 rounded-lg border border-rose-500/30 bg-rose-500/10 px-2.5 py-1 text-[11px] text-rose-300 hover:bg-rose-500/20 transition"
                          >
                            <Trash2 class="h-3 w-3" /> Delete
                          </button>
                        {:else}
                          <span class="text-[10px] text-neutral-500">Current Session</span>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      {:else if activeTab === 'teams'}
        <!-- Teams & Vaults -->
        <div class="space-y-6 max-w-5xl">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div>
              <h2 class="text-lg font-bold text-white tracking-tight">Team Vaults & Member Access</h2>
              <p class="text-xs text-neutral-400 mt-0.5">Create shared E2EE Team Vaults and assign registered users.</p>
            </div>
            <div class="flex items-center gap-2">
              <button
                type="button"
                onclick={() => {
                  createVaultError = null;
                  isCreateVaultOpen = true;
                }}
                class="flex items-center gap-1.5 px-3 py-2 rounded-lg border border-[#2a2a3c] bg-[#161624] hover:bg-[#1d1d30] text-white font-medium text-xs transition"
              >
                <FolderPlus class="h-4 w-4 text-indigo-400" /> New Team Vault
              </button>
              <button
                type="button"
                onclick={() => {
                  inviteError = null;
                  isInviteOpen = true;
                }}
                disabled={teamVaults.length === 0}
                class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs shadow transition disabled:opacity-40"
              >
                <UserPlus class="h-4 w-4" /> Invite Member
              </button>
            </div>
          </div>

          {#if teamVaults.length > 0}
            <!-- Team Vault Selector -->
            <div class="flex items-center gap-2 flex-wrap">
              {#each teamVaults as tv (tv.id)}
                <button
                  type="button"
                  onclick={() => {
                    selectedTeamVaultId = tv.id;
                    loadVaultMembers(tv.id);
                  }}
                  class="rounded-xl px-3.5 py-2 text-xs font-medium border transition {selectedTeamVaultId === tv.id ? 'bg-emerald-500/15 border-emerald-500/40 text-emerald-300' : 'bg-[#12121c] border-[#222233] text-neutral-400 hover:text-white'}"
                >
                  {tv.name}
                </button>
              {/each}
            </div>
          {/if}

          <!-- Members Table -->
          <div class="rounded-xl border border-[#222233] bg-[#12121c] overflow-hidden text-xs">
            <table class="w-full text-left">
              <thead class="bg-[#151524] text-neutral-400 border-b border-[#222233]">
                <tr>
                  <th class="p-3">User Email / ID</th>
                  <th class="p-3">Role</th>
                  <th class="p-3">Vault ID</th>
                  <th class="p-3">Added Date</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[#1e1e2c]">
                {#if teamVaults.length === 0}
                  <tr>
                    <td colspan="4" class="p-6 text-center text-neutral-500">
                      No Team Vaults created yet. Click "New Team Vault" above to create one.
                    </td>
                  </tr>
                {:else if teamMembers.length === 0}
                  <tr>
                    <td colspan="4" class="p-6 text-center text-neutral-500">
                      No members in this vault yet. Click "Invite Member" to grant access.
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
                      No session logs recorded yet.
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
            <p class="text-xs text-neutral-400 mt-0.5">Configure backend REST API connection.</p>
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
            <button
              type="button"
              onclick={() => {
                gatewayApi.setApiBase(backendUrl);
                showBanner('Gateway API endpoint saved.');
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

<!-- Register New User Modal -->
{#if isCreateUserOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="w-full max-w-md rounded-2xl border border-[#252538] bg-[#141420] p-6 shadow-2xl space-y-4 text-xs animate-in fade-in zoom-in-95 duration-150">
      <div class="flex items-center justify-between border-b border-[#222233] pb-3">
        <h3 class="text-sm font-bold text-white">Register New User Account</h3>
        <button type="button" onclick={() => (isCreateUserOpen = false)} class="text-neutral-400 hover:text-white">✕</button>
      </div>

      {#if createUserError}
        <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
          {createUserError}
        </div>
      {/if}

      <div class="space-y-3">
        <div>
          <label for="new-user-name" class="block text-neutral-300 font-medium mb-1">Full Name</label>
          <input
            id="new-user-name"
            type="text"
            bind:value={newUserName}
            placeholder="Budi Santoso"
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          />
        </div>

        <div>
          <label for="new-user-email" class="block text-neutral-300 font-medium mb-1">Email Address</label>
          <input
            id="new-user-email"
            type="email"
            bind:value={newUserEmail}
            placeholder="budi@company.com"
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          />
        </div>

        <div>
          <label for="new-user-pw" class="block text-neutral-300 font-medium mb-1">Initial Password</label>
          <input
            id="new-user-pw"
            type="password"
            bind:value={newUserPassword}
            placeholder="Minimum 8 characters"
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          />
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-3 border-t border-[#222233]">
        <button
          type="button"
          onclick={() => (isCreateUserOpen = false)}
          class="px-3 py-1.5 rounded-lg border border-[#262638] text-neutral-400 hover:text-white transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleCreateUser}
          disabled={isCreatingUser}
          class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold shadow transition disabled:opacity-50"
        >
          {isCreatingUser ? 'Creating...' : 'Create User'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Create Team Vault Modal -->
{#if isCreateVaultOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="w-full max-w-md rounded-2xl border border-[#252538] bg-[#141420] p-6 shadow-2xl space-y-4 text-xs animate-in fade-in zoom-in-95 duration-150">
      <div class="flex items-center justify-between border-b border-[#222233] pb-3">
        <h3 class="text-sm font-bold text-white">Create Shared Team Vault</h3>
        <button type="button" onclick={() => (isCreateVaultOpen = false)} class="text-neutral-400 hover:text-white">✕</button>
      </div>

      {#if createVaultError}
        <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
          {createVaultError}
        </div>
      {/if}

      <div>
        <label for="new-vault-name" class="block text-neutral-300 font-medium mb-1">Team Vault Name</label>
        <input
          id="new-vault-name"
          type="text"
          bind:value={newVaultName}
          placeholder="DevOps Production Servers"
          class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
        />
      </div>

      <div class="flex justify-end gap-2 pt-3 border-t border-[#222233]">
        <button
          type="button"
          onclick={() => (isCreateVaultOpen = false)}
          class="px-3 py-1.5 rounded-lg border border-[#262638] text-neutral-400 hover:text-white transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleCreateTeamVault}
          disabled={isCreatingVault}
          class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold shadow transition disabled:opacity-50"
        >
          {isCreatingVault ? 'Creating...' : 'Create Vault'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Invite Member Modal -->
{#if isInviteOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="w-full max-w-md rounded-2xl border border-[#252538] bg-[#141420] p-6 shadow-2xl space-y-4 text-xs animate-in fade-in zoom-in-95 duration-150">
      <div class="flex items-center justify-between border-b border-[#222233] pb-3">
        <h3 class="text-sm font-bold text-white">Invite User to Team Vault</h3>
        <button type="button" onclick={() => (isInviteOpen = false)} class="text-neutral-400 hover:text-white">✕</button>
      </div>

      {#if inviteError}
        <div class="rounded-lg bg-rose-950/40 border border-rose-500/30 p-2.5 text-rose-300 text-[11px]">
          {inviteError}
        </div>
      {/if}

      <div class="space-y-3">
        <div>
          <label for="inv-email" class="block text-neutral-300 font-medium mb-1">Select or Enter Registered User Email</label>
          <input
            id="inv-email"
            type="email"
            list="registered-users-list"
            bind:value={inviteEmail}
            placeholder="colleague@company.com"
            class="w-full rounded-lg border border-[#262638] bg-[#0c0c14] px-3 py-2 text-white outline-none focus:border-emerald-500"
          />
          <datalist id="registered-users-list">
            {#each users as u}
              <option value={u.email}>{u.name}</option>
            {/each}
          </datalist>
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
          {isInviting ? 'Adding...' : 'Add to Vault'}
        </button>
      </div>
    </div>
  </div>
{/if}
