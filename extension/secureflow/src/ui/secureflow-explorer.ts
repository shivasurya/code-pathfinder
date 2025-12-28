import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import { StoredProfile } from '../models/profile-store';
import { ProfileStorageService } from '../services/profile-storage-service';
import { ScanStorageService } from '../services/scan-storage-service';
import { ProfileScanService } from '../services/profile-scan-service';
import { ScanResult } from '../models/scan-result';
import { AnalyticsService } from '../services/analytics';
import { SettingsManager } from '../settings/settings-manager';
import { ModelConfig } from '../generated/model-config';

export class SecureFlowExplorer {
  private static instance: SecureFlowExplorer;
  private _view?: vscode.WebviewView;
  private static _provider?: SecureFlowWebViewProvider;

  private constructor(private readonly context: vscode.ExtensionContext) {}

  public static getInstance(
    context: vscode.ExtensionContext
  ): SecureFlowExplorer {
    if (!SecureFlowExplorer.instance) {
      SecureFlowExplorer.instance = new SecureFlowExplorer(context);
    }
    return SecureFlowExplorer.instance;
  }

  public static register(context: vscode.ExtensionContext): void {
    const provider = new SecureFlowWebViewProvider(
      context.extensionUri,
      context
    );
    SecureFlowExplorer._provider = provider;
    context.subscriptions.push(
      vscode.window.registerWebviewViewProvider('secureflow.mainView', provider)
    );
  }

  public static refreshScanList(): void {
    if (SecureFlowExplorer._provider) {
      SecureFlowExplorer._provider.refreshScanList();
    }
  }
}

class SecureFlowWebViewProvider implements vscode.WebviewViewProvider {
  private _view?: vscode.WebviewView;
  private _profiles: StoredProfile[] = [];
  private _scans: ScanResult[] = [];
  private _profileService: ProfileStorageService;
  private _scanService: ScanStorageService;
  private _profileScanService: ProfileScanService;

  constructor(
    private readonly _extensionUri: vscode.Uri,
    private readonly _context: vscode.ExtensionContext
  ) {
    this._profileService = new ProfileStorageService(this._context);
    this._scanService = new ScanStorageService(this._context);
    this._profileScanService = new ProfileScanService(this._context);
    this.loadProfiles();
    this.loadScans();
  }

  private async loadProfiles() {
    try {
      // Get profiles only for the current workspace folders
      const workspaceFolders = vscode.workspace.workspaceFolders || [];
      let profiles: StoredProfile[] = [];

      for (const folder of workspaceFolders) {
        const workspaceProfiles = this._profileService.getWorkspaceProfiles(
          folder.uri.toString()
        );
        profiles = profiles.concat(workspaceProfiles);
      }

      this._profiles = profiles;

      if (this._view) {
        this._view.webview.postMessage({
          type: 'updateProfiles',
          profiles: this._profiles
        });
      }
    } catch (error) {
      if (this._view) {
        this._view.webview.postMessage({
          type: 'error',
          message: 'Failed to load profiles'
        });
      }
    }
  }

  private async loadScans() {
    try {
      const scans = this._scanService.getAllScans();
      this._scans = scans;

      if (this._view) {
        this._view.webview.postMessage({
          type: 'updateScans',
          scans: this._scans
        });
      }
    } catch (error) {
      if (this._view) {
        this._view.webview.postMessage({
          type: 'error',
          message: 'Failed to load scan history'
        });
      }
    }
  }

  public async refreshScanList() {
    await this.loadScans();
  }

