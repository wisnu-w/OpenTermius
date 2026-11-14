<script lang="ts">
  import {
    X,
    FileText,
    Upload,
    CheckCircle2,
    AlertCircle,
    Server,
    Download
  } from 'lucide-svelte';

  interface ParsedHost {
    label: string;
    address: string;
    port: number;
    username: string;
    group: string;
    tags: string[];
    authType?: 'password' | 'key';
    privateKey?: string;
  }

  interface Props {
    isOpen: boolean;
    onClose: () => void;
    onImport: (hosts: ParsedHost[]) => Promise<void>;
  }

  let { isOpen, onClose, onImport }: Props = $props();

  let importType = $state<'ssh-config' | 'json-csv'>('ssh-config');
  let rawContent = $state('');
  let parsedHosts = $state<ParsedHost[]>([]);
  let parseError = $state<string | null>(null);
  let isSubmitting = $state(false);

  function parseSSHConfig(text: string): ParsedHost[] {
    const lines = text.split('\n');
    const hosts: ParsedHost[] = [];
    let currentHost: Partial<ParsedHost> | null = null;

    for (let rawLine of lines) {
      const line = rawLine.trim();
      if (!line || line.startsWith('#')) continue;

      const [key, ...rest] = line.split(/\s+/);
      const val = rest.join(' ');
      const lowerKey = key.toLowerCase();

      if (lowerKey === 'host') {
        if (currentHost && currentHost.label && currentHost.address) {
          hosts.push({
            label: currentHost.label,
            address: currentHost.address,
            port: currentHost.port || 22,
            username: currentHost.username || 'root',
            group: currentHost.group || 'SSH Config',
            tags: ['imported'],
            authType: currentHost.authType || 'password'
          });
        }
        // Exclude wildcard host declarations like 'Host *'
        if (val === '*') {
          currentHost = null;
        } else {
          currentHost = {
            label: val,
            address: val, // fallback if HostName is omitted
            port: 22,
            username: 'root',
            group: 'SSH Config'
          };
        }
      } else if (currentHost) {
        if (lowerKey === 'hostname') {
          currentHost.address = val;
        } else if (lowerKey === 'user') {
          currentHost.username = val;
        } else if (lowerKey === 'port') {
          const p = parseInt(val, 10);
          if (!isNaN(p)) currentHost.port = p;
        } else if (lowerKey === 'identityfile') {
          currentHost.authType = 'key';
        }
      }
    }

    if (currentHost && currentHost.label && currentHost.address) {
      hosts.push({
        label: currentHost.label,
        address: currentHost.address,
        port: currentHost.port || 22,
        username: currentHost.username || 'root',
        group: currentHost.group || 'SSH Config',
        tags: ['imported'],
        authType: currentHost.authType || 'password'
      });
    }

    return hosts;
  }

  function parseJSONorCSV(text: string): ParsedHost[] {
    const trimmed = text.trim();
    if (trimmed.startsWith('[') || trimmed.startsWith('{')) {
      const parsed = JSON.parse(trimmed);
      const items = Array.isArray(parsed) ? parsed : [parsed];
      return items.map((item: any) => ({
        label: item.label || item.name || item.address || 'Imported Host',
        address: item.address || item.host || item.hostname || '127.0.0.1',
        port: parseInt(item.port, 10) || 22,
        username: item.username || item.user || 'root',
        group: item.group || item.folder || 'Default',
        tags: Array.isArray(item.tags) ? item.tags : ['imported'],
        authType: item.authType || 'password'
      }));
    }

    // CSV format parsing
    const lines = trimmed.split('\n').filter((l) => l.trim().length > 0);
    if (lines.length === 0) return [];

    const headers = lines[0].split(',').map((h) => h.trim().toLowerCase());
    const labelIdx = headers.indexOf('label');
    const addrIdx = headers.indexOf('address') !== -1 ? headers.indexOf('address') : headers.indexOf('host');
    const portIdx = headers.indexOf('port');
    const userIdx = headers.indexOf('username') !== -1 ? headers.indexOf('username') : headers.indexOf('user');
    const groupIdx = headers.indexOf('group') !== -1 ? headers.indexOf('group') : headers.indexOf('folder');

    const hosts: ParsedHost[] = [];
    for (let i = 1; i < lines.length; i++) {
      const cols = lines[i].split(',').map((c) => c.trim());
      if (cols.length < 2) continue;

      const address = addrIdx !== -1 ? cols[addrIdx] : cols[1];
      const label = labelIdx !== -1 ? cols[labelIdx] : cols[0] || address;
      const port = portIdx !== -1 ? parseInt(cols[portIdx], 10) || 22 : 22;
      const username = userIdx !== -1 ? cols[userIdx] : 'root';
      const group = groupIdx !== -1 ? cols[groupIdx] : 'Imported';

      hosts.push({
        label,
        address,
        port,
        username,
        group,
        tags: ['imported'],
        authType: 'password'
      });
    }

    return hosts;
  }

  function handleParse() {
    parseError = null;
    if (!rawContent.trim()) {
      parseError = 'Please paste configuration content or upload a file.';
      parsedHosts = [];
      return;
    }

    try {
      if (importType === 'ssh-config') {
        parsedHosts = parseSSHConfig(rawContent);
      } else {
        parsedHosts = parseJSONorCSV(rawContent);
      }

      if (parsedHosts.length === 0) {
        parseError = 'No valid hosts detected in the input.';
      }
    } catch (err: unknown) {
      parseError = err instanceof Error ? err.message : 'Failed to parse configuration format.';
      parsedHosts = [];
    }
  }

  async function handleFileUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    const text = await file.text();
    rawContent = text;
    handleParse();
  }

  async function handleSubmit() {
    if (parsedHosts.length === 0) return;
    isSubmitting = true;
    try {
      await onImport(parsedHosts);
      rawContent = '';
      parsedHosts = [];
      onClose();
    } catch (err: unknown) {
      parseError = err instanceof Error ? err.message : 'Import failed';
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4 overflow-y-auto animate-in fade-in duration-100">
    <div class="w-full max-w-2xl max-h-[85dvh] overflow-y-auto rounded-2xl border border-[#252535] bg-[#12121a] p-4 sm:p-6 shadow-2xl space-y-5 animate-in zoom-in-95 duration-100 my-auto">
      <!-- Modal Header -->
      <div class="flex items-center justify-between border-b border-[#1e1e2d] pb-4">
        <div class="flex items-center gap-2.5">
          <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Download class="h-4 w-4" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-white">Import Hosts</h3>
            <p class="text-xs text-neutral-400">Import servers from SSH config or CSV/JSON</p>
          </div>
        </div>
        <button
          onclick={onClose}
          class="rounded-lg p-1 text-neutral-400 hover:bg-[#1f1f2e] hover:text-white transition"
          aria-label="Close"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <!-- Format Selector -->
      <div class="flex gap-2 border-b border-[#1c1c28] pb-3">
        <button
          type="button"
          onclick={() => {
            importType = 'ssh-config';
            handleParse();
          }}
          class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition {importType === 'ssh-config' ? 'bg-[#222233] text-white border border-[#303046]' : 'text-neutral-400 hover:text-white'}"
        >
          <FileText class="h-3.5 w-3.5" />
          <span>SSH Config (~/.ssh/config)</span>
        </button>
        <button
          type="button"
          onclick={() => {
            importType = 'json-csv';
            handleParse();
          }}
          class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition {importType === 'json-csv' ? 'bg-[#222233] text-white border border-[#303046]' : 'text-neutral-400 hover:text-white'}"
        >
          <Server class="h-3.5 w-3.5" />
          <span>JSON / CSV</span>
        </button>
      </div>

      <!-- Input Textarea & File Upload -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <label for="import-paste-area" class="text-xs font-medium text-neutral-300">
            Paste content or upload file
          </label>
          <label class="flex items-center gap-1.5 text-xs text-emerald-400 hover:underline cursor-pointer">
            <Upload class="h-3.5 w-3.5" />
            <span>Choose File</span>
            <input type="file" onchange={handleFileUpload} class="hidden" accept=".conf,.config,.txt,.json,.csv" />
          </label>
        </div>

        <textarea
          id="import-paste-area"
          bind:value={rawContent}
          oninput={handleParse}
          placeholder={importType === 'ssh-config' ? "Host my-server\n  HostName 192.168.1.50\n  User ubuntu\n  Port 22" : "label,address,port,username,group\nWeb 1,192.168.1.10,22,ubuntu,Production"}
          class="w-full h-32 rounded-xl border border-[#222230] bg-[#0c0c12] p-3 text-xs font-mono text-neutral-200 placeholder-neutral-600 outline-none focus:border-emerald-500 transition resize-none"
        ></textarea>
      </div>

      <!-- Errors -->
      {#if parseError}
        <div class="rounded-xl border border-rose-500/30 bg-rose-950/20 p-3 text-xs text-rose-300 flex items-center gap-2">
          <AlertCircle class="h-4 w-4 shrink-0 text-rose-400" />
          <span>{parseError}</span>
        </div>
      {/if}

      <!-- Parsed Preview -->
      {#if parsedHosts.length > 0}
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-white flex items-center gap-1">
              <CheckCircle2 class="h-3.5 w-3.5 text-emerald-400" /> Detected {parsedHosts.length} hosts
            </span>
            <span class="text-neutral-400 text-[11px]">Preview below</span>
          </div>
          <div class="max-h-40 overflow-y-auto rounded-xl border border-[#20202d] bg-[#0c0c12]">
            <table class="w-full text-left text-[11px]">
              <thead class="border-b border-[#20202d] bg-[#14141d] text-neutral-400 uppercase font-mono">
                <tr>
                  <th class="p-2">Label</th>
                  <th class="p-2">Address</th>
                  <th class="p-2">User</th>
                  <th class="p-2">Group</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[#181822] text-neutral-200">
                {#each parsedHosts as h}
                  <tr>
                    <td class="p-2 font-medium text-white">{h.label}</td>
                    <td class="p-2 font-mono text-emerald-400">{h.address}:{h.port}</td>
                    <td class="p-2 font-mono text-neutral-400">{h.username}</td>
                    <td class="p-2 text-neutral-400">{h.group}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}

      <!-- Footer Buttons -->
      <div class="flex items-center justify-end gap-2 border-t border-[#1c1c28] pt-4">
        <button
          type="button"
          onclick={onClose}
          class="rounded-lg border border-[#262635] px-4 py-2 text-xs font-semibold text-neutral-300 hover:text-white transition"
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={parsedHosts.length === 0 || isSubmitting}
          onclick={handleSubmit}
          class="rounded-lg bg-emerald-600 px-5 py-2 text-xs font-semibold text-white hover:bg-emerald-500 transition shadow disabled:opacity-40"
        >
          {isSubmitting ? 'Importing...' : `Import ${parsedHosts.length > 0 ? parsedHosts.length + ' Hosts' : ''}`}
        </button>
      </div>
    </div>
  </div>
{/if}
