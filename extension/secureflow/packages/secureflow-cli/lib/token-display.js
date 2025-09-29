const { cyan, yellow, red, green, dim, magenta } = require('colorette');

/**
 * CLI Token Display Utility
 * Handles all token usage display formatting for CLI applications
 * Separated from TokenTracker for reusability across different UI contexts
 */
class TokenDisplay {
  /**
   * Format numbers in k/M format for compact display
   */
  static formatNumber(num) {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'k';
    } else {
      return num.toString();
    }
  }
  /**
   * Display current session state before LLM call (compact single line)
   */
  static displayPreCallUsage(usageData) {
    // Validate input
    if (!usageData) {
      console.log(yellow('⚠️  Invalid usage data for token display'));
      return;
    }

    const { iteration, model, session, available, warnings } = usageData;
    
    // Validate nested objects with defaults
    const safeModel = model || {};
    const safeSession = session || {};
    const safeAvailable = available || {};
    const safeWarnings = warnings || {};
    
    // Calculate usage percentage
    const contextWindow = safeModel.contextWindow || 0;
    const totalUsed = (safeSession.inputTokens || 0) + (safeSession.outputTokens || 0);
    const usagePercentage = contextWindow > 0 ? ((totalUsed / contextWindow) * 100).toFixed(1) : '0.0';
    
    // Build compact display
    const iterationText = iteration ? ` (${iteration})` : '';
    const modelName = safeModel.name || 'Unknown';
    const inputTokens = this.formatNumber(safeSession.inputTokens || 0);
    const outputTokens = this.formatNumber(safeSession.outputTokens || 0);
    const availableContext = this.formatNumber(safeAvailable.context || 0);
    const availableOutput = this.formatNumber(safeAvailable.output || 0);
    
    // Color coding for available tokens
    const contextColor = (safeAvailable.context || 0) > 10000 ? green : ((safeAvailable.context || 0) > 1000 ? yellow : red);
    const outputColor = (safeAvailable.output || 0) > 1000 ? green : ((safeAvailable.output || 0) > 0 ? yellow : red);
    
    // Warning indicators
    const warningText = (safeWarnings.lowContext || safeWarnings.lowOutput) ? red(' ⚠️') : '';
    
    // add new line before display
    console.log();
    console.log(
      cyan(`${modelName}`) + ' | ' +
      magenta(`I:${inputTokens}`) + ' ' + cyan(`O:${outputTokens}`) + ' | ' +
      contextColor(`C:${availableContext}`) + ' ' +
      outputColor(`O:${availableOutput}`) + ' | ' +
      green(`${usagePercentage}%`) +
      warningText
    );
    console.log();
  }
  /**
   * Display token usage from API response (compact single line)
   */
  static displayUsageResponse(usageData) {
    if (!usageData.success) {
      console.log(yellow(`   ⚠️  ${usageData.error}`));
      return;
    }

    const { iteration, current, totals, remaining, percentages, warnings } = usageData;
    
    // Validate nested objects with defaults
    const safeCurrent = current || {};
    const safeTotals = totals || {};
    const safeRemaining = remaining || {};
    const safePercentages = percentages || {};
    
    // Build compact display values
    const iterationText = iteration ? ` (${iteration})` : '';
    const currentInput = this.formatNumber(safeCurrent.inputTokens || 0);
    const currentOutput = this.formatNumber(safeCurrent.outputTokens || 0);
    const totalInput = this.formatNumber(safeTotals.inputTokens || 0);
    const totalOutput = this.formatNumber(safeTotals.outputTokens || 0);
    const remainingContext = this.formatNumber(safeRemaining.context || 0);
    const remainingOutput = this.formatNumber(safeRemaining.output || 0);
    const contextPercentage = (safePercentages.context || 0).toFixed(1);
    
    // Color coding for remaining tokens
    const contextColor = (safeRemaining.context || 0) > 10000 ? green : ((safeRemaining.context || 0) > 1000 ? yellow : red);
    const outputColor = (safeRemaining.output || 0) > 1000 ? green : ((safeRemaining.output || 0) > 0 ? yellow : red);
    
    // Reasoning tokens (if present)
    const reasoningText = (safeCurrent.reasoningTokens || 0) > 0 ? ` R:${this.formatNumber(safeCurrent.reasoningTokens)}` : '';
    
    console.log();
    console.log(
      green(`Input :${currentInput} Output :${currentOutput} ${reasoningText}`) + ' | ' +
      cyan(`Total Input :${totalInput} Output :${totalOutput}`) + ' | ' +
      contextColor(`Context :${remainingContext}`) + ' ' +
      outputColor(`Output :${remainingOutput}`) + ' | ' +
      magenta(`Context Usage :${contextPercentage}%`)
    );
    console.log();
  }

  /**
   * Display final usage summary (compact single line)
   */
  static displayFinalSummary(summaryData) {
    // Validate input
    if (!summaryData) {
      console.log(yellow('⚠️  Invalid summary data for token display'));
      return;
    }

    const { model, summary, limits, percentages, breakdown } = summaryData;
    
    // Validate nested objects with defaults
    const safeModel = model || {};
    const safeSummary = summary || {};
    const safePercentages = percentages || {};
    
    // Format numbers compactly
    const totalInput = this.formatNumber(safeSummary.totalInputTokens || 0);
    const totalOutput = this.formatNumber(safeSummary.totalOutputTokens || 0);
    const totalUsed = this.formatNumber(safeSummary.totalTokensUsed || 0);
    const contextUsage = (safePercentages.contextUsage || 0).toFixed(1);
    const outputUsage = (safePercentages.outputUsage || 0).toFixed(1);
    
    // Reasoning tokens (if present)
    const reasoningText = (safeSummary.totalReasoningTokens || 0) > 0 
      ? ` R:${this.formatNumber(safeSummary.totalReasoningTokens)}` : '';
    
    console.log();
    console.log(
      magenta(`📊 ${safeModel.name || 'Unknown'}`) + ' | ' +
      cyan(`${safeSummary.totalIterations || 0} iterations`) + ' | ' +
      green(`I:${totalInput} O:${totalOutput}${reasoningText}`) + ' | ' +
      yellow(`Total:${totalUsed}`) + ' | ' +
      red(`${contextUsage}% ${outputUsage}%`)
    );
    console.log();

    // TODO: Per-iteration breakdown for cost analysis
    // Commented out for now to keep output compact
    /*
    if (breakdown && breakdown.length > 1) {
      console.log('\nPer-iteration breakdown:');
      breakdown.forEach((usage) => {
        const reasoningText = usage.reasoningTokens > 0 ? `, Reasoning: ${this.formatNumber(usage.reasoningTokens)}` : '';
        console.log(`  ${usage.iteration}. Input: ${this.formatNumber(usage.inputTokens)}, Output: ${this.formatNumber(usage.outputTokens)}${reasoningText}`);
      });
    }
    */
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