  public resolveWebviewView(
    webviewView: vscode.WebviewView,
    context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken
  ) {
    console.log('SecureFlow: resolveWebviewView called');
    this._view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this._extensionUri]
    };

    console.log('SecureFlow: Webview options set, generating HTML...');

    webviewView.webview.onDidReceiveMessage(async (message) => {
      console.log('SecureFlow: Received message from webview:', message.type);
      // Initialize analytics service for all message handlers
      const analytics = AnalyticsService.getInstance();

      switch (message.type) {
        case 'scanWorkspace':
          // Track UI action
          analytics.trackEvent('Workspace Security Scan Started', {
            scan_trigger: 'manual_button_click'
          });

          try {
            // Start the workspace scan
            await this._profileService.scanWorkspace();
            // Load the new profiles
            await this.loadProfiles();
            // Track API success
            analytics.trackEvent('Workspace Security Scan Completed', {
              scan_result: 'success',
              profiles_found: this._profiles.length
            });

            // Update the UI with success state
            if (this._view) {
              this._view.webview.postMessage({
                type: 'scanComplete',
                success: true
              });
            }
          } catch (error) {
            // Track API failure
            analytics.trackEvent('Workspace Security Scan Failed', {
              error_type: 'scan_execution_error'
            });

            if (this._view) {
              this._view.webview.postMessage({
                type: 'scanComplete',
                success: false,
                error: 'Failed to scan workspace'
              });
            }
          }
          break;
        case 'getProfiles':
          analytics.trackEvent('Security Profiles Refreshed', {
            refresh_trigger: 'user_request'
          });
          await this.loadProfiles();
          break;
        case 'getScans':
          analytics.trackEvent('Scan History Viewed', {
            view_trigger: 'user_request'
          });
          await this.loadScans();
          break;
        case 'showSettings':
          analytics.trackEvent('Settings Opened', {
            trigger: 'user_request'
          });
          if (this._view) {
            this._view.webview.postMessage({
              type: 'showSettings'
            });
          }
          break;
        case 'backToProfiles':
          if (this._view) {
            this._view.webview.postMessage({
              type: 'backToProfiles'
            });
          }
          break;
        case 'getCurrentConfig':
          if (this._view) {
            const settingsManager = new SettingsManager(this._context);
            const currentModel = settingsManager.getSelectedAIModel();
            const currentApiKey = await settingsManager.getApiKey();
            const currentProvider = settingsManager.getSelectedProvider();

            this._view.webview.postMessage({
              type: 'currentConfig',
              config: {
                model: currentModel,
                apiKey: currentApiKey || '',
                provider: currentProvider
              }
            });
          }
          break;
        case 'viewScan':
          analytics.trackEvent('Scan Results Opened', {
            scan_number: message.scanNumber
          });
          // Get scan from storage service (not filtered _scans)
          const scan = this._scanService.getScanByNumber(message.scanNumber);
          if (scan) {
            console.log('SecureFlow: Opening scan results', {
              scanNumber: scan.scanNumber,
              issuesCount: scan.issues?.length || 0
            });
            this.openScanResultsWebview(scan);
          } else {
            console.error('SecureFlow: Scan not found:', message.scanNumber);
            vscode.window.showErrorMessage(`Scan #${message.scanNumber} not found`);
          }
          break;
        case 'openVulnerabilityDetails':
          analytics.trackEvent('Vulnerability Details Opened', {
            vulnerability_title: message.vulnerability.title
          });
          this.openVulnerabilityDetailsWebview(message.vulnerability);
          break;
        case 'openFile':
          try {
            const document = await vscode.workspace.openTextDocument(message.filePath);
            const editor = await vscode.window.showTextDocument(document);
            const position = new vscode.Position(message.line - 1, 0);
            editor.selection = new vscode.Selection(position, position);
            editor.revealRange(new vscode.Range(position, position));
          } catch (error) {
            vscode.window.showErrorMessage(`Failed to open file: ${message.filePath}`);
          }
          break;
        case 'scanGitChanges':
          analytics.trackEvent('Git Changes Scan Triggered', {
            trigger: 'profile_action'
          });
          try {
            await vscode.commands.executeCommand('secureflow.reviewChanges');
          } catch (error) {
            vscode.window.showErrorMessage('Failed to scan git changes');
          }
          break;
        case 'profileSelected':
          analytics.trackEvent('Security Profile Selected', {
            profile_action: 'view_details'
          });
          const profile = this._profiles.find(
            (p) => p.id === message.profileId
          );
          if (profile) {
            // Load scans for this specific profile
            const profileScans = this._scanService.getScansForProfile(profile.id);

            // Send profile details and scans to webview
            if (this._view) {
              this._view.webview.postMessage({
                type: 'profileDetails',
                profile: profile
              });
              this._view.webview.postMessage({
                type: 'updateScans',
                scans: profileScans
              });
            }
          }
          break;

        case 'rescanProfile':
          analytics.trackEvent('Profile Rescan Started', {
            profile_id: message.profileId,
            scan_trigger: 'manual_button_click'
          });

          const profileToRescan = this._profiles.find(
            (p) => p.id === message.profileId
          );

          if (profileToRescan) {
            // Run scan with progress notification
            await vscode.window.withProgress(
              {
                location: vscode.ProgressLocation.Notification,
                title: `Scanning ${profileToRescan.name}`,
                cancellable: false
              },
              async (progress) => {
                try {
                  progress.report({ message: 'Initializing scan...' });

                  // Run the scan using CLI scanner
                  const savedScan = await this._profileScanService.scanProfile(
                    profileToRescan,
                    (progressMessage) => {
                      progress.report({ message: progressMessage });
                      console.log('ProfileScan:', progressMessage);
                    }
                  );

                  // Track success
                  analytics.trackEvent('Profile Rescan Completed', {
                    profile_id: profileToRescan.id,
                    scan_result: 'success',
                    issues_found: savedScan.issues?.length || 0
                  });

                  // Reload scans for this profile
                  const profileScans = this._scanService.getScansForProfile(profileToRescan.id);

                  // Update UI
                  if (this._view) {
                    this._view.webview.postMessage({
                      type: 'updateScans',
                      scans: profileScans
                    });
                  }

                  // Show completion message
                  const issuesCount = savedScan.issues?.length || 0;
                  const message = `✓ Scan complete! Found ${issuesCount} issue${issuesCount !== 1 ? 's' : ''}.`;
                  vscode.window.showInformationMessage(message);
                } catch (error: any) {
                  // Track failure
                  analytics.trackEvent('Profile Rescan Failed', {
                    profile_id: message.profileId,
                    error_type: error.message || 'unknown'
                  });

                  console.error('Error rescanning profile:', error);
                  vscode.window.showErrorMessage(
                    `Failed to scan profile: ${error.message || error}`
                  );
                }
              }
            );
          }
          break;

        case 'deleteProfile':
          analytics.trackEvent('Profile Deletion Requested', {
            profile_id: message.profileId
          });

          try {
            await this._profileService.deleteProfile(message.profileId);
            await this.loadProfiles();

            if (this._view) {
              this._view.webview.postMessage({
                type: 'profileDeleted',
                profileId: message.profileId
              });
              this._view.webview.postMessage({
                type: 'updateProfiles',
                profiles: this._profiles
              });
            }

            vscode.window.showInformationMessage('Profile deleted successfully');
          } catch (error) {
            console.error('Error deleting profile:', error);
            vscode.window.showErrorMessage('Failed to delete profile');
          }
          break;
        case 'confirmDelete':
          analytics.trackEvent('Security Profile Delete Requested', {
            delete_trigger: 'user_confirmation'
          });

          const answer = await vscode.window.showWarningMessage(
            'Are you sure you want to delete this profile?',
            { modal: true },
            'Delete',
            'Cancel'
          );
          if (answer === 'Delete') {
            try {
              // Delete the profile
              await this._profileService.deleteProfile(message.profileId);

              // Delete all scan history as requested
              await this._scanService.clearAllScans();

              // Reload both profiles and scans to reflect changes
              await this.loadProfiles();
              await this.loadScans();

              // Track API success
              analytics.trackEvent('Security Profile Deleted Successfully', {
                profile_id: message.profileId,
                scan_history_cleared: true
              });

              if (this._view) {
                this._view.webview.postMessage({
                  type: 'deleteSuccess',
                  profileId: message.profileId
                });
              }
            } catch (error) {
              // Track API failure
              analytics.trackEvent('Security Profile Delete Failed', {
                error_type: 'deletion_error'
              });

              if (this._view) {
                this._view.webview.postMessage({
                  type: 'error',
                  message: 'Failed to delete profile and scan history'
                });
              }
            }
          }
          break;
        case 'rescanAll':
          try {
            // Implement logic to rescan all profiles
            await this._profileService.rescanAllProfiles();
            await this.loadProfiles();
          } catch (error) {
            if (this._view) {
              this._view.webview.postMessage({
                type: 'error',
                message: 'Failed to rescan profiles'
              });
            }
          }
          break;
        case 'checkOnboardingStatus':
          try {
            const settingsManager = new SettingsManager(this._context);
            const apiKey = await settingsManager.getApiKey();
            const model = settingsManager.getSelectedAIModel();

            const isConfigured = !!(apiKey && model);

            if (this._view) {
              this._view.webview.postMessage({
                type: 'onboardingStatus',
                isConfigured: isConfigured
              });
            }
          } catch (error) {
            if (this._view) {
              this._view.webview.postMessage({
                type: 'onboardingStatus',
                isConfigured: false
              });
            }
          }
          break;
        case 'saveConfig':
          console.log('SecureFlow: saveConfig message received', {
            provider: message.provider,
            model: message.model,
            hasApiKey: !!message.apiKey
          });

          analytics.trackEvent('Configuration Saved', {
            model: message.model,
            provider: message.provider
          });

          try {
            const config = vscode.workspace.getConfiguration('secureflow');
            if (message.provider) {
              console.log('SecureFlow: Updating provider to:', message.provider);
              await config.update(
                'Provider',
                message.provider,
                vscode.ConfigurationTarget.Global
              );
            }
            console.log('SecureFlow: Updating AIModel to:', message.model);
            await config.update(
              'AIModel',
              message.model,
              vscode.ConfigurationTarget.Global
            );

            // Verify it was saved
            const savedModel = config.get<string>('AIModel');
            console.log('SecureFlow: Verified AIModel saved as:', savedModel);

            console.log('SecureFlow: Updating APIKey');
            await config.update(
              'APIKey',
              message.apiKey,
              vscode.ConfigurationTarget.Global
            );

            console.log('SecureFlow: Configuration saved successfully');

            if (this._view) {
              this._view.webview.postMessage({
                type: 'configSaved',
                success: true
              });

              // Update onboarding status to configured
              this._view.webview.postMessage({
                type: 'onboardingStatus',
                isConfigured: true
              });
            }

            // Only start workspace scan if not skipped (e.g., from Settings page)
            if (!message.skipScan) {
              // Show success notification
              vscode.window.showInformationMessage(
                'SecureFlow configuration saved! Starting workspace security scan...'
              );

              // Automatically start workspace scan
              console.log('SecureFlow: Starting automatic workspace scan...');
              try {
                await this._profileService.scanWorkspace();
                await this.loadProfiles();
                console.log('SecureFlow: Automatic workspace scan completed');

                // Notify webview that scan is complete
                if (this._view) {
                  this._view.webview.postMessage({
                    type: 'scanComplete',
                    success: true,
                    profileCount: this._profiles.length
                  });

                  // Wait a moment for the success message to be visible
                  await new Promise(resolve => setTimeout(resolve, 2000));

                  // Send profiles data to update the view
                  this._view.webview.postMessage({
                    type: 'updateProfiles',
                    profiles: this._profiles
                  });
                }
              } catch (error) {
                console.error('SecureFlow: Workspace scan failed:', error);
              }
            } else {
              // Just show success notification for settings update
              vscode.window.showInformationMessage(
                'SecureFlow settings saved successfully!'
              );
            }
          } catch (error) {
            console.error('SecureFlow: Error saving configuration:', error);

            vscode.window.showErrorMessage(
              'Failed to save SecureFlow configuration. Please try again.'
            );

            if (this._view) {
              this._view.webview.postMessage({
                type: 'configSaved',
                success: false,
                error: 'Failed to save configuration'
              });
            }
          }
          break;
      }
    });

    const htmlContent = this._getHtmlContent(webviewView.webview);
    console.log('SecureFlow: HTML content generated, length:', htmlContent.length);
    console.log('SecureFlow: HTML preview:', htmlContent.substring(0, 200));
    webviewView.webview.html = htmlContent;
    console.log('SecureFlow: Webview HTML set successfully');
  }

  private openScanResultsWebview(scan: ScanResult) {
    // Create a new webview panel for displaying scan results
    const panel = vscode.window.createWebviewPanel(
      'secureflowScanResult',
      `SecureFlow Scan #${scan.scanNumber}`,
      vscode.ViewColumn.Two,
      {
        enableScripts: true,
        retainContextWhenHidden: true
      }
    );

    panel.webview.html = this.generateScanResultHtml(scan);
  }

  private openVulnerabilityDetailsWebview(vulnerability: any) {
    // Create a new webview panel for displaying vulnerability details
    const panel = vscode.window.createWebviewPanel(
      'secureflowVulnerabilityDetails',
      vulnerability.title,
      vscode.ViewColumn.Two,
      {
        enableScripts: true,
        retainContextWhenHidden: true
      }
    );

    // Handle messages from the webview
    panel.webview.onDidReceiveMessage(async (message) => {
      switch (message.type) {
        case 'openFile':
          try {
            const document = await vscode.workspace.openTextDocument(message.filePath);
            const editor = await vscode.window.showTextDocument(document, vscode.ViewColumn.One);
            const position = new vscode.Position(message.line - 1, 0);
            editor.selection = new vscode.Selection(position, position);
            editor.revealRange(new vscode.Range(position, position));
          } catch (error) {
            vscode.window.showErrorMessage(`Failed to open file: ${message.filePath}`);
          }
          break;
        case 'closeVulnerabilityDetails':
          panel.dispose();
          break;
      }
    });

    panel.webview.html = this.generateVulnerabilityDetailsHtml(panel.webview, vulnerability);
  }

  private generateVulnerabilityDetailsHtml(webview: vscode.Webview, vulnerability: any): string {
    const occurrencesHtml = vulnerability.occurrences
      .map((occurrence: any) => `
        <button class="occurrence-card" onclick="openFile('${occurrence.filePath}', ${occurrence.startLine})">
          <svg class="file-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
            <polyline points="13 2 13 9 20 9"></polyline>
          </svg>
          <div class="occurrence-details">
            <div class="occurrence-file">${this.getRelativePath(occurrence.filePath)}</div>
            <div class="occurrence-line">Line ${occurrence.startLine}</div>
          </div>
          <svg class="arrow-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </button>
      `)
      .join('');

    return `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${vulnerability.title}</title>
  <style>
    body {
      font-family: var(--vscode-font-family);
      font-size: var(--vscode-font-size);
      color: var(--vscode-foreground);
      background-color: var(--vscode-editor-background);
      margin: 0;
      padding: 20px;
    }
    .vulnerability-details-container {
      max-width: 900px;
      margin: 0 auto;
    }
    .details-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--vscode-widget-border);
    }
    .details-title {
      margin: 0;
      font-size: 20px;
      font-weight: 600;
    }
    .details-body {
      display: flex;
      flex-direction: column;
      gap: 24px;
    }
    .details-section {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .section-label {
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      opacity: 0.6;
      letter-spacing: 0.5px;
    }
    .section-value {
      font-size: 14px;
      line-height: 1.6;
    }
    .issue-title {
      font-size: 16px;
      font-weight: 600;
    }
    .recommendation {
      background: var(--vscode-textBlockQuote-background);
      border-left: 3px solid var(--vscode-button-background);
      padding: 12px;
      border-radius: 4px;
    }
    .severity-badge {
      display: inline-block;
      padding: 4px 12px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.3px;
      width: fit-content;
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
      background: rgba(59, 130, 246, 0.2);
      color: #93c5fd;
      border: 1px solid rgba(59, 130, 246, 0.3);
    }
    .occurrences-grid {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .occurrence-card {
      background: var(--vscode-button-secondaryBackground);
      border: 1px solid var(--vscode-widget-border);
      border-radius: 6px;
      padding: 12px;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 12px;
      transition: all 0.2s;
      font-family: var(--vscode-font-family);
      color: var(--vscode-foreground);
      text-align: left;
      width: 100%;
    }
    .occurrence-card:hover {
      border-color: var(--vscode-button-background);
      background: var(--vscode-button-background);
      color: var(--vscode-button-foreground);
      transform: translateX(4px);
    }
    .file-icon, .arrow-icon {
      flex-shrink: 0;
      stroke: currentColor;
    }
    .file-icon {
      width: 18px;
      height: 18px;
    }
    .arrow-icon {
      width: 16px;
      height: 16px;
      opacity: 0.5;
    }
    .occurrence-card:hover .arrow-icon {
      opacity: 1;
    }
    .occurrence-details {
      flex: 1;
      min-width: 0;
    }
    .occurrence-file {
      font-size: 13px;
      font-family: var(--vscode-editor-font-family);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      margin-bottom: 4px;
    }
    .occurrence-line {
      font-size: 11px;
      opacity: 0.7;
    }
  </style>
</head>
<body>
  <div class="vulnerability-details-container">
    <div class="details-header">
      <h2 class="details-title">Vulnerability Details</h2>
    </div>

    <div class="details-body">
      <div class="details-section">
        <div class="section-label">Issue</div>
        <div class="section-value issue-title">${vulnerability.title}</div>
      </div>

      <div class="details-section">
        <div class="section-label">Severity</div>
        <span class="severity-badge severity-${vulnerability.severity.toLowerCase()}">
          ${vulnerability.severity}
        </span>
      </div>

      <div class="details-section">
        <div class="section-label">Description</div>
        <div class="section-value">${vulnerability.description}</div>
      </div>

      <div class="details-section">
        <div class="section-label">Recommendation</div>
        <div class="section-value recommendation">${vulnerability.recommendation}</div>
      </div>

      <div class="details-section">
        <div class="section-label">Affected Locations (${vulnerability.occurrences.length})</div>
        <div class="occurrences-grid">
          ${occurrencesHtml}
        </div>
      </div>
    </div>
  </div>

  <script>
    const vscode = acquireVsCodeApi();

    function openFile(filePath, line) {
      vscode.postMessage({
        type: 'openFile',
        filePath: filePath,
        line: line
      });
    }
  </script>
</body>
</html>
    `;
  }

  private formatTimestamp(timestamp: number | string): string {
    const date = new Date(timestamp);
    const now = new Date();
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);

    const isToday = date.toDateString() === now.toDateString();
    const isYesterday = date.toDateString() === yesterday.toDateString();

    const timeStr = date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    });

    if (isToday) {
      return `Today at ${timeStr}`;
    } else if (isYesterday) {
      return `Yesterday at ${timeStr}`;
    } else {
      return date.toLocaleDateString('en-US', {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
      });
    }
  }

  private getRelativePath(absolutePath: string): string {
    if (!vscode.workspace.workspaceFolders) {
      return absolutePath;
    }

    for (const folder of vscode.workspace.workspaceFolders) {
      const relativePath = vscode.workspace.asRelativePath(absolutePath, false);
      if (relativePath !== absolutePath) {
        return relativePath;
      }
    }

    return absolutePath;
  }

  private generateScanResultHtml(scan: ScanResult): string {
    const issuesHtml = scan.issues
      .map(
        (item) => `
            <div class="issue">
                <div class="issue-header">
                    <span class="severity severity-${item.issue.severity.toLowerCase()}">${item.issue.severity}</span>
                    <h3 class="issue-title">${item.issue.title}</h3>
                </div>
                <div class="issue-meta">
                    <span class="file">${this.getRelativePath(item.filePath)}:${item.startLine}</span>
                </div>
                <p class="description">${item.issue.description}</p>
                <div class="recommendation">
                    <strong>Recommendation:</strong> ${item.issue.recommendation}
                </div>
            </div>
        `
      )
      .join('');

    return `
            <!DOCTYPE html>
            <html>
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>SecureFlow Scan #${scan.scanNumber}</title>
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
                        padding: 4px 8px; 
                        border-radius: 4px; 
                        font-size: 12px;
                        color: var(--vscode-textLink-foreground);
                        border: 1px solid rgba(255, 255, 255, 0.1);
                    }
                    .description { 
                        margin: 16px 0; 
                        line-height: 1.6;
                        color: var(--vscode-foreground);
                        font-size: 14px;
                    }
                    .recommendation { 
                        background: rgba(0, 122, 204, 0.1); 
                        padding: 16px; 
                        border-radius: 6px; 
                        margin-top: 16px; 
                        font-size: 14px;
                        line-height: 1.5;
                    }
                    .recommendation strong {
                        color: var(--vscode-textLink-foreground);
                        font-weight: 600;
                    }
                    .no-issues {
                        text-align: center;
                        padding: 40px;
                        color: var(--vscode-descriptionForeground);
                    }
                </style>
            </head>
            <body>
                <div class="header">
                    <h1>SecureFlow Scan #${scan.scanNumber}</h1>
                    <div class="scan-info">
                        <div class="info-item">
                            <span class="info-label">Scan Time</span>
                            <span class="info-value">${this.formatTimestamp(scan.timestamp)}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Files Analyzed</span>
                            <span class="info-value">${scan.fileCount}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Issues Found</span>
                            <span class="info-value">${scan.issues.length}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">AI Model</span>
                            <span class="info-value">${scan.model}</span>
                        </div>
                    </div>
                </div>
                
                ${
                  scan.issues.length > 0
                    ? `
                    <h2>Security Issues</h2>
                    ${issuesHtml}
                `
                    : '<div class="no-issues"><h2>✅ No Security Issues Found</h2><p>This scan completed successfully with no security issues detected.</p></div>'
                }
            </body>
            </html>
        `;
  }

  private _getHtmlContent(webview: vscode.Webview): string {
    try {
      // Get paths to Svelte webview resources
      const htmlPath = vscode.Uri.joinPath(
        this._extensionUri,
        'dist',
        'svelte-webview',
        'index.html'
      );
      const scriptPath = webview.asWebviewUri(
        vscode.Uri.joinPath(this._extensionUri, 'dist', 'svelte-webview', 'webview.js')
      );

      console.log('SecureFlow: Paths:', {
        htmlPath: htmlPath.fsPath,
        scriptPath: scriptPath.toString()
      });

      // Check if files exist
      if (!fs.existsSync(htmlPath.fsPath)) {
        throw new Error(`HTML file not found: ${htmlPath.fsPath}`);
      }

      const scriptFsPath = vscode.Uri.joinPath(this._extensionUri, 'dist', 'svelte-webview', 'webview.js').fsPath;
      if (!fs.existsSync(scriptFsPath)) {
        throw new Error(`Script file not found: ${scriptFsPath}`);
      }

      // Read the HTML template
      const htmlContent = fs.readFileSync(htmlPath.fsPath, 'utf-8');
      console.log('SecureFlow: HTML template read, length:', htmlContent.length);

      // Prepare model configuration for injection
      const modelConfigData = {
        models: ModelConfig.getAllActive(),
        providers: {
          openai: ModelConfig.getProviderInfo('openai'),
          anthropic: ModelConfig.getProviderInfo('anthropic'),
          google: ModelConfig.getProviderInfo('google'),
          xai: ModelConfig.getProviderInfo('xai'),
          ollama: ModelConfig.getProviderInfo('ollama'),
          openrouter: ModelConfig.getProviderInfo('openrouter')
        }
      };

      console.log('SecureFlow: Injecting model config:', {
        modelCount: modelConfigData.models.length,
        providers: Object.keys(modelConfigData.providers),
        openrouterInfo: modelConfigData.providers.openrouter
      });

      // Replace placeholders with actual values
      const html = htmlContent
        .replace(/\$\{cspSource\}/g, webview.cspSource)
        .replace(/\$\{scriptUri\}/g, scriptPath.toString())
        .replace(/\$\{modelConfig\}/g, JSON.stringify(modelConfigData));

      console.log('SecureFlow: HTML placeholders replaced');
      return html;
    } catch (error) {
      console.error('SecureFlow: Error loading webview:', error);
      // Return fallback HTML with error message
      return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body {
      font-family: var(--vscode-font-family);
      color: var(--vscode-foreground);
      background-color: var(--vscode-editor-background);
      padding: 20px;
    }
    .error {
      color: var(--vscode-errorForeground);
      background: var(--vscode-inputValidation-errorBackground);
      border: 1px solid var(--vscode-inputValidation-errorBorder);
      padding: 10px;
      border-radius: 4px;
    }
  </style>
</head>
<body>
  <h3>SecureFlow - Error Loading View</h3>
  <div class="error">
    <p>Failed to load SecureFlow webview.</p>
    <p>Error: ${error instanceof Error ? error.message : String(error)}</p>
    <p>Please check the Extension Host output for more details.</p>
  </div>
</body>
</html>`;
    }
  }
}
