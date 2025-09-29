const { cyan, yellow, red, green, dim, magenta } = require('colorette');
const { FileRequestHandler } = require('./file-request-handler');
const { loadPrompt } = require('../lib/prompts/prompt-loader');
const { TokenDisplay } = require('../lib/token-display');

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
  // console.log(magenta('🔍 Starting AI-driven security analysis...'));
    
    let iteration = 0;
    let currentContext = await this._buildInitialContext(profileInfo, projectSummary, reviewPrompt);
    let allFileContents = [];
    let fileRequestsHistory = [];
    let finalAnalysis = null;
    let messages = [{ role: 'user', content: currentContext }];

    while (iteration < this.maxIterations) {
      iteration++;

      // Send context to AI and get response
      const aiResponse = await this._sendToAI(null, iteration, messages);
      messages.push({ role: 'assistant', content: aiResponse });
      
      this.analysisLog.push({
        iteration,
        timestamp: new Date().toISOString(),
        contextLength: currentContext.length,
        response: aiResponse
      });

      // Check if AI wants to request files
      const fileRequests = this._extractFileRequests(aiResponse);
      const newFileRequests = fileRequests.filter(request => !fileRequestsHistory.some(f => f.path === request.path));
      fileRequestsHistory.push(...newFileRequests);

      // Check if AI wants to list files
      const listFileRequests = this._extractListFileRequests(aiResponse);
      
      if (fileRequests.length > 0 || listFileRequests.length > 0) {
        let fileResults = [];
        let listResults = [];
        
        // Process file requests first
        if (fileRequests.length > 0) {
          fileResults = await this.fileHandler.processFileRequests(aiResponse);
        }
        
        // Process list file requests separately
        if (listFileRequests.length > 0) {
          listResults = await this.fileHandler.processListFileRequests(aiResponse);
        }
        
        // Add successful file contents to context, avoiding duplicates
        const newFileContents = fileResults
          .filter(result => result.status === 'success')
          .map(result => ({
            path: result.path,
            content: result.content,
            reason: allFileContents.some(f => f.path === result.path) ? 'duplicate' : result.reason
          }))
          .filter(file => !allFileContents.some(f => f.path === file.path));

        allFileContents.push(...newFileContents);

        // Build context for next iteration with both file contents and directory listings
        currentContext = await this._buildIterativeContext(
          newFileContents,
          fileResults,
          listResults,
          iteration,
          fileRequestsHistory,
          projectSummary
        );
        messages.push({ role: 'user', content: currentContext });

      } else {
        // No more file or list requests, this should be the final analysis
        finalAnalysis = aiResponse;
        break;
      }
    }

    // If we reached max iterations, get final analysis
    if (!finalAnalysis) {
      console.log(yellow('⚠️  Reached maximum iterations, requesting final analysis'));
      finalAnalysis = await this._getFinalAnalysis(currentContext, messages);
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

    // context += `Files in the project:\n${projectSummary.directoryStructure}\n`;

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
    context += '\n' + await this.fileHandler.getListFileRequestInstructions();
    context += '\n' + await this.fileHandler.getWordpressFileRequestInstructions();

    return context;
  }

  /**
   * Build context for iterative requests
   */
  async _buildIterativeContext(fileContents, fileResults, listResults, iteration, fileRequestsHistory, projectSummary) {
    let context = "\n";

    // Add file contents from previous requests
    if (fileContents.length > 0) {
      context += `[ANALYZED FILES - Iteration ${iteration}]\n`;
      fileContents.forEach(file => {
        context += `\n=== FILE: ${file.path} ===\n`;
        context += `Reason: ${file.reason}\n`;
        if (file.reason === 'duplicate') {
          context += `[File was already analyzed in previous conversation]\n`;
        } else {
          context += `${file.content}\n`;
        }
        context += `=== END FILE: ${file.path} ===\n`;
      });
    }

    // Add directory listings from previous requests
    if (listResults && listResults.length > 0) {
      context += `\n[DIRECTORY LISTINGS - Iteration ${iteration}]\n`;
      listResults.forEach(result => {
        if (result.status === 'success') {
          context += `\n=== DIRECTORY: ${result.path} ===\n`;
          context += `Reason: ${result.reason}\n`;
          context += `Total items: ${result.totalCount} (${result.directoryCount} directories, ${result.fileCount} files)\n`;
          
          // List directories first
          if (result.directories && result.directories.length > 0) {
            context += `\nDirectories:\n`;
            result.directories.forEach(dir => {
              context += `📁 ${dir.name} (path: ${dir.relativePath})\n`;
            });
          }
          
          // Then list files
          if (result.files && result.files.length > 0) {
            context += `\nFiles:\n`;
            result.files.forEach(file => {
              context += `📄 ${file.name} (path: ${file.relativePath})\n`;
            });
          }
          
          context += `=== END DIRECTORY: ${result.path} ===\n`;
        } else {
          context += `❌ Directory listing failed for ${result.path}: ${result.reason}\n`;
        }
      });
    }

    // Add file request results summary
    if (fileResults.length > 0) {
      context += `\n[FILE REQUEST RESULTS]\n`;
      fileResults.forEach(result => {
        if (result.status === 'success') {
          context += `✅ ${result.path}: Successfully read (${result.lines} lines)\n`;
        } else {
          context += `❌ This file doesn't exist or is not a source file ${result.path}: ${result.reason}. Don't request it again\n`;
        }
      });
    }

    // Add list request results summary
    if (listResults && listResults.length > 0) {
      context += `\n[DIRECTORY LISTING RESULTS]\n`;
      listResults.forEach(result => {
        if (result.status === 'success') {
          context += `✅ ${result.path}: Successfully listed (${result.totalCount} total items: ${result.directoryCount} directories, ${result.fileCount} files)\n`;
        } else {
          context += `❌ Directory listing failed for ${result.path}: ${result.reason}. Don't request it again\n`;
        }
      });
    }

    // Add file request history
    if (fileRequestsHistory.length > 0) {
      context += `\n[FILE REQUEST HISTORY]\n`;
      context += `So far, you have requested ${fileRequestsHistory.length} files\n`;
      context += `Project has ${projectSummary.totalFiles} files\n`;
      context += `\n`;
    }

    // Add instructions for next step
    if (iteration < this.maxIterations) {
      // include files in the project list
      // context += `\nFiles in the project:\n${projectSummary.directoryStructure}\n`;
      // context += '\n' + await this.fileHandler.getFileRequestInstructions();
      const iterativePrompt = await loadPrompt('scanner/iterative-analysis.txt');
      context += `\n${iterativePrompt}`;
    } else {
      const finalIterationPrompt = await loadPrompt('scanner/final-iteration.txt');
      context += `\n${finalIterationPrompt}`;
    }
    //console.log(context);
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
   * Extract list file requests from AI response
   */
  _extractListFileRequests(response) {
    const listFileRequestRegex = /<list_file_request\s+path="([^"]+)"(?:\s+reason="([^"]*)")?\s*\/>/g;
    const requests = [];
    
    let match;
    while ((match = listFileRequestRegex.exec(response)) !== null) {
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
  async _sendToAI(context, iteration, messages) {
    //console.log(dim(`📤 Sending context to AI (${context.length} characters)`));
    
    // Display session state before the call
    // if (this.tokenTracker) {
    //   const preCallData = this.tokenTracker.getPreCallUsageData(iteration);
    //   TokenDisplay.displayPreCallUsage(preCallData);
    // }
    
    try {
      const response = await this.aiClient.analyze(context, messages);
      
      // Extract content from response object
      const content = response.content || response;
      
      // Record token usage from API response if available
      if (this.tokenTracker && response.usage) {
        const usageData = this.tokenTracker.recordUsage(response.usage, iteration);
        TokenDisplay.displayUsageResponse(usageData);
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
  async _getFinalAnalysis(context, messages) {
    const finalAnalysisPrompt = await loadPrompt('scanner/final-analysis.txt');
    const finalContext = context + `\n\n${finalAnalysisPrompt}`;
    messages.push({ role: 'user', content: finalContext });
    return await this._sendToAI(null, 'final', messages);
  }

  /**
   * Get analysis log for debugging
   */
  getAnalysisLog() {
    return this.analysisLog;
  }
}

module.exports = { AISecurityAnalyzer };
