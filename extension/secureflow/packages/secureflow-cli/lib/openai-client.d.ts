import { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
import { HttpClient } from './http-client';

export declare class OpenAIClient extends HttpClient implements AIClient {
  constructor();
  sendRequest(prompt: string, options?: AIClientOptions): Promise<AIResponse>;
  sendStreamingRequest(
    prompt: string,
    callback: (chunk: AIResponseChunk) => void,
    options?: AIClientOptions
  ): Promise<void>;
}
