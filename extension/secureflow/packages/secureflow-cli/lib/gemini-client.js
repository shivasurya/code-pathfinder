const { AIClient } = require('./ai-client');
const { HttpClient } = require('./http-client');

class GeminiClient extends HttpClient {
  constructor() {
    super();
    this.defaultModel = 'gemini-2.5-pro';
  }

  /**
   * Send a request to the Google Gemini API
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options Gemini-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('Google Gemini API key is required');
    }

    const model = options.model || this.defaultModel;
    const response = await this.post(
      `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent`,
      {
        contents: this._convertMessagesToContents(messages, prompt)
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

    const content = response.candidates[0].content.parts
      .map((part) => part.text)
      .join('');

    return {
      content,
      model: model,
      provider: 'gemini',
      usage: response.usageMetadata
    };
  }

  /**
   * Convert messages array to Gemini contents format
   * @param {import('./ai-client').AIMessage[]} [messages] Messages array
   * @param {string} prompt Fallback prompt if no messages
   * @returns {any[]} Gemini contents array
   */
  _convertMessagesToContents(messages, prompt) {
    if (!messages) {
      return [
        {
          parts: [
            {
              text: prompt
            }
          ]
        }
      ];
    }

    return messages.map(message => ({
      role: message.role === 'assistant' ? 'model' : 'user',
      parts: [
        {
          text: message.content
        }
      ]
    }));
  }

  /**
   * Send a streaming request to the Google Gemini API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options Gemini-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    // throw not implemented error
    throw new Error('sendStreamingRequest not implemented for GeminiClient');
  }

  /**
   * Parse multiple JSON objects from a stream chunk
   * @param {string} chunk Raw stream chunk that may contain multiple JSON objects
   * @returns {any[]} Array of parsed JSON objects
   */
  parseStreamChunk(chunk) {
    // throw not implemented error
    throw new Error('parseStreamChunk not implemented for GeminiClient');
  }
}

module.exports = {
  GeminiClient
};
