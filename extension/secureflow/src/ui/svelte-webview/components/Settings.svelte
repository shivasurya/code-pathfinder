<script lang="ts">
  import Select from './ui/Select.svelte';
  import Input from './ui/Input.svelte';
  import Button from './ui/Button.svelte';
  import Card from './ui/Card.svelte';
  import FormField from './ui/FormField.svelte';

  export let vscode: any;
  export let modelConfig: any;

  // State
  let selectedProvider = 'auto';
  let selectedModel = 'claude-sonnet-4-5-20250929';
  let customModelId = '';
  let apiKey = '';
  let showCustomModelInput = false;
  let isSubmitting = false;
  let saveStatus: 'idle' | 'success' | 'error' = 'idle';
  let errorMessage = '';

  // Listen for messages from extension
  if (typeof window !== 'undefined') {
    window.addEventListener('message', (event) => {
      const message = event.data;

      if (message.type === 'configSaved') {
        isSubmitting = false;
        if (message.success) {
          saveStatus = 'success';
          setTimeout(() => {
            saveStatus = 'idle';
          }, 3000);
        } else {
          saveStatus = 'error';
          errorMessage = message.error || 'Unknown error';
        }
      }

      if (message.type === 'currentConfig') {
        apiKey = message.config.apiKey || '';
        selectedModel = message.config.model || 'claude-sonnet-4-5-20250929';
        selectedProvider = message.config.provider || 'auto';

        // Show custom model input if provider is openrouter or if model contains "/"
        if (selectedProvider === 'openrouter' || selectedModel.includes('/')) {
          showCustomModelInput = true;
          customModelId = selectedModel;
        }
      }
    });
  }

  // Request current config when component mounts
  if (vscode) {
    vscode.postMessage({ type: 'getCurrentConfig' });
  }

  // Provider options
  const providers = [
    { value: 'auto', label: 'Auto-detect from model', description: 'Automatically detect provider from your model selection' },
    { value: 'anthropic', label: 'Anthropic (Claude)', description: 'Direct API access to Claude models' },
    { value: 'openai', label: 'OpenAI (GPT)', description: 'Direct API access to GPT models' },
    { value: 'google', label: 'Google (Gemini)', description: 'Direct API access to Gemini models' },
    { value: 'xai', label: 'xAI (Grok)', description: 'Direct API access to Grok models' },
    { value: 'openrouter', label: 'OpenRouter', description: 'Access 300+ models with one API key' },
    { value: 'ollama', label: 'Ollama (Local)', description: 'Local models, no API key needed' }
  ];

  // Group models by provider
  $: groupedModels = groupModelsByProvider(modelConfig?.models || []);

  function groupModelsByProvider(models: any[]) {
    const groups = {
      anthropic: { label: 'Anthropic (Direct API)', models: [] },
      openai: { label: 'OpenAI (Direct API)', models: [] },
      google: { label: 'Google (Direct API)', models: [] },
      xai: { label: 'xAI (Direct API)', models: [] },
      openrouter: { label: 'OpenRouter (Unified Access)', models: [] },
      ollama: { label: 'Ollama (Local)', models: [] }
    };

    models.forEach((model: any) => {
      if (groups[model.provider]) {
        groups[model.provider].models.push(model);
      }
    });

    return Object.entries(groups)
      .filter(([_, group]: any) => group.models.length > 0)
      .map(([key, group]: any) => ({ key, ...group }));
  }

  // Handle provider change
  function handleProviderChange(event: CustomEvent) {
    selectedProvider = event.detail.value;
    showCustomModelInput = selectedProvider === 'openrouter';

    if (!showCustomModelInput) {
      customModelId = '';
    }
  }

  // Handle model selection
  function handleModelChange(event: CustomEvent) {
    selectedModel = event.detail.value;
  }

  // Handle save
  function handleSave() {
    const finalModel = showCustomModelInput && customModelId ? customModelId : selectedModel;

    if (!apiKey.trim()) {
      saveStatus = 'error';
      errorMessage = 'API key is required';
      return;
    }

    if (!finalModel) {
      saveStatus = 'error';
      errorMessage = 'Please select or enter a model';
      return;
    }

    isSubmitting = true;
    saveStatus = 'idle';

    vscode.postMessage({
      type: 'saveConfig',
      apiKey: apiKey.trim(),
      model: finalModel,
      provider: selectedProvider === 'auto' ? undefined : selectedProvider,
      skipScan: true  // Don't trigger scan from Settings
    });
  }

  // Handle back
  function handleBack() {
    if (vscode) {
      vscode.postMessage({ type: 'backToProfiles' });
    }
  }
