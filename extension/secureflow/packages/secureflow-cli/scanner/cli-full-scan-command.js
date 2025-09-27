const fs = require('fs');
const path = require('path');
const { cyan, yellow, red, green, dim, magenta } = require('colorette');
const { ProjectAnalyzer } = require('./project-analyzer');
const { AISecurityAnalyzer } = require('./ai-security-analyzer');
const { loadPrompt } = require('../lib/prompts/prompt-loader');
const { loadConfig } = require('../lib/config');
const { AIClientFactory } = require('../lib/ai-client-factory');
const { TokenTracker } = require('../lib/token-tracker');
const { TokenDisplay } = require('../lib/token-display');
const { withLoader, withSecurityLoader, createLoader } = require('../lib/animated-loader');
const { DefectDojoFormatter } = require('../lib/formatters/defectdojo-formatter');
const { DefectDojoClient } = require('../lib/defectdojo-client');

/**
 * CLI Full Scan Command - performs comprehensive security analysis
 */
class CLIFullScanCommand {
  constructor(options = {}) {
    this.selectedModel = options.selectedModel;
    this.outputFormat = options.outputFormat || 'text';
    this.outputFile = options.outputFile;
    this.defectDojoOptions = options.defectDojoOptions || {};
    this.tokenTracker = null; // Will be initialized when model is known
  }

  /**
   * Execute full scan command
   */
  async execute(projectPath, options = {}) {
    const startTime = Date.now();
    console.log(magenta('🛡️  SecureFlow Full Security Scan'));
    console.log(dim(`Project: ${path.resolve(projectPath)}`));
    console.log(dim(`Model: ${this.selectedModel || 'default'}`));
    console.log('');

    try {
      // Load configuration
      const config = loadConfig();
      if (!config.apiKey) {
        throw new Error('API key not configured. Please run: secureflow config --show');
      }

      // Initialize project analyzer
      const projectSummary = await withLoader('📁 Analyzing project structure...', async () => {
        const projectAnalyzer = new ProjectAnalyzer(projectPath);
        return await projectAnalyzer.getProjectSummary();
      }, {
        successMessage: green('✅ Project structure analyzed'),
        errorMessage: red('❌ Project analysis failed')
      });
      
      console.log(green(`✅ Found ${projectSummary.totalFiles} files in project`));
      console.log(dim(`   Extensions: ${Object.keys(projectSummary.filesByExtension).join(', ')}`));
      console.log(dim(`   Important files: ${projectSummary.importantFiles.length}`));
      
      // // Display directory tree
      // console.log(cyan('\n📂 Project Structure:'));
      // const directoryStructure = await projectAnalyzer.getDirectoryStructure();
      // console.log(projectAnalyzer.formatDirectoryTreeForCLI(directoryStructure));

      // Load profile information if available
      const profileInfo = await this._loadProfileInfo(projectPath);
      if (profileInfo) {
        console.log(green(`✅ Using profile: ${profileInfo.name} (${profileInfo.category})`));
      } else {
        console.log(yellow('⚠️  No profile found, proceeding without context'));
      }
      
      const reviewPrompt = await loadPrompt('common/security-review-cli.txt');

      // Initialize token tracker
      const model = this.selectedModel || config.model || 'claude-3-5-sonnet-20241022';
      this.tokenTracker = new TokenTracker(model);

      // Initialize AI client
      const aiClient = this._createAIClient(config);

      // Initialize AI security analyzer with token tracking
      const aiAnalyzer = new AISecurityAnalyzer(aiClient, projectPath, {
        maxIterations: 20,
        maxFileLines: 5000,
        partialReadLines: 500,
        tokenTracker: this.tokenTracker
      });

      const analysisResult = await aiAnalyzer.analyzeProject(
        profileInfo,
        projectSummary,
        reviewPrompt
      );

      // Parse security issues from final analysis
      const securityIssues = this._parseSecurityIssues(analysisResult.analysis);

      // Generate results
      const scanResult = {
        timestamp: new Date().toISOString(),
        projectPath: path.resolve(projectPath),
        model: this.selectedModel || config.model || 'default',
        totalFiles: projectSummary.totalFiles,
        filesAnalyzed: analysisResult.fileContents.length,
        iterations: analysisResult.iterations,
        issues: securityIssues,
        summary: {
          critical: securityIssues.filter(i => i.severity === 'Critical').length,
          high: securityIssues.filter(i => i.severity === 'High').length,
          medium: securityIssues.filter(i => i.severity === 'Medium').length,
          low: securityIssues.filter(i => i.severity === 'Low').length
        }
      };

      // Output results
      await this._outputResults(scanResult, analysisResult);

      // Display final token usage summary
      if (this.tokenTracker) {
        const summaryData = this.tokenTracker.getFinalSummaryData();
        TokenDisplay.displayFinalSummary(summaryData);
      }

      const duration = ((Date.now() - startTime) / 1000).toFixed(1);
      console.log(green(`\n✅ Scan completed in ${duration}s`));
      console.log(dim(`   Files analyzed: ${scanResult.filesAnalyzed}/${scanResult.totalFiles}`));
      console.log(dim(`   AI iterations: ${scanResult.iterations}`));
      
      if (scanResult.issues.length > 0) {
        console.log(red(`   Security issues found: ${scanResult.issues.length}`));
      } else {
        console.log(green(`   No security issues found`));
      }

    } catch (error) {
      console.error(red('❌ Full scan failed:'));
      console.error(error.message);
      if (process.env.DEBUG) {
        console.error(error.stack);
      }
      process.exit(1);
    }
  }

