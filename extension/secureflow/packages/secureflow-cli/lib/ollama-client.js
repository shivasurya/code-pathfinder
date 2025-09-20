const { AIClient } = require('./ai-client');
const { HttpClient } = require('./http-client');

class OllamaClient extends HttpClient {
  constructor() {
    super();
    this.API_URL = 'http://localhost:11434/api/chat';
    this.defaultModel = 'qwen3:4b';
  }

  /**
   * Send a request to the Ollama API
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options Ollama-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    const model = options?.model || this.defaultModel;
    
    const response = await this.post(
      this.API_URL,
      {
        model: model,
        messages: messages || [{ role: 'user', content: prompt }],
        stream: false,
        options: {
          temperature: options?.temperature || 0,
          num_predict: options?.maxTokens || 2000
        }
      },
      {
        'Content-Type': 'application/json'
      }
    );

    console.log(response.message.content);

    // remove text between <think> and </think> tags
    response.message.content = response.message.content.replace(/<think>.*?<\/think>/gs, '');

    return {
      content: response.message.content,
      model: response.model,
      provider: 'ollama',
      usage: {
        prompt_tokens: response.prompt_eval_count || 0,
        completion_tokens: response.eval_count || 0,
        total_tokens: (response.prompt_eval_count || 0) + (response.eval_count || 0)
      }
    };
  }

  /**
   * Send a streaming request to the Ollama API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options Ollama-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    const model = options?.model || this.defaultModel;
    let contentSoFar = '';

    await this.streamingPost(
      this.API_URL,
      {
        model: model,
        messages: messages || [{ role: 'user', content: prompt }],
        stream: true,
        options: {
          temperature: options?.temperature || 0,
          num_predict: options?.maxTokens || 2000
        }
      },
      (chunk) => {
        try {
          // Ollama sends newline-delimited JSON
          const lines = chunk.split('\n').filter((line) => line.trim() !== '');

          for (const line of lines) {
            try {
              const parsed = JSON.parse(line);
              
              if (parsed.message && parsed.message.content) {
                const content = parsed.message.content;
                contentSoFar += content;
                callback({ content: contentSoFar, isComplete: false });
              }
              
              if (parsed.done) {
                callback({ content: contentSoFar, isComplete: true });
                return;
              }
            } catch (e) {
              console.error('Error parsing Ollama response line:', e);
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
}

module.exports = {
  OllamaClient
};
