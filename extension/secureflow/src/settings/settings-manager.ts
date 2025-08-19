import * as vscode from 'vscode';

/**
 * TODO(CLI): This settings manager depends on VS Code configuration and secret storage.
 * - Treat this file as EXTENSION-ONLY.
 * - The CLI will introduce a separate CLIConfigManager that reads from env/flags/~/.secureflow/config.json.
 * - Keep API surface unchanged for the extension; do not import this module from the CLI.
 */

export type AIModel =
  | 'gpt-4o'
  | 'gpt-4o-mini'
  | 'o1-mini'
  | 'o1'
  | 'gpt-4.1-2025-04-14'
  | 'o3-mini-2025-01-31'
  | 'gemini-2.5-pro'
  | 'gemini-2.5-flash'
  | 'claude-opus-4-1-20250805'
  | 'claude-opus-4-20250514'
  | 'claude-sonnet-4-20250514'
  | 'claude-3-7-sonnet-20250219'
  | 'claude-3-5-sonnet-20241022'
  | 'claude-3-5-haiku-20241022';

/**
 * Settings manager for SecureFlow extension
 */
export class SettingsManager {
  private context: vscode.ExtensionContext;

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
  }

  /**
   * Get the selected AI Model from user preferences
   */
  public getSelectedAIModel(): AIModel {
    const config = vscode.workspace.getConfiguration('secureflow');
    return config.get<AIModel>('AIModel') || 'claude-3-5-sonnet-20241022';
  }

  /**
   * Get the API Key from secure storage
   * @returns The API key if found, otherwise undefined
   */
  public async getApiKey(): Promise<string | undefined> {
    // just get from workspace settings
    const config = vscode.workspace.getConfiguration('secureflow');
    return config.get<string>('APIKey');
  }
}
