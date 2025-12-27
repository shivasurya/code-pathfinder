<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let value: string;
  export let options: Array<{value: string, label: string}> | undefined = undefined;
  export let groups: Array<{key: string, label: string, models: any[]}> | undefined = undefined;
  export let grouped: boolean = false;
  export let placeholder: string = 'Select...';

  function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    dispatch('change', { value: target.value });
  }
</script>

<div class="select-wrapper">
  <select class="select" {value} on:change={handleChange}>
    {#if grouped && groups}
      {#each groups as group}
        <optgroup label={group.label}>
          {#each group.models as model}
            <option value={model.id}>
              {model.displayName}
            </option>
          {/each}
        </optgroup>
      {/each}
    {:else if options}
      {#each options as option}
        <option value={option.value}>{option.label}</option>
      {/each}
    {/if}
  </select>
  <div class="select-icon">
    <svg width="12" height="8" viewBox="0 0 12 8" fill="currentColor">
      <path d="M1 1L6 6L11 1" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
    </svg>
  </div>
</div>

<style>
  .select-wrapper {
    position: relative;
    width: 100%;
  }

  .select {
    width: 100%;
    padding: 10px 36px 10px 12px;
    background-color: var(--vscode-input-background);
    color: var(--vscode-input-foreground);
    border: 1px solid var(--vscode-input-border);
    border-radius: 4px;
    font-size: 13px;
    font-family: var(--vscode-font-family);
    cursor: pointer;
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    transition: border-color 0.2s, background-color 0.2s;
  }

  .select:hover {
    border-color: var(--vscode-focusBorder);
  }

  .select:focus {
    outline: none;
    border-color: var(--vscode-focusBorder);
    background-color: var(--vscode-input-background);
  }

  .select-icon {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
    opacity: 0.6;
  }

  .select option {
    background-color: var(--vscode-dropdown-background);
    color: var(--vscode-dropdown-foreground);
  }
</style>
