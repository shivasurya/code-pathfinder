const { cyan, yellow, red, green, dim, magenta } = require('colorette');
const { FileRequestHandler } = require('./file-request-handler');
const { loadPrompt } = require('../lib/prompts/prompt-loader');

/**
 * AI-powered security analyzer with iterative file request capability
 */
class AISecurityAnalyzer {
  constructor(aiClient, projectPath, options = {}) {
    this.aiClient = aiClient;
    this.projectPath = projectPath;
    this.fileHandler = new FileRequestHandler(projectPath, options);
    this.maxIterations = options.maxIterations || 3;
    this.analysisLog = [];
    this.tokenTracker = options.tokenTracker || null;
  }

  /**
   * Perform iterative security analysis with file requests
   */
  async analyzeProject(profileInfo, projectSummary, reviewPrompt) {
    console.log(magenta('🔍 Starting AI-driven security analysis...'));
    
    let iteration = 0;
    let currentContext = await this._buildInitialContext(profileInfo, projectSummary, reviewPrompt);
    let allFileContents = [];
    let finalAnalysis = null;

    while (iteration < this.maxIterations) {
      iteration++;
      console.log(cyan(`\n📋 Analysis iteration ${iteration}/${this.maxIterations}`));

      // Send context to AI and get response
      const aiResponse = await this._sendToAI(currentContext, iteration);
      
      this.analysisLog.push({
        iteration,
        timestamp: new Date().toISOString(),
        contextLength: currentContext.length,
        response: aiResponse
      });

      // Check if AI wants to request files
      const fileRequests = this._extractFileRequests(aiResponse);
      
      if (fileRequests.length > 0) {
        console.log(yellow(`🤖 LLM requesting ${fileRequests.length} files for analysis`));
        
        // Process file requests
        const fileResults = await this.fileHandler.processFileRequests(aiResponse);
        
        // Add successful file contents to context
        const newFileContents = fileResults
          .filter(result => result.status === 'success')
          .map(result => ({
            path: result.path,
            content: result.content,
            reason: result.reason
          }));

        allFileContents.push(...newFileContents);

        // Build context for next iteration
        currentContext = await this._buildIterativeContext(
          profileInfo,
          projectSummary,
          reviewPrompt,
          allFileContents,
          fileResults,
          iteration
        );

      } else {
        // No more file requests, this should be the final analysis
        console.log(green('✅ AI completed analysis without additional file requests'));
        finalAnalysis = aiResponse;
        break;
      }
    }

    // If we reached max iterations, get final analysis
    if (!finalAnalysis) {
      console.log(yellow('⚠️  Reached maximum iterations, requesting final analysis'));
      finalAnalysis = await this._getFinalAnalysis(currentContext);
    }

    return {
      analysis: finalAnalysis,
      fileContents: allFileContents,
      iterations: iteration,
      requestLog: this.fileHandler.getRequestLog(),
      analysisLog: this.analysisLog
    };
  }

  /**
   * Build initial context for first AI request
   */
  async _buildInitialContext(profileInfo, projectSummary, reviewPrompt) {
    let context = reviewPrompt + '\n\n';

    // Add profile information
    if (profileInfo) {
      context += `[SECUREFLOW PROFILE CONTEXT]\n`;
      context += `Name: ${profileInfo.name}\n`;
      context += `Category: ${profileInfo.category}\n`;
      context += `Languages: ${profileInfo.languages.join(', ')}\n`;
      context += `Frameworks: ${profileInfo.frameworks.join(', ')}\n`;
      context += `Build Tools: ${profileInfo.buildTools.join(', ')}\n`;
      context += `Evidence: ${profileInfo.evidence.join('; ')}\n\n`;
    }

    // Add project summary
    context += `[PROJECT STRUCTURE]\n`;
    context += `Project Path: ${projectSummary.projectPath}\n`;
    context += `Total Files: ${projectSummary.totalFiles}\n\n`;

    context += `Directory Structure:\n${projectSummary.directoryStructure}\n`;

    context += `Files by Extension:\n`;
    Object.entries(projectSummary.filesByExtension).forEach(([ext, files]) => {
      context += `${ext}: ${files.length} files\n`;
    });

    context += `\nImportant Files Identified:\n`;
    projectSummary.importantFiles.forEach(file => {
      context += `- ${file.relativePath} (${file.lineCount} lines)\n`;
    });

    // Add file request instructions
    context += '\n' + await this.fileHandler.getFileRequestInstructions();

    const initialAnalysisPrompt = await loadPrompt('scanner/initial-analysis.txt');
    context += `\n${initialAnalysisPrompt}`;
    return context;
  }

