import * as vscode from 'vscode';

export type AIModel = 
    | 'gpt-4o' 
    | 'gpt-4o-mini' 
    | 'o1-mini' 
    | 'o1' 
    | 'gpt-4.1-2025-04-14' 
    | 'o3-mini-2025-01-31' 
    | 'gemini-2.5-pro' 
    | 'gemini-2.5-flash' 
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
