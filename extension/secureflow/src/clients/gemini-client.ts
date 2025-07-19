import { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
import { HttpClient } from './http-client';

interface GeminiCompletionResponse {
    candidates: Array<{
        content: {
            parts: Array<{
                text: string;
            }>;
        }
    }>;
    promptFeedback?: any;
}

export class GeminiClient extends HttpClient implements AIClient {
    private static readonly API_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models';
    private defaultModel = 'gemini-pro';

    /**
     * Send a request to the Google Gemini API
     * @param prompt The prompt to send
     * @param options Gemini-specific options
     * @returns The AI response
     */
    public async sendRequest(prompt: string, options?: AIClientOptions): Promise<AIResponse> {
        if (!options?.apiKey) {
            throw new Error('Google Gemini API key is required');
        }

        const model = options.model || this.defaultModel;
        const url = `${GeminiClient.API_BASE_URL}/${model}:generateContent?key=${options.apiKey}`;

        const response = await this.post(
            url,
            {
                contents: [
                    {
                        parts: [
                            { text: prompt }
                        ]
                    }
                ],
                generationConfig: {
                    temperature: options.temperature || 0.7,
                    maxOutputTokens: options.maxTokens || 500,
                }
            },
            {
                'Content-Type': 'application/json'
            }
        ) as GeminiCompletionResponse;

        // Extract text from Gemini's response
        const content = response.candidates[0]?.content?.parts
            .map(part => part.text)
            .join('') || '';

        return {
            content,
            model: model,
            provider: 'gemini'
        };
    }

    /**
     * Send a streaming request to the Google Gemini API
     * @param prompt The prompt to send
     * @param callback Callback function for each chunk
     * @param options Gemini-specific options
     */
    public async sendStreamingRequest(
        prompt: string,
        callback: (chunk: AIResponseChunk) => void,
        options?: AIClientOptions
    ): Promise<void> {
        if (!options?.apiKey) {
            throw new Error('Google Gemini API key is required');
        }

        const model = options.model || this.defaultModel;
        const url = `${GeminiClient.API_BASE_URL}/${model}:streamGenerateContent?key=${options.apiKey}`;

        let contentSoFar = '';
        
        await this.streamingPost(
            url,
            {
                contents: [
                    {
                        parts: [
                            { text: prompt }
                        ]
                    }
                ],
                generationConfig: {
                    temperature: options.temperature || 0.7,
                    maxOutputTokens: options.maxTokens || 500,
                }
            },
            (chunk: string) => {
                try {
                    // Gemini may return multiple JSON objects in a single chunk
                    const jsonObjects = this.parseStreamChunk(chunk);
                    
                    for (const obj of jsonObjects) {
                        if (obj.candidates && obj.candidates[0]?.content?.parts) {
                            const parts = obj.candidates[0].content.parts;
                            for (const part of parts) {
                                if (part.text) {
                                    contentSoFar += part.text;
                                    callback({ content: contentSoFar, isComplete: false });
                                }
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
                'Content-Type': 'application/json'
            }
        );
    }

    /**
     * Parse multiple JSON objects from a stream chunk
     * @param chunk Raw stream chunk that may contain multiple JSON objects
     * @returns Array of parsed JSON objects
     */
    private parseStreamChunk(chunk: string): any[] {
        const results: any[] = [];
        
        // Split by newlines and attempt to parse each line as JSON
        const lines = chunk.split('\n');
        
        for (const line of lines) {
            if (line.trim()) {
                try {
                    const parsed = JSON.parse(line);
                    results.push(parsed);
                } catch (e) {
                    console.warn('Failed to parse JSON line:', line);
                }
            }
        }
        
        return results;
    }
}
