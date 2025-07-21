import * as vscode from 'vscode';

export type AIModel = 'claude-3-5-sonnet-20241022' | 'claude' | 'gemini-2.5-pro' | 'gemini-2.5-flash' | 'openai';

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
        return config.get<AIModel>('AIModel') || 'openai';
    }
    
    /**
     * Get the API Key for the selected model
     */
    public async getApiKey(): Promise<string | undefined> {
        const key = `secureflow.APIKey`;

        // Try to get the key from secure storage first
        let apiKey = await this.context.secrets.get(key);
        
        // If not found in secure storage, check if it's in settings
        if (!apiKey) {
            const config = vscode.workspace.getConfiguration('secureflow');
            const configKey = config.get<string>('APIKey');
            
            // If found in settings, store it securely and clear from settings
            if (configKey) {
                await this.context.secrets.store(key, configKey);
                // Clear the key from settings to keep it secure
                await config.update('APIKey', '', vscode.ConfigurationTarget.Global);
                apiKey = configKey;
            }
        }
        
        return apiKey;
    }
    
    /**
     * Store API Key securely
     */
    public async storeApiKey(apiKey: string): Promise<void> {
        const key = `secureflow.APIKey`;
        await this.context.secrets.store(key, apiKey);
    }
}
