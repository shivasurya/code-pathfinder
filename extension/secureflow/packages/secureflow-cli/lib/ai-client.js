/**
 * Interface for AI model providers
 */

/**
 * @typedef {Object} AIClientOptions
 * @property {string} apiKey
 * @property {string} [model]
 * @property {number} [temperature]
 * @property {number} [maxTokens]
 */

/**
 * @typedef {Object} AIResponseChunk
 * @property {string} content
 * @property {boolean} isComplete
 */

/**
 * @typedef {Object} AIMessage
 * @property {string} role - The role of the message sender ('user', 'assistant', 'system')
 * @property {string} content - The content of the message
 */

/**
 * @typedef {Object} AIResponse
 * @property {string} content
 * @property {string} model
 * @property {string} provider
 */

/**
 * Interface that all AI model providers must implement
 */
class AIClient {
  /**
   * Send a request to the AI model
   * @param {string} prompt The prompt to send to the AI model
   * @param {AIClientOptions} [options] Options for the request
   * @param {AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<AIResponse>} The AI model response
   */
  async sendRequest(prompt, options, messages) {
    throw new Error('sendRequest method must be implemented');
  }

  /**
   * Send a streaming request to the AI model
   * @param {string} prompt The prompt to send to the AI model
   * @param {function(AIResponseChunk): void} callback Callback function for each chunk of the response
   * @param {AIClientOptions} [options] Options for the request
   * @param {AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    throw new Error('sendStreamingRequest method must be implemented');
  }
}

module.exports = {
  AIClient
};
