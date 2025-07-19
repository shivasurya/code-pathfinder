import { AIClient } from './ai-client';
import { ClaudeClient } from './claude-client';
import { GeminiClient } from './gemini-client';
import { OpenAIClient } from './openai-client';
import { AIModel } from '../settings-manager';

/**
 * Factory class for creating AI clients
 */
export class AIClientFactory {
    /**
     * Get the appropriate AI client based on the model
     * @param model The AI model to use
     * @returns The AI client for the specified model
     */
    public static getClient(model: AIModel): AIClient {
        switch (model) {
            case 'openai':
                return new OpenAIClient();
            case 'claude':
                return new ClaudeClient();
            case 'gemini':
                return new GeminiClient();
            case 'claude-3-5-sonnet-20241022':
                return new ClaudeClient();
            default:
                throw new Error(`Unsupported AI model: ${model}`);
        }
    }
}
