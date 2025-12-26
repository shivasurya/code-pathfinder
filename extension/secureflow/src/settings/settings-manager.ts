import * as vscode from 'vscode';

/**
 * TODO(CLI): This settings manager depends on VS Code configuration and secret storage.
 * - Treat this file as EXTENSION-ONLY.
 * - The CLI will introduce a separate CLIConfigManager that reads from env/flags/~/.secureflow/config.json.
 * - Keep API surface unchanged for the extension; do not import this module from the CLI.
 * 
 * NOTE: AIModel type is auto-generated from config/models.json
 * Run `npm run generate:models` from the CLI package to update
 */

// Import and re-export AIModel type from generated configuration
import type { AIModel } from '../generated/model-config';
export type { AIModel };

/**
 * AI Provider type
 */
export type AIProvider = 'auto' | 'openai' | 'anthropic' | 'google' | 'xai' | 'ollama' | 'openrouter';

/**
 * Settings manager for SecureFlow extension
 */
export class SettingsManager {
  private context: vscode.ExtensionContext;

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
  }

  /**
   * Get the selected AI Provider from user preferences
   */
  public getSelectedProvider(): AIProvider {
    const config = vscode.workspace.getConfiguration('secureflow');
    return config.get<AIProvider>('Provider') || 'auto';
  }

  /**
   * Get the selected AI Model from user preferences
   * Note: For OpenRouter, this can be any model ID (e.g., "anthropic/claude-3-5-sonnet")
   */
  public getSelectedAIModel(): string {
    const config = vscode.workspace.getConfiguration('secureflow');
    return config.get<string>('AIModel') || 'claude-sonnet-4-5-20250929';
  }

  /**
   * Get the API Key from workspace settings
   * @returns The API key if found, otherwise undefined
   */
  public async getApiKey(): Promise<string | undefined> {
    // just get from workspace settings
    const config = vscode.workspace.getConfiguration('secureflow');
    return config.get<string>('APIKey');
  }
}
