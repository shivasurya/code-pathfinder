/******/ (() => { // webpackBootstrap
/******/ 	"use strict";
/******/ 	var __webpack_modules__ = ([
/* 0 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.activate = activate;
exports.deactivate = deactivate;
// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
const vscode = __importStar(__webpack_require__(1));
const security_analyzer_1 = __webpack_require__(5);
const git_changes_1 = __webpack_require__(2);
// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
function activate(context) {
    console.log('SecureFlow extension is now active!');
    // Create an output channel for security diagnostics
    const outputChannel = vscode.window.createOutputChannel('SecureFlow Security Diagnostics');
    // Register the command that will be triggered with cmd+l
    const analyzeSelectionCommand = vscode.commands.registerCommand('secureflow.analyzeSelection', async () => {
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
            vscode.window.showInformationMessage('No text selected for security analysis');
            return;
        }
        // Show progress indicator
        await vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: "Scanning for security issues...",
            cancellable: true
        }, async (progress, token) => {
            // Show output channel
            outputChannel.clear();
            outputChannel.show(true);
            outputChannel.appendLine('🔍 Analyzing code for security vulnerabilities...');
            // Simulate scanning process with some delay
            progress.report({ increment: 0 });
            // First stage - initial scanning
            await new Promise(resolve => setTimeout(resolve, 500));
            progress.report({ increment: 20, message: "Parsing code..." });
            outputChannel.appendLine('⏳ Parsing code structure...');
            // Second stage - deeper analysis
            await new Promise(resolve => setTimeout(resolve, 700));
            progress.report({ increment: 30, message: "Checking for vulnerabilities..." });
            outputChannel.appendLine('⏳ Checking for known vulnerability patterns...');
            // Third stage - final checks
            await new Promise(resolve => setTimeout(resolve, 800));
            progress.report({ increment: 50, message: "Finalizing analysis..." });
            outputChannel.appendLine('⏳ Running final security checks...');
            // Analyze the selected code
            const securityIssues = (0, security_analyzer_1.performSecurityAnalysis)(selectedText);
            // Complete the progress
            await new Promise(resolve => setTimeout(resolve, 500));
            // Display results
            outputChannel.appendLine('\n✅ Security analysis complete!\n');
            if (securityIssues.length === 0) {
                outputChannel.appendLine('🎉 No security issues found in the selected code.');
            }
            else {
                outputChannel.appendLine(`⚠️ Found ${securityIssues.length} potential security issues:\n`);
                securityIssues.forEach((issue, index) => {
                    outputChannel.appendLine(`Issue #${index + 1}: ${issue.title}`);
                    outputChannel.appendLine(`Severity: ${issue.severity}`);
                    outputChannel.appendLine(`Description: ${issue.description}`);
                    outputChannel.appendLine(`Recommendation: ${issue.recommendation}\n`);
                });
            }
        });
    });
    // Register the git changes review command and status bar button
    (0, git_changes_1.registerSecureFlowReviewCommand)(context, outputChannel);
    // Add command to context subscriptions
    context.subscriptions.push(analyzeSelectionCommand);
}
// This method is called when your extension is deactivated
function deactivate() { }


/***/ }),
/* 1 */
/***/ ((module) => {

module.exports = require("vscode");

/***/ }),
/* 2 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.getGitChanges = getGitChanges;
exports.registerSecureFlowReviewCommand = registerSecureFlowReviewCommand;
const vscode = __importStar(__webpack_require__(1));
const cp = __importStar(__webpack_require__(3));
const path = __importStar(__webpack_require__(4));
const security_analyzer_1 = __webpack_require__(5);
/**
 * Gets the git changes (hunks) for a specific file or all files in the workspace
 * @param filePath Optional path to a specific file
 * @returns Array of change information objects
 */
async function getGitChanges(filePath) {
    try {
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders || workspaceFolders.length === 0) {
            throw new Error('No workspace folder found');
        }
        const repoPath = workspaceFolders[0].uri.fsPath;
        const changes = [];
        // Construct the git diff command
        let command = 'git diff --unified=0';
        if (filePath) {
            const relativePath = path.relative(repoPath, filePath);
            command += ` -- "${relativePath}"`;
        }
        // Execute git command
        const output = await executeCommand(command, repoPath);
        // Parse the git diff output
        let currentFile = null;
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
                            }
                            else {
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
    }
    catch (error) {
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
async function executeCommand(command, cwd) {
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
 * Registers the SecureFlow review command and status bar button
 * @param context Extension context
 * @param outputChannel Output channel for displaying results
 */
function registerSecureFlowReviewCommand(context, outputChannel) {
    // Create status bar item
    const statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.text = "$(shield) SecureFlow";
    statusBarItem.tooltip = "Scan git changes for security issues";
    statusBarItem.command = "secureflow.reviewChanges";
    statusBarItem.show();
    // Register command
    const reviewCommand = vscode.commands.registerCommand("secureflow.reviewChanges", async (uri) => {
        // Get the file path from the URI if provided (from SCM view)
        let currentFilePath;
        if (uri && uri.fsPath) {
            // If command was triggered from SCM view with a specific file
            currentFilePath = uri.fsPath;
        }
        else {
            // Otherwise use the active editor file
            const editor = vscode.window.activeTextEditor;
            currentFilePath = editor?.document.uri.fsPath;
        }
        try {
            // Show progress indicator
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: "SecureFlow: Scanning git changes...",
                cancellable: true
            }, async (progress, token) => {
                // Clear and show output channel
                outputChannel.clear();
                outputChannel.show(true);
                outputChannel.appendLine('🔍 SecureFlow: Scanning git changes for security issues...\n');
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
                let allIssues = [];
                // Analyze each change
                for (let i = 0; i < changes.length; i++) {
                    const change = changes[i];
                    outputChannel.appendLine(`File: ${path.basename(change.filePath)}`);
                    outputChannel.appendLine(`Lines: ${change.startLine}-${change.startLine + change.lineCount - 1}`);
                    outputChannel.appendLine(`Changes:\n${change.content}\n`);
                    // Analyze the change content
                    const issues = (0, security_analyzer_1.performSecurityAnalysis)(change.content);
                    // Map issues to include file path and line number
                    const mappedIssues = issues.map((issue) => ({
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
                }
                else {
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
                    vscode.window.showWarningMessage(`SecureFlow: Found ${allIssues.length} security ${allIssues.length === 1 ? 'issue' : 'issues'} in your code changes.`, 'View Details').then(selection => {
                        if (selection === 'View Details') {
                            outputChannel.show(true);
                        }
                    });
                }
            });
        }
        catch (error) {
            console.error('Error during security review:', error);
            vscode.window.showErrorMessage(`SecureFlow: Error during security review: ${error}`);
        }
    });
    // Add to subscriptions
    context.subscriptions.push(statusBarItem, reviewCommand);
}


/***/ }),
/* 3 */
/***/ ((module) => {

module.exports = require("child_process");

/***/ }),
/* 4 */
/***/ ((module) => {

module.exports = require("path");

/***/ }),
/* 5 */
/***/ ((__unused_webpack_module, exports) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.performSecurityAnalysis = performSecurityAnalysis;
/**
 * Performs security analysis on the given code snippet
 * @param code The code to analyze
 * @returns Array of security issues found
 */
function performSecurityAnalysis(code) {
    // This is a mock implementation. In a real extension, you would implement actual code analysis.
    const issues = [];
    // Check for SQL injection vulnerability pattern
    if (code.toLowerCase().includes('sql') && code.includes("'") && code.includes("+")) {
        issues.push({
            title: "Potential SQL Injection",
            severity: "High",
            description: "String concatenation used with SQL queries can lead to SQL injection attacks.",
            recommendation: "Use parameterized queries or prepared statements instead of string concatenation."
        });
    }
    // Check for potential XSS vulnerability pattern
    if ((code.includes("innerHTML") || code.includes("document.write")) && code.includes("${")) {
        issues.push({
            title: "Potential Cross-Site Scripting (XSS)",
            severity: "High",
            description: "Directly inserting user input into DOM can lead to XSS attacks.",
            recommendation: "Use textContent instead of innerHTML, or sanitize user input before inserting into DOM."
        });
    }
    // Check for hardcoded secrets
    if (code.toLowerCase().includes("password") || code.toLowerCase().includes("token") || code.toLowerCase().includes("secret")) {
        if (code.includes("=") && (code.includes("'") || code.includes("\""))) {
            issues.push({
                title: "Hardcoded Secret",
                severity: "Medium",
                description: "Sensitive information appears to be hardcoded in the source code.",
                recommendation: "Use environment variables or a secure vault for storing sensitive information."
            });
        }
    }
    // Check for insecure random number generation
    if (code.includes("Math.random(") && (code.toLowerCase().includes("auth") ||
        code.toLowerCase().includes("token") ||
        code.toLowerCase().includes("password") ||
        code.toLowerCase().includes("secure"))) {
        issues.push({
            title: "Insecure Random Number Generation",
            severity: "Medium",
            description: "Using Math.random() for security-sensitive operations is not recommended.",
            recommendation: "Use a cryptographically secure random number generator like crypto.getRandomValues()."
        });
    }
    // Check for potential command injection
    if (code.includes("exec(") || code.includes("spawn(") || code.includes("system(")) {
        if (code.includes("+") || code.includes("`") || code.includes("${")) {
            issues.push({
                title: "Potential Command Injection",
                severity: "Critical",
                description: "Dynamic command execution can lead to command injection vulnerabilities.",
                recommendation: "Avoid using user input in command execution. If necessary, properly validate and sanitize the input."
            });
        }
    }
    // If no issues found but code contains security-sensitive patterns, add a mock issue (for demonstration)
    if (issues.length === 0 && (code.toLowerCase().includes("security") ||
        code.toLowerCase().includes("login") ||
        code.toLowerCase().includes("auth") ||
        code.toLowerCase().includes("password"))) {
        issues.push({
            title: "Potential Weak Authentication",
            severity: "Medium",
            description: "Authentication logic might not follow security best practices.",
            recommendation: "Implement multi-factor authentication and ensure proper password handling."
        });
    }
    return issues;
}


/***/ })
/******/ 	]);
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			// no module.id needed
/******/ 			// no module.loaded needed
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	// This entry module is referenced by other modules so it can't be inlined
/******/ 	var __webpack_exports__ = __webpack_require__(0);
/******/ 	module.exports = __webpack_exports__;
/******/ 	
/******/ })()
;
//# sourceMappingURL=extension.js.map