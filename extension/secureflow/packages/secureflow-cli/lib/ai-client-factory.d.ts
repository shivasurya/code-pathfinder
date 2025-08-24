import { AIClient } from './ai-client';
import { AIModel } from './types';

/**
 * Factory class for creating AI clients
 */
export declare class AIClientFactory {
  /**
   * Get the appropriate AI client based on the model
   * @param model The AI model to use
   * @returns The AI client for the specified model
   */
  static getClient(model: AIModel): AIClient;
}
