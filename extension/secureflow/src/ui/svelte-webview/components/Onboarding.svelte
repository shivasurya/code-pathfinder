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
  let profileCount = 0;
  let scanCompleted = false;

  // Listen for messages from extension
  if (typeof window !== 'undefined') {
    window.addEventListener('message', (event) => {
      const message = event.data;
      console.log('Received message from extension:', message);

      if (message.type === 'configSaved') {
        isSubmitting = false;
        if (message.success) {
          console.log('Configuration saved successfully!');
          saveStatus = 'success';
        } else {
          console.error('Failed to save configuration:', message.error);
          saveStatus = 'error';
          errorMessage = message.error || 'Unknown error';
        }
      }

      // Handle scan completion
      if (message.type === 'scanComplete') {
        scanCompleted = true;
        if (message.success) {
          saveStatus = 'success';
          profileCount = message.profileCount || 0;
        } else {
          saveStatus = 'error';
          errorMessage = message.error || 'Scan failed';
        }
      }
    });
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

  // Handle custom model input
  function handleCustomModelInput(event: CustomEvent) {
    customModelId = event.detail;
  }

  // Handle form submission
  function handleGetStarted() {
    console.log('Get Started clicked', { selectedProvider, selectedModel, customModelId, hasApiKey: !!apiKey });

    // Reset status
    saveStatus = 'idle';
    errorMessage = '';

    // Use custom model if provided, otherwise use dropdown selection
    const finalModel = customModelId.trim() || selectedModel;

    // Validation
    if (!finalModel) {
      console.error('Validation failed: No model selected');
      saveStatus = 'error';
      errorMessage = 'Please select or enter an AI model';
      return;
    }

    if (!apiKey.trim() && selectedProvider !== 'ollama') {
      console.error('Validation failed: No API key');
      saveStatus = 'error';
      errorMessage = 'Please enter an API key (not required for Ollama)';
      return;
    }

    console.log('Sending saveConfig message:', {
      type: 'saveConfig',
      provider: selectedProvider,
      model: finalModel,
      apiKey: apiKey ? '***' : ''
    });

    isSubmitting = true;

    // Send to extension
    if (vscode) {
      vscode.postMessage({
        type: 'saveConfig',
        provider: selectedProvider,
        model: finalModel,
        apiKey: apiKey
      });
      console.log('Message sent to extension');
    } else {
      console.error('VSCode API not available!');
      saveStatus = 'error';
      errorMessage = 'VSCode API not available. Please reload the window.';
      isSubmitting = false;
    }
  }

  // Get recommended model
  $: recommendedModel = modelConfig?.models?.find((m: any) => m.recommended);
</script>

<div class="onboarding">
  <Card>
    <div class="header">
      <div class="icon">
        <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
          <rect width="48" height="48" rx="12" fill="var(--vscode-button-background)" opacity="0.1"/>
          <path d="M24 14L16 20V28L24 34L32 28V20L24 14Z" stroke="var(--vscode-button-background)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M24 22L28 24V28L24 30L20 28V24L24 22Z" fill="var(--vscode-button-background)"/>
        </svg>
      </div>
      <h1>Welcome to SecureFlow AI</h1>
      <p class="subtitle">powered by codepathfinder.dev</p>
    </div>

    <div class="form">
      <FormField label="Provider" description={providers.find(p => p.value === selectedProvider)?.description}>
        <Select
          options={providers}
          value={selectedProvider}
          on:change={handleProviderChange}
        />
      </FormField>

      <FormField
        label="AI Model"
        description={showCustomModelInput ? "Select a popular model or enter any OpenRouter model ID below" : "Choose your AI model from the list"}
      >
        <Select
          grouped={true}
          groups={groupedModels}
          value={selectedModel}
          on:change={handleModelChange}
          placeholder="Select AI model..."
        />

        {#if !showCustomModelInput}
          <p class="hint">Missing a model? <a href="https://github.com/shivasurya/code-pathfinder/issues" target="_blank">Request to include it here</a></p>
        {/if}

        {#if showCustomModelInput}
          <div class="custom-model-input">
            <Input
              type="text"
              placeholder="Or enter any OpenRouter model ID (e.g., cohere/command-r-plus)"
              bind:value={customModelId}
            />
            <p class="hint">Browse all models at <a href="https://openrouter.ai/models" target="_blank">openrouter.ai/models</a></p>
          </div>
        {/if}
      </FormField>

      <FormField
        label="API Key"
        description="Your API key is stored securely in VSCode settings"
      >
        <Input
          type="password"
          placeholder="Enter your API key..."
          bind:value={apiKey}
          icon="🔒"
        />
      </FormField>

      {#if saveStatus === 'success'}
        <div class="status-message success">
          <span class="status-icon">✅</span>
          <div class="status-text">
            <strong>Setup Complete!</strong>
            {#if scanCompleted}
              <p>Found {profileCount} application profile{profileCount !== 1 ? 's' : ''} in your workspace.</p>
              <p style="margin-top: 12px; font-weight: 500;">👈 Check the <strong>SecureFlow</strong> panel in the Activity Bar (left sidebar) to view your security profiles!</p>
            {:else}
              <p>Configuration saved. Analyzing your workspace...</p>
            {/if}
          </div>
        </div>
      {/if}

      {#if saveStatus === 'error'}
        <div class="status-message error">
          <span class="status-icon">❌</span>
          <div class="status-text">
            <strong>Error</strong>
            <p>{errorMessage}</p>
          </div>
        </div>
      {/if}

      <div class="actions">
        <Button variant="primary" size="large" on:click={handleGetStarted} disabled={isSubmitting || saveStatus === 'success'}>
          <span slot="icon">{isSubmitting ? '⏳' : '🚀'}</span>
          {isSubmitting ? 'Saving...' : 'Get Started'}
        </Button>
      </div>
    </div>

    <div class="footer">
      <p class="disclaimer">
        SecureFlow is powered by AI. While our analysis strives to be thorough,
        please review all suggestions carefully before implementation.
      </p>
    </div>
  </Card>
</div>

<style>
  .onboarding {
    width: 100%;
    max-width: 600px;
    margin: 0 auto;
  }

  .header {
    text-align: center;
    margin-bottom: 32px;
  }

  .icon {
    margin-bottom: 16px;
    display: flex;
    justify-content: center;
  }

  h1 {
    font-size: 24px;
    font-weight: 600;
    margin: 0 0 8px 0;
    color: var(--vscode-foreground);
  }

  .subtitle {
    font-size: 14px;
    margin: 0;
    opacity: 0.8;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .custom-model-input {
    margin-top: 12px;
  }

  .hint {
    margin: 8px 0 0 0;
    font-size: 12px;
    opacity: 0.7;
  }

  .hint a {
    color: var(--vscode-textLink-foreground);
    text-decoration: none;
  }

  .hint a:hover {
    text-decoration: underline;
  }

  .status-message {
    display: flex;
    gap: 12px;
    padding: 16px;
    border-radius: 6px;
    margin-bottom: 16px;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .status-message.success {
    background-color: rgba(22, 163, 74, 0.1);
    border: 1px solid rgba(22, 163, 74, 0.3);
  }

  .status-message.error {
    background-color: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .status-icon {
    font-size: 20px;
    flex-shrink: 0;
  }

  .status-text {
    flex: 1;
  }

  .status-text strong {
    display: block;
    margin-bottom: 4px;
    font-size: 14px;
  }

  .status-text p {
    margin: 0;
    font-size: 13px;
    opacity: 0.9;
  }

  .actions {
    margin-top: 8px;
  }

  .footer {
    margin-top: 32px;
    padding-top: 24px;
    border-top: 1px solid var(--vscode-widget-border);
  }

  .disclaimer {
    text-align: center;
    font-size: 12px;
    margin: 0;
    opacity: 0.6;
    line-height: 1.5;
  }
</style>
