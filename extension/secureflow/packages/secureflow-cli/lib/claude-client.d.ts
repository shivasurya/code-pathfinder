import { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
import { HttpClient } from './http-client';

export declare class ClaudeClient extends HttpClient implements AIClient {
  constructor();
  sendRequest(prompt?: string, options?: AIClientOptions, messages?: any): Promise<AIResponse>;
  sendStreamingRequest(
    prompt?: string,
    callback: (chunk: AIResponseChunk) => void,
    options?: AIClientOptions,
    messages?: any
  ): Promise<void>;
}