  /**
   * Load profile information for the project
   */
  async _loadProfileInfo(projectPath) {
    try {
      // Try to find profile in common locations
      const profilePaths = [
        path.join(projectPath, '.secureflow', 'profile.json'),
        path.join(projectPath, 'secureflow-profile.json')
      ];

      for (const profilePath of profilePaths) {
        if (fs.existsSync(profilePath)) {
          const profileData = JSON.parse(fs.readFileSync(profilePath, 'utf8'));
          return profileData;
        }
      }

      return null;
    } catch (error) {
      console.log(yellow(`⚠️  Could not load profile: ${error.message}`));
      return null;
    }
  }

  /**
   * Create AI client based on configuration
   */
  _createAIClient(config) {
    const model = this.selectedModel || config.model || 'claude-3-5-sonnet-20241022';
    const aiClient = AIClientFactory.getClient(model);
    
    // Create wrapper that matches our expected interface
    return {
      analyze: async (context, messages) => {
        return await withSecurityLoader(async () => {
          const response = await aiClient.sendRequest(context, {
            apiKey: config.apiKey,
            model: model,
            temperature: 0,
            maxTokens: 4000
          }, messages);
          
          // Return response object with both content and usage for token tracking
          return response;
        }, {
          successMessage: "",
          errorMessage: red('❌ Security analysis failed')
        });
      }
    };
  }

  /**
   * Parse security issues from AI analysis
   */
  _parseSecurityIssues(analysisText) {
    const issues = [];
    const issueRegex = /<issue>\s*<title>(.*?)<\/title>\s*<severity>(.*?)<\/severity>\s*<description>(.*?)<\/description>\s*<recommendation>(.*?)<\/recommendation>\s*<\/issue>/gs;
    
    let match;
    while ((match = issueRegex.exec(analysisText)) !== null) {
      issues.push({
        title: match[1].trim(),
        severity: match[2].trim(),
        description: match[3].trim(),
        recommendation: match[4].trim()
      });
    }

    return issues;
  }

