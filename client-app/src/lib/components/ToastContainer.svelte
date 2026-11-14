<script lang="ts">
  import { AlertCircle, CheckCircle2, Info, X } from 'lucide-svelte';
  import { toast } from '../stores/toast.svelte';
  import { platform } from '../stores/platform.svelte';
</script>

{#if toast.items.length > 0}
  <div
    class="fixed z-[100] flex flex-col gap-2 pointer-events-none
      {platform.isMobile
        ? 'left-3 right-3 bottom-[calc(4.75rem+var(--sab))]'
        : 'top-4 right-4 w-full max-w-sm'}"
    aria-live="polite"
  >
    {#each toast.items as item (item.id)}
      <div
        role="alert"
        class="pointer-events-auto flex items-start gap-2.5 rounded-2xl border px-3.5 py-3 shadow-2xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-2 sm:slide-in-from-top-2 duration-150
          {item.kind === 'error'
            ? 'border-rose-500/35 bg-[#1b1116]/95 text-rose-100'
            : item.kind === 'success'
              ? 'border-emerald-500/35 bg-[#101c16]/95 text-emerald-100'
              : 'border-indigo-500/35 bg-[#121420]/95 text-indigo-100'}"
      >
        {#if item.kind === 'error'}
          <AlertCircle class="h-4.5 w-4.5 text-rose-400 shrink-0 mt-0.5" />
        {:else if item.kind === 'success'}
          <CheckCircle2 class="h-4.5 w-4.5 text-emerald-400 shrink-0 mt-0.5" />
        {:else}
          <Info class="h-4.5 w-4.5 text-indigo-400 shrink-0 mt-0.5" />
        {/if}

        <p class="flex-1 text-xs leading-relaxed break-words">{item.message}</p>

        <button
          type="button"
          onclick={() => toast.dismiss(item.id)}
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg text-neutral-400 hover:text-white hover:bg-white/10 transition"
          aria-label="Dismiss notification"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>
    {/each}
  </div>
{/if}
