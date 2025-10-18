// Export all AI client functionality
export { AIClient, AIClientOptions, AIResponse, AIResponseChunk } from './ai-client';
export { AIClientFactory } from './ai-client-factory';
export { ClaudeClient } from './claude-client';
export { GeminiClient } from './gemini-client';
export { OpenAIClient } from './openai-client';
export { OllamaClient } from './ollama-client';
export { HttpClient } from './http-client';
export { AIModel } from './types';

// Export prompts functionality
export { getPromptPath, getAppProfilerPrompt } from './prompts';
export { loadPrompt, getPromptForAppType, getApplicationProfilerPrompt, getThreatModelingPrompt } from './prompts/prompt-loader';

export { WorkspaceAnalyzer, ApplicationProfile, WorkspaceAnalyzerOptions } from './workspace-analyzer';

// Export analytics service
export { AnalyticsService, StorageAdapter, FileStorageAdapter, VSCodeStorageAdapter } from './services/analytics';
