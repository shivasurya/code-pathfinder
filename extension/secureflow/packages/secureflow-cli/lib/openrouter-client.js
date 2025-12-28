const { AIClient } = require('./ai-client');

// OpenRouter SDK is an ES module, we need to use dynamic import
let OpenRouter;

// Async function to load the OpenRouter SDK
async function loadOpenRouterSDK() {
  if (!OpenRouter) {
    const module = await import('@openrouter/sdk');
    OpenRouter = module.OpenRouter;
  }
  return OpenRouter;
}

/**
 * OpenRouter client implementation
 * Supports 300+ models from multiple providers through a unified API
 */
class OpenRouterClient extends AIClient {
  constructor() {
    super();
    this.defaultModel = 'anthropic/claude-sonnet-4.5-20250929';
    this.client = null;
  }

  /**
   * Initialize the OpenRouter SDK client
   * @param {string} apiKey The OpenRouter API key
   * @returns {Promise<OpenRouter>} The initialized client
   */
  async getClient(apiKey) {
    const OpenRouterClass = await loadOpenRouterSDK();

    if (!this.client || this.client.apiKey !== apiKey) {
      this.client = new OpenRouterClass({
        apiKey: apiKey
      });
    }
    return this.client;
  }

  /**
   * Send a request to the OpenRouter API
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options OpenRouter-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('OpenRouter API key is required');
    }

    const client = await this.getClient(options.apiKey);
    const model = options.model || this.defaultModel;

    try {
      const response = await client.chat.send({
        model: model,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature !== undefined ? options.temperature : 0,
        max_tokens: options.maxTokens || 2000,
        stream: false
      });

      // Extract content from response
      const content = response.choices?.[0]?.message?.content || '';

      return {
        content: content,
        model: response.model || model,
        provider: 'openrouter',
        usage: response.usage
      };
    } catch (error) {
      // Enhance error messages with OpenRouter-specific details
      if (error.message) {
        throw new Error(`OpenRouter API error: ${error.message}`);
      }
      throw error;
    }
  }

  /**
   * Send a streaming request to the OpenRouter API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options OpenRouter-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    if (!options?.apiKey) {
      throw new Error('OpenRouter API key is required');
    }

    const client = await this.getClient(options.apiKey);
    const model = options.model || this.defaultModel;

    try {
      const stream = await client.chat.send({
        model: model,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature !== undefined ? options.temperature : 0,
        max_tokens: options.maxTokens || 2000,
        stream: true
      });

      let contentSoFar = '';

      // Process streaming response
      for await (const chunk of stream) {
        const delta = chunk.choices?.[0]?.delta?.content;

        if (delta) {
          contentSoFar += delta;
          callback({
            content: contentSoFar,
            isComplete: false
          });
        }
      }

      // Send final completion signal
      callback({
        content: contentSoFar,
        isComplete: true
      });
    } catch (error) {
      // Enhance error messages with OpenRouter-specific details
      if (error.message) {
        throw new Error(`OpenRouter API error: ${error.message}`);
      }
      throw error;
    }
  }
}

module.exports = {
  OpenRouterClient
};
