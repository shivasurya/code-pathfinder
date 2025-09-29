const { HttpClient } = require('./http-client');

class GrokClient extends HttpClient {
  constructor() {
    super();
    this.API_URL = 'https://api.x.ai/v1/chat/completions';
    this.defaultModel = 'grok-4-fast-reasoning';
  }

  /**
   * Send a request to the xAI Grok API (OpenAI-compatible)
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options Options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('xAI Grok API key is required');
    }

    const response = await this.post(
      this.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature ?? 0,
        max_tokens: options.maxTokens ?? 2000,
        stream: false
      },
      {
        Authorization: `Bearer ${options.apiKey}`,
        'Content-Type': 'application/json'
      }
    );

    return {
      content: response?.choices?.[0]?.message?.content ?? '',
      model: response.model,
      provider: 'grok',
      usage: response.usage
    };
  }

  /**
   * Send a streaming request to the xAI Grok API (SSE, OpenAI-compatible)
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback for each chunk
   * @param {import('./ai-client').AIClientOptions} options Options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    if (!options?.apiKey) {
      throw new Error('xAI Grok API key is required');
    }

    let contentSoFar = '';

    await this.streamingPost(
      this.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature ?? 0,
        max_tokens: options.maxTokens ?? 2000,
        stream: true
      },
      (chunk) => {
        try {
          const lines = chunk.split('\n').filter((line) => line.trim() !== '');
          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6);
              if (data === '[DONE]') {
                callback({ content: contentSoFar, isComplete: true });
                return;
              }
              try {
                const parsed = JSON.parse(data);
                if (parsed.choices && parsed.choices[0]?.delta?.content) {
                  const piece = parsed.choices[0].delta.content;
                  contentSoFar += piece;
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

module.exports = {
  GrokClient
};