</script>

<div class="settings-container">
  <Card>
    <div class="settings-header">
      <h1>Settings</h1>
      <p class="subtitle">Configure your AI provider and model preferences</p>
    </div>

    <form on:submit|preventDefault={handleSave}>
      <div class="form-section">
        <FormField label="AI Provider">
          <Select
            options={providers}
            value={selectedProvider}
            on:change={handleProviderChange}
          />
        </FormField>

        {#if !showCustomModelInput}
          <FormField label="AI Model">
            <Select
              grouped={true}
              groups={groupedModels}
              value={selectedModel}
              on:change={handleModelChange}
            />
            <div class="help-text">
              Missing a model? <a href="https://github.com/shivasurya/code-pathfinder/issues" target="_blank">Request to include it here</a>
            </div>
          </FormField>
        {/if}

        {#if showCustomModelInput}
          <FormField label="Model ID">
            <Input
              type="text"
              bind:value={customModelId}
              placeholder="e.g., anthropic/claude-3.5-sonnet"
            />
            <div class="help-text">
              Enter any model available on OpenRouter (e.g., anthropic/claude-3.5-sonnet, google/gemini-pro-1.5).
              See <a href="https://openrouter.ai/models" target="_blank">OpenRouter Models</a> for the full list.
            </div>
          </FormField>
        {/if}

        <FormField label="API Key">
          <Input
            type="password"
            bind:value={apiKey}
            placeholder="Enter your API key"
          />
          <div class="help-text">
            {#if selectedProvider === 'openrouter'}
              Get your API key from <a href="https://openrouter.ai/keys" target="_blank">OpenRouter</a>
            {:else if selectedProvider === 'anthropic'}
              Get your API key from <a href="https://console.anthropic.com/" target="_blank">Anthropic Console</a>
            {:else if selectedProvider === 'openai'}
              Get your API key from <a href="https://platform.openai.com/api-keys" target="_blank">OpenAI</a>
            {:else if selectedProvider === 'google'}
              Get your API key from <a href="https://makersuite.google.com/app/apikey" target="_blank">Google AI Studio</a>
            {:else if selectedProvider === 'xai'}
              Get your API key from <a href="https://console.x.ai/" target="_blank">xAI Console</a>
            {:else}
              API key from your chosen provider
            {/if}
          </div>
        </FormField>
      </div>

      <div class="form-actions">
        <Button variant="secondary" type="button" on:click={handleBack}>
          Back to Profiles
        </Button>
        <Button variant="primary" type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Saving...' : 'Save Settings'}
        </Button>
      </div>

      {#if saveStatus === 'success'}
        <div class="status-message success">
          ✓ Settings saved successfully!
        </div>
      {/if}

      {#if saveStatus === 'error'}
        <div class="status-message error">
          ✗ {errorMessage}
        </div>
      {/if}
    </form>
  </Card>
</div>

<style>
  .settings-container {
    padding: 2rem;
    max-width: 600px;
    margin: 0 auto;
  }

  .settings-header {
    margin-bottom: 2rem;
  }

  .settings-header h1 {
    margin: 0 0 0.5rem 0;
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--vscode-foreground);
  }

  .subtitle {
    margin: 0;
    color: var(--vscode-descriptionForeground);
    font-size: 0.9rem;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .help-text {
    margin-top: 0.5rem;
    font-size: 0.85rem;
    color: var(--vscode-descriptionForeground);
  }

  .help-text a {
    color: var(--vscode-textLink-foreground);
    text-decoration: none;
  }

  .help-text a:hover {
    text-decoration: underline;
  }

  .form-actions {
    display: flex;
    gap: 1rem;
    margin-top: 1.5rem;
  }

  .status-message {
    margin-top: 1rem;
    padding: 0.75rem;
    border-radius: 4px;
    font-size: 0.9rem;
  }

  .status-message.success {
    background-color: var(--vscode-testing-iconPassed);
    color: var(--vscode-editor-background);
  }

  .status-message.error {
    background-color: var(--vscode-errorForeground);
    color: var(--vscode-editor-background);
  }
</style>
