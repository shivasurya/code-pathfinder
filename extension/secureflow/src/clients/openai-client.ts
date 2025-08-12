import {
  AIClient,
  AIClientOptions,
  AIResponse,
  AIResponseChunk
} from './ai-client';
import { HttpClient } from './http-client';

interface OpenAICompletionResponse {
  choices: Array<{
    message: {
      content: string;
    };
  }>;
  model: string;
}

export class OpenAIClient extends HttpClient implements AIClient {
  private static readonly API_URL =
    'https://api.openai.com/v1/chat/completions';
  private defaultModel = 'gpt-3.5-turbo';

  /**
   * Send a request to the OpenAI API
   * @param prompt The prompt to send
   * @param options OpenAI-specific options
   * @returns The AI response
   */
  public async sendRequest(
    prompt: string,
    options?: AIClientOptions
  ): Promise<AIResponse> {
    if (!options?.apiKey) {
      throw new Error('OpenAI API key is required');
    }

    const response = (await this.post(
      OpenAIClient.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0,
        max_tokens: options.maxTokens || 2000,
        stream: false
      },
      {
        Authorization: `Bearer ${options.apiKey}`,
        'Content-Type': 'application/json'
      }
    )) as OpenAICompletionResponse;

    return {
      content: response.choices[0].message.content,
      model: response.model,
      provider: 'openai'
    };
  }

  /**
   * Send a streaming request to the OpenAI API
   * @param prompt The prompt to send
   * @param callback Callback function for each chunk
   * @param options OpenAI-specific options
   */
  public async sendStreamingRequest(
    prompt: string,
    callback: (chunk: AIResponseChunk) => void,
    options?: AIClientOptions
  ): Promise<void> {
    if (!options?.apiKey) {
      throw new Error('OpenAI API key is required');
    }

    let contentSoFar = '';

    await this.streamingPost(
      OpenAIClient.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0,
        max_tokens: options.maxTokens || 2000,
        stream: true
      },
      (chunk: string) => {
        try {
          // OpenAI sends "data: " prefixed SSE
          const lines = chunk.split('\n').filter((line) => line.trim() !== '');

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6); // Remove 'data: ' prefix

              if (data === '[DONE]') {
                callback({ content: contentSoFar, isComplete: true });
                return;
              }

              try {
                const parsed = JSON.parse(data);
                if (parsed.choices && parsed.choices[0]?.delta?.content) {
                  const content = parsed.choices[0].delta.content;
                  contentSoFar += content;
                  callback({ content: contentSoFar, isComplete: false });
                }
              } catch (e) {
                console.error('Error parsing SSE data:', e);
              }
            }
          }
        } catch (error) {
          console.error('Error processing chunk:', error);
        }
      },
      () => {
        // Final completion callback
        callback({ content: contentSoFar, isComplete: true });
      },
      {
        Authorization: `Bearer ${options.apiKey}`,
        'Content-Type': 'application/json',
        Accept: 'text/event-stream'
      }
    );
  }
}
