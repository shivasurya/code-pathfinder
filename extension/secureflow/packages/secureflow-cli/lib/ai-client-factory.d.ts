import { AIClient } from './ai-client';

/**
 * Factory class for creating AI clients
 */
export declare class AIClientFactory {
  /**
   * Get the appropriate AI client based on the model
   * @param model The AI model to use (can be any string including custom OpenRouter model IDs)
   * @returns The AI client for the specified model
   */
  static getClient(model: string): AIClient;
}
