const { AIClient } = require('./ai-client');
const { ClaudeClient } = require('./claude-client');
const { GeminiClient } = require('./gemini-client');
const { OpenAIClient } = require('./openai-client');
const { OllamaClient } = require('./ollama-client');
const { GrokClient } = require('./grok-client');

/**
 * Factory class for creating AI clients
 */
class AIClientFactory {
  /**
   * Get the appropriate AI client based on the model
   * @param {import('./types').AIModel} model The AI model to use
   * @returns {AIClient} The AI client for the specified model
   */
  static getClient(model) {
    switch (model) {
      // OpenAI models
      case 'gpt-5-pro':
      case 'gpt-5':
      case 'gpt-5-mini':
      case 'gpt-5-nano':
      case 'o3':
      case 'o3-pro':
      case 'o3-mini':
      case 'o4-mini':
      case 'gpt-4.1':
      case 'gpt-4.1-mini':
      case 'gpt-4o':
      case 'gpt-4o-mini':
      case 'o1':
        return new OpenAIClient();

      // Google models
      case 'gemini-2.5-pro':
      case 'gemini-2.5-flash':
        return new GeminiClient();

      // Anthropic models
      case 'claude-sonnet-4-5-20250929':
      case 'claude-opus-4-1-20250805':
      case 'claude-opus-4-20250514':
      case 'claude-sonnet-4-20250514':
      case 'claude-3-7-sonnet-20250219':
      case 'claude-haiku-4-5':
      case 'claude-3-5-haiku-20241022':
        return new ClaudeClient();

      // Grok (xAI) models
      case 'grok-4-fast-reasoning':
        return new GrokClient();

      // Ollama models
      case 'qwen3:4b':
        return new OllamaClient();

      default:
        throw new Error(`Unsupported AI model: ${model}`);
    }
  }
}

module.exports = {
  AIClientFactory
};
