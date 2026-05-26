<script lang="ts">
  import Button from './ui/Button.svelte';

  export let vscode: any;
  export let user: {
    sub: string;
    name?: string;
    nickname?: string;
    email?: string;
    picture?: string;
  } | null = null;

  $: displayName = user
    ? user.nickname || user.name || user.email || user.sub
    : '';

  function handleSignIn() {
    vscode?.postMessage({ type: 'auth:login' });
  }

  function handleSignOut() {
    vscode?.postMessage({ type: 'auth:logout' });
  }
</script>

<header class="auth-header" class:signed-in={!!user}>
  {#if user}
    <div class="user">
      {#if user.picture}
        <img class="avatar" src={user.picture} alt={displayName} />
      {:else}
        <div class="avatar avatar-fallback">
          {displayName.charAt(0).toUpperCase()}
        </div>
      {/if}
      <span class="handle" title={user.email || displayName}>{displayName}</span>
    </div>
    <button class="link" on:click={handleSignOut} type="button">Sign out</button>
  {:else}
    <span class="status">Not signed in</span>
    <Button variant="primary" size="small" on:click={handleSignIn}>
      Sign in with GitHub
    </Button>
  {/if}
</header>

<style>
  .auth-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--vscode-widget-border, rgba(255, 255, 255, 0.1));
    background-color: var(--vscode-sideBar-background);
    font-size: 12px;
  }

  .user {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .avatar {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    flex-shrink: 0;
    object-fit: cover;
    background-color: var(--vscode-button-secondaryBackground);
  }

  .avatar-fallback {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--vscode-button-secondaryForeground);
    font-weight: 600;
    font-size: 11px;
  }

  .handle {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--vscode-foreground);
    font-weight: 500;
  }

  .status {
    color: var(--vscode-descriptionForeground);
  }

  .link {
    background: none;
    border: none;
    padding: 0;
    color: var(--vscode-textLink-foreground);
    cursor: pointer;
    font-family: var(--vscode-font-family);
    font-size: 12px;
    text-decoration: none;
  }

  .link:hover {
    text-decoration: underline;
  }
</style>
