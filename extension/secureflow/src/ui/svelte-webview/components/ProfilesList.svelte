<script lang="ts">
  import Button from './ui/Button.svelte';

  export let vscode: any;
  export let profiles: any[];

  let isScanning = false;

  function handleScanWorkspace() {
    if (vscode) {
      isScanning = true;
      vscode.postMessage({ type: 'scanWorkspace' });
    }
  }

  function handleProfileClick(profile: any, event: Event) {
    event.stopPropagation();
    console.log('Profile clicked:', profile);
    if (vscode) {
      vscode.postMessage({
        type: 'profileSelected',
        profileId: profile.id
      });
    }
  }

  function openSettings() {
    // Send message to show settings view
    if (vscode) {
      vscode.postMessage({ type: 'showSettings' });
    }
  }

  // Listen for scan completion
  window.addEventListener('message', (event) => {
    const message = event.data;
    if (message.type === 'scanComplete') {
      isScanning = false;
    }
  });
</script>

<!-- Profiles List View -->
<div class="profiles-container">
    <div class="header">
      <div class="header-content">
        <div class="header-text">
          <h1>Security Profiles</h1>
          <p class="subtitle">Application profiles detected in your workspace</p>
        </div>
        <button class="settings-icon-btn" on:click={openSettings} title="Settings">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
            <circle cx="12" cy="12" r="3"></circle>
          </svg>
        </button>
      </div>
    </div>

    <div class="actions">
      <Button variant="primary" size="medium" on:click={handleScanWorkspace} disabled={isScanning}>
        <span slot="icon">
          {#if isScanning}
            <svg class="spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 16px; height: 16px; display: block;">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 16px; height: 16px; display: block;">
              <path d="M3 7V5a2 2 0 0 1 2-2h2"></path>
              <path d="M17 3h2a2 2 0 0 1 2 2v2"></path>
              <path d="M21 17v2a2 2 0 0 1-2 2h-2"></path>
              <path d="M7 21H5a2 2 0 0 1-2-2v-2"></path>
              <circle cx="12" cy="12" r="3"></circle>
            </svg>
          {/if}
        </span>
        {isScanning ? 'Scanning...' : 'Scan Workspace'}
      </Button>
    </div>

    <div class="profiles-list">
      {#if profiles.length === 0}
        <div class="empty-state-wrapper">
          <div class="empty-state-compact">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
            </svg>
            <h4>No Profiles Found</h4>
            <p>Click "Scan Workspace" to analyze your project and detect application profiles.</p>
          </div>
        </div>
      {:else}
        {#each profiles as profile}
          <div class="profile-card-wrapper">
            <div class="profile-card-compact">
              <div class="profile-header-compact">
                <div class="profile-title-group">
                  <h4 class="profile-name-compact">{profile.name || 'Unknown Application'}</h4>
                  <span class="profile-category-compact">{profile.category || 'Unknown'}</span>
                </div>
                {#if profile.confidence}
                  <div class="profile-confidence-compact">{profile.confidence}%</div>
                {/if}
              </div>

              <div class="profile-info-compact">
                {#if profile.subcategory}
                  <div class="profile-subcategory-compact">{profile.subcategory}</div>
                {/if}

                {#if profile.technology}
                  <div class="profile-tech-compact">
                    <span class="tech-badge-compact">{profile.technology}</span>
                  </div>
                {/if}

                {#if profile.languages && profile.languages.length > 0}
                  <div class="profile-meta-row">
                    <span class="meta-label-compact">Languages</span>
                    <span class="meta-value-compact">{profile.languages.join(', ')}</span>
                  </div>
                {/if}

                {#if profile.frameworks && profile.frameworks.length > 0}
                  <div class="profile-meta-row">
                    <span class="meta-label-compact">Frameworks</span>
                    <span class="meta-value-compact">{profile.frameworks.join(', ')}</span>
                  </div>
                {/if}
              </div>

              <button class="view-details-btn" on:click={(e) => handleProfileClick(profile, e)}>
                <span>View Details</span>
                <svg class="arrow-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
              </button>
            </div>
          </div>
        {/each}
      {/if}
    </div>
</div>

<style>
  .profiles-container {
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
    padding: 20px;
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .header {
    margin-bottom: 16px;
  }

  .header-content {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
  }

  .header-text {
    flex: 1;
  }

  h1 {
    font-size: 20px;
    font-weight: 600;
    margin: 0 0 4px 0;
    color: var(--vscode-foreground);
  }

  .subtitle {
    font-size: 13px;
    margin: 0;
    opacity: 0.7;
  }

  .settings-icon-btn {
    background: var(--vscode-button-secondaryBackground);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 4px;
    padding: 8px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    color: var(--vscode-foreground);
  }

  .settings-icon-btn:hover {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border-color: var(--vscode-button-background);
  }

  .settings-icon-btn svg {
    width: 18px;
    height: 18px;
    stroke: currentColor;
  }

  .actions {
    margin-bottom: 16px;
  }

  .profiles-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    overflow-y: auto;
    flex: 1;
    min-height: 0;
    max-height: calc(100vh - 200px);
    padding-right: 4px;
  }

  .profiles-list::-webkit-scrollbar {
    width: 6px;
  }

  .profiles-list::-webkit-scrollbar-track {
    background: transparent;
  }

  .profiles-list::-webkit-scrollbar-thumb {
    background: var(--vscode-scrollbarSlider-background);
    border-radius: 3px;
  }

  .profiles-list::-webkit-scrollbar-thumb:hover {
    background: var(--vscode-scrollbarSlider-hoverBackground);
  }

  /* Empty State */
  .empty-state-wrapper {
    background-color: var(--vscode-editor-background);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 6px;
    padding: 10px;
  }

  .empty-state-compact {
    text-align: center;
    padding: 30px 20px;
  }

  .empty-icon {
    width: 40px;
    height: 40px;
    stroke: currentColor;
    opacity: 0.5;
    margin: 0 auto 12px;
  }

  .empty-state-compact h4 {
    font-size: 15px;
    margin: 0 0 6px 0;
    font-weight: 500;
  }

  .empty-state-compact p {
    margin: 0;
    font-size: 13px;
    opacity: 0.7;
  }

  /* Profile Card Wrapper */
  .profile-card-wrapper {
    background-color: var(--vscode-editor-background);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 6px;
    padding: 10px;
    transition: all 0.2s;
  }

  .profile-card-wrapper:hover {
    border-color: var(--vscode-button-background);
  }

  .profile-card-compact {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  /* Profile Header */
  .profile-header-compact {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
  }

  .profile-title-group {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
  }

  .profile-name-compact {
    font-size: 14px;
    margin: 0;
    font-weight: 600;
    color: var(--vscode-foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .profile-category-compact {
    background-color: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
    padding: 2px 8px;
    border-radius: 8px;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    white-space: nowrap;
  }

  .profile-confidence-compact {
    font-size: 11px;
    font-weight: 600;
    color: var(--vscode-button-background);
    padding: 2px 6px;
    background: var(--vscode-button-secondaryBackground);
    border-radius: 4px;
    white-space: nowrap;
  }

  /* Profile Info */
  .profile-info-compact {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .profile-subcategory-compact {
    font-size: 12px;
    opacity: 0.7;
    margin: 0;
  }

  .profile-tech-compact {
    margin: 0;
  }

  .tech-badge-compact {
    display: inline-block;
    background-color: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
  }

  .profile-meta-row {
    display: flex;
    gap: 6px;
    font-size: 12px;
  }

  .meta-label-compact {
    opacity: 0.6;
    font-weight: 500;
    white-space: nowrap;
  }

  .meta-value-compact {
    opacity: 0.9;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* View Details Button */
  .view-details-btn {
    background: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 4px;
    padding: 6px 10px;
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: all 0.2s;
    margin-top: 2px;
    font-family: var(--vscode-font-family);
    font-weight: 500;
  }

  .view-details-btn:hover {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border-color: var(--vscode-button-background);
  }

  .arrow-icon {
    width: 14px;
    height: 14px;
    stroke: currentColor;
    opacity: 0.7;
    transition: transform 0.2s;
  }

  .view-details-btn:hover .arrow-icon {
    transform: translateX(2px);
    opacity: 1;
  }

  /* Spinner animation */
  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .spinner {
    animation: spin 1s linear infinite;
  }
</style>
