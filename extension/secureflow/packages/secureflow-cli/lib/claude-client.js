const { AIClient } = require('./ai-client');
const { HttpClient } = require('./http-client');

class ClaudeClient extends HttpClient {
  constructor() {
    super();
    this.API_URL = 'https://api.anthropic.com/v1/messages';
    this.defaultModel = 'claude-sonnet-4-5-20250929';
  }

  /**
   * Send a request to the Anthropic Claude API
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options Claude-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('Anthropic Claude API key is required');
    }

    // Show deprecation warning for claude-3-5-sonnet-20241022
    if (options.model === 'claude-3-5-sonnet-20241022') {
      console.warn('⚠️  WARNING: claude-3-5-sonnet-20241022 is deprecated. Please upgrade to claude-sonnet-4-5-20250929 for better performance and latest features.');
    }

    const response = await this.post(
      this.API_URL,
      {
        model: options.model || "claude-sonnet-4-5-20250929",
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0,
        max_tokens: options.maxTokens || 4000,
        stream: false
      },
      {
        'x-api-key': options.apiKey,
        'anthropic-version': '2023-06-01', // Use appropriate API version
        'Content-Type': 'application/json'
      }
    );

    // Extract text content from Claude's response
    const content = response.content
      .filter((item) => item.type === 'text')
      .map((item) => item.text)
      .join('');

    return {
      content,
      model: response.model,
      provider: 'claude',
      usage: response.usage
    };
  }

  /**
   * Send a streaming request to the Anthropic Claude API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options Claude-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    if (!options?.apiKey) {
      throw new Error('Anthropic Claude API key is required');
    }

    let contentSoFar = '';

    await this.streamingPost(
      this.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0.7,
        max_tokens: options.maxTokens || 500,
        stream: true
      },
      (chunk) => {
        try {
          // Claude also uses SSE format with data: prefix
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
                if (
                  parsed.type === 'content_block_delta' &&
                  parsed.delta &&
                  parsed.delta.text
                ) {
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
        Accept: 'text/event-stream'
      }
    );
  }
}

module.exports = {
  ClaudeClient
};
