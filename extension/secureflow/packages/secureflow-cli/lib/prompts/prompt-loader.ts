import * as fs from 'fs';
import * as path from 'path';
import { getPromptPath, getAppProfilerPrompt } from './index';

// Try to import vscode, but handle gracefully if not available (CLI context)
let vscode: any;
try {
  vscode = require('vscode');
} catch {
  vscode = null;
}

/**
 * Load a prompt file from the prompts directory
 * @param promptPath Relative path to the prompt file
 * @returns The prompt content as string
 */
export async function loadPrompt(promptPath: string): Promise<string> {
  try {
    let fullPath: string;
    
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
 * @param category Main application category
 * @param subcategory Optional subcategory
 * @param technology Optional specific technology
 * @returns The loaded prompt content
 */
export async function getPromptForAppType(
  category: string,
  subcategory?: string,
  technology?: string
): Promise<string> {
  const promptPath = getPromptPath(category, subcategory, technology);
  return loadPrompt(promptPath);
}

/**
 * Load the application profiler prompt
 * @returns The application profiler prompt content
 */
export async function getApplicationProfilerPrompt(): Promise<string> {
  return loadPrompt(getAppProfilerPrompt());
}

/**
 * Get the threat modeling prompt
 * @returns The threat modeling prompt content
 */
export async function getThreatModelingPrompt(): Promise<string> {
  return loadPrompt('common/threat-modeling.txt');
}

export default {
  loadPrompt,
  getPromptForAppType,
  getApplicationProfilerPrompt,
  getThreatModelingPrompt
};
