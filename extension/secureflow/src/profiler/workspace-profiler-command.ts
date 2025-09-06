import * as vscode from 'vscode';
import { ProjectProfiler, ApplicationProfile } from './project-profiler';
import { getPromptPath } from '@codepathfinder/secureflow-cli';
import { SettingsManager } from '../settings/settings-manager';
import { ProfileStorageService } from '../services/profile-storage-service';
import { SentryService } from '../services/sentry-service';

/**
 * Command handler for workspace profiling and analysis
 */
export class WorkspaceProfilerCommand {
  private settingsManager: SettingsManager;

  constructor(private readonly context: vscode.ExtensionContext) {
    this.settingsManager = new SettingsManager(context);
  }

  /**
   * Register the workspace profiler command
   */
  public register(): vscode.Disposable {
    const sentry = SentryService.getInstance();
    return vscode.commands.registerCommand(
      'secureflow.profileWorkspace',
      sentry.withErrorHandling(
        'secureflow.profileWorkspace',
        this.executeCommand.bind(this)
      )
    );
  }

  /**
   * Execute the workspace profiler command
   */
  private async executeCommand(): Promise<void> {
    const workspaceFolders = vscode.workspace.workspaceFolders;

    if (!workspaceFolders || workspaceFolders.length === 0) {
      vscode.window.showErrorMessage('No workspace folder open');
      return;
    }

    // For now, just use the first workspace folder
    const workspaceFolder = workspaceFolders[0];

    // Create output channel if it doesn't exist
    const outputChannel = vscode.window.createOutputChannel(
      'SecureFlow Workspace Profiler'
    );
    outputChannel.clear();
    outputChannel.show(true);

    // Show progress indicator
    await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: 'Profiling workspace...',
        cancellable: true
      },
      async (progress, token) => {
        try {
          outputChannel.appendLine(
            '🔍 Profiling workspace to identify application types...'
          );

          outputChannel.appendLine(`📂 Workspace: ${workspaceFolder.name}`);

          // Create profiler instance with settings manager
          const profiler = new ProjectProfiler(
            workspaceFolder,
            this.settingsManager
          );

          // Profile the workspace
          progress.report({
            increment: 0,
            message: 'Scanning project structure...'
          });

          // Progress callback
          const updateProgress = (message: string) => {
            outputChannel.appendLine(`⏳ ${message}`);
            progress.report({ increment: 20, message });
          };

          // Get the API key from the settings manager
          const apiKey = await this.settingsManager.getApiKey();
          if (!apiKey) {
            outputChannel.appendLine(
              '❌ No API key configured. Please set your AI API key in the settings.'
            );
            vscode.window.showErrorMessage(
              'No API key configured. Please set your AI API key in the settings.'
            );
            return;
          }
          // Run the profiler
          const applications = await profiler.profileWorkspace(
            apiKey,
            updateProgress
          );

          // No applications detected
          if (applications.length === 0) {
            outputChannel.appendLine(
              '❓ Could not determine the application type'
            );
            vscode.window.showWarningMessage(
              'Could not determine the application type. Manual configuration may be required.'
            );
            return;
          }

          // Multiple applications detected - let the user choose
          let selectedApplications: ApplicationProfile[] = applications;

          if (applications.length > 1) {
            outputChannel.appendLine(
              `📊 Detected ${applications.length} applications in this workspace`
            );

            // Create quickpick for user to select
            const selected =
              await this.promptUserForApplicationSelection(applications);
            if (!selected) {
              outputChannel.appendLine('❌ Application selection cancelled');
              vscode.window.showInformationMessage(
                'Application selection cancelled.'
              );
              return;
            }

            selectedApplications = [selected];
          }

          // Show the detected applications
          outputChannel.appendLine(`\n✅ Workspace profiling complete!`);
          for (const app of selectedApplications) {
            outputChannel.appendLine(`\n----- Application Profile -----`);
            outputChannel.appendLine(`Name: ${app.name}`);
            outputChannel.appendLine(`Path: ${app.path}`);
            outputChannel.appendLine(
              `Type: ${app.category}${app.subcategory ? '/' + app.subcategory : ''}`
            );
            outputChannel.appendLine(`Technology: ${app.technology || 'N/A'}`);
            outputChannel.appendLine(`Languages: ${app.languages.join(', ')}`);
            outputChannel.appendLine(
              `Frameworks: ${app.frameworks.join(', ')}`
            );
            outputChannel.appendLine(
              `Build Tools: ${app.buildTools.join(', ')}`
            );
            outputChannel.appendLine(`Confidence: ${app.confidence}%`);
            outputChannel.appendLine(`\nEvidence:`);
            app.evidence.forEach((e) => outputChannel.appendLine(`- ${e}`));

            // Get the appropriate prompt path
            const promptPath = getPromptPath(
              app.category,
              app.subcategory,
              app.technology
            );
            outputChannel.appendLine(`\nSelected prompt: ${promptPath}`);

            // store the profile using profile storage service
            const profileStorageService = new ProfileStorageService(
              this.context
            );
            const storedProfile = await profileStorageService.storeProfile(
              app,
              workspaceFolder.uri.toString(),
              true
            );
            outputChannel.appendLine(
              `Stored profile with ID: ${storedProfile.id}`
            );
          }

          vscode.window.showInformationMessage(
            `Workspace profiling complete. ${selectedApplications.length === 1 ? '1 application' : `${selectedApplications.length} applications`} detected.`
          );
        } catch (error) {
          outputChannel.appendLine(
            `❌ Error during workspace profiling: ${error}`
          );
          vscode.window.showErrorMessage(
            'Error during workspace profiling. See output for details.'
          );
        }
      }
    );
  }

  /**
   * Prompt user to select which application to analyze
   */
  private async promptUserForApplicationSelection(
    applications: ApplicationProfile[]
  ): Promise<ApplicationProfile | undefined> {
    const items = applications.map((app) => ({
      label: app.name,
      description: `${app.category}${app.subcategory ? '/' + app.subcategory : ''}`,
      detail: `Path: ${app.path}, Confidence: ${app.confidence}%, Languages: ${app.languages.join(', ')}`,
      application: app
    }));

    const selected = await vscode.window.showQuickPick(items, {
      placeHolder: 'Select an application to analyze',
      canPickMany: false
    });

    return selected ? selected.application : undefined;
  }
}

export default WorkspaceProfilerCommand;
