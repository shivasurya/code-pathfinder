/**
 * Interface for AI model providers
 */

export interface AIClientOptions {
  apiKey: string;
  model?: string;
  temperature?: number;
  maxTokens?: number;
}

export interface AIResponseChunk {
  content: string;
  isComplete: boolean;
}

export interface AIResponse {
  content: string;
  model: string;
  provider: string;
}

/**
 * Interface that all AI model providers must implement
 */
export declare class AIClient {
  /**
   * Send a request to the AI model
   * @param prompt The prompt to send to the AI model
   * @param options Options for the request
   * @param messages Messages for the request
   * @returns The AI model response
   */
  sendRequest(prompt?: string, options?: AIClientOptions, messages?: any): Promise<AIResponse>;

  /**
   * Send a streaming request to the AI model
   * @param prompt The prompt to send to the AI model
   * @param callback Callback function for each chunk of the response
   * @param options Options for the request
   * @param messages Messages for the request
   */
  sendStreamingRequest(
    prompt?: string,
    callback: (chunk: AIResponseChunk) => void,
    options?: AIClientOptions,
    messages?: any
  ): Promise<void>;
}
