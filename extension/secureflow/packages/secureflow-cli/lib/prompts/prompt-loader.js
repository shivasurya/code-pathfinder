const fs = require('fs');
const path = require('path');
const { getPromptPath, getAppProfilerPrompt } = require('./index');

// Try to import vscode, but handle gracefully if not available (CLI context)
let vscode;
try {
  vscode = require('vscode');
} catch {
  vscode = null;
}

/**
 * Load a prompt file from the prompts directory
 * @param {string} promptPath Relative path to the prompt file
 * @returns {Promise<string>} The prompt content as string
 */
async function loadPrompt(promptPath) {
  try {
    let fullPath;
    
    if (vscode) {
      // VS Code extension context
      const extensionPath = vscode.extensions.getExtension(
        'codepathfinder.secureflow'
      )?.extensionPath;

      if (!extensionPath) {
        throw new Error('Could not find extension path');
      }

      fullPath = path.join(extensionPath, 'dist', 'prompts', promptPath);
    } else {
      // CLI context - prompts are in the same directory as this file
      fullPath = path.join(__dirname, promptPath);
    }
    
    return fs.readFileSync(fullPath, 'utf8');
  } catch (error) {
    console.error(`Error loading prompt: ${error}`);
    throw error;
  }
}

/**
 * Get and load the appropriate prompt based on application profile
 * @param {string} category Main application category
 * @param {string} subcategory Optional subcategory
 * @param {string} technology Optional specific technology
 * @returns {Promise<string>} The loaded prompt content
 */
async function getPromptForAppType(category, subcategory, technology) {
  const promptPath = getPromptPath(category, subcategory, technology);
  return loadPrompt(promptPath);
}

/**
 * Load the application profiler prompt
 * @returns {Promise<string>} The application profiler prompt content
 */
async function getApplicationProfilerPrompt() {
  return loadPrompt(getAppProfilerPrompt());
}

/**
 * Get the threat modeling prompt
 * @returns {Promise<string>} The threat modeling prompt content
 */
async function getThreatModelingPrompt() {
  return loadPrompt('common/threat-modeling.txt');
}

module.exports = {
  loadPrompt,
  getPromptForAppType,
  getApplicationProfilerPrompt,
  getThreatModelingPrompt
};
