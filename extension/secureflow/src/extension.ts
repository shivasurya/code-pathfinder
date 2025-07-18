// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import { SecurityIssue } from './models/security-issue';
import { performSecurityAnalysis } from './security-analyzer';
import { registerSecureFlowReviewCommand } from './git-changes';
import { SettingsManager } from './settings-manager';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {
	console.log('SecureFlow extension is now active!');

	// Create an output channel for security diagnostics
	const outputChannel = vscode.window.createOutputChannel('SecureFlow Security Diagnostics');
	
	// Initialize the settings manager
	const settingsManager = new SettingsManager(context);
	
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
			
			// Get the selected AI Model
			const selectedModel = settingsManager.getSelectedAIModel();
			outputChannel.appendLine(`🤖 Using AI Model: ${selectedModel}`);
			
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
			
			// Analyze the selected code with the chosen AI Model
			const aiModel = settingsManager.getSelectedAIModel();
			const securityIssues = performSecurityAnalysis(selectedText, aiModel);
			
			// Complete the progress
			await new Promise(resolve => setTimeout(resolve, 500));
			
			// Display results
			outputChannel.appendLine('\n✅ Security analysis complete!\n');
			
			if (securityIssues.length === 0) {
				outputChannel.appendLine('🎉 No security issues found in the selected code.');
			} else {
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
	registerSecureFlowReviewCommand(context, outputChannel, settingsManager);

	// Add command to context subscriptions
	context.subscriptions.push(analyzeSelectionCommand);
}

// This method is called when your extension is deactivated
export function deactivate() {}
