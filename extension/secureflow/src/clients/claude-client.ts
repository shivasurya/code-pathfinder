import { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
import { HttpClient } from './http-client';

interface ClaudeCompletionResponse {
    content: Array<{
        text: string;
        type: string;
    }>;
    model: string;
}

export class ClaudeClient extends HttpClient implements AIClient {
    private static readonly API_URL = 'https://api.anthropic.com/v1/messages';
    private defaultModel = 'claude-3-5-sonnet-20241022';

    /**
     * Send a request to the Anthropic Claude API
     * @param prompt The prompt to send
     * @param options Claude-specific options
     * @returns The AI response
     */
    public async sendRequest(prompt: string, options?: AIClientOptions): Promise<AIResponse> {
        if (!options?.apiKey) {
            throw new Error('Anthropic Claude API key is required');
        }

        const response = await this.post(
            ClaudeClient.API_URL,
            {
                model: options.model || this.defaultModel,
                messages: [{ role: 'user', content: prompt }],
                temperature: options.temperature || 0,
                max_tokens: options.maxTokens || 500,
                stream: false
            },
            {
                'x-api-key': options.apiKey,
                'anthropic-version': '2023-06-01', // Use appropriate API version
                'Content-Type': 'application/json'
            }
        ) as ClaudeCompletionResponse;

        // Extract text content from Claude's response
        const content = response.content
            .filter(item => item.type === 'text')
            .map(item => item.text)
            .join('');

        return {
            content,
            model: response.model,
            provider: 'claude'
        };
    }

    /**
     * Send a streaming request to the Anthropic Claude API
     * @param prompt The prompt to send
     * @param callback Callback function for each chunk
     * @param options Claude-specific options
     */
    public async sendStreamingRequest(
        prompt: string,
        callback: (chunk: AIResponseChunk) => void,
        options?: AIClientOptions
    ): Promise<void> {
        if (!options?.apiKey) {
            throw new Error('Anthropic Claude API key is required');
        }

        let contentSoFar = '';
        
        await this.streamingPost(
            ClaudeClient.API_URL,
            {
                model: options.model || this.defaultModel,
                messages: [{ role: 'user', content: prompt }],
                temperature: options.temperature || 0.7,
                max_tokens: options.maxTokens || 500,
                stream: true
            },
            (chunk: string) => {
                try {
                    // Claude also uses SSE format with data: prefix
                    const lines = chunk.split('\n').filter(line => line.trim() !== '');
                    
                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const data = line.slice(6); // Remove 'data: ' prefix
                            
                            if (data === '[DONE]') {
                                callback({ content: contentSoFar, isComplete: true });
                                return;
                            }
                            
                            try {
                                const parsed = JSON.parse(data);
                                if (parsed.type === 'content_block_delta' && 
                                    parsed.delta && 
                                    parsed.delta.text) {
                                    const content = parsed.delta.text;
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
                'x-api-key': options.apiKey,
                'anthropic-version': '2023-06-01',
                'Content-Type': 'application/json',
                'Accept': 'text/event-stream'
            }
        );
    }
}
