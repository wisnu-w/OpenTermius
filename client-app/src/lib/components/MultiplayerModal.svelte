<script lang="ts">
  import { Users, Copy, Check, X, Shield, Radio } from 'lucide-svelte';

  interface Props {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: Props = $props();

  let copied = $state(false);
  const shareUrl = 'ws://localhost:8080/ws/sessions/session-live-77b3/join';

  function handleCopy() {
    navigator.clipboard.writeText(shareUrl);
    copied = true;
    setTimeout(() => {
      copied = false;
    }, 2000);
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4 overflow-y-auto">
    <div class="w-full max-w-md max-h-[85dvh] overflow-y-auto rounded-2xl border border-[#262635] bg-[#14141d] p-4 sm:p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-150 my-auto">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Radio class="h-5 w-5 animate-pulse" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Share Live Multiplayer Session</h3>
            <p class="text-xs text-neutral-400">Collaborate with teammates in real-time inside this terminal.</p>
          </div>
        </div>
        <button onclick={onClose} class="text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-[#20202c]">
          <X class="h-4 w-4" />
        </button>
      </div>

      <div class="rounded-xl border border-[#22222f] bg-[#0c0c12] p-3 space-y-2">
        <div class="flex items-center justify-between text-[11px] text-neutral-400">
          <span>Collaboration Link (WebSocket PTY)</span>
          <span class="text-emerald-400 font-medium">Read & Write Access</span>
        </div>
        <div class="flex items-center gap-2">
          <input
            type="text"
            readonly
            value={shareUrl}
            class="flex-1 rounded-lg border border-[#252535] bg-[#161622] px-3 py-1.5 text-xs font-mono text-neutral-200 outline-none"
          />
          <button
            onclick={handleCopy}
            class="flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500 transition active:scale-95"
          >
            {#if copied}
              <Check class="h-3.5 w-3.5" /> Copied
            {:else}
              <Copy class="h-3.5 w-3.5" /> Copy
            {/if}
          </button>
        </div>
      </div>

      <div class="rounded-lg bg-[#181824] p-3 text-xs text-neutral-400 flex items-start gap-2.5">
        <Shield class="h-4 w-4 text-emerald-400 mt-0.5 shrink-0" />
        <span>No extra Termius Team subscription required. Multiplayer sessions use direct self-hosted WebSocket replication.</span>
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <button
          onclick={onClose}
          class="rounded-lg bg-[#222230] px-4 py-2 text-xs font-semibold text-white hover:bg-[#2a2a3c] transition"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}
