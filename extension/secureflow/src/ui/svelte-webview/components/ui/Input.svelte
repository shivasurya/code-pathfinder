<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let type: string = 'text';
  export let value: string = '';
  export let placeholder: string = '';
  export let icon: string | undefined = undefined;

  function handleInput(event: Event) {
    const target = event.target as HTMLInputElement;
    value = target.value;
    dispatch('input', target.value);
  }
</script>

<div class="input-wrapper">
  {#if icon}
    <span class="input-icon">{icon}</span>
  {/if}
  <input
    class="input"
    class:with-icon={icon}
    {type}
    {value}
    {placeholder}
    on:input={handleInput}
  />
</div>

<style>
  .input-wrapper {
    position: relative;
    width: 100%;
  }

  .input {
    width: 100%;
    padding: 10px 12px;
    background-color: var(--vscode-input-background);
    color: var(--vscode-input-foreground);
    border: 1px solid var(--vscode-input-border);
    border-radius: 4px;
    font-size: 13px;
    font-family: var(--vscode-font-family);
    transition: border-color 0.2s, background-color 0.2s;
    box-sizing: border-box;
  }

  .input.with-icon {
    padding-left: 36px;
  }

  .input::placeholder {
    color: var(--vscode-input-placeholderForeground);
    opacity: 0.6;
  }

  .input:hover {
    border-color: var(--vscode-focusBorder);
  }

  .input:focus {
    outline: none;
    border-color: var(--vscode-focusBorder);
    background-color: var(--vscode-input-background);
  }

  .input-icon {
    position: absolute;
    left: 12px;
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
    font-size: 14px;
  }
</style>
