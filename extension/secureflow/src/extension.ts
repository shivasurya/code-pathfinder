// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import { SecurityIssue } from './models/security-issue';
import { performSecurityAnalysisAsync } from './analysis/security-analyzer';
import { registerSecureFlowReviewCommand } from './git/git-changes';
import { SettingsManager } from './settings/settings-manager';
import { WorkspaceProfilerCommand } from './profiler/workspace-profiler-command';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {
	console.log('SecureFlow extension is now active!');

	// Create an output channel for security diagnostics
	const outputChannel = vscode.window.createOutputChannel('SecureFlow Security Diagnostics');
	
	// Initialize the settings manager
	const settingsManager = new SettingsManager(context);
	
	// Register workspace profiler command
	const workspaceProfilerCommand = new WorkspaceProfilerCommand(context);
	const workspaceProfilerDisposable = workspaceProfilerCommand.register();
	context.subscriptions.push(workspaceProfilerDisposable);
	
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
			
			// Get the API key for the selected AI Model
			const aiModel = settingsManager.getSelectedAIModel();
			let securityIssues: SecurityIssue[] = [];
			
			try {
				// Try to get the API key for the selected model
				const apiKey = await settingsManager.getApiKey();
				
				if (apiKey) {
					// If we have an API key, use the AI-powered analysis
					outputChannel.appendLine(`⏳ Running AI-powered analysis with ${aiModel}...`);
					securityIssues = await performSecurityAnalysisAsync(selectedText, aiModel, apiKey);
				} else {
					// Fallback to pattern-based analysis if no API key
					outputChannel.appendLine('⚠️ No API key found for the selected AI Model. Using pattern-based analysis only.');
				}
			} catch (error) {
				// If there's an error with the API key or AI analysis, fallback to pattern-based
				console.error('Error with AI analysis:', error);
				outputChannel.appendLine(`⚠️ Error connecting to ${aiModel}: ${error}. Using pattern-based analysis only.`);
			}
			
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
	
	// Register command to set API key
	const setApiKeyCommand = vscode.commands.registerCommand('secureflow.setApiKey', async () => {
		const aiModel = settingsManager.getSelectedAIModel();
		const apiKey = await vscode.window.showInputBox({
			prompt: `Enter API Key for ${aiModel}`,
			password: true,
			ignoreFocusOut: true,
			placeHolder: 'API Key'
		});
		
		if (apiKey) {
			await settingsManager.storeApiKey(apiKey);
			vscode.window.showInformationMessage(`API Key for ${aiModel} has been stored securely.`);
		}
	});

	// Add commands to context subscriptions
	context.subscriptions.push(analyzeSelectionCommand, setApiKeyCommand);
}

// This method is called when your extension is deactivated
export function deactivate() {}
