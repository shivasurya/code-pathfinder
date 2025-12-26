import * as vscode from 'vscode';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { AIModel, SettingsManager } from '../settings/settings-manager';
import { analyzeSecurityWithAI } from './security-analyzer-ai';
import { AnalyticsService } from '../services/analytics';
import { SentryService } from '../services/sentry-service';
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
  const fencedBlockRe = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g;
  text = text.replace(fencedBlockRe, (_m, lang, code) => {
    const language = lang ? ` language-${lang}` : '';
    return `<pre class="code-block"><code class="${language}">${code}</code></pre>`;
  });

  // Inline code: `code`
  const inlineCodeRe = /`([^`]+)`/g;
  text = text.replace(
    inlineCodeRe,
    (_m, code) => `<code class="inline-code">${code}</code>`
  );

  // Paragraph and line breaks
  const paragraphs = text
    .split(/\n\n+/)
    .map((para) => para.replace(/\n/g, '<br/>'));
  return paragraphs.map((p) => `<p>${p}</p>`).join('');
}

/**
 * TODO(VS Code): This function is VS Code UI specific and MUST remain extension-only.
 * TODO(CLI): Extract a UI-free analysis helper for reuse in CLI.
 * - performSecurityAnalysisAsync is almost pure except for the optional vscode.ExtensionContext
 *   parameter and analytics/sentry in callers. For CLI, plan to introduce a wrapper:
 *   runSecurityAnalysis(code, aiModel, apiKey, filePath, { isGitDiff }): Promise<SecurityIssue[]>
 * - The following functions are VS Code UI specific and MUST remain extension-only:
 *   - generateSelectionAnalysisHtml()
 *   - updateSelectionWebview()
 *   - registerAnalyzeSelectionCommand()
 */
/**
 * Performs security analysis on the given code snippet asynchronously,
 * utilizing both pattern-based detection and AI-based analysis if an API key is provided
 * @param code The code to analyze
 * @param aiModel The AI Model to use
 * @param apiKey API key for the AI Model
 * @param filePath Optional file path for profile context
 * @param context Optional VS Code extension context for profile service
 * @returns Promise with array of security issues found
 */
export async function performSecurityAnalysisAsync(
  code: string,
  aiModel: string,
  apiKey?: string,
  filePath?: string,
  context?: vscode.ExtensionContext,
  isGitDiff?: boolean
): Promise<SecurityIssue[]> {
  // If no API key is provided, just return the pattern-based results
  if (!apiKey) {
    return [];
  }

  try {
    // Run the AI-based analysis
    const aiIssues = await analyzeSecurityWithAI(
      code,
      aiModel,
      apiKey,
      filePath,
      context,
      isGitDiff
    );

    // Merge the results, removing any duplicates
    const allIssues = [];

    // Add AI issues that don't overlap with pattern issues
    for (const aiIssue of aiIssues) {
      allIssues.push(aiIssue);
    }

    return allIssues;
  } catch (error) {
    console.error('Error in AI-based analysis:', error);
    // If AI analysis fails, return just the pattern-based results
    return [];
  }
}

/**
 * Generates HTML for displaying security analysis results
 * @param scanNumber The scan number for this analysis
 * @param timestamp The timestamp of the analysis
 * @param issues Array of security issues found
 * @param filePath The file path where the selection was analyzed
 * @param startLine The starting line of the selection
 * @returns HTML string for the webview
 */
// EXTENSION-ONLY: Webview HTML renderer
function generateSelectionAnalysisHtml(
  scanNumber: number,
  timestamp: Date,
  issues: SecurityIssue[],
  filePath: string,
  startLine: number,
  aiModel: string
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
      (issue) => `
        <div class="issue">
            <div class="issue-header">
                <span class="severity severity-${issue.severity.toLowerCase()}">${issue.severity}</span>
                <h3 class="issue-title">${issue.title}</h3>
            </div>
            <div class="issue-meta">
                <span class="file">${getRelativePath(filePath)}:${startLine}</span>
            </div>
            <div class="description">${renderMarkdownBasic(issue.description)}</div>
            <div class="recommendation">
                <strong>Recommendation:</strong>
                ${renderMarkdownBasic(issue.recommendation)}
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
            <title>SecureFlow Selection Analysis</title>
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
                    padding: 2px 6px; 
                    border-radius: 3px; 
                    font-size: 12px;
                    color: var(--vscode-textLink-foreground);
                }
                .description { 
                    line-height: 1.6; 
                    margin-bottom: 16px;
                    color: var(--vscode-foreground);
                }
                .recommendation { 
                    background: rgba(0, 122, 204, 0.1); 
                    border-left: 3px solid var(--vscode-textLink-foreground); 
                    padding: 12px 16px; 
                    border-radius: 0 4px 4px 0;
                    font-size: 14px;
                    line-height: 1.5;
                }
                /* Code styling for markdown */
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
                .no-issues {
                    text-align: center;
                    padding: 40px 20px;
                    color: var(--vscode-descriptionForeground);
                }
                .no-issues h3 {
                    color: var(--vscode-foreground);
                    margin-bottom: 10px;
                }
            </style>
        </head>
        <body>
            <div class="header">
                <h1>SecureFlow Selection Analysis</h1>
                <div class="scan-info">
                    <div class="info-item">
                        <span class="info-label">Scan #</span>
                        <span class="info-value">${scanNumber}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Timestamp</span>
                        <span class="info-value">${formatTimestamp(timestamp)}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Issues Found</span>
                        <span class="info-value">${issues.length}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">File</span>
                        <span class="info-value">${getRelativePath(filePath)}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">AI Model</span>
                        <span class="info-value">${aiModel}</span>
                    </div>
                </div>
            </div>
            ${
              issues.length > 0
                ? issuesHtml
                : `
                <div class="no-issues">
                    <h3>✅ No Security Issues Found</h3>
                    <p>The selected code appears to be secure based on our analysis.</p>
                </div>
            `
            }
        </body>
        </html>`;
}

/**
 * Updates the webview with security analysis results
 * @param panel The webview panel to update
 * @param scanNumber The scan number
 * @param timestamp The analysis timestamp
 * @param issues Array of security issues
 * @param filePath The file path
 * @param startLine The starting line
 */
// EXTENSION-ONLY: Webview panel updater
function updateSelectionWebview(
  panel: vscode.WebviewPanel,
  scanNumber: number,
  timestamp: Date,
  issues: SecurityIssue[],
  filePath: string,
  startLine: number,
  aiModel: string
) {
  panel.webview.html = generateSelectionAnalysisHtml(
    scanNumber,
    timestamp,
    issues,
    filePath,
    startLine,
    aiModel
  );
}

/**
 * Registers the analyze selection command for VS Code
 * @param outputChannel The output channel for logging
 * @param settingsManager The settings manager instance
 * @param context The VS Code extension context for profile services
 * @returns Disposable for the registered command
 */
// EXTENSION-ONLY: Command registration for VS Code
export function registerAnalyzeSelectionCommand(
  outputChannel: vscode.OutputChannel,
  settingsManager: SettingsManager,
  context: vscode.ExtensionContext
): vscode.Disposable {
  let resultsPanel: vscode.WebviewPanel | undefined;

  const sentry = SentryService.getInstance();
  return vscode.commands.registerCommand(
    'secureflow.analyzeSelection',
    sentry.withErrorHandling('secureflow.analyzeSelection', async () => {
      // Track command usage
      const analytics = AnalyticsService.getInstance();
      analytics.trackEvent('Code Analysis Started', {
        analysis_type: 'selected_text'
      });

      // Get the active text editor
      const editor = vscode.window.activeTextEditor;
      if (!editor) {
        vscode.window.showErrorMessage('No active editor found');
        return;
      }

      // Get the selected text
      const selection = editor.selection;
      const selectedText = editor.document.getText(selection);

      if (!selectedText) {
        vscode.window.showInformationMessage(
          'No text selected for security analysis'
        );
        return;
      }

      // Create or show WebView panel
      if (!resultsPanel) {
        resultsPanel = vscode.window.createWebviewPanel(
          'secureflowSelectionResults',
          'SecureFlow Selection Analysis',
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
          title: 'SecureFlow: Analyzing selected code...',
          cancellable: true
        },
        async (progress, token) => {
          // Get file information first
          const filePath = editor.document.fileName;
          const startLine = selection.start.line + 1; // Convert to 1-based line numbers
          const timestamp = new Date();
          const scanNumber = Math.floor(Date.now() / 1000); // Simple scan number based on timestamp

          // Get the selected AI Model and API key
          const aiModel = settingsManager.getSelectedAIModel();
          let securityIssues: SecurityIssue[] = [];

          try {
            const apiKey = await settingsManager.getApiKey();
            if (apiKey) {
              securityIssues = await performSecurityAnalysisAsync(
                selectedText,
                aiModel,
                apiKey,
                filePath,
                context,
                false
              );
            }
          } catch (error) {
            console.error('Error with AI analysis:', error);
            try {
              const sentry = SentryService.getInstance();
              sentry.captureException(error as Error, {
                context: 'ai_analysis_error',
                component: 'analyze_selection_command',
                ai_model: aiModel,
                selected_text_length: selectedText.length
              });
            } catch (sentryError) {
              console.error(
                'Failed to capture AI analysis error:',
                sentryError
              );
            }
          }

          // Update WebView with results
          if (resultsPanel) {
            updateSelectionWebview(
              resultsPanel,
              scanNumber,
              timestamp,
              securityIssues,
              filePath,
              startLine,
              aiModel
            );
            resultsPanel.reveal(vscode.ViewColumn.Two);
          }

          // Show appropriate message
          if (securityIssues.length === 0) {
            vscode.window
              .showInformationMessage(
                `SecureFlow Analysis #${scanNumber}: No security issues found in the selected code.`,
                'View Results'
              )
              .then((selection) => {
                if (selection === 'View Results' && resultsPanel) {
                  resultsPanel.reveal(vscode.ViewColumn.Two);
                }
              });
          } else {
            vscode.window
              .showWarningMessage(
                `SecureFlow Analysis #${scanNumber}: Found ${securityIssues.length} potential security issues.`,
                'View Results'
              )
              .then((selection) => {
                if (selection === 'View Results' && resultsPanel) {
                  resultsPanel.reveal(vscode.ViewColumn.Two);
                }
              });
          }
        }
      );
    })
  );
}
