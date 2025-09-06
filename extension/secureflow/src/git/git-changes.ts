import * as vscode from 'vscode';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { performSecurityAnalysisAsync } from '../analysis/security-analyzer';
import { AnalyticsService } from '../services/analytics';
import { SettingsManager } from '../settings/settings-manager';
import { SentryService } from '../services/sentry-service';
import { ProfileStorageService } from '../services/profile-storage-service';
import { StoredProfile } from '../models/profile-store';
import { ScanStorageService } from '../services/scan-storage-service';
import { loadPrompt } from '@codepathfinder/secureflow-cli';
import { SecureFlowExplorer } from '../ui/secureflow-explorer';
// Shared, VS Code-agnostic git helpers (CommonJS) from local workspace package
import * as cliGit from '@codepathfinder/secureflow-cli/lib/git';

/**
 * TODO(CLI): This module mixes core git parsing with VS Code UI and services.
 * Extraction plan:
 * - Create a pure helper: getGitChangesAtRepo(repoPath: string, opts: { staged?: boolean }): Promise<GitChangeInfo[]>
 *   that replaces usage of vscode.workspace and works in Node.
 * - Keep Webview-related functions and command registration as EXTENSION-ONLY.
 * - CLI will reuse only the pure helper and call performSecurityAnalysisAsync directly.
 */

/**
 * Gets the git changes (hunks) for a specific file or all files in the workspace
 * @param filePath Optional path to a specific file
 * @returns Array of change information objects
 */
export async function getGitChanges(): Promise<GitChangeInfo[]> {
  try {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders || workspaceFolders.length === 0) {
      throw new Error('No workspace folder found');
    }

    const repoPath = workspaceFolders[0].uri.fsPath;
    const changes = await cliGit.getGitChangesAtRepo(repoPath, {
      staged: true,
      unstaged: true
    });
    return changes as GitChangeInfo[];
  } catch (error) {
    console.error('Error getting git changes:', error);
    return [];
  }
}

// Note: command execution and parsing now lives in `packages/secureflow-cli/lib/git`

/**
 * Registers the SecureFlow review command for git changes
 * @param context VSCode extension context
 * @param outputChannel Output channel for displaying results
 * @param settingsManager Settings manager for the extension
 */
