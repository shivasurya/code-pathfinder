<script lang="ts">
  export let label: string;
  export let description: string | undefined = undefined;
</script>

<div class="form-field">
  <label class="label">{label}</label>
  <div class="control">
    <slot />
  </div>
  {#if description}
    <p class="description">{description}</p>
  {/if}
</div>

<style>
  .form-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .label {
    font-size: 13px;
    font-weight: 500;
    color: var(--vscode-foreground);
    display: block;
  }

  .control {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .description {
    margin: 0;
    font-size: 12px;
    opacity: 0.7;
    line-height: 1.4;
  }
</style>
