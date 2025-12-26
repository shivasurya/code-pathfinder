const { AIClient } = require('./ai-client');
const { ClaudeClient } = require('./claude-client');
const { GeminiClient } = require('./gemini-client');
const { OpenAIClient } = require('./openai-client');
const { OllamaClient } = require('./ollama-client');
const { GrokClient } = require('./grok-client');
const { OpenRouterClient } = require('./openrouter-client');
const { ModelConfig } = require('./generated/model-config');

/**
 * Map of client class names to actual client classes
 */
const CLIENT_MAP = {
  'OpenAIClient': OpenAIClient,
  'ClaudeClient': ClaudeClient,
  'GeminiClient': GeminiClient,
  'GrokClient': GrokClient,
  'OllamaClient': OllamaClient,
  'OpenRouterClient': OpenRouterClient
};

/**
 * Factory class for creating AI clients
 */
class AIClientFactory {
  /**
   * Get the appropriate AI client based on the model
   * @param {string} model The AI model to use (can be a standard model or OpenRouter model ID like "provider/model")
   * @returns {AIClient} The AI client for the specified model
   */
  static getClient(model) {
    // Check if this is an OpenRouter model (contains "/" in the model name)
    if (/^[a-z0-9-]+\/[a-z0-9-]+/i.test(model)) {
      // This is an OpenRouter model ID (e.g., "anthropic/claude-3-5-sonnet")
      return new OpenRouterClient();
    }

    // Get model configuration from generated config for standard models
    const modelConfig = ModelConfig.get(model);

    if (!modelConfig) {
      throw new Error(`Unsupported AI model: ${model}. For OpenRouter models, use the format "provider/model-name" (e.g., "anthropic/claude-3-5-sonnet")`);
    }

    // Get the client class from the map
    const ClientClass = CLIENT_MAP[modelConfig.client];

    if (!ClientClass) {
      throw new Error(`Client class ${modelConfig.client} not found for model: ${model}`);
    }

    return new ClientClass();
  }
}

module.exports = {
  AIClientFactory
};
