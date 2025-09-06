// Export all AI client functionality
const { AIClient } = require('./ai-client');
const { AIClientFactory } = require('./ai-client-factory');
const { ClaudeClient } = require('./claude-client');
const { GeminiClient } = require('./gemini-client');
const { OpenAIClient } = require('./openai-client');
const { HttpClient } = require('./http-client');
const { AIModel } = require('./types');

// Export prompts functionality
const { getPromptPath, getAppProfilerPrompt } = require('./prompts');
const { loadPrompt, getPromptForAppType, getApplicationProfilerPrompt, getThreatModelingPrompt } = require('./prompts/prompt-loader');

// Export workspace analyzer functionality
const { WorkspaceAnalyzer, ApplicationProfile } = require('./workspace-analyzer');

module.exports = {
  AIClient,
  AIClientFactory,
  ClaudeClient,
  GeminiClient,
  OpenAIClient,
  HttpClient,
  AIModel,
  getPromptPath,
  getAppProfilerPrompt,
  loadPrompt,
  getPromptForAppType,
  getApplicationProfilerPrompt,
  getThreatModelingPrompt,
  WorkspaceAnalyzer,
  ApplicationProfile
};
