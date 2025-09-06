/**
 * Interface representing a detected application in the workspace
 */
export interface ApplicationProfile {
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

/**
 * Options for configuring the WorkspaceAnalyzer
 */
export interface WorkspaceAnalyzerOptions {
  selectedModel?: string;
}

/**
 * Core workspace analyzer that handles AI communication and workspace nature determination
 * This is platform-agnostic and can be used by both CLI and VS Code extension
 */
export declare class WorkspaceAnalyzer {
  constructor(options?: WorkspaceAnalyzerOptions);

  /**
   * Use AI to determine the project types based on structure and key files
   * @param projectStructure - The collected project structure
   * @param keyFileContents - Contents of key project files
   * @param secretApiKey - API key for the AI service
   * @param progressCallback - Optional progress callback
   * @returns Array of detected application profiles
   */
  determineProjectTypes(
    projectStructure: any,
    keyFileContents: any[],
    secretApiKey: string,
    progressCallback?: (message: string) => void
  ): Promise<ApplicationProfile[]>;

  /**
   * Set the AI model to use for analysis
   * @param modelName - Name of the AI model
   */
  setModel(modelName: string): void;
}

export declare class ApplicationProfile {
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

  constructor(data: any);
}
