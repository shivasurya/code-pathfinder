const { AIClientFactory } = require('../ai-client-factory');
const { loadPrompt } = require('../prompts/prompt-loader');
const { TokenTracker } = require('../token-tracker');
const { TokenDisplay } = require('../token-display');

/**
 * Interface representing a detected application in the workspace
 */
class ApplicationProfile {
  constructor(data) {
    this.name = data.name;
    this.path = data.path;
    this.category = data.category;
    this.subcategory = data.subcategory;
    this.technology = data.technology;
    this.confidence = data.confidence;
    this.languages = data.languages || [];
    this.frameworks = data.frameworks || [];
    this.buildTools = data.buildTools || [];
    this.evidence = data.evidence || [];
  }
}

/**
 * Core workspace analyzer that handles AI communication and workspace nature determination
 * This is platform-agnostic and can be used by both CLI and VS Code extension
 */
class WorkspaceAnalyzer {
  constructor(options = {}) {
    this.selectedModel = options.selectedModel || 'claude-sonnet-4-5-20250929';
    console.log('WorkspaceAnalyzer: Initializing with model:', this.selectedModel);
    this.aiClient = AIClientFactory.getClient(this.selectedModel);
    console.log('WorkspaceAnalyzer: Client type:', this.aiClient.constructor.name);
    this.tokenTracker = new TokenTracker(this.selectedModel);
  }

  /**
   * Use AI to determine the project types based on structure and key files
   * @param {Object} projectStructure - The collected project structure
   * @param {Array} keyFileContents - Contents of key project files
   * @param {string} secretApiKey - API key for the AI service
   * @param {Function} progressCallback - Optional progress callback
   * @returns {Promise<ApplicationProfile[]>} Array of detected application profiles
   */
  async determineProjectTypes(
    projectStructure,
    keyFileContents,
    secretApiKey,
    progressCallback
  ) {
    try {
      // Create a condensed representation of the project
      const projectData = {
        structure: {
          directories: projectStructure.directories.map((dir) => ({
            name: dir.name,
            path: dir.path,
            depth: dir.depth
          })),
          files: projectStructure.files.map((file) => ({
            name: file.name,
            path: file.path,
            extension: file.extension
          }))
        },
        keyFiles: keyFileContents
      };

      // Load the app profiler prompt
      let promptTemplate = '';
      try {
        promptTemplate = await loadPrompt('common/app-profiler.txt');
      } catch (error) {
        console.error('Error loading app profiler prompt:', error);
        // Fallback to a basic prompt if the file can't be loaded
        promptTemplate = `You are an expert application profiler. Analyze the following project structure and key file contents to determine the type of application(s).`;
      }

      // Create a prompt for the AI
      const prompt = `
      ${promptTemplate}
      
      PROJECT STRUCTURE:
      ${JSON.stringify(projectData, null, 2)}
      
      Based on this information, determine:
      1. The type of application(s) in this workspace
      2. If it's a monorepo, identify each distinct application
      3. The primary programming languages and frameworks used
      4. For each identified application, provide category, subcategory, and technology
      
      Respond in the following JSON format:
      {
        "applications": [
          {
            "name": "application name",
            "path": "relative/path/to/app",
            "category": "category",
            "subcategory": "subcategory",
            "technology": "specific technology",
            "confidence": confidence percentage,
            "languages": ["language1", "language2"],
            "frameworks": ["framework1", "framework2"],
            "buildTools": ["tool1", "tool2"],
            "evidence": ["reason1", "reason2"]
          }
        ]
      }
      `;

      progressCallback?.('Sending request to AI service...');

      // Display session state before the call
      const preCallData = this.tokenTracker.getPreCallUsageData();
      TokenDisplay.displayPreCallUsage(preCallData);

      try {
        // Call the AI client to analyze the workspace
        console.log('WorkspaceAnalyzer: Sending request with model:', this.selectedModel);
        const response = await this.aiClient.sendRequest(prompt, {
          temperature: 0, // Lower temperature for more deterministic results
          maxTokens: 2048, // Ensure enough tokens for the response
          apiKey: secretApiKey, // The API key should be managed by the client
          model: this.selectedModel // Explicitly pass the model
        });

        // Record token usage from API response
        const usageData = this.tokenTracker.recordUsage(response.usage);
        TokenDisplay.displayUsageResponse(usageData);

        // Parse the JSON response
        try {
          // Extract the JSON part from the response
          const jsonMatch = response.content.match(/\{[\s\S]*\}/);
          if (!jsonMatch) {
            console.error('No JSON found in response:', response.content);
            return []; // Fallback to empty array
          }

          const jsonContent = jsonMatch[0];
          const result = JSON.parse(jsonContent);

          if (!result.applications || !Array.isArray(result.applications)) {
            console.error(
              'Invalid response format, missing applications array:',
              result
            );
            return []; // Fallback to empty array
          }

          // Convert to ApplicationProfile instances
          return result.applications.map(app => new ApplicationProfile(app));
        } catch (parseError) {
          console.error('Error parsing AI response:', parseError);
          console.error('Response content:', response.content);
          return []; // Fallback to empty array
        }
      } catch (aiError) {
        console.error('Error calling AI service:', aiError);
        return []; // Fallback to empty array
      }
    } catch (error) {
      console.error('Error determining project types:', error);
      throw error;
    }
  }

  /**
   * Set the AI model to use for analysis
   * @param {string} modelName - Name of the AI model
   */
  setModel(modelName) {
    this.selectedModel = modelName;
    this.aiClient = AIClientFactory.getClient(modelName);
    this.tokenTracker = new TokenTracker(modelName);
  }

  /**
   * Get token usage statistics
   */
  getTokenUsage() {
    return this.tokenTracker.getUsageStats();
  }

  /**
   * Display final token usage summary
   */
  displayTokenSummary() {
    const summaryData = this.tokenTracker.getFinalSummaryData();
    TokenDisplay.displayFinalSummary(summaryData);
  }
}

module.exports = {
  WorkspaceAnalyzer,
  ApplicationProfile
};