  /**
   * Output scan results in specified format
   */
  async _outputResults(scanResult, analysisResult) {
    if (this.outputFormat === 'json') {
      const output = JSON.stringify(scanResult, null, 2);
      
      if (this.outputFile) {
        fs.writeFileSync(this.outputFile, output);
        console.log(green(`📄 Results saved to: ${this.outputFile}`));
      } else {
        console.log('\n' + output);
      }
    } else if (this.outputFormat === 'defectdojo') {
      // DefectDojo format output
      const defectDojoFindings = DefectDojoFormatter.formatFindings(scanResult);
      
      // Validate the findings format
      const validation = DefectDojoFormatter.validateFindings(defectDojoFindings);
      if (!validation.valid) {
        console.error(red('❌ DefectDojo format validation failed:'));
        validation.errors.forEach(error => console.error(red(`   ${error}`)));
        throw new Error('DefectDojo format validation failed');
      }
      
      const output = JSON.stringify(defectDojoFindings, null, 2);
      
      // Save to file if specified
      if (this.outputFile) {
        fs.writeFileSync(this.outputFile, output);
        console.log(green(`📄 DefectDojo findings saved to: ${this.outputFile}`));
      }
      
      // Submit to DefectDojo API if configured
      await this._submitToDefectDojo(defectDojoFindings, scanResult);
      
      // Also show a summary in text format for user convenience
      this._outputDefectDojoSummary(scanResult, defectDojoFindings);
    } else {
      // Text format output
      this._outputTextResults(scanResult, analysisResult);
      
      if (this.outputFile) {
        const textOutput = this._generateTextOutput(scanResult, analysisResult);
        fs.writeFileSync(this.outputFile, textOutput);
        console.log(green(`📄 Results saved to: ${this.outputFile}`));
      }
    }
  }

  /**
   * Output results in text format
   */
  _outputTextResults(scanResult, analysisResult) {
    console.log('\n' + '='.repeat(60));
    console.log(magenta('🛡️  SECUREFLOW SECURITY SCAN RESULTS'));
    console.log('='.repeat(60));
    
    console.log(`📅 Timestamp: ${scanResult.timestamp}`);
    console.log(`📁 Project: ${scanResult.projectPath}`);
    console.log(`🤖 Model: ${scanResult.model}`);
    console.log(`📊 Files: ${scanResult.filesAnalyzed}/${scanResult.totalFiles} analyzed`);
    console.log(`🔄 Iterations: ${scanResult.iterations}`);
    
    console.log('\n📈 ISSUE SUMMARY:');
    console.log(`   Critical: ${scanResult.summary.critical}`);
    console.log(`   High:     ${scanResult.summary.high}`);
    console.log(`   Medium:   ${scanResult.summary.medium}`);
    console.log(`   Low:      ${scanResult.summary.low}`);
    console.log(`   Total:    ${scanResult.issues.length}`);

    if (scanResult.issues.length > 0) {
      console.log('\n🔍 SECURITY ISSUES:');
      console.log('-'.repeat(60));
      
      scanResult.issues.forEach((issue, index) => {
        const severityColor = this._getSeverityColor(issue.severity);
        console.log(`\n${index + 1}. ${issue.title}`);
        console.log(`   Severity: ${severityColor(issue.severity)}`);
        console.log(`   Description: ${issue.description}`);
        console.log(`   Recommendation: ${issue.recommendation}`);
      });
    }

    console.log('\n' + '='.repeat(60));
  }

  /**
   * Generate text output for file saving
   */
  _generateTextOutput(scanResult, analysisResult) {
    let output = 'SECUREFLOW SECURITY SCAN RESULTS\n';
    output += '='.repeat(60) + '\n\n';
    
    output += `Timestamp: ${scanResult.timestamp}\n`;
    output += `Project: ${scanResult.projectPath}\n`;
    output += `Model: ${scanResult.model}\n`;
    output += `Files: ${scanResult.filesAnalyzed}/${scanResult.totalFiles} analyzed\n`;
    output += `Iterations: ${scanResult.iterations}\n\n`;
    
    output += 'ISSUE SUMMARY:\n';
    output += `Critical: ${scanResult.summary.critical}\n`;
    output += `High: ${scanResult.summary.high}\n`;
    output += `Medium: ${scanResult.summary.medium}\n`;
    output += `Low: ${scanResult.summary.low}\n`;
    output += `Total: ${scanResult.issues.length}\n\n`;

    if (scanResult.issues.length > 0) {
      output += 'SECURITY ISSUES:\n';
      output += '-'.repeat(60) + '\n\n';
      
      scanResult.issues.forEach((issue, index) => {
        output += `${index + 1}. ${issue.title}\n`;
        output += `   Severity: ${issue.severity}\n`;
        output += `   Description: ${issue.description}\n`;
        output += `   Recommendation: ${issue.recommendation}\n\n`;
      });
    }

    return output;
  }

