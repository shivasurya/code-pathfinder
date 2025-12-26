import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
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
  private logFile: string | null = null;
  private logStream: fs.WriteStream | null = null;

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.settingsManager = new SettingsManager(context);
    this.scanStorageService = new ScanStorageService(context);
  }

  /**
   * Initialize log file for this scan
   */
  private async initLogFile(profileName: string): Promise<string> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const logFileName = `scan-${profileName.replace(/[^a-z0-9]/gi, '_')}-${timestamp}.log`;
    const logsDir = path.join(this.context.globalStorageUri.fsPath, 'scan-logs');

    // Ensure logs directory exists
    await vscode.workspace.fs.createDirectory(vscode.Uri.file(logsDir));

    this.logFile = path.join(logsDir, logFileName);
    this.logStream = fs.createWriteStream(this.logFile, { flags: 'a' });

    this.log(`========================================`);
    this.log(`SecureFlow Scan Log`);
    this.log(`Profile: ${profileName}`);
    this.log(`Started: ${new Date().toISOString()}`);
    this.log(`========================================\n`);

    return this.logFile;
  }

  /**
   * Write log message to file and console
   */
  private log(message: string, data?: any) {
    const timestamp = new Date().toISOString();
    const logLine = data
      ? `[${timestamp}] ${message} ${JSON.stringify(data, null, 2)}`
      : `[${timestamp}] ${message}`;

    // Write to file
    if (this.logStream) {
      this.logStream.write(logLine + '\n');
    }

    // Also log to console
    if (data) {
      console.log(message, data);
    } else {
      console.log(message);
    }
  }

  /**
   * Close log file
   */
  private closeLogFile() {
    if (this.logStream) {
      this.log(`\n========================================`);
      this.log(`Scan completed: ${new Date().toISOString()}`);
      this.log(`========================================`);
      this.logStream.end();
      this.logStream = null;
    }
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
    // Initialize log file
    const logFilePath = await this.initLogFile(profile.name);
    this.log(`📝 Log file created: ${logFilePath}`);

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
      this.log('ProfileScanService: Starting scan', {
        profile: profile.name,
        profileId: profile.id,
        path: projectPath,
        model: model,
        workspaceFolder: workspaceFolder.uri.fsPath
      });

      // Create temporary output file for JSON results
      const outputFile = path.join(
        this.context.globalStorageUri.fsPath,
        `scan-${profile.id}-${Date.now()}.json`
      );

      this.log('Output file will be:', outputFile);

      // Ensure global storage directory exists
      await vscode.workspace.fs.createDirectory(this.context.globalStorageUri);

      // Infer provider from model name
      // OpenRouter models contain "/" (e.g., "anthropic/claude-3-5-sonnet")
      let provider: string | undefined = undefined;
      if (/^[a-z0-9-]+\/[a-z0-9-]+/i.test(model)) {
        provider = 'openrouter';
        this.log('Detected OpenRouter model format, setting provider to openrouter');
      }

      // Initialize CLI scanner with config passed directly
      const scanner = new CLIScanner.CLIFullScanCommand({
        selectedModel: model,
        outputFormat: 'json',
        outputFile: outputFile,
        maxIterations: 20, // Maximum iterations for thorough analysis
        config: {
          apiKey: apiKey,
          model: model,
          provider: provider,
          analytics: {
            enabled: false // Disable analytics for extension scans
          }
        }
      });

      this.log('CLI Scanner initialized with config', {
        selectedModel: model,
        outputFormat: 'json',
        maxIterations: 20,
        provider: provider || 'auto (inferred from model)',
        apiKeyMasked: apiKey.substring(0, 8) + '...'
      });

      progressCallback?.('Analyzing project structure...');

      // Capture console output
      const originalConsoleLog = console.log;
      const originalConsoleError = console.error;
      const originalConsoleWarn = console.warn;

      try {
        // Execute scan with stdout/stderr capture
        this.log('🚀 Executing CLI scanner...', {
          projectPath,
          model,
          outputFile
        });

        this.log('\n========== CLI SCANNER OUTPUT START ==========');

        // Override console methods to capture all output
        console.log = (...args: any[]) => {
          const timestamp = new Date().toISOString();
          const message = args.map(arg =>
            typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
          ).join(' ');

          // Write to log file directly
          if (this.logStream) {
            this.logStream.write(`[${timestamp}] [STDOUT] ${message}\n`);
          }

          // Also output to original console
          originalConsoleLog(...args);
        };

        console.error = (...args: any[]) => {
          const timestamp = new Date().toISOString();
          const message = args.map(arg =>
            typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
          ).join(' ');

          // Write to log file directly
          if (this.logStream) {
            this.logStream.write(`[${timestamp}] [STDERR] ${message}\n`);
          }

          // Also output to original console
          originalConsoleError(...args);
        };

        console.warn = (...args: any[]) => {
          const timestamp = new Date().toISOString();
          const message = args.map(arg =>
            typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
          ).join(' ');

          // Write to log file directly
          if (this.logStream) {
            this.logStream.write(`[${timestamp}] [WARN] ${message}\n`);
          }

          // Also output to original console
          originalConsoleWarn(...args);
        };

        await scanner.execute(projectPath);

        this.log('========== CLI SCANNER OUTPUT END ==========\n');
        this.log('✅ CLI scanner completed, reading results from:', outputFile);
        progressCallback?.('Scan complete, processing results...');

        // Read results
        const resultsUri = vscode.Uri.file(outputFile);

        // Check if file exists
        try {
          await vscode.workspace.fs.stat(resultsUri);
          this.log('✓ Results file exists');
        } catch (error) {
          this.log('❌ Results file not found!', { outputFile, error });
          throw new Error(`Scan results file not found: ${outputFile}`);
        }

        const resultsData = await vscode.workspace.fs.readFile(resultsUri);
        const resultsText = Buffer.from(resultsData).toString('utf8');

        this.log('📄 Raw JSON length:', resultsText.length);
        this.log('📄 First 500 chars:', resultsText.substring(0, 500));

        const scanResults = JSON.parse(resultsText);

        this.log('📊 Parsed scan results', {
          totalIssues: scanResults.issues?.length || 0,
          filesAnalyzed: scanResults.filesAnalyzed,
          iterations: scanResults.iterations,
          summary: scanResults.summary
        });

        // Map CLI results to our format
        const mappedIssues = this.mapCliResultsToIssues(scanResults, workspaceFolder.uri.fsPath, profile.path);

        this.log('🔄 Mapped issues', {
          count: mappedIssues.length,
          firstIssue: mappedIssues[0],
          severityBreakdown: {
            critical: mappedIssues.filter(i => i.issue.severity === 'Critical').length,
            high: mappedIssues.filter(i => i.issue.severity === 'High').length,
            medium: mappedIssues.filter(i => i.issue.severity === 'Medium').length,
            low: mappedIssues.filter(i => i.issue.severity === 'Low').length
          }
        });

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

        this.log('💾 Scan saved to storage', {
          scanNumber: savedScan.scanNumber,
          profileId: profile.id,
          issuesCount: savedScan.issues.length,
          timestamp: savedScan.timestamp
        });

        progressCallback?.(`✓ Scan #${savedScan.scanNumber} saved with ${mappedIssues.length} findings`);

        // Clean up temporary JSON file
        try {
          await vscode.workspace.fs.delete(resultsUri);
          this.log('🗑️ Temporary JSON file deleted');
        } catch (error) {
          this.log('⚠️ Failed to delete temporary scan file:', error);
        }

        this.log(`\n✅ SCAN COMPLETE - Scan #${savedScan.scanNumber}`);
        this.log(`📝 Full logs saved to: ${logFilePath}`);

        return savedScan;
      } finally {
        // Restore console methods
        console.log = originalConsoleLog;
        console.error = originalConsoleError;
        console.warn = originalConsoleWarn;

        this.log('🔐 Console methods restored');
      }
    } catch (error: any) {
      this.log('❌ SCAN FAILED', {
        error: error.message,
        stack: error.stack
      });
      progressCallback?.(`✗ Scan failed: ${error.message || error}`);
      throw error;
    } finally {
      // Always close log file
      this.closeLogFile();
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