  /**
   * Build context for iterative requests
   */
  async _buildIterativeContext(profileInfo, projectSummary, reviewPrompt, fileContents, fileResults, iteration) {
    let context = reviewPrompt + '\n\n';

    // Add profile information
    if (profileInfo) {
      context += `[SECUREFLOW PROFILE CONTEXT]\n`;
      context += `Name: ${profileInfo.name}\n`;
      context += `Category: ${profileInfo.category}\n`;
      context += `Languages: ${profileInfo.languages.join(', ')}\n`;
      context += `Frameworks: ${profileInfo.frameworks.join(', ')}\n\n`;
    }

    // Add file contents from previous requests
    context += `[ANALYZED FILES - Iteration ${iteration}]\n`;
    fileContents.forEach(file => {
      context += `\n=== FILE: ${file.path} ===\n`;
      context += `Reason: ${file.reason}\n`;
      context += `${file.content}\n`;
      context += `=== END FILE: ${file.path} ===\n`;
    });

    // Add file request results summary
    context += `\n[FILE REQUEST RESULTS]\n`;
    fileResults.forEach(result => {
      if (result.status === 'success') {
        context += `✅ ${result.path}: Successfully read (${result.lines} lines)\n`;
      } else {
        context += `❌ ${result.path}: ${result.reason}\n`;
      }
    });

    // Add instructions for next step
    if (iteration < this.maxIterations) {
      context += '\n' + await this.fileHandler.getFileRequestInstructions();
      const iterativePrompt = await loadPrompt('scanner/iterative-analysis.txt');
      context += `\n${iterativePrompt}`;
    } else {
      const finalIterationPrompt = await loadPrompt('scanner/final-iteration.txt');
      context += `\n${finalIterationPrompt}`;
    }

    return context;
  }

  /**
   * Extract file requests from AI response
   */
  _extractFileRequests(response) {
    const fileRequestRegex = /<file_request\s+path="([^"]+)"(?:\s+reason="([^"]*)")?\s*\/>/g;
    const requests = [];
    
    let match;
    while ((match = fileRequestRegex.exec(response)) !== null) {
      requests.push({
        path: match[1],
        reason: match[2] || 'No reason provided'
      });
    }

    return requests;
  }

  /**
   * Send context to AI and get response
   */
  async _sendToAI(context, iteration) {
    console.log(dim(`📤 Sending context to AI (${context.length} characters)`));
    
    // Display session state before the call
    if (this.tokenTracker) {
      this.tokenTracker.displayPreCallUsage(iteration);
    }
    
    try {
      const response = await this.aiClient.analyze(context);
      
      // Extract content from response object
      const content = response.content || response;
      console.log(dim(`📥 Received AI response (${content.length} characters)`));
      
      // Record token usage from API response if available
      if (this.tokenTracker && response.usage) {
        this.tokenTracker.recordUsage(response.usage, iteration);
      }
      
      return content;
    } catch (error) {
      console.error(red(`❌ AI request failed in iteration ${iteration}: ${error.message}`));
      throw error;
    }
  }

  /**
   * Get final analysis when max iterations reached
   */
  async _getFinalAnalysis(context) {
    console.log(context);
    const finalAnalysisPrompt = await loadPrompt('scanner/final-analysis.txt');
    const finalContext = context + `\n\n${finalAnalysisPrompt}`;
    return await this._sendToAI(finalContext, 'final');
  }

  /**
   * Get analysis log for debugging
   */
  getAnalysisLog() {
    return this.analysisLog;
  }
}

module.exports = { AISecurityAnalyzer };