  /**
   * Output DefectDojo summary for user convenience
   */
  _outputDefectDojoSummary(scanResult, defectDojoFindings) {    
    console.log(`🤖 Model: ${scanResult.model}`);
    console.log(`📊 Files: ${scanResult.filesAnalyzed}/${scanResult.totalFiles} analyzed`);
    
    const summaryCounts = {
      critical: defectDojoFindings.findings.filter(f => f.severity.toLowerCase() === 'critical').length,
      high: defectDojoFindings.findings.filter(f => f.severity.toLowerCase() === 'high').length,
      medium: defectDojoFindings.findings.filter(f => f.severity.toLowerCase() === 'medium').length,
      low: defectDojoFindings.findings.filter(f => f.severity.toLowerCase() === 'low').length,
      info: defectDojoFindings.findings.filter(f => f.severity.toLowerCase() === 'info').length
    };

    console.log(
      `\n${magenta(`Critical:${summaryCounts.critical}`)} ${cyan(`High:${summaryCounts.high}`)} ${yellow(`Medium:${summaryCounts.medium}`)} ${green(`Low:${summaryCounts.low}`)} ${dim(`Info:${summaryCounts.info}`)}`
    );

    if (defectDojoFindings.findings.length > 0) {
      // Show first 3 findings as examples
      defectDojoFindings.findings.forEach((finding, index) => {
        const severityColor = this._getSeverityColor(finding.severity);
        console.log(`\n${index + 1}. ${finding.title}`);
        console.log(`   Severity: ${severityColor(finding.severity)}`);
        console.log(`   ID: ${finding.unique_id_from_tool}`);
        if (finding.file_path) {
          console.log(`   File: ${finding.file_path}${finding.line ? `:${finding.line}` : ''}`);
        }
        if (finding.tags && finding.tags.length > 0) {
          console.log(`   Tags: ${finding.tags.join(', ')}`);
        }
      });
    }
  }

  /**
   * Submit findings to DefectDojo API if configured
   */
  async _submitToDefectDojo(defectDojoFindings, scanResult) {
    // Check if DefectDojo API options are provided (engagement ID is optional)
    if (!this.defectDojoOptions.url || !this.defectDojoOptions.token || 
        !this.defectDojoOptions.productId) {
      console.log(dim('ℹ️  DefectDojo API not configured - skipping automatic submission'));
      console.log(dim('    Required: --defectdojo-url, --defectdojo-token, --defectdojo-product-id'));
      return;
    }
    try {
      // console.log(cyan('\n🔗 Submitting findings to DefectDojo...'));
      
      // Create DefectDojo client and validate configuration
      const defectDojoClient = new DefectDojoClient(this.defectDojoOptions);
      const validation = await defectDojoClient.validateConfiguration();
      
      if (!validation.valid) {
        throw new Error(`Configuration validation failed: ${validation.errors.join(', ')}`);
      }
      
      // Notify if engagement was created
      // if (validation.engagementCreated) {
      //   console.log(yellow('   📝 Created new engagement (ID: ' + validation.engagementId + ')'));
      // }
      
      // Submit findings
      const result = await defectDojoClient.submitFindings(defectDojoFindings);
      if (result.success) {
        console.log(green('✅ Findings submitted to DefectDojo'));
      } else {
        console.error(red('❌ DefectDojo submission failed'));
        console.error(red(`   ${result.errorMessage}`));
      }
      
    } catch (error) {
      console.error(red('❌ DefectDojo submission failed:'));
      console.error(red(`   ${error.message}`));
      if (process.env.DEBUG) {
        console.error(error.stack);
      }
    }
  }

  /**
   * Get color function for severity level
   */
  _getSeverityColor(severity) {
    switch (severity.toLowerCase()) {
      case 'critical': return red;
      case 'high': return red;
      case 'medium': return yellow;
      case 'low': return green;
      case 'info': return dim;
      default: return dim;
    }
  }
}

module.exports = { CLIFullScanCommand };
