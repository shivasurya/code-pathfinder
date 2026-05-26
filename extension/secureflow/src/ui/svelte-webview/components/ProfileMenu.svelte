<script lang="ts">
  import { onDestroy } from 'svelte';

  export let vscode: any;
  export let session: {
    kind: 'user';
    user: {
      sub: string;
      name?: string;
      nickname?: string;
      email?: string;
      picture?: string;
    };
  } | { kind: 'guest' };

  let open = false;
  let rootEl: HTMLDivElement;

  $: displayName =
    session.kind === 'guest'
      ? 'Guest'
      : session.user.nickname ||
        session.user.name ||
        session.user.email ||
        session.user.sub;

  $: tooltip =
    session.kind === 'guest'
      ? 'Browsing as guest'
      : session.user.email || displayName;

  function toggle() {
    open = !open;
    if (open) {
      // Attach outside-click listener only while open.
      setTimeout(() => document.addEventListener('mousedown', onOutside), 0);
    }
  }

  function close() {
    open = false;
    document.removeEventListener('mousedown', onOutside);
  }

  function onOutside(e: MouseEvent) {
    if (rootEl && !rootEl.contains(e.target as Node)) close();
  }

  function signOut() {
    close();
    vscode?.postMessage({ type: 'auth:logout' });
  }

  onDestroy(() => document.removeEventListener('mousedown', onOutside));
</script>

<div class="profile-menu" bind:this={rootEl}>
  <button
    class="trigger"
    type="button"
    on:click={toggle}
    aria-expanded={open}
    aria-haspopup="menu"
    title={tooltip}
  >
    <span class="avatar">
      {#if session.kind === 'user' && session.user.picture}
        <img src={session.user.picture} alt="" />
      {:else}
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      {/if}
    </span>
    <span class="name">{displayName}</span>
  </button>

  {#if open}
    <div class="dropdown" role="menu">
      <button class="item" type="button" role="menuitem" on:click={signOut}>
        <svg
          viewBox="0 0 24 24"
          width="14"
          height="14"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
          <polyline points="16 17 21 12 16 7" />
          <line x1="21" y1="12" x2="9" y2="12" />
        </svg>
        Sign out
      </button>
    </div>
  {/if}
</div>

<style>
  .profile-menu {
    position: relative;
    display: inline-block;
  }

  .trigger {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    background: transparent;
    border: none;
    padding: 4px 6px;
    cursor: pointer;
    color: var(--vscode-foreground);
    font-family: var(--vscode-font-family);
    border-radius: 4px;
  }

  .trigger:hover {
    background: var(--vscode-list-hoverBackground);
  }

  .avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
  }

  .avatar img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .avatar svg {
    width: 18px;
    height: 18px;
  }

  .name {
    font-size: 11px;
    max-width: 80px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 140px;
    background: var(--vscode-menu-background, var(--vscode-editor-background));
    color: var(--vscode-menu-foreground, var(--vscode-foreground));
    border: 1px solid var(--vscode-menu-border, var(--vscode-widget-border, rgba(128, 128, 128, 0.4)));
    border-radius: 4px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
    padding: 4px;
    z-index: 10;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 10px;
    background: transparent;
    border: none;
    border-radius: 3px;
    font-family: var(--vscode-font-family);
    font-size: 13px;
    color: inherit;
    cursor: pointer;
    text-align: left;
  }

  .item:hover {
    background: var(--vscode-menu-selectionBackground, var(--vscode-list-hoverBackground));
    color: var(--vscode-menu-selectionForeground, inherit);
  }
</style>
