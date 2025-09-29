const fs = require('fs');
const path = require('path');

/**
 * Token tracking and management utility
 * Pure calculation logic without display dependencies - reusable across CLI and VS Code
 * Uses only API response data for accurate token counting across all providers
 */
class TokenTracker {
  constructor(modelName) {
    this.modelName = modelName;
    this.modelLimits = this._loadModelLimits();
    this.currentLimits = this._getModelLimits(modelName);
    this.usageHistory = [];
    this.totalInputTokens = 0;
    this.totalOutputTokens = 0;
    this.totalReasoningTokens = 0;
  }

  /**
   * Load model context limits from configuration
   */
  _loadModelLimits() {
    try {
      const configPath = path.join(__dirname, '../config/model-context-limits.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      return config.modelContextLimits;
    } catch (error) {
      console.warn('Could not load model limits configuration:', error.message);
      return {};
    }
  }

  /**
   * Get limits for specific model
   */
  _getModelLimits(modelName) {
    // Find the model in the configuration
    for (const provider in this.modelLimits) {
      if (this.modelLimits[provider][modelName]) {
        return {
          provider,
          ...this.modelLimits[provider][modelName]
        };
      }
    }

    // Default fallback limits
    return {
      provider: 'unknown',
      contextWindow: 128000,
      maxOutput: 4000,
      description: 'Default limits (model not found in configuration)'
    };
  }


  /**
   * Calculate available tokens for input based on used tokens
   */
  getAvailableInputTokens() {
    const totalUsed = this.totalInputTokens + this.totalOutputTokens;
    return Math.max(0, this.currentLimits.contextWindow - totalUsed);
  }

  /**
   * Calculate available tokens for output
   */
  getAvailableOutputTokens() {
    return Math.max(0, this.currentLimits.maxOutput - this.totalOutputTokens);
  }

  /**
   * Get current session state data for display
   */
  getPreCallUsageData(iteration = null) {
    const availableInput = this.getAvailableInputTokens();
    const availableOutput = this.getAvailableOutputTokens();
    
    return {
      iteration,
      model: {
        name: this.modelName,
        provider: this.currentLimits.provider,
        contextWindow: this.currentLimits.contextWindow,
        maxOutput: this.currentLimits.maxOutput
      },
      session: {
        inputTokens: this.totalInputTokens,
        outputTokens: this.totalOutputTokens,
        reasoningTokens: this.totalReasoningTokens
      },
      available: {
        context: availableInput,
        output: availableOutput
      },
      warnings: {
        lowContext: availableInput < 1000,
        lowOutput: availableOutput < 1000
      }
    };
  }

  /**
   * Record token usage from API response and return structured data
   */
  recordUsage(apiUsage, iteration = null) {
    if (!apiUsage) {
      return {
        success: false,
        error: 'No token usage data available from API response'
      };
    }

    // Extract token counts from API response - handle different provider formats
    let inputTokens = 0;
    let outputTokens = 0;
    let reasoningTokens = 0;

    // Handle nested usageMetadata (Gemini full response format)
    const usage = apiUsage.usageMetadata || apiUsage;

    // OpenAI/Anthropic format
    if (usage.input_tokens !== undefined) {
      inputTokens = usage.input_tokens || 0;
      outputTokens = usage.output_tokens || 0;
      reasoningTokens = usage.reasoning_tokens || 0; // For O1/O3 models
    }
    // Google Gemini format
    else if (usage.promptTokenCount !== undefined) {
      inputTokens = usage.promptTokenCount || 1000;
      outputTokens = usage.candidatesTokenCount || 0;
      reasoningTokens = usage.thoughtsTokenCount || 0; // For thinking models
    }
    // ollama format
    else if (usage.total_tokens !== undefined) {
      inputTokens = usage.prompt_tokens || 0;
      outputTokens = usage.completion_tokens || 0;
    }
    // xAI Grok format (OpenAI-compatible but may have specific fields)
    else if (usage.prompt_tokens !== undefined && usage.completion_tokens !== undefined) {
      inputTokens = usage.prompt_tokens || 0;
      outputTokens = usage.completion_tokens || 0;
      reasoningTokens = usage.reasoning_tokens || 0; // Grok reasoning tokens
    }
    // Fallback - try both naming conventions
    else {
      inputTokens = usage.input_tokens || usage.promptTokenCount || usage.prompt_tokens || 0;
      outputTokens = usage.output_tokens || usage.candidatesTokenCount || usage.completion_tokens || 0;
      reasoningTokens = usage.reasoning_tokens || usage.thoughtsTokenCount || 0;
    }
    
    // Update totals
    this.totalInputTokens += inputTokens;
    this.totalOutputTokens += outputTokens;
    this.totalReasoningTokens += reasoningTokens;
    
    // Record usage
    const usageRecord = {
      iteration: iteration || this.usageHistory.length + 1,
      inputTokens,
      outputTokens,
      reasoningTokens,
      timestamp: new Date().toISOString()
    };
    
    this.usageHistory.push(usageRecord);
    
    // Calculate remaining tokens
    const remainingContext = this.currentLimits.contextWindow - this.totalInputTokens - this.totalOutputTokens;
    const remainingOutput = this.currentLimits.maxOutput - this.totalOutputTokens;
    
    // Calculate usage percentages
    const contextUsagePercent = ((this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow * 100);
    const outputUsagePercent = (this.totalOutputTokens / this.currentLimits.maxOutput * 100);
    
    return {
      success: true,
      iteration,
      current: {
        inputTokens,
        outputTokens,
        reasoningTokens
      },
      totals: {
        inputTokens: this.totalInputTokens,
        outputTokens: this.totalOutputTokens,
        reasoningTokens: this.totalReasoningTokens,
        totalUsed: this.totalInputTokens + this.totalOutputTokens
      },
      remaining: {
        context: remainingContext,
        output: remainingOutput
      },
      percentages: {
        context: contextUsagePercent,
        output: outputUsagePercent
      },
      warnings: {
        lowContext: remainingContext < 1000,
        lowOutput: remainingOutput < 1000
      }
    };
  }

  /**
   * Get final usage summary data for display
   */
  getFinalSummaryData() {
    const contextUsagePercent = ((this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow * 100);
    const outputUsagePercent = (this.totalOutputTokens / this.currentLimits.maxOutput * 100);
    
    return {
      model: {
        name: this.modelName,
        provider: this.currentLimits.provider
      },
      summary: {
        totalIterations: this.usageHistory.length,
        totalInputTokens: this.totalInputTokens,
        totalOutputTokens: this.totalOutputTokens,
        totalReasoningTokens: this.totalReasoningTokens,
        totalTokensUsed: this.totalInputTokens + this.totalOutputTokens
      },
      limits: {
        contextWindow: this.currentLimits.contextWindow,
        maxOutput: this.currentLimits.maxOutput
      },
      percentages: {
        contextUsage: contextUsagePercent,
        outputUsage: outputUsagePercent
      },
      breakdown: this.usageHistory.map((usage, index) => ({
        iteration: index + 1,
        inputTokens: usage.inputTokens,
        outputTokens: usage.outputTokens,
        reasoningTokens: usage.reasoningTokens,
        timestamp: usage.timestamp
      }))
    };
  }

  /**
   * Get usage statistics for programmatic access
   */
  getUsageStats() {
    return {
      model: this.modelName,
      provider: this.currentLimits.provider,
      limits: {
        contextWindow: this.currentLimits.contextWindow,
        maxOutput: this.currentLimits.maxOutput
      },
      usage: {
        totalInputTokens: this.totalInputTokens,
        totalOutputTokens: this.totalOutputTokens,
        totalReasoningTokens: this.totalReasoningTokens,
        totalTokens: this.totalInputTokens + this.totalOutputTokens,
        iterations: this.usageHistory.length
      },
      percentages: {
        contextUsage: ((this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow * 100),
        outputUsage: (this.totalOutputTokens / this.currentLimits.maxOutput * 100)
      },
      history: this.usageHistory
    };
  }

  /**
   * Reset usage tracking
   */
  reset() {
    this.usageHistory = [];
    this.totalInputTokens = 0;
    this.totalOutputTokens = 0;
    this.totalReasoningTokens = 0;
  }

  /**
   * Check if we're approaching limits
   */
  isApproachingLimits() {
    const contextUsage = (this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow;
    const outputUsage = this.totalOutputTokens / this.currentLimits.maxOutput;
    
    return {
      context: contextUsage > 0.8,
      output: outputUsage > 0.8,
      contextPercentage: contextUsage * 100,
      outputPercentage: outputUsage * 100
    };
  }
}

module.exports = { TokenTracker };
