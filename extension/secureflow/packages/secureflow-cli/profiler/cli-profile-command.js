const path = require('path');
const fs = require('fs');
const { CLIProjectProfiler } = require('./cli-project-profiler');
const { loadConfig } = require('../lib/config');
const { AIClientFactory } = require('../lib/ai-client-factory');
const { TokenTracker } = require('../lib/token-tracker');
const { TokenDisplay } = require('../lib/token-display');
const { yellow, green, red, cyan, blue } = require('colorette');

/**
 * CLI command handler for workspace profiling
 */
class CLIProfileCommand {
  constructor(options = {}) {
    this.selectedModel = options.selectedModel;
  }

  /**
   * Execute the profile command
   * @param {string} projectPath - Path to the project to profile
   * @param {Object} options - Command options
   */
  async execute(projectPath, options = {}) {
    try {
      // Resolve and validate project path
      const resolvedPath = path.resolve(projectPath || process.cwd());
      
      if (!fs.existsSync(resolvedPath)) {
        console.error(red(`❌ Project path does not exist: ${resolvedPath}`));
        process.exit(1);
      }

      const stats = fs.statSync(resolvedPath);
      if (!stats.isDirectory()) {
        console.error(red(`❌ Project path is not a directory: ${resolvedPath}`));
        process.exit(1);
      }

      console.log(cyan('🔍 SecureFlow Workspace Profiler'));
      console.log(`📂 Analyzing project at: ${resolvedPath}`);
      console.log('');

      // Load configuration to get API key and model
      const config = loadConfig();
      if (!config.apiKey) {
        console.error(red('❌ No API key configured. Please run "secureflow config" to set up your configuration.'));
        process.exit(1);
      }

      // Create profiler instance
      const profilerOptions = {
        selectedModel: this.selectedModel || config.model || 'claude-sonnet-4-5-20250929'
      };
      const profiler = new CLIProjectProfiler(profilerOptions);

      console.log(blue(`🤖 Using AI model: ${profilerOptions.selectedModel}`));
      console.log('');

      // Progress callback for user feedback
      const updateProgress = (message) => {
        console.log(yellow(`⏳ ${message}`));
      };

      // Profile the workspace
      const applications = await profiler.profileWorkspace(
        resolvedPath,
        config.apiKey,
        updateProgress
      );

      // Display token usage summary if available
      if (profiler.getTokenUsage) {
        profiler.displayTokenSummary();
      }

      // Handle results
      if (applications.length === 0) {
        console.log(yellow('❓ Could not determine the application type'));
        console.log('Manual configuration may be required.');
        return;
      }

      // Display results
      console.log('');
      console.log(green('✅ Workspace profiling complete!'));
      console.log(`📊 Detected ${applications.length} application${applications.length > 1 ? 's' : ''}`);
      console.log('');

      // Show detailed information for each application
      applications.forEach((app, index) => {
        console.log(cyan(`----- Application ${index + 1} -----`));
        console.log(`Name: ${app.name}`);
        console.log(`Path: ${app.path}`);
        console.log(`Type: ${app.category}${app.subcategory ? '/' + app.subcategory : ''}`);
        console.log(`Technology: ${app.technology || 'N/A'}`);
        console.log(`Languages: ${app.languages.join(', ')}`);
        console.log(`Frameworks: ${app.frameworks.join(', ')}`);
        console.log(`Build Tools: ${app.buildTools.join(', ')}`);
        console.log(`Confidence: ${app.confidence}%`);
        console.log('');
        console.log('Evidence:');
        app.evidence.forEach((e) => console.log(`  • ${e}`));
        console.log('');
      });

      // Output format handling
      if (options.format === 'json') {
        console.log(cyan('JSON Output:'));
        console.log(JSON.stringify(applications, null, 2));
      }

      // Save results if requested
      if (options.output) {
        const outputPath = path.resolve(options.output);
        const outputData = {
          timestamp: new Date().toISOString(),
          projectPath: resolvedPath,
          model: profilerOptions.selectedModel,
          applications
        };
        
        fs.writeFileSync(outputPath, JSON.stringify(outputData, null, 2));
        console.log(green(`💾 Results saved to: ${outputPath}`));
      }

    } catch (error) {
      console.error(red('❌ Error during workspace profiling:'));
      console.error(error.message);
      if (process.env.DEBUG) {
        console.error(error.stack);
      }
      process.exit(1);
    }
  }

  /**
   * Set the AI model to use
   * @param {string} modelName - Name of the AI model
   */
  setModel(modelName) {
    this.selectedModel = modelName;
  }
}

module.exports = {
  CLIProfileCommand
};
