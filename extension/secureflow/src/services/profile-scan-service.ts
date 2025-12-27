import * as vscode from 'vscode';
import * as path from 'path';
import { StoredProfile } from '../models/profile-store';
import { SettingsManager } from '../settings/settings-manager';
import { ScanStorageService } from './scan-storage-service';

// Import CLI scanner - note: this is a CommonJS module
const CLIScanner = require('@codepathfinder/secureflow-cli/scanner/cli-full-scan-command');

/**
 * Service for running full security scans on profiles using the CLI scanner
 */
export class ProfileScanService {
  private context: vscode.ExtensionContext;
  private settingsManager: SettingsManager;
  private scanStorageService: ScanStorageService;

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.settingsManager = new SettingsManager(context);
    this.scanStorageService = new ScanStorageService(context);
  }

  /**
   * Run a full security scan on a profile
   * @param profile The profile to scan
   * @param progressCallback Optional callback for progress updates
   * @returns The saved scan result
   */
  async scanProfile(
    profile: StoredProfile,
    progressCallback?: (message: string) => void
  ): Promise<any> {
    try {
      // Get workspace folder
      const workspaceFolder = vscode.workspace.getWorkspaceFolder(
        vscode.Uri.parse(profile.workspaceFolderUri)
      );

      if (!workspaceFolder) {
        throw new Error('Workspace folder not found for profile');
      }

      // Build absolute path to scan
      const projectPath = path.join(
        workspaceFolder.uri.fsPath,
        profile.path === '/' ? '' : profile.path
      );

      // Get AI configuration
      const apiKey = await this.settingsManager.getApiKey();
      const model = this.settingsManager.getSelectedAIModel();

      if (!apiKey || !model) {
        throw new Error('AI model and API key must be configured. Please check settings.');
      }

      progressCallback?.(`Starting security scan for ${profile.name}...`);

      // Create temporary output file for JSON results
      const outputFile = path.join(
        this.context.globalStorageUri.fsPath,
        `scan-${profile.id}-${Date.now()}.json`
      );

      // Ensure global storage directory exists
      await vscode.workspace.fs.createDirectory(this.context.globalStorageUri);

      // Infer provider from model name
      // OpenRouter models contain "/" (e.g., "anthropic/claude-3-5-sonnet")
      let provider: string | undefined = undefined;
      if (/^[a-z0-9-]+\/[a-z0-9-]+/i.test(model)) {
        provider = 'openrouter';
      }

      // Initialize CLI scanner with config passed directly
      const scanner = new CLIScanner.CLIFullScanCommand({
        selectedModel: model,
        outputFormat: 'json',
        outputFile: outputFile,
        maxIterations: 20, // Maximum iterations for thorough analysis
        silent: true, // Disable console output for extension usage
        config: {
          apiKey: apiKey,
          model: model,
          provider: provider,
          analytics: {
            enabled: false // Disable analytics for extension scans
          }
        }
      });

      progressCallback?.('Analyzing project structure...');

      await scanner.execute(projectPath);

      progressCallback?.('Scan complete, processing results...');

      // Read results
      const resultsUri = vscode.Uri.file(outputFile);

      // Check if file exists
      try {
        await vscode.workspace.fs.stat(resultsUri);
      } catch (error) {
        throw new Error(`Scan results file not found: ${outputFile}`);
      }

      const resultsData = await vscode.workspace.fs.readFile(resultsUri);
      const resultsText = Buffer.from(resultsData).toString('utf8');
      const scanResults = JSON.parse(resultsText);

      // Map CLI results to our format
      const mappedIssues = this.mapCliResultsToIssues(scanResults, workspaceFolder.uri.fsPath, profile.path);

      // Save scan to storage with profileId
      const scanSummary = `Full scan complete! Found ${mappedIssues.length} issue${mappedIssues.length !== 1 ? 's' : ''}.`;
      const savedScan = await this.scanStorageService.saveScan(
        mappedIssues,
        scanSummary,
        `Scanned ${scanResults.filesAnalyzed} files in ${scanResults.iterations} iterations`,
        scanResults.filesAnalyzed || 0,
        model,
        profile.id // Link scan to profile
      );

      progressCallback?.(`✓ Scan #${savedScan.scanNumber} saved with ${mappedIssues.length} findings`);

      // Clean up temporary JSON file
      try {
        await vscode.workspace.fs.delete(resultsUri);
      } catch (error) {
        // Ignore cleanup errors
      }

      return savedScan;
    } catch (error: any) {
      progressCallback?.(`✗ Scan failed: ${error.message || error}`);
      throw error;
    }
  }

  /**
   * Map CLI scan results to our issue format
   */
  private mapCliResultsToIssues(
    scanResults: any,
    workspacePath: string,
    profilePath: string
  ): Array<{
    issue: {
      title: string;
      severity: 'Low' | 'Medium' | 'High' | 'Critical';
      description: string;
      recommendation: string;
    };
    filePath: string;
    startLine: number;
  }> {
    if (!scanResults.issues || scanResults.issues.length === 0) {
      return [];
    }

    return scanResults.issues.map((issue: any) => {
      // Extract file path and line number from description if present
      const fileMatch = issue.description?.match(/(?:in|at|file:)\s+([^\s:]+):?(\d+)?/i);
      const extractedFile = fileMatch ? fileMatch[1] : '';
      const extractedLine = fileMatch && fileMatch[2] ? parseInt(fileMatch[2], 10) : 1;

      // Clean up file path
      let filePath = extractedFile || 'unknown';
      if (filePath.startsWith('./')) {
        filePath = filePath.substring(2);
      }
      if (filePath.startsWith('/')) {
        filePath = filePath.substring(1);
      }

      // Validate severity
      const validSeverities = ['Low', 'Medium', 'High', 'Critical'];
      const severity = validSeverities.includes(issue.severity)
        ? issue.severity
        : 'Medium';

      return {
        issue: {
          title: issue.title || 'Security Issue',
          severity: severity as 'Low' | 'Medium' | 'High' | 'Critical',
          description: issue.description || '',
          recommendation: issue.recommendation || 'Review and fix this security issue.'
        },
        filePath: filePath,
        startLine: extractedLine
      };
    });
  }
}
