import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { performSecurityAnalysisAsync } from '../analysis/security-analyzer';
import { SettingsManager } from '../settings/settings-manager';
import { ProfileStorageService } from '../services/profile-storage-service';
import { StoredProfile } from '../models/profile-store';

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

        // Process both staged and unstaged changes
        const processDiff = async (staged: boolean): Promise<void> => {
            const command = `git diff ${staged ? '--cached' : ''} --unified=0 --no-color`;
            const output = await executeCommand(command, repoPath);
            
            if (!output.trim()) return;
            
            let currentFile: string | null = null;
            const lines = output.split('\n');
            let i = 0;
            
            while (i < lines.length) {
                const line = lines[i];
                
                // Check for file header
                if (line.startsWith('diff --git')) {
                    const match = line.match(/diff --git a\/(.*?) b\/(.*)/);
                    if (match && match[2]) {
                        currentFile = match[2].trim();
                    }
                    i++;
                    continue;
                }
                
                // Check for hunk header
                if (line.startsWith('@@') && currentFile) {
                    const match = line.match(/@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/);
                    if (match) {
                        const startLine = parseInt(match[3], 10);
                        let addedLines = 0;
                        let content = '';
                        
                        // Process the hunk content
                        i++; // Move to first line of hunk
                        while (i < lines.length && !lines[i].startsWith('diff --git') && !lines[i].startsWith('@@')) {
                            const hunkLine = lines[i];
                            if (hunkLine.startsWith('+') && !hunkLine.startsWith('+++')) {
                                content += hunkLine.substring(1) + '\n';
                                addedLines++;
                            }
                            i++;
                        }
                        
                        if (addedLines > 0) {
                            changes.push({
                                filePath: path.join(repoPath, currentFile),
                                startLine,
                                lineCount: addedLines,
                                content: content.trim()
                            });
                        }
                        
                        continue; // Don't increment i again as we already did in the loop
                    }
                }
                
                i++;
            }
        };
        
        // Process both staged and unstaged changes
        await processDiff(true);  // Staged changes
        await processDiff(false); // Unstaged changes
        
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
    // Initialize profile storage service
    const profileService = new ProfileStorageService(context);
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

                        // Get all stored profiles
                        const allProfiles: StoredProfile[] = profileService.getAllProfiles();

                        // Helper function to find the most relevant profile for a file path
                        const findRelevantProfile = (filePath: string): StoredProfile | undefined => {
                            if (allProfiles.length === 0) {
                                return undefined;
                            }

                            // Check if file is in a workspace folder
                            const workspaceFolder = vscode.workspace.getWorkspaceFolder(vscode.Uri.file(filePath));
                            if (!workspaceFolder) {
                                return undefined;
                            }

                            // Find profiles within the workspace
                            const relevantProfiles = allProfiles.filter(profile => {
                                const profileUri = vscode.Uri.parse(profile.workspaceFolderUri);
                                const profileWorkspace = vscode.workspace.getWorkspaceFolder(profileUri);
                                return profileWorkspace && profileWorkspace.uri.fsPath === workspaceFolder.uri.fsPath;
                            });

                            if (relevantProfiles.length === 0) {
                                return undefined;
                            }

                            // Get relative path within workspace
                            const relativePath = path.relative(workspaceFolder.uri.fsPath, filePath);

                            // Find profile where file path falls under profile path
                            return relevantProfiles.find(profile => {
                                const normalizedProfilePath = profile.path === '/' ? '' : profile.path;
                                const profilePathParts = normalizedProfilePath.split('/').filter(p => p);
                                const filePathParts = relativePath.split(path.sep).filter(p => p);

                                // Check if file path starts with profile path parts
                                return profilePathParts.every((part, index) => filePathParts[index] === part);
                            });
                        };

                        // Group changes by file
                        const changesByFile: { [filePath: string]: {diffs: string[], starts: number[]} } = {};
                        for (const change of changes) {
                            if (!changesByFile[change.filePath]) {
                                changesByFile[change.filePath] = { diffs: [], starts: [] };
                            }
                            changesByFile[change.filePath].diffs.push(change.content);
                            changesByFile[change.filePath].starts.push(change.startLine);
                        }

                        let allIssues: Array<{issue: SecurityIssue, filePath: string, startLine: number}> = [];
                        const workspaceFolders = vscode.workspace.workspaceFolders || [];
                        let consolidatedReviewContent = '';
                        const fileMetadata: Array<{filePath: string, startLine: number}> = [];

                        // Collect all profiles used
                        const usedProfiles = new Set<StoredProfile>();

                        // Build consolidated review content for all files
                        for (const [filePath, {diffs, starts}] of Object.entries(changesByFile)) {
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
                                fullFileContent = (await vscode.workspace.openTextDocument(filePath)).getText();
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
                            usedProfiles.forEach(profile => {
                                profileSummary += `- ${profile.name} (${profile.category}) at ${profile.path}\n`;
                            });
                            profileSummary += `*/\n\n`;
                        }

                        const finalReviewContent = profileSummary + consolidatedReviewContent;
                        outputChannel.appendLine(finalReviewContent);

                        progress.report({ 
                            increment: 30,
                            message: `Performing security analysis...`
                        });

                        // Make single API call with all consolidated content
                        const issues = await performSecurityAnalysisAsync(finalReviewContent, selectedModel, await settingsManager.getApiKey());
                        
                        // Map issues back to files (best effort - use first file if unable to determine)
                        const mappedIssues = issues.map((issue: SecurityIssue, index: number) => {
                            // Try to find the most relevant file based on issue content or use first file as fallback
                            let relevantFile = fileMetadata[0]; // fallback to first file
                            
                            // Simple heuristic: if issue mentions a filename, use that
                            for (const fileMeta of fileMetadata) {
                                const fileName = path.basename(fileMeta.filePath);
                                if (issue.description.includes(fileName) || issue.title.includes(fileName)) {
                                    relevantFile = fileMeta;
                                    break;
                                }
                            }
                            
                            return {
                                issue,
                                filePath: relevantFile.filePath,
                                startLine: relevantFile.startLine
                            };
                        });
                        
                        allIssues = mappedIssues;

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
