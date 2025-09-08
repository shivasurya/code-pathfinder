const { cyan, yellow, red, green, dim, magenta } = require('colorette');

/**
 * CLI Token Display Utility
 * Handles all token usage display formatting for CLI applications
 * Separated from TokenTracker for reusability across different UI contexts
 */
class TokenDisplay {
  /**
   * Display current session state before LLM call
   */
  static displayPreCallUsage(usageData) {
    const { iteration, model, session, available, warnings } = usageData;
    
    const iterationText = iteration ? ` (Iteration ${iteration})` : '';
    console.log(cyan(`\n🔢 Token Usage${iterationText} - Session State:`));
    console.log(dim(`   Model: ${model.name} (${model.provider})`));
    console.log(dim(`   Context Window: ${model.contextWindow.toLocaleString()} tokens`));
    console.log(dim(`   Max Output: ${model.maxOutput.toLocaleString()} tokens`));
    
    console.log(`   📊 Session input so far: ${session.inputTokens.toLocaleString()}`);
    console.log(`   📊 Session output so far: ${session.outputTokens.toLocaleString()}`);
    if (session.reasoningTokens > 0) {
      console.log(`   🧠 Session reasoning so far: ${session.reasoningTokens.toLocaleString()}`);
    }
    
    const inputColor = available.context > 10000 ? green : (available.context > 1000 ? yellow : red);
    const outputColor = available.output > 1000 ? green : (available.output > 0 ? yellow : red);
    
    console.log(`   ⚡ Available context: ${inputColor(available.context.toLocaleString())} tokens`);
    console.log(`   ⚡ Available output: ${outputColor(available.output.toLocaleString())} tokens`);
    
    if (warnings.lowContext) {
      console.log(red(`   ⚠️  WARNING: Low context tokens remaining`));
    }
    
    if (warnings.lowOutput) {
      console.log(yellow(`   ⚠️  WARNING: Low output tokens remaining`));
    }
  }

  /**
   * Display token usage from API response
   */
  static displayUsageResponse(usageData) {
    if (!usageData.success) {
      console.log(yellow(`   ⚠️  ${usageData.error}`));
      return;
    }

    const { iteration, current, totals, remaining, percentages, warnings } = usageData;
    
    const iterationText = iteration ? ` (Iteration ${iteration})` : '';
    console.log(cyan(`\n📊 Token Usage${iterationText} - API Response:`));
    
    console.log(green(`   ✅ Actual API usage:`));
    console.log(`      📤 Input: ${current.inputTokens.toLocaleString()} tokens`);
    console.log(`      📥 Output: ${current.outputTokens.toLocaleString()} tokens`);
    if (current.reasoningTokens > 0) {
      console.log(`      🧠 Reasoning: ${current.reasoningTokens.toLocaleString()} tokens`);
    }
    
    console.log(`   📈 Session totals:`);
    console.log(`      📤 Total input: ${totals.inputTokens.toLocaleString()} tokens`);
    console.log(`      📥 Total output: ${totals.outputTokens.toLocaleString()} tokens`);
    if (totals.reasoningTokens > 0) {
      console.log(`      🧠 Total reasoning: ${totals.reasoningTokens.toLocaleString()} tokens`);
    }
    console.log(`      🎯 Total used: ${totals.totalUsed.toLocaleString()} tokens`);
    
    const contextColor = remaining.context > 10000 ? green : (remaining.context > 1000 ? yellow : red);
    const outputColor = remaining.output > 1000 ? green : (remaining.output > 0 ? yellow : red);
    
    console.log(`   ⚡ Remaining context: ${contextColor(remaining.context.toLocaleString())} tokens`);
    console.log(`   ⚡ Remaining output: ${outputColor(remaining.output.toLocaleString())} tokens`);
    
    console.log(dim(`   📊 Context usage: ${percentages.context.toFixed(1)}%`));
    console.log(dim(`   📊 Output usage: ${percentages.output.toFixed(1)}%`));
  }

  /**
   * Display final usage summary
   */
  static displayFinalSummary(summaryData) {
    const { model, summary, limits, percentages, breakdown } = summaryData;
    
    console.log(magenta('\n📊 FINAL TOKEN USAGE SUMMARY'));
    console.log('='.repeat(50));
    console.log(`Model: ${model.name} (${model.provider})`);
    console.log(`Total iterations: ${summary.totalIterations}`);
    console.log(`Total input tokens: ${summary.totalInputTokens.toLocaleString()}`);
    console.log(`Total output tokens: ${summary.totalOutputTokens.toLocaleString()}`);
    if (summary.totalReasoningTokens > 0) {
      console.log(`Total reasoning tokens: ${summary.totalReasoningTokens.toLocaleString()}`);
    }
    console.log(`Total tokens used: ${summary.totalTokensUsed.toLocaleString()}`);
    
    console.log(`Context window usage: ${percentages.contextUsage.toFixed(1)}% of ${limits.contextWindow.toLocaleString()}`);
    console.log(`Output limit usage: ${percentages.outputUsage.toFixed(1)}% of ${limits.maxOutput.toLocaleString()}`);
    
    if (breakdown.length > 1) {
      console.log('\nPer-iteration breakdown:');
      breakdown.forEach((usage) => {
        const reasoningText = usage.reasoningTokens > 0 ? `, Reasoning: ${usage.reasoningTokens.toLocaleString()}` : '';
        console.log(`  ${usage.iteration}. Input: ${usage.inputTokens.toLocaleString()}, Output: ${usage.outputTokens.toLocaleString()}${reasoningText}`);
      });
    }
    console.log('='.repeat(50));
  }

  /**
   * Display compact usage info (for single-line updates)
   */
  static displayCompactUsage(usageData) {
    if (!usageData.success) {
      console.log(yellow(`⚠️  ${usageData.error}`));
      return;
    }

    const { current, totals } = usageData;
    const totalText = totals.reasoningTokens > 0 
      ? `${totals.inputTokens.toLocaleString()}+${totals.outputTokens.toLocaleString()}+${totals.reasoningTokens.toLocaleString()}`
      : `${totals.inputTokens.toLocaleString()}+${totals.outputTokens.toLocaleString()}`;
    
    console.log(dim(`   📊 Tokens: ${current.inputTokens.toLocaleString()}→${current.outputTokens.toLocaleString()} (Total: ${totalText})`));
  }

  /**
   * Display warning messages for token limits
   */
  static displayLimitWarnings(warningsData) {
    const { context, output, contextPercentage, outputPercentage } = warningsData;
    
    if (context) {
      console.log(red(`⚠️  Context limit warning: ${contextPercentage.toFixed(1)}% used`));
    }
    
    if (output) {
      console.log(yellow(`⚠️  Output limit warning: ${outputPercentage.toFixed(1)}% used`));
    }
  }
}

module.exports = { TokenDisplay };
