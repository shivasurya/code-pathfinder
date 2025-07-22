import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { performSecurityAnalysisAsync } from '../analysis/security-analyzer';
import { SettingsManager } from '../settings/settings-manager';

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
        const changes: GitChangeInfo[] = [];

        // Construct the git diff command
        let command = 'git diff --unified=0';

        // Execute git command
        const output = await executeCommand(command, repoPath);
        
        // Parse the git diff output
        let currentFile: string | null = null;
        
        const lines = output.split('\n');
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            
            // Check for file header
            if (line.startsWith('diff --git')) {
                const match = line.match(/diff --git a\/(.*) b\/(.*)/);
                if (match && match[2]) {
                    currentFile = match[2];
                }
                continue;
            }
            
            // Check for hunk header
            if (line.startsWith('@@') && currentFile) {
                const match = line.match(/@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/);
                if (match) {
                    const startLine = parseInt(match[3], 10);
                    const lineCount = match[4] ? parseInt(match[4], 10) : 1;
                    
                    // Collect the changed lines content
                    let content = '';
                    let j = i + 1;
                    let collectedLines = 0;
                    
                    while (j < lines.length && collectedLines < lineCount) {
                        const nextLine = lines[j];
                        if (!nextLine.startsWith('-') && 
                            !nextLine.startsWith('diff --git') && 
                            !nextLine.startsWith('@@')) {
                            // Include the line if it's an addition or context line
                            if (nextLine.startsWith('+')) {
                                content += nextLine.substring(1) + '\n';
                                collectedLines++;
                            } else {
                                content += nextLine + '\n';
                                collectedLines++;
                            }
                        }
                        j++;
                    }
                    
                    changes.push({
                        filePath: path.join(repoPath, currentFile),
                        startLine,
                        lineCount,
                        content: content.trim()
                    });
                }
            }
        }
        
        return changes;
    } catch (error) {
        console.error('Error getting git changes:', error);
        return [];
    }
}

/**
 * Executes a shell command and returns the output
 * @param command Command to execute
 * @param cwd Current working directory
 * @returns Command output as string
 */
async function executeCommand(command: string, cwd: string): Promise<string> {
    return new Promise((resolve, reject) => {
        cp.exec(command, { cwd }, (error, stdout, stderr) => {
            if (error) {
                reject(error);
                return;
            }
            resolve(stdout);
        });
    });
}

/**
 * Registers the SecureFlow review command for git changes
 * @param context VSCode extension context
 * @param outputChannel Output channel for displaying results
 * @param settingsManager Settings manager for the extension
 */
export function registerSecureFlowReviewCommand(
    context: vscode.ExtensionContext, 
    outputChannel: vscode.OutputChannel,
    settingsManager: SettingsManager
): void {
    // Create status bar item
    const statusBarItem = vscode.window.createStatusBarItem(
        vscode.StatusBarAlignment.Right,
        100
    );
    statusBarItem.text = "$(shield) SecureFlow";
    statusBarItem.tooltip = "Scan git changes for security issues";
    statusBarItem.command = "secureflow.reviewChanges";
    statusBarItem.show();
    
    let resultsPanel: vscode.WebviewPanel | undefined;

    // Register command
    const reviewCommand = vscode.commands.registerCommand(
        "secureflow.reviewChanges",
        async (uri?: vscode.Uri) => {
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
                        title: "SecureFlow: Scanning git changes...",
                        cancellable: true
                    },
                    async (progress, token) => {
                        // Get the selected AI Model
                        const selectedModel = settingsManager.getSelectedAIModel();
                        
                        // Get git changes
                        const changes = await getGitChanges();
                        
                        if (changes.length === 0) {
                            updateWebview(resultsPanel!, 'No git changes found to scan.', []);
                            vscode.window.showInformationMessage('SecureFlow: No git changes found to scan.');
                            return;
                        }
                        
                        progress.report({ increment: 30, message: "Analyzing changes..." });
                        
                        let allIssues: Array<{issue: SecurityIssue, filePath: string, startLine: number}> = [];
                        
                        // Analyze each change
                        for (let i = 0; i < changes.length; i++) {
                            const change = changes[i];
                            const issues = await performSecurityAnalysisAsync(change.content, selectedModel, await settingsManager.getApiKey());
                            
                            const mappedIssues = issues.map((issue: SecurityIssue) => ({
                                issue,
                                filePath: change.filePath,
                                startLine: change.startLine
                            }));
                            
                            allIssues = [...allIssues, ...mappedIssues];
                            
                            progress.report({ 
                                increment: 60 / changes.length, 
                                message: `Analyzed ${i + 1}/${changes.length} chunks...` 
                            });
                        }
                        
                        // Update WebView with results
                        updateWebview(resultsPanel!, `Scan complete! Found ${allIssues.length} issues.`, allIssues);
                        
                        if (allIssues.length > 0) {
                            vscode.window.showWarningMessage(
                                `SecureFlow: Found ${allIssues.length} security ${allIssues.length === 1 ? 'issue' : 'issues'} in your code changes.`
                            );
                        }
                    }
                );
            } catch (error) {
                console.error('Error during security review:', error);
                vscode.window.showErrorMessage(`SecureFlow: Error during security review: ${error}`);
            }
        }
    );
    
    context.subscriptions.push(statusBarItem, reviewCommand);
}

function updateWebview(panel: vscode.WebviewPanel, summary: string, issues: Array<{issue: SecurityIssue, filePath: string, startLine: number}>) {
    const html = `<!DOCTYPE html>
    <html>
        <head>
            <style>
                body { font-family: Arial, sans-serif; padding: 20px; }
                .summary { margin-bottom: 20px; font-size: 1.2em; }
                .issue { 
                    background: #f3f3f3; 
                    padding: 15px; 
                    margin-bottom: 15px; 
                    border-radius: 5px;
                    border-left: 4px solid #cc0000;
                }
                .issue-title { font-weight: bold; color: #cc0000; }
                .issue-location { color: #666; margin: 5px 0; }
                .issue-severity { 
                    display: inline-block;
                    padding: 3px 8px;
                    border-radius: 3px;
                    background: #ff9999;
                    color: white;
                    font-size: 0.9em;
                }
            </style>
        </head>
        <body>
            <div class="summary">${summary}</div>
            ${issues.map((item, index) => `
                <div class="issue">
                    <div class="issue-title">${item.issue.title}</div>
                    <div class="issue-location">
                        ${path.basename(item.filePath)} (Line ${item.startLine})
                    </div>
                    <div class="issue-severity">${item.issue.severity}</div>
                    <p>${item.issue.description}</p>
                    <p><strong>Recommendation:</strong> ${item.issue.recommendation}</p>
                </div>
            `).join('')}
        </body>
    </html>`;

    panel.webview.html = html;
}

// Interface for git change information
export interface GitChangeInfo {
    filePath: string;
    startLine: number;
    lineCount: number;
    content: string;
}
