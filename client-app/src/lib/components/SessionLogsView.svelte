<script lang="ts">
  import { ScrollText, Play, Clock, Server, User, Search, Download } from 'lucide-svelte';

  interface LogEntry {
    id: string;
    hostLabel: string;
    user: string;
    duration: string;
    startedAt: string;
    vault: string;
  }

  const logs: LogEntry[] = [
    {
      id: 'log-1',
      hostLabel: 'AWS Kubernetes Master (EKS)',
      user: 'devops-lead@company.internal',
      duration: '42m 15s',
      startedAt: 'Today at 16:30',
      vault: 'DevOps & Infrastructure'
    },
    {
      id: 'log-2',
      hostLabel: 'Primary PostgreSQL Node',
      user: 'dba@company.internal',
      duration: '18m 02s',
      startedAt: 'Today at 14:10',
      vault: 'Database Cluster Vault'
    },
    {
      id: 'log-3',
      hostLabel: 'Staging App Server 01',
      user: 'jane.doe@opentermius.local',
      duration: '05m 44s',
      startedAt: 'Yesterday at 19:22',
      vault: 'Personal Cloud Vault'
    }
  ];
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-white">Terminal Session Logs</h2>
        <span class="rounded bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400 font-medium">
          Compliance & Audit Ready
        </span>
      </div>
      <p class="text-xs text-neutral-400 mt-1">
        Full terminal session recordings, command history, and playback for team collaboration and audit compliance.
      </p>
    </div>
  </div>

  <div class="rounded-xl border border-[#22222a] bg-[#14141a] overflow-hidden">
    <table class="w-full text-left text-xs">
      <thead class="border-b border-[#22222a] bg-[#181822] text-neutral-400 font-semibold">
        <tr>
          <th class="py-3 px-4">Server Target</th>
          <th class="py-3 px-4">Operator / User</th>
          <th class="py-3 px-4">Vault Origin</th>
          <th class="py-3 px-4">Duration</th>
          <th class="py-3 px-4">Timestamp</th>
          <th class="py-3 px-4 text-right">Actions</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-[#1e1e28] text-neutral-200">
        {#each logs as log}
          <tr class="hover:bg-[#181824] transition-colors">
            <td class="py-3 px-4 font-medium text-white flex items-center gap-2">
              <Server class="h-3.5 w-3.5 text-emerald-400" />
              {log.hostLabel}
            </td>
            <td class="py-3 px-4 text-neutral-300">
              {log.user}
            </td>
            <td class="py-3 px-4 text-neutral-400">
              {log.vault}
            </td>
            <td class="py-3 px-4 font-mono text-neutral-400">
              {log.duration}
            </td>
            <td class="py-3 px-4 text-neutral-400">
              {log.startedAt}
            </td>
            <td class="py-3 px-4 text-right">
              <button class="inline-flex items-center gap-1 rounded bg-[#20202c] px-2.5 py-1 text-xs text-emerald-400 hover:bg-emerald-600 hover:text-white transition">
                <Play class="h-3 w-3" /> Playback
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
