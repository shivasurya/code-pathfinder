<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let variant: 'primary' | 'secondary' = 'primary';
  export let size: 'small' | 'medium' | 'large' = 'medium';
  export let disabled: boolean = false;
  export let type: 'button' | 'submit' | 'reset' = 'button';

  function handleClick() {
    if (!disabled) {
      dispatch('click');
    }
  }
</script>

<button
  class="button {variant} {size}"
  on:click={handleClick}
  {disabled}
  {type}
>
  {#if $$slots.icon}
    <span class="icon">
      <slot name="icon" />
    </span>
  {/if}
  <slot />
</button>

<style>
  .button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-family: var(--vscode-font-family);
    font-weight: 500;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
  }

  .button:hover:not(:disabled) {
    transform: translateY(-1px);
  }

  .button:active:not(:disabled) {
    transform: translateY(0);
  }

  .button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Variants */
  .button.primary {
    background-color: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
  }

  .button.primary:hover:not(:disabled) {
    background-color: var(--vscode-button-hoverBackground);
  }

  .button.secondary {
    background-color: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
  }

  .button.secondary:hover:not(:disabled) {
    background-color: var(--vscode-button-secondaryHoverBackground);
  }

  /* Sizes */
  .button.small {
    padding: 6px 12px;
    font-size: 12px;
  }

  .button.medium {
    padding: 8px 16px;
    font-size: 13px;
  }

  .button.large {
    padding: 12px 24px;
    font-size: 14px;
    width: 100%;
  }

  .icon {
    display: flex;
    align-items: center;
    font-size: 16px;
  }
</style>
