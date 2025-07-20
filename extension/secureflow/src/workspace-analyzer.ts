import * as vscode from 'vscode';
import WorkspaceProfiler from './workspace-profiler';
import { getPromptForAppType } from './prompts/prompt-loader';

/**
 * Class that manages the workspace profiling and security analysis process
 */
export class WorkspaceAnalyzer {
  private profiler: WorkspaceProfiler | undefined;
  
  constructor() {}
  
  /**
   * Initialize the analyzer with the active workspace
   */
  public async initialize(workspaceFolder: vscode.WorkspaceFolder): Promise<void> {
    this.profiler = new WorkspaceProfiler(workspaceFolder);
  }
  
  /**
   * Profile the workspace and analyze for security issues
   */
  public async analyzeWorkspace(): Promise<any> {
    if (!this.profiler) {
      throw new Error('WorkspaceAnalyzer not initialized');
    }
    
    // Step 1: Profile the workspace to identify application types
    const profilerResult = await this.profiler.profileWorkspace();
    
    // If multiple applications were found, let the user choose which to analyze
    let applicationsToAnalyze = profilerResult.applications;
    
    if (applicationsToAnalyze.length > 1) {
      const selected = await this.promptUserForApplicationSelection(applicationsToAnalyze);
      if (!selected) {
        return null; // User cancelled
      }
      applicationsToAnalyze = [selected];
    }
    
    // No applications detected
    if (applicationsToAnalyze.length === 0) {
      vscode.window.showWarningMessage('Could not determine the application type. Using generic analysis.');
      return this.runGenericAnalysis();
    }
    
    // Run analysis for each selected application
    const results = [];
    for (const app of applicationsToAnalyze) {
      const result = await this.analyzeApplication(app);
      results.push(result);
    }
    
    return results;
  }
  
  /**
   * Prompt user to select which application to analyze
   */
  private async promptUserForApplicationSelection(applications: any[]): Promise<any | undefined> {
    const items = applications.map(app => ({
      label: app.name,
      description: `${app.category}${app.subcategory ? '/' + app.subcategory : ''}`,
      detail: `Path: ${app.path}, Confidence: ${app.confidence}%`,
      application: app
    }));
    
    const selected = await vscode.window.showQuickPick(items, {
      placeHolder: 'Select an application to analyze',
      canPickMany: false
    });
    
    return selected ? selected.application : undefined;
  }
  
  /**
   * Analyze a specific application using its profile
   */
  private async analyzeApplication(application: any): Promise<any> {
    try {
      // Get the appropriate prompt for this application type
      const prompt = await getPromptForAppType(
        application.category,
        application.subcategory,
        application.technology
      );
      
      // TODO: Implement the actual security analysis using the prompt
      // This is where we'd send the code to the AI for security review
      
      return {
        application: application,
        securityIssues: [] // Placeholder for actual security analysis results
      };
    } catch (error) {
      console.error('Error analyzing application:', error);
      throw error;
    }
  }
  
  /**
   * Fallback to generic analysis when app type can't be determined
   */
  private async runGenericAnalysis(): Promise<any> {
    // TODO: Implement generic security analysis
    return {
      application: {
        name: "Unknown Application",
        category: "unknown"
      },
      securityIssues: [] // Placeholder for actual security analysis results
    };
  }
}

export default WorkspaceAnalyzer;
