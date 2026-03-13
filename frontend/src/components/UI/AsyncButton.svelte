<script>
  let {
    onclick = () => {},
    disabled = false,
    class: className = '',
    successDuration = 1500,
    children,
    ...restProps
  } = $props();

  let phase = $state('idle'); // 'idle' | 'loading' | 'success'

  async function handleClick(e) {
    if (phase !== 'idle' || disabled) return;
    phase = 'loading';
    try {
      await onclick(e);
      phase = 'success';
      setTimeout(() => { phase = 'idle'; }, successDuration);
    } catch (err) {
      phase = 'idle';
      throw err;
    }
  }
</script>

<button
  class={className}
  onclick={handleClick}
  disabled={disabled || phase !== 'idle'}
  {...restProps}
>
  {#if phase === 'loading'}
    <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
    </svg>
  {:else if phase === 'success'}
    <svg class="h-3.5 w-3.5 text-emerald-500" viewBox="0 0 20 20" fill="currentColor">
      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
    </svg>
  {:else}
    {@render children()}
  {/if}
</button>
