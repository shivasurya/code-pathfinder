<script lang="ts">
  import Onboarding from './components/Onboarding.svelte';
  import Settings from './components/Settings.svelte';
  import ProfilesList from './components/ProfilesList.svelte';
  import ProfileDetails from './components/ProfileDetails.svelte';
  import { onMount } from 'svelte';

  // VSCode API - acquire it safely
  let vscode: any;

  // Model configuration injected by the extension
  let modelConfig: any = null;

  // Application state
  type View = 'onboarding' | 'settings' | 'profiles' | 'profileDetails';
  let currentView: View = 'onboarding';
  let profiles: any[] = [];
  let selectedProfile: any = null;
  let scans: any[] = [];
  let isConfigured: boolean = false;

  onMount(() => {
    // Acquire VSCode API
    try {
      vscode = window.acquireVsCodeApi();
    } catch (e) {
      console.error('Failed to acquire VSCode API:', e);
    }

    // Get model config from window (injected by extension)
    if (window.modelConfig) {
      modelConfig = window.modelConfig;
      console.log('Model config loaded:', modelConfig);
    } else {
      console.error('No modelConfig found on window');
    }

    // Listen for messages from extension
    window.addEventListener('message', (event) => {
      const message = event.data;

      switch (message.type) {
        case 'modelConfig':
          modelConfig = message.config;
          break;

        case 'onboardingStatus':
          console.log('Received onboarding status:', message.isConfigured);
          isConfigured = message.isConfigured;
          // If already configured but on onboarding, switch to profiles view
          if (isConfigured && currentView === 'onboarding') {
            currentView = 'profiles';
          }
          break;

        case 'updateProfiles':
          console.log('Received profiles:', message.profiles);
          profiles = message.profiles || [];
          // If configured (API key & model set), show profiles view even if no profiles yet
          // If not configured, stay on onboarding unless there are profiles
          if (currentView === 'onboarding') {
            if (isConfigured || profiles.length > 0) {
              currentView = 'profiles';
            }
          }
          break;

        case 'profileDetails':
          console.log('Received profile details:', message.profile);
          selectedProfile = message.profile;
          currentView = 'profileDetails';
          break;

        case 'updateScans':
          console.log('Received scans:', message.scans);
          scans = message.scans || [];
          break;

        case 'profileDeleted':
          // Go back to profiles list
          currentView = 'profiles';
          selectedProfile = null;
          // Request updated profiles list
          if (vscode) {
            vscode.postMessage({ type: 'getProfiles' });
          }
          break;

        case 'showSettings':
          currentView = 'settings';
          break;

        case 'backToProfiles':
          currentView = 'profiles';
          break;
      }
    });

    // Request initial state
    if (vscode) {
      vscode.postMessage({ type: 'checkOnboardingStatus' });
      vscode.postMessage({ type: 'getProfiles' });
    }
  });

  function handleBackToProfiles() {
    currentView = 'profiles';
    selectedProfile = null;
  }
</script>

<main>
  {#if modelConfig}
    {#if currentView === 'onboarding'}
      <Onboarding {vscode} {modelConfig} />
    {:else if currentView === 'settings'}
      <Settings {vscode} {modelConfig} />
    {:else if currentView === 'profiles'}
      <ProfilesList {vscode} {profiles} />
    {:else if currentView === 'profileDetails' && selectedProfile}
      <ProfileDetails {vscode} profile={selectedProfile} {scans} onBack={handleBackToProfiles} />
    {/if}
  {:else}
    <div class="loading">
      <p>Loading configuration...</p>
      <p style="font-size: 12px; margin-top: 10px;">If this persists, check the Developer Tools console (Help > Toggle Developer Tools)</p>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: var(--vscode-font-family);
    font-size: var(--vscode-font-size);
    color: var(--vscode-foreground);
    background-color: var(--vscode-editor-background);
  }

  /* Global severity badge styles */
  :global(.severity-critical) {
    background: rgba(220, 38, 38, 0.2);
    color: #fca5a5;
    border: 1px solid rgba(220, 38, 38, 0.3);
  }

  :global(.severity-high) {
    background: rgba(251, 146, 60, 0.2);
    color: #fdba74;
    border: 1px solid rgba(251, 146, 60, 0.3);
  }

  :global(.severity-medium) {
    background: rgba(245, 158, 11, 0.2);
    color: #fbbf24;
    border: 1px solid rgba(245, 158, 11, 0.3);
  }

  :global(.severity-low) {
    background: rgba(34, 197, 94, 0.2);
    color: #86efac;
    border: 1px solid rgba(34, 197, 94, 0.3);
  }

  main {
    width: 100%;
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    box-sizing: border-box;
  }

  .loading {
    text-align: center;
    opacity: 0.6;
  }
</style>