// EXTENSION-ONLY: Command registration + Webview UI
export function registerSecureFlowReviewCommand(
  context: vscode.ExtensionContext,
  outputChannel: vscode.OutputChannel,
  settingsManager: SettingsManager
): void {
  // Initialize profile storage service
  const profileService = new ProfileStorageService(context);

  // Initialize scan storage service
  const scanService = new ScanStorageService(context);
  // Create status bar item
  const statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100
  );
  statusBarItem.text = '$(shield) SecureFlow';
  statusBarItem.tooltip = 'Scan git changes for security issues';
  statusBarItem.command = 'secureflow.reviewChanges';
  statusBarItem.show();

  let resultsPanel: vscode.WebviewPanel | undefined;

  // Register command with Sentry error handling
  const sentry = SentryService.getInstance();
  const reviewCommand = vscode.commands.registerCommand(
    'secureflow.reviewChanges',
    sentry.withErrorHandling(
      'secureflow.reviewChanges',
      async (uri?: vscode.Uri) => {
        // Track command usage
        const analytics = AnalyticsService.getInstance();
        analytics.trackEvent('Git Security Review Started', {
          review_type: 'git_changes'
        });

        try {
          // Create or show WebView panel
          if (!resultsPanel) {
            resultsPanel = vscode.window.createWebviewPanel(
              'secureflowResults',
              'SecureFlow Results',
              vscode.ViewColumn.Two,
              {
                enableScripts: true,
                retainContextWhenHidden: true
              }
            );

            resultsPanel.onDidDispose(() => {
              resultsPanel = undefined;
            });
          }

          // Show progress indicator
          await vscode.window.withProgress(
            {
              location: vscode.ProgressLocation.Notification,
              title: 'SecureFlow: Scanning git changes...',
              cancellable: true
            },
            async (progress, token) => {
              // Get the selected AI Model
              const selectedModel = settingsManager.getSelectedAIModel();

              // Get git changes
              const changes = await getGitChanges();

              if (changes.length === 0) {
                updateWebview(resultsPanel!, 0, new Date(), []);
                vscode.window.showInformationMessage(
                  'SecureFlow: No git changes found to scan.'
                );
                return;
              }

              // Get all stored profiles
              const allProfiles: StoredProfile[] =
                profileService.getAllProfiles();

              // Helper function to find the most relevant profile for a file path
              const findRelevantProfile = (
                filePath: string
              ): StoredProfile | undefined => {
                if (allProfiles.length === 0) {
                  return undefined;
                }

                // Check if file is in a workspace folder
                const workspaceFolder = vscode.workspace.getWorkspaceFolder(
                  vscode.Uri.file(filePath)
                );
                if (!workspaceFolder) {
                  return undefined;
                }

                // Find profiles within the workspace
                const relevantProfiles = allProfiles.filter((profile) => {
                  const profileUri = vscode.Uri.parse(
                    profile.workspaceFolderUri
                  );
                  const profileWorkspace =
                    vscode.workspace.getWorkspaceFolder(profileUri);
                  return (
                    profileWorkspace &&
                    profileWorkspace.uri.fsPath === workspaceFolder.uri.fsPath
                  );
                });

                if (relevantProfiles.length === 0) {
                  return undefined;
                }

                // Get relative path within workspace
                const relativePath = path.relative(
                  workspaceFolder.uri.fsPath,
                  filePath
                );

                // Find profile where file path falls under profile path
                return relevantProfiles.find((profile) => {
                  const normalizedProfilePath =
                    profile.path === '/' ? '' : profile.path;
                  const profilePathParts = normalizedProfilePath
                    .split('/')
                    .filter((p) => p);
                  const filePathParts = relativePath
                    .split(path.sep)
                    .filter((p) => p);

                  // Check if file path starts with profile path parts
                  return profilePathParts.every(
                    (part, index) => filePathParts[index] === part
                  );
                });
              };

              // Group changes by file
              const changesByFile: {
                [filePath: string]: { diffs: string[]; starts: number[] };
              } = {};
              for (const change of changes) {
                if (!changesByFile[change.filePath]) {
                  changesByFile[change.filePath] = { diffs: [], starts: [] };
                }
                changesByFile[change.filePath].diffs.push(change.content);
                changesByFile[change.filePath].starts.push(change.startLine);
              }

              let allIssues: Array<{
                issue: SecurityIssue;
                filePath: string;
                startLine: number;
              }> = [];
              const workspaceFolders = vscode.workspace.workspaceFolders || [];
              let consolidatedReviewContent = await loadPrompt(
                'common/review-changes.txt'
              );
              const fileMetadata: Array<{
                filePath: string;
                startLine: number;
              }> = [];

              // Collect all profiles used
              const usedProfiles = new Set<StoredProfile>();

              // Build consolidated review content for all files
              for (const [filePath, { diffs, starts }] of Object.entries(
                changesByFile
              )) {
                const profile = findRelevantProfile(filePath);
                if (profile) {
                  usedProfiles.add(profile);
                }

                // Store file metadata for later mapping
                fileMetadata.push({ filePath, startLine: starts[0] || 1 });

                consolidatedReviewContent += `\n=== FILE: ${filePath} ===\n`;

                if (profile) {
                  consolidatedReviewContent += `/*\n[SECUREFLOW PROFILE CONTEXT]\nName: ${profile.name}\nCategory: ${profile.category}\nPath: ${profile.path}\nLanguages: ${profile.languages.join(', ')}\nFrameworks: ${profile.frameworks.join(', ')}\nBuild Tools: ${profile.buildTools.join(', ')}\nEvidence: ${profile.evidence.join('; ')}\n*/\n`;
                }

                // Attach all diffs for the file
                for (const diff of diffs) {
                  consolidatedReviewContent += `<DIFF>\n${diff}\n</DIFF>\n`;
                }

                // Attach full file content
                let fullFileContent = '';
                try {
                  fullFileContent = (
                    await vscode.workspace.openTextDocument(filePath)
                  ).getText();
                } catch (e) {
                  fullFileContent = '// Unable to load full file content';
                }
                consolidatedReviewContent += `\n/* [FULL FILE CONTENT] */\n${fullFileContent}\n`;
                consolidatedReviewContent += `\n=== END FILE: ${filePath} ===\n\n`;

                progress.report({
                  increment: 30 / Object.keys(changesByFile).length,
                  message: `Preparing analysis...`
                });
              }

              // Add summary of all profiles at the beginning
              let profileSummary = '';
              if (usedProfiles.size > 0) {
                profileSummary = `/*\n[SECUREFLOW ANALYSIS CONTEXT]\nAnalyzing ${Object.keys(changesByFile).length} files across ${usedProfiles.size} application profile(s):\n`;
                usedProfiles.forEach((profile) => {
                  profileSummary += `- ${profile.name} (${profile.category}) at ${profile.path}\n`;
                });
                profileSummary += `*/\n\n`;
              }

              const finalReviewContent =
                profileSummary + consolidatedReviewContent;
              outputChannel.appendLine(finalReviewContent);

              progress.report({
                increment: 30,
                message: `Performing security analysis...`
              });

              // Make single API call with all consolidated content
              const issues = await performSecurityAnalysisAsync(
                finalReviewContent,
                selectedModel,
                await settingsManager.getApiKey(),
                undefined,
                context,
                true
              );

              // Map issues back to files (best effort - use first file if unable to determine)
              const mappedIssues = issues.map(
                (issue: SecurityIssue, index: number) => {
                  // Try to find the most relevant file based on issue content or use first file as fallback
                  let relevantFile = fileMetadata[0]; // fallback to first file

                  // Simple heuristic: if issue mentions a filename, use that
                  for (const fileMeta of fileMetadata) {
                    const fileName = path.basename(fileMeta.filePath);
                    if (
                      issue.description.includes(fileName) ||
                      issue.title.includes(fileName)
                    ) {
                      relevantFile = fileMeta;
                      break;
                    }
                  }

                  return {
                    issue,
                    filePath: relevantFile.filePath,
                    startLine: relevantFile.startLine
                  };
                }
              );

              allIssues = mappedIssues;

              // Save the scan result
              const scanSummary = `Scan complete! Found ${allIssues.length} issues.`;
              const savedScan = await scanService.saveScan(
                allIssues,
                scanSummary,
                finalReviewContent,
                Object.keys(changesByFile).length,
                selectedModel
              );

              // Update WebView with results including scan number
              const webviewSummary = `Scan #${savedScan.scanNumber} complete! Found ${allIssues.length} issues. (${savedScan.timestampFormatted})`;
              updateWebview(
                resultsPanel!,
                savedScan.scanNumber,
                new Date(savedScan.timestamp),
                allIssues
              );

              // Refresh the scan list in the main webview
              SecureFlowExplorer.refreshScanList();

              if (allIssues.length > 0) {
                vscode.window
                  .showWarningMessage(
                    `SecureFlow Scan #${savedScan.scanNumber}: Found ${allIssues.length} security issues in your git changes.`,
                    'View Results'
                  )
                  .then((selection) => {
                    if (selection === 'View Results' && resultsPanel) {
                      resultsPanel.reveal(vscode.ViewColumn.Two);
                    }
                  });
              } else {
                vscode.window.showInformationMessage(
                  `SecureFlow Scan #${savedScan.scanNumber}: No security issues found in your git changes.`
                );
              }
            }
          );
        } catch (error) {
          console.error('Error during security review:', error);
          vscode.window.showErrorMessage(
            `SecureFlow: Error during security review: ${error}`
          );

          // Capture the error with Sentry (additional error handling within the wrapped function)
          try {
            sentry.captureException(error as Error, {
              context: 'git_review_command',
              component: 'git_changes',
              uri: uri?.toString()
            });
          } catch (sentryError) {
            console.error('Failed to capture git review error:', sentryError);
          }
        }
      }
    )
  );
  context.subscriptions.push(statusBarItem, reviewCommand);
}

