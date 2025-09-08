const fs = require('fs');
const path = require('path');
const { cyan, yellow, red, green, dim, magenta } = require('colorette');

/**
 * Token tracking and management utility
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
   * Display current session state before LLM call
   */
  displayPreCallUsage(iteration = null) {
    const availableInput = this.getAvailableInputTokens();
    const availableOutput = this.getAvailableOutputTokens();
    
    const iterationText = iteration ? ` (Iteration ${iteration})` : '';
    console.log(cyan(`\n🔢 Token Usage${iterationText} - Session State:`));
    console.log(dim(`   Model: ${this.modelName} (${this.currentLimits.provider})`));
    console.log(dim(`   Context Window: ${this.currentLimits.contextWindow.toLocaleString()} tokens`));
    console.log(dim(`   Max Output: ${this.currentLimits.maxOutput.toLocaleString()} tokens`));
    
    console.log(`   📊 Session input so far: ${this.totalInputTokens.toLocaleString()}`);
    console.log(`   📊 Session output so far: ${this.totalOutputTokens.toLocaleString()}`);
    if (this.totalReasoningTokens > 0) {
      console.log(`   🧠 Session reasoning so far: ${this.totalReasoningTokens.toLocaleString()}`);
    }
    
    const inputColor = availableInput > 10000 ? green : (availableInput > 1000 ? yellow : red);
    const outputColor = availableOutput > 1000 ? green : (availableOutput > 0 ? yellow : red);
    
    console.log(`   ⚡ Available context: ${inputColor(availableInput.toLocaleString())} tokens`);
    console.log(`   ⚡ Available output: ${outputColor(availableOutput.toLocaleString())} tokens`);
    
    if (availableInput < 1000) {
      console.log(red(`   ⚠️  WARNING: Low context tokens remaining`));
    }
    
    if (availableOutput < 1000) {
      console.log(yellow(`   ⚠️  WARNING: Low output tokens remaining`));
    }
  }

  /**
   * Record and display token usage from API response
   */
  recordUsage(apiUsage, iteration = null) {
    if (!apiUsage) {
      console.log(yellow(`   ⚠️  No token usage data available from API response`));
      return;
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
    // Fallback - try both naming conventions
    else {
      inputTokens = usage.input_tokens || usage.promptTokenCount || 0;
      outputTokens = usage.output_tokens || usage.candidatesTokenCount || 0;
      reasoningTokens = usage.reasoning_tokens || usage.thoughtsTokenCount || 0;
    }
    
    // Update totals
    this.totalInputTokens += inputTokens;
    this.totalOutputTokens += outputTokens;
    this.totalReasoningTokens += reasoningTokens;
    
    // Record usage
    this.usageHistory.push({
      iteration: iteration || this.usageHistory.length + 1,
      inputTokens,
      outputTokens,
      reasoningTokens,
      timestamp: new Date().toISOString()
    });
    
    const iterationText = iteration ? ` (Iteration ${iteration})` : '';
    console.log(cyan(`\n📊 Token Usage${iterationText} - API Response:`));
    
    console.log(green(`   ✅ Actual API usage:`));
    console.log(`      📤 Input: ${inputTokens.toLocaleString()} tokens`);
    console.log(`      📥 Output: ${outputTokens.toLocaleString()} tokens`);
    if (reasoningTokens > 0) {
      console.log(`      🧠 Reasoning: ${reasoningTokens.toLocaleString()} tokens`);
    }
    
    console.log(`   📈 Session totals:`);
    console.log(`      📤 Total input: ${this.totalInputTokens.toLocaleString()} tokens`);
    console.log(`      📥 Total output: ${this.totalOutputTokens.toLocaleString()} tokens`);
    if (this.totalReasoningTokens > 0) {
      console.log(`      🧠 Total reasoning: ${this.totalReasoningTokens.toLocaleString()} tokens`);
    }
    console.log(`      🎯 Total used: ${(this.totalInputTokens + this.totalOutputTokens).toLocaleString()} tokens`);
    
    const remainingContext = this.currentLimits.contextWindow - this.totalInputTokens - this.totalOutputTokens;
    const remainingOutput = this.currentLimits.maxOutput - this.totalOutputTokens;
    
    const contextColor = remainingContext > 10000 ? green : (remainingContext > 1000 ? yellow : red);
    const outputColor = remainingOutput > 1000 ? green : (remainingOutput > 0 ? yellow : red);
    
    console.log(`   ⚡ Remaining context: ${contextColor(remainingContext.toLocaleString())} tokens`);
    console.log(`   ⚡ Remaining output: ${outputColor(remainingOutput.toLocaleString())} tokens`);
    
    // Usage percentage
    const contextUsagePercent = ((this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow * 100).toFixed(1);
    const outputUsagePercent = (this.totalOutputTokens / this.currentLimits.maxOutput * 100).toFixed(1);
    
    console.log(dim(`   📊 Context usage: ${contextUsagePercent}%`));
    console.log(dim(`   📊 Output usage: ${outputUsagePercent}%`));
  }

  /**
   * Display final usage summary
   */
  displayFinalSummary() {
    console.log(magenta('\n📊 FINAL TOKEN USAGE SUMMARY'));
    console.log('='.repeat(50));
    console.log(`Model: ${this.modelName} (${this.currentLimits.provider})`);
    console.log(`Total iterations: ${this.usageHistory.length}`);
    console.log(`Total input tokens: ${this.totalInputTokens.toLocaleString()}`);
    console.log(`Total output tokens: ${this.totalOutputTokens.toLocaleString()}`);
    if (this.totalReasoningTokens > 0) {
      console.log(`Total reasoning tokens: ${this.totalReasoningTokens.toLocaleString()}`);
    }
    console.log(`Total tokens used: ${(this.totalInputTokens + this.totalOutputTokens).toLocaleString()}`);
    
    const contextUsagePercent = ((this.totalInputTokens + this.totalOutputTokens) / this.currentLimits.contextWindow * 100).toFixed(1);
    const outputUsagePercent = (this.totalOutputTokens / this.currentLimits.maxOutput * 100).toFixed(1);
    
    console.log(`Context window usage: ${contextUsagePercent}% of ${this.currentLimits.contextWindow.toLocaleString()}`);
    console.log(`Output limit usage: ${outputUsagePercent}% of ${this.currentLimits.maxOutput.toLocaleString()}`);
    
    if (this.usageHistory.length > 1) {
      console.log('\nPer-iteration breakdown:');
      this.usageHistory.forEach((usage, index) => {
        const reasoningText = usage.reasoningTokens > 0 ? `, Reasoning: ${usage.reasoningTokens.toLocaleString()}` : '';
        console.log(`  ${index + 1}. Input: ${usage.inputTokens.toLocaleString()}, Output: ${usage.outputTokens.toLocaleString()}${reasoningText}`);
      });
    }
    console.log('='.repeat(50));
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
