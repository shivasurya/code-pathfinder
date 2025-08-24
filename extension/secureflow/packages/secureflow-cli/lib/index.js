// Export all AI client functionality
const { AIClient } = require('./ai-client');
const { AIClientFactory } = require('./ai-client-factory');
const { ClaudeClient } = require('./claude-client');
const { GeminiClient } = require('./gemini-client');
const { OpenAIClient } = require('./openai-client');
const { HttpClient } = require('./http-client');

module.exports = {
  AIClient,
  AIClientFactory,
  ClaudeClient,
  GeminiClient,
  OpenAIClient,
  HttpClient
};
