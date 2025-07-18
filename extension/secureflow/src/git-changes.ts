import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';
import { SecurityIssue } from './models/security-issue';
import { performSecurityAnalysis } from './security-analyzer';
import { SettingsManager } from './settings-manager';

/**
 * Gets the git changes (hunks) for a specific file or all files in the workspace
 * @param filePath Optional path to a specific file
 * @returns Array of change information objects
 */
export async function getGitChanges(filePath?: string): Promise<GitChangeInfo[]> {
    try {
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders || workspaceFolders.length === 0) {
            throw new Error('No workspace folder found');
        }

        const repoPath = workspaceFolders[0].uri.fsPath;
        const changes: GitChangeInfo[] = [];

        // Construct the git diff command
        let command = 'git diff --unified=0';
        if (filePath) {
            const relativePath = path.relative(repoPath, filePath);
            command += ` -- "${relativePath}"`;
        }

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
    
    // Register command
    const reviewCommand = vscode.commands.registerCommand(
        "secureflow.reviewChanges",
        async (uri?: vscode.Uri) => {
            // Get the file path from the URI if provided (from SCM view)
            let currentFilePath: string | undefined;
            
            if (uri && uri.fsPath) {
                // If command was triggered from SCM view with a specific file
                currentFilePath = uri.fsPath;
            } else {
                // Otherwise use the active editor file
                const editor = vscode.window.activeTextEditor;
                currentFilePath = editor?.document.uri.fsPath;
            }
            
            try {
                // Show progress indicator
                await vscode.window.withProgress(
                    {
                        location: vscode.ProgressLocation.Notification,
                        title: "SecureFlow: Scanning git changes...",
                        cancellable: true
                    },
                    async (progress, token) => {
                        // Clear and show output channel
                        outputChannel.clear();
                        outputChannel.show(true);
                        outputChannel.appendLine('🔍 SecureFlow: Scanning git changes for security issues...\n');
                        
                        // Get the selected AI Model
                        const selectedModel = settingsManager.getSelectedAIModel();
                        outputChannel.appendLine(`🤖 Using AI Model: ${selectedModel}`);
                        
                        // Report progress
                        progress.report({ increment: 0 });
                        
                        // Get git changes
                        outputChannel.appendLine('⏳ Collecting git changes...');
                        const changes = await getGitChanges(currentFilePath);
                        
                        if (changes.length === 0) {
                            outputChannel.appendLine('\n⚠️ No git changes found to scan.');
                            vscode.window.showInformationMessage('SecureFlow: No git changes found to scan.');
                            return;
                        }
                        
                        // Report progress
                        progress.report({ increment: 30, message: "Analyzing changes..." });
                        
                        // Print file and change information
                        outputChannel.appendLine(`\n📄 Found ${changes.length} changed chunks in ${new Set(changes.map(c => c.filePath)).size} files\n`);
                        
                        let allIssues: Array<{issue: SecurityIssue, filePath: string, startLine: number}> = [];
                        
                        // Analyze each change
                        for (let i = 0; i < changes.length; i++) {
                            const change = changes[i];
                            outputChannel.appendLine(`File: ${path.basename(change.filePath)}`);
                            outputChannel.appendLine(`Lines: ${change.startLine}-${change.startLine + change.lineCount - 1}`);
                            outputChannel.appendLine(`Changes:\n${change.content}\n`);
                            
                            // Analyze the change content with the selected AI Model
                            const issues = performSecurityAnalysis(change.content, selectedModel);
                            
                            // Map issues to include file path and line number
                            const mappedIssues = issues.map((issue: SecurityIssue) => ({
                                issue,
                                filePath: change.filePath,
                                startLine: change.startLine
                            }));
                            
                            allIssues = [...allIssues, ...mappedIssues];
                            
                            // Update progress
                            progress.report({ 
                                increment: 60 / changes.length, 
                                message: `Analyzed ${i + 1}/${changes.length} chunks...` 
                            });
                        }
                        
                        // Finalize progress
                        progress.report({ increment: 10, message: "Finalizing scan..." });
                        await new Promise(resolve => setTimeout(resolve, 500));
                        
                        // Display results
                        outputChannel.appendLine('\n✅ Security scan complete!\n');
                        
                        if (allIssues.length === 0) {
                            outputChannel.appendLine('🎉 No security issues found in the scanned changes.');
                            vscode.window.showInformationMessage('SecureFlow: No security issues found in the scanned changes.');
                        } else {
                            outputChannel.appendLine(`⚠️ Found ${allIssues.length} potential security issues:\n`);
                            
                            allIssues.forEach((item, index) => {
                                const { issue, filePath, startLine } = item;
                                outputChannel.appendLine(`Issue #${index + 1}: ${issue.title}`);
                                outputChannel.appendLine(`File: ${path.basename(filePath)}`);
                                outputChannel.appendLine(`Location: Line ${startLine}`);
                                outputChannel.appendLine(`Severity: ${issue.severity}`);
                                outputChannel.appendLine(`Description: ${issue.description}`);
                                outputChannel.appendLine(`Recommendation: ${issue.recommendation}\n`);
                            });
                            
                            // Show notification
                            vscode.window.showWarningMessage(
                                `SecureFlow: Found ${allIssues.length} security ${allIssues.length === 1 ? 'issue' : 'issues'} in your code changes.`,
                                'View Details'
                            ).then(selection => {
                                if (selection === 'View Details') {
                                    outputChannel.show(true);
                                }
                            });
                        }
                    }
                );
            } catch (error) {
                console.error('Error during security review:', error);
                vscode.window.showErrorMessage(`SecureFlow: Error during security review: ${error}`);
            }
        }
    );
    
    // Add to subscriptions
    context.subscriptions.push(statusBarItem, reviewCommand);
}

// Interface for git change information
export interface GitChangeInfo {
    filePath: string;
    startLine: number;
    lineCount: number;
    content: string;
}
