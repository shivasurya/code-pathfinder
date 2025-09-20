const { AIClient } = require('./ai-client');
const { HttpClient } = require('./http-client');

class OpenAIClient extends HttpClient {
  constructor() {
    super();
    this.API_URL = 'https://api.openai.com/v1/chat/completions';
    this.defaultModel = 'gpt-3.5-turbo';
  }

  /**
   * Send a request to the OpenAI API
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options OpenAI-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('OpenAI API key is required');
    }

    const response = await this.post(
      this.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0,
        max_tokens: options.maxTokens || 2000,
        stream: false
      },
      {
        Authorization: `Bearer ${options.apiKey}`,
        'Content-Type': 'application/json'
      }
    );

    return {
      content: response.choices[0].message.content,
      model: response.model,
      provider: 'openai',
      usage: response.usage
    };
  }

  /**
   * Send a streaming request to the OpenAI API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options OpenAI-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    if (!options?.apiKey) {
      throw new Error('OpenAI API key is required');
    }

    let contentSoFar = '';

    await this.streamingPost(
      this.API_URL,
      {
        model: options.model || this.defaultModel,
        messages: messages || [{ role: 'user', content: prompt }],
        temperature: options.temperature || 0,
        max_tokens: options.maxTokens || 2000,
        stream: true
      },
      (chunk) => {
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

module.exports = {
  OpenAIClient
};
