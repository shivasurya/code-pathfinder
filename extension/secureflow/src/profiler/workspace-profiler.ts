import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { getPromptPath } from '../prompts';
import { loadPrompt } from '../prompts/prompt-loader';
import { AIClientFactory } from '../clients';
import { AIClient } from '../clients/ai-client';
import { SettingsManager } from '../settings/settings-manager';

interface ProjectApplication {
  name: string;
  path: string;
  category: string;
  subcategory?: string;
  technology?: string;
  confidence: number;
  languages: string[];
  frameworks: string[];
  buildTools: string[];
  evidence: string[];
}

interface ProfilerResult {
  applications: ProjectApplication[];
}

/**
 * Class responsible for workspace profiling to identify application types
 */
export class WorkspaceProfiler {
  private readonly ignorePatterns = [
    'node_modules',
    'dist',
    'build',
    '.git',
    'coverage',
    'generated',
    '.next',
    '.nuxt',
    'venv',
    '__pycache__',
    '.DS_Store',
    '*.lock',
    '*.log'
  ];

  /**
   * Maximum file size to read in bytes (1MB)
   */
  private readonly MAX_FILE_SIZE = 1024 * 1024;

  /**
   * Files that provide strong signals about project type
   */
  private readonly highSignalFiles = [
    'package.json',
    'tsconfig.json',
    'webpack.config.js',
    'next.config.js',
    'angular.json',
    'vue.config.js',
    'nuxt.config.js',
    'AndroidManifest.xml',
    'Info.plist',
    'Podfile',
    'pubspec.yaml',
    'pom.xml',
    'build.gradle',
    'Cargo.toml',
    'go.mod',
    'pyproject.toml',
    'setup.py',
    'requirements.txt',
    'Gemfile',
    'composer.json',
    'serverless.yml',
    'Dockerfile',
    '.csproj',
    '.fsproj',
    'project.json'
  ];

  // AI client for workspace profiling
  private aiClient: AIClient;

  constructor(
    private readonly workspaceFolder: vscode.WorkspaceFolder,
    private readonly settingsManager?: SettingsManager
  ) {
    // Initialize AI client
    if (this.settingsManager) {
      const selectedModel = this.settingsManager.getSelectedAIModel();
      this.aiClient = AIClientFactory.getClient(selectedModel);
    } else {
      // Default to GPT-4o if settings manager is not provided
      this.aiClient = AIClientFactory.getClient('gpt-4o');
    }
  }

  /**
   * Profile the workspace to determine application types
   */
  public async profileWorkspace(): Promise<ProfilerResult> {
    // Start with workspace structure analysis
    const workspaceStructure = await this.getWorkspaceStructure();
    
    // Use AI to profile the workspace based on structure
    const applications = await this.analyzeWorkspaceWithAI(workspaceStructure);
    
    return { applications };
  }

  /**
   * Get a high-level view of the workspace structure
   * Returns a tree-like structure of directories and key files
   */
  private async getWorkspaceStructure(): Promise<any> {
    const structure: any = {
      root: this.workspaceFolder.uri.fsPath,
      directories: {},
      keyFiles: []
    };
    
    await this.scanDirectory(this.workspaceFolder.uri.fsPath, structure, 0, 3);
    
    return structure;
  }
  
  /**
   * Recursively scan a directory to build structure
   * @param dirPath Directory to scan
   * @param structure Structure object to populate
   * @param currentDepth Current depth in the tree
   * @param maxDepth Maximum depth to scan
   */
  private async scanDirectory(
    dirPath: string, 
    structure: any, 
    currentDepth: number, 
    maxDepth: number
  ): Promise<void> {
    if (currentDepth > maxDepth) {
      return;
    }
    
    try {
      const entries = fs.readdirSync(dirPath, { withFileTypes: true });
      
      for (const entry of entries) {
        const entryPath = path.join(dirPath, entry.name);
        
        // Skip ignored patterns
        if (this.shouldIgnore(entry.name)) {
          continue;
        }
        
        if (entry.isDirectory()) {
          if (currentDepth < maxDepth) {
            structure.directories[entry.name] = { 
              directories: {}, 
              keyFiles: [] 
            };
            await this.scanDirectory(
              entryPath, 
              structure.directories[entry.name], 
              currentDepth + 1, 
              maxDepth
            );
          } else {
            structure.directories[entry.name] = "[Directory contents not scanned]";
          }
        } else if (entry.isFile()) {
          // Only include high-signal files in the structure
          if (this.isHighSignalFile(entry.name)) {
            // For key files, also store their content if not too large
            const stats = fs.statSync(entryPath);
            if (stats.size <= this.MAX_FILE_SIZE) {
              const content = fs.readFileSync(entryPath, 'utf8');
              structure.keyFiles.push({
                name: entry.name,
                path: entryPath,
                content: content
              });
            } else {
              structure.keyFiles.push({
                name: entry.name,
                path: entryPath,
                content: "[File too large to include]"
              });
            }
          }
        }
      }
    } catch (error) {
      console.error(`Error scanning directory ${dirPath}:`, error);
    }
  }
  
