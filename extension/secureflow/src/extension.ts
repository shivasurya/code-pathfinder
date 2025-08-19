// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import { SecurityIssue } from './models/security-issue';
import {
  performSecurityAnalysisAsync,
  registerAnalyzeSelectionCommand
} from './analysis/security-analyzer';
import { registerSecureFlowReviewCommand } from './git/git-changes';
import { SettingsManager } from './settings/settings-manager';
import { WorkspaceProfilerCommand } from './profiler/workspace-profiler-command';
import { SecureFlowExplorer } from './ui/secureflow-explorer';
import { AnalyticsService } from './services/analytics';
import { SentryService } from './services/sentry-service';

/**
 * TODO(CLI): This file is EXTENSION-ONLY. It wires up VS Code activation,
 * command registration, webviews, analytics, and sentry. The CLI will have
 * its own entry point and must NOT import this module.
 */

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
    const analyticsEnabled = vscode.workspace
      .getConfiguration('secureflow')
      .get('analytics.enabled', true);

    if (analyticsEnabled) {
      await analytics.initialize(context);
      analytics.trackEvent('SecureFlow Extension Started', {
        extension_version: context.extension.packageJSON.version,
        vscode_version: vscode.version
      });
    } else {
      console.log(
        '📊 Analytics: Disabled in settings, skipping initialization'
      );
    }

    // Rely on Sentry's default global handlers; filtering is handled in beforeSend
  } catch (error) {
    console.error('Failed to initialize SecureFlow services:', error);
    // Even if Sentry fails, we should still try to capture this error
    try {
      const sentry = SentryService.getInstance();
      sentry.captureException(error as Error, {
        context: 'extension_activation'
      });
    } catch (sentryError) {
      console.error('Failed to capture initialization error:', sentryError);
    }
  }

  SecureFlowExplorer.register(context);

  const outputChannel = vscode.window.createOutputChannel(
    'SecureFlow Security Diagnostics'
  );

  try {
    // Initialize the settings manager
    const settingsManager = new SettingsManager(context);

    // Register workspace profiler command
    const workspaceProfilerCommand = new WorkspaceProfilerCommand(context);
    const workspaceProfilerDisposable = workspaceProfilerCommand.register();
    context.subscriptions.push(workspaceProfilerDisposable);

    // Register analyze selection command
    context.subscriptions.push(
      registerAnalyzeSelectionCommand(outputChannel, settingsManager, context)
    );

    // Register the git changes review command and status bar button
    registerSecureFlowReviewCommand(context, outputChannel, settingsManager);

    console.log('SecureFlow extension fully activated!');
  } catch (error) {
    console.error(`SecureFlow activation failed: ${error}`);

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

    vscode.window.showErrorMessage(`SecureFlow activation failed: ${error}`);
  }
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
      sentry.captureException(error as Error, {
        context: 'extension_deactivation'
      });
    } catch (sentryError) {
      console.error('Failed to capture deactivation error:', sentryError);
    }
  }
}