import { formatTimestamp } from '../utils/format-timestamp';

// Basic, safe markdown renderer for our webview needs.
// Supports fenced code blocks (```lang) and inline code (`code`).
// Everything is HTML-escaped by default to avoid XSS.
function escapeHtml(input: string): string {
  return input
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderMarkdownBasic(md: string): string {
  if (!md) {
    return '';
  }

  // Normalize line endings
  let text = md.replace(/\r\n/g, '\n');

  // Escape HTML first
  text = escapeHtml(text);

  // Fenced code blocks: ```lang\n...\n```
  // Use a replacer that preserves newlines and optional language
  const fencedBlockRe = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g;
  text = text.replace(fencedBlockRe, (_m, lang, code) => {
    const language = lang ? ` language-${lang}` : '';
    return `<pre class="code-block"><code class="${language}">${code}</code></pre>`;
  });

  // Inline code: `code` (avoid matching inside already converted blocks)
  // Since we've already converted fenced blocks, a simple regex works here
  const inlineCodeRe = /`([^`]+)`/g;
  text = text.replace(
    inlineCodeRe,
    (_m, code) => `<code class="inline-code">${code}</code>`
  );

  // Basic paragraph/line breaks: convert double newlines to paragraphs, single to <br>
  const paragraphs = text
    .split(/\n\n+/)
    .map((para) => para.replace(/\n/g, '<br/>'));
  return paragraphs.map((p) => `<p>${p}</p>`).join('');
}

function generateGitChangesHtml(
  scanNumber: number,
  timestamp: Date,
  issues: Array<{ issue: SecurityIssue; filePath: string; startLine: number }>
): string {
  // Helper to get relative path from workspace root
  function getRelativePath(filePath: string): string {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (workspaceFolders && workspaceFolders.length > 0) {
      const root = workspaceFolders[0].uri.fsPath;
      return path.relative(root, filePath);
    }
    return filePath;
  }

  const issuesHtml = issues
    .map(
      (item) => `
        <div class="issue">
            <div class="issue-header">
                <span class="severity severity-${item.issue.severity.toLowerCase()}">${item.issue.severity}</span>
                <h3 class="issue-title">${item.issue.title}</h3>
            </div>
            <div class="issue-meta">
                <span class="file">${getRelativePath(item.filePath)}:${item.startLine}</span>
            </div>
            <div class="description">${renderMarkdownBasic(item.issue.description)}</div>
            <div class="recommendation">
                <strong>Recommendation:</strong>
                ${renderMarkdownBasic(item.issue.recommendation)}
            </div>
        </div>
    `
    )
    .join('');

  return `<!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Git Changes Security Review</title>
            <style>
                body { 
                    font-family: var(--vscode-font-family); 
                    padding: 20px; 
                    color: var(--vscode-foreground);
                    background: var(--vscode-editor-background);
                }
                .header { 
                    border-bottom: 1px solid var(--vscode-panel-border); 
                    padding-bottom: 15px; 
                    margin-bottom: 20px; 
                }
                .scan-info { 
                    display: flex; 
                    gap: 20px; 
                    margin-bottom: 10px; 
                    flex-wrap: wrap;
                }
                .info-item { 
                    display: flex; 
                    flex-direction: column; 
                }
                .info-label { 
                    font-size: 12px; 
                    color: var(--vscode-descriptionForeground); 
                }
                .info-value { 
                    font-weight: bold; 
                }
                .issue { 
                    border: 1px solid var(--vscode-panel-border); 
                    border-radius: 8px; 
                    padding: 20px; 
                    margin-bottom: 20px; 
                    background: var(--vscode-editor-background);
                    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
                    transition: box-shadow 0.2s ease;
                }
                .issue:hover {
                    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
                }
                .issue-header {
                    display: flex;
                    align-items: center;
                    gap: 12px;
                    margin-bottom: 12px;
                }
                .issue-title { 
                    margin: 0;
                    font-size: 16px;
                    font-weight: 600;
                    color: var(--vscode-foreground);
                    line-height: 1.4;
                }
                .issue-meta { 
                    display: flex; 
                    gap: 16px; 
                    margin-bottom: 16px; 
                    font-size: 13px;
                    align-items: center;
                }
                .severity { 
                    padding: 3px 8px; 
                    border-radius: 4px; 
                    font-weight: 600;
                    font-size: 12px;
                    text-transform: uppercase;
                    letter-spacing: 0.5px;
                }
                .severity-critical { 
                    background: rgba(220, 38, 38, 0.2); 
                    color: #fca5a5; 
                    border: 1px solid rgba(220, 38, 38, 0.3);
                }
                .severity-high { 
                    background: rgba(251, 146, 60, 0.2); 
                    color: #fdba74; 
                    border: 1px solid rgba(251, 146, 60, 0.3);
                }
                .severity-medium { 
                    background: rgba(245, 158, 11, 0.2); 
                    color: #fbbf24; 
                    border: 1px solid rgba(245, 158, 11, 0.3);
                }
                .severity-low { 
                    background: rgba(34, 197, 94, 0.2); 
                    color: #86efac; 
                    border: 1px solid rgba(34, 197, 94, 0.3);
                }
                .file { 
                    font-family: var(--vscode-editor-font-family); 
                    background: rgba(255, 255, 255, 0.05); 
                    padding: 4px 8px; 
                    border-radius: 4px; 
                    font-size: 12px;
                    color: var(--vscode-textLink-foreground);
                    border: 1px solid rgba(255, 255, 255, 0.1);
                }
                .description { 
                    margin: 16px 0; 
                    line-height: 1.6;
                    color: var(--vscode-foreground);
                    font-size: 14px;
                }
                .recommendation {
                    background: rgba(0, 128, 255, 0.04);
                    border-left: 3px solid #007acc;
                    padding: 8px 12px;
                    border-radius: 4px;
                    font-size: 13px;
                    color: rgb(244, 250, 253);
                }
                /* Code styling */
                pre.code-block {
                    background: var(--vscode-editor-background);
                    border: 1px solid var(--vscode-panel-border);
                    border-radius: 6px;
                    padding: 12px;
                    overflow: auto;
                    font-family: var(--vscode-editor-font-family);
                    font-size: 12px;
                    line-height: 1.5;
                    margin: 10px 0;
                }
                code.inline-code {
                    background: rgba(255, 255, 255, 0.06);
                    border: 1px solid var(--vscode-panel-border);
                    border-radius: 4px;
                    padding: 1px 4px;
                    font-family: var(--vscode-editor-font-family);
                    font-size: 12px;
                }
            </style>
        </head>
        <body>
            <div class="header">
                <h2>Git Changes Security Review</h2>
                <div class="scan-info">
                    <div class="info-item">
                        <span class="info-label">Scan #</span>
                        <span class="info-value">${scanNumber}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Completed</span>
                        <span class="info-value">${formatTimestamp(timestamp)}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Issues Found</span>
                        <span class="info-value">${issues.length}</span>
                    </div>
                </div>
            </div>
            ${issuesHtml}
        </body>
        </html>`;
}

function updateWebview(
  panel: vscode.WebviewPanel,
  scanNumber: number,
  timestamp: Date,
  issues: Array<{ issue: SecurityIssue; filePath: string; startLine: number }>
) {
  panel.webview.html = generateGitChangesHtml(scanNumber, timestamp, issues);
}

// Interface for git change information
export interface GitChangeInfo {
  filePath: string;
  startLine: number;
  lineCount: number;
  content: string;
}
