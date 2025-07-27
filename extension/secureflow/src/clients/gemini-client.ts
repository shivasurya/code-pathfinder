import { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
import { HttpClient } from './http-client';

// interface GeminiCompletionResponse {
//     candidates: Array<{
//         content: {
//             parts: Array<{
//                 text: string;
//             }>;
//         }
//     }>;
//     promptFeedback?: any;
// }

export class GeminiClient extends HttpClient implements AIClient {
    private defaultModel = 'gemini-2.5-pro';

    /**
     * Send a request to the Google Gemini API
     * @param prompt The prompt to send
     * @param options Gemini-specific options
     * @returns The AI response
     */
    public async sendRequest(prompt: string, options?: AIClientOptions): Promise<AIResponse> {
        console.log("Using API key: ", options?.apiKey);
        if (!options?.apiKey) {
            throw new Error('Google Gemini API key is required');
        }

        const model = options.model || this.defaultModel;
        const response = await this.post(
            `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent`,
            {
                contents: [{
                    parts: [{
                        text: prompt
                    }]
                }]
            },
            {
                'x-goog-api-key': options.apiKey,
                'Content-Type': 'application/json'
            }
        );

        // check if response is valid
        if (!response.candidates || response.candidates.length === 0) {
            throw new Error('No candidates found in response');
        }

        const content = response.candidates[0].content.parts.map((part: any) => part.text).join('');
        
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
        // throw not implemented error
        throw new Error('sendStreamingRequest not implemented for GeminiClient');
    }

    /**
     * Parse multiple JSON objects from a stream chunk
     * @param chunk Raw stream chunk that may contain multiple JSON objects
     * @returns Array of parsed JSON objects
     */
    private parseStreamChunk(chunk: string): any[] {
        // throw not implemented error
        throw new Error('parseStreamChunk not implemented for GeminiClient');
    }
}
