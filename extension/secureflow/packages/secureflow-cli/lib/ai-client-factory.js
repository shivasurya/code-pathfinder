const { AIClient } = require('./ai-client');
const { ClaudeClient } = require('./claude-client');
const { GeminiClient } = require('./gemini-client');
const { OpenAIClient } = require('./openai-client');
const { OllamaClient } = require('./ollama-client');
const { GrokClient } = require('./grok-client');
const { ModelConfig } = require('./generated/model-config');

/**
 * Map of client class names to actual client classes
 */
const CLIENT_MAP = {
  'OpenAIClient': OpenAIClient,
  'ClaudeClient': ClaudeClient,
  'GeminiClient': GeminiClient,
  'GrokClient': GrokClient,
  'OllamaClient': OllamaClient
};

/**
 * Factory class for creating AI clients
 */
class AIClientFactory {
  /**
   * Get the appropriate AI client based on the model
   * @param {import('./generated/model-types').AIModel} model The AI model to use
   * @returns {AIClient} The AI client for the specified model
   */
  static getClient(model) {
    // Get model configuration from generated config
    const modelConfig = ModelConfig.get(model);
    
    if (!modelConfig) {
      throw new Error(`Unsupported AI model: ${model}`);
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