  /**
   * Check if a file or directory should be ignored
   */
  private shouldIgnore(name: string): boolean {
    for (const pattern of this.ignorePatterns) {
      if (pattern.startsWith('*')) {
        // Handle extension pattern
        const ext = pattern.substring(1); 
        if (name.endsWith(ext)) {
          return true;
        }
      } else if (name === pattern) {
        return true;
      }
    }
    return false;
  }
  
  /**
   * Check if a file provides high signal about project type
   */
  private isHighSignalFile(fileName: string): boolean {
    return this.highSignalFiles.some(pattern => {
      if (pattern.startsWith('*.')) {
        // Handle extension pattern
        const ext = pattern.substring(1);
        return fileName.endsWith(ext);
      }
      return fileName === pattern;
    });
  }
  
  /**
   * Use AI to analyze the workspace structure and determine application types
   */
  private async analyzeWorkspaceWithAI(workspaceStructure: any): Promise<ProjectApplication[]> {
    try {
      // Create a condensed representation of the project for the AI
      const projectData = {
        root: workspaceStructure.root,
        directories: Object.keys(workspaceStructure.directories),
        keyFiles: workspaceStructure.keyFiles.map((file: any) => ({
          name: file.name,
          path: file.path,
          content: file.content.length > 1000 ? file.content.substring(0, 1000) + "..." : file.content
        }))
      };

      // Load the app profiler prompt
      let promptTemplate = '';
      try {
        promptTemplate = await loadPrompt('common/app-profiler.txt');
      } catch (error) {
        console.error('Error loading app profiler prompt:', error);
        // Fallback to a basic prompt
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

      // If no AI client is initialized, return mock data
      if (!this.aiClient) {
        return this.getMockProfiles();
      }

      try {
        // Call the AI client to analyze the workspace
        const response = await this.aiClient.sendRequest(prompt, {
          temperature: 0.1, // Lower temperature for more deterministic results
          maxTokens: 2048,  // Ensure enough tokens for the response
          apiKey: '' // The API key should be managed by the client
        });
        
        // Parse the JSON response
        try {
          // Extract the JSON part from the response
          const jsonMatch = response.content.match(/\{[\s\S]*\}/);
          if (!jsonMatch) {
            console.error('No JSON found in response:', response.content);
            return this.getMockProfiles(); // Fallback to mock profiles
          }
          
          const jsonContent = jsonMatch[0];
          const result = JSON.parse(jsonContent);
          
          if (!result.applications || !Array.isArray(result.applications)) {
            console.error('Invalid response format, missing applications array:', result);
            return this.getMockProfiles(); // Fallback to mock profiles
          }
          
          return result.applications;
        } catch (parseError) {
          console.error('Error parsing AI response:', parseError);
          console.error('Response content:', response.content);
          return this.getMockProfiles(); // Fallback to mock profiles
        }
      } catch (aiError) {
        console.error('Error calling AI service:', aiError);
        return this.getMockProfiles(); // Fallback to mock profiles
      }
    } catch (error) {
      console.error('Error analyzing workspace with AI:', error);
      return this.getMockProfiles();
    }
  }
  
  /**
   * Get mock application profiles for testing or fallback
   */
  private getMockProfiles(): ProjectApplication[] {
    const mockApplications: ProjectApplication[] = [
      {
        name: "Sample Express API",
        path: "/",
        category: "backend",
        subcategory: "http",
        technology: "express",
        confidence: 85,
        languages: ["javascript", "typescript"],
        frameworks: ["express"],
        buildTools: ["webpack"],
        evidence: ["Found package.json with express dependency", "Server routing patterns detected"]
      }
    ];
    
    return mockApplications;
  }
  
  /**
   * Get the appropriate security prompt for an application
   */
  public getPromptForApplication(app: ProjectApplication): string {
    return getPromptPath(app.category, app.subcategory, app.technology);
  }
}

export default WorkspaceProfiler;
