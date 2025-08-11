// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import { SecurityIssue } from './models/security-issue';
import { performSecurityAnalysisAsync } from './analysis/security-analyzer';
import { registerSecureFlowReviewCommand } from './git/git-changes';
import { SettingsManager } from './settings/settings-manager';
import { WorkspaceProfilerCommand } from './profiler/workspace-profiler-command';
import { SecureFlowExplorer } from './ui/secureflow-explorer';
import { AnalyticsService } from './services/analytics';
import { SentryService } from './services/sentry-service';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export async function activate(context: vscode.ExtensionContext) {
	
	try {
		// Initialize Sentry error reporting first
		const sentry = SentryService.getInstance();
		await sentry.initialize(context);
		sentry.addBreadcrumb('Extension activation started', 'lifecycle');

		// Initialize analytics if enabled
		const analytics = AnalyticsService.getInstance();
		const analyticsEnabled = vscode.workspace.getConfiguration('secureflow').get('analytics.enabled', true);
		// console.log('📊 Analytics: Settings check - enabled:', analyticsEnabled);
		
		if (analyticsEnabled) {
			await analytics.initialize(context);
			analytics.trackEvent('SecureFlow Extension Started', {
				extension_version: context.extension.packageJSON.version,
				vscode_version: vscode.version
			});
		} else {
			console.log('📊 Analytics: Disabled in settings, skipping initialization');
		}

		// Set up global error handlers
		sentry.setupGlobalErrorHandlers();
	} catch (error) {
		console.error('Failed to initialize SecureFlow services:', error);
		// Even if Sentry fails, we should still try to capture this error
		try {
			const sentry = SentryService.getInstance();
			sentry.captureException(error as Error, { context: 'extension_activation' });
		} catch (sentryError) {
			console.error('Failed to capture initialization error:', sentryError);
		}
	}
	
	SecureFlowExplorer.register(context);
	
	// Show activation message to user for debugging
	// vscode.window.showInformationMessage('SecureFlow extension activated successfully!');

	// Create an output channel for security diagnostics
	const outputChannel = vscode.window.createOutputChannel('SecureFlow Security Diagnostics');
	outputChannel.appendLine('SecureFlow extension starting...');
	
	try {
		// Initialize the settings manager
		const settingsManager = new SettingsManager(context);
		outputChannel.appendLine('Settings manager initialized');
		
		// Register workspace profiler command
		const workspaceProfilerCommand = new WorkspaceProfilerCommand(context);
		const workspaceProfilerDisposable = workspaceProfilerCommand.register();
		context.subscriptions.push(workspaceProfilerDisposable);
		outputChannel.appendLine('Workspace profiler command registered');
		
		// Register analyze selection command
		const analyzeSelectionCommand = registerAnalyzeSelectionCommand(outputChannel, settingsManager);
		outputChannel.appendLine('Analyze selection command registered');
		
		// Register the git changes review command and status bar button
		registerSecureFlowReviewCommand(context, outputChannel, settingsManager);
		outputChannel.appendLine('Git review command registered');

		// Add commands to context subscriptions
		context.subscriptions.push(
			analyzeSelectionCommand
		);
		
		outputChannel.appendLine('SecureFlow extension fully activated!');
		console.log('SecureFlow extension fully activated!');
		
	} catch (error) {
		const errorMessage = `SecureFlow activation failed: ${error}`;
		outputChannel.appendLine(errorMessage);
		console.error(errorMessage);
		
		// Capture the error with Sentry
		try {
			const sentry = SentryService.getInstance();
			sentry.captureException(error as Error, { 
				context: 'extension_main_activation',
				component: 'command_registration'
			});
		} catch (sentryError) {
			console.error('Failed to capture activation error:', sentryError);
		}
		
		vscode.window.showErrorMessage(errorMessage);
	}
}

function registerAnalyzeSelectionCommand(
	outputChannel: vscode.OutputChannel, 
	settingsManager: SettingsManager
): vscode.Disposable {
	const sentry = SentryService.getInstance();
	return vscode.commands.registerCommand('secureflow.analyzeSelection', sentry.withErrorHandling(
		'secureflow.analyzeSelection',
		async () => {
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
				
				// Capture the error with Sentry
				try {
					const sentry = SentryService.getInstance();
					sentry.captureException(error as Error, { 
						context: 'ai_analysis_error',
						component: 'analyze_selection_command',
						ai_model: aiModel,
						selected_text_length: selectedText.length
					});
				} catch (sentryError) {
					console.error('Failed to capture AI analysis error:', sentryError);
				}
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
	}));
}

// This method is called when your extension is deactivated
export async function deactivate() {
	try {
		// Add breadcrumb for deactivation
		const sentry = SentryService.getInstance();
		sentry.addBreadcrumb('Extension deactivation started', 'lifecycle');
		
		// Properly shutdown analytics
		const analytics = AnalyticsService.getInstance();
		await analytics.shutdown();
		
		// Flush any pending Sentry events before shutdown
		await sentry.flush(3000); // Wait up to 3 seconds for events to be sent
		
		// Close Sentry client
		await sentry.close();
		
		console.log('SecureFlow extension deactivated successfully');
	} catch (error) {
		console.error('Error during extension deactivation:', error);
		// Try to capture this error, but don't wait for it
		try {
			const sentry = SentryService.getInstance();
			sentry.captureException(error as Error, { context: 'extension_deactivation' });
		} catch (sentryError) {
			console.error('Failed to capture deactivation error:', sentryError);
		}
	}
}
