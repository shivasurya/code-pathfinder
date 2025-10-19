import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import { StoredProfile } from '../models/profile-store';
import { ProfileStorageService } from '../services/profile-storage-service';
import { ScanStorageService } from '../services/scan-storage-service';
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

  constructor(
    private readonly _extensionUri: vscode.Uri,
    private readonly _context: vscode.ExtensionContext
  ) {
    this._profileService = new ProfileStorageService(this._context);
    this._scanService = new ScanStorageService(this._context);
    this.loadProfiles();
    this.loadScans();
  }

  private async loadProfiles() {
    try {
      const profiles = await this._profileService.getAllProfiles();
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
    this._view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this._extensionUri]
    };

    webviewView.webview.onDidReceiveMessage(async (message) => {
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
        case 'viewScan':
          analytics.trackEvent('Scan Results Opened', {
            scan_number: message.scanNumber
          });
          const scan = this._scans.find(
            (s) => s.scanNumber === message.scanNumber
          );
          if (scan) {
            this.openScanResultsWebview(scan);
          }
          break;
        case 'profileSelected':
          analytics.trackEvent('Security Profile Selected', {
            profile_action: 'view_details'
          });
          const profile = this._profiles.find(
            (p) => p.id === message.profileId
          );
          if (profile && this._view) {
            this._view.webview.postMessage({
              type: 'profileDetails',
              profile: {
                id: profile.id,
                name: profile.name,
                path: profile.path,
                category: profile.category,
                subcategory: profile.subcategory || 'N/A',
                technology: profile.technology || 'N/A',
                confidence: profile.confidence,
                languages: profile.languages || [],
                isActive: profile.isActive,
                timestamp: new Date(profile.timestamp).toLocaleString()
              }
            });
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
        case 'rescanProfile':
          analytics.trackEvent('Security Profile Rescan Started', {
            rescan_trigger: 'manual_request'
          });

          try {
            const profile = this._profiles.find(
              (p) => p.id === message.profileId
            );
            if (profile) {
              // Trigger a rescan for the specific profile path
              // You'll need to implement the actual rescan logic in your service
              await this._profileService.rescanProfile(profile);
              await this.loadProfiles();

              // Track API success
              analytics.trackEvent('Security Profile Rescan Completed', {
                profile_id: message.profileId,
                rescan_result: 'success'
              });
            }
          } catch (error) {
            // Track API failure
            analytics.trackEvent('Security Profile Rescan Failed', {
              error_type: 'rescan_execution_error'
            });

            if (this._view) {
              this._view.webview.postMessage({
                type: 'error',
                message: 'Failed to rescan profile'
              });
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
          analytics.trackEvent('Configuration Saved', {
            model: message.model
          });

          try {
            const config = vscode.workspace.getConfiguration('secureflow');
            await config.update(
              'AIModel',
              message.model,
              vscode.ConfigurationTarget.Global
            );
            await config.update(
              'APIKey',
              message.apiKey,
              vscode.ConfigurationTarget.Global
            );

            if (this._view) {
              this._view.webview.postMessage({
                type: 'configSaved',
                success: true
              });
            }
          } catch (error) {
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

    webviewView.webview.html = this._getHtmlContent(webviewView.webview);
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
    // Get paths to resource files
    const htmlPath = vscode.Uri.joinPath(
      this._extensionUri,
      'dist',
      'webview',
      'index.html'
    );
    const mainScriptPath = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, 'dist', 'webview', 'main.js')
    );
    const stylesPath = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, 'dist', 'webview', 'styles.css')
    );
    const iconPath = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, 'resources', 'icon.png')
    );

    // Read and return the HTML content
    const htmlContent = fs.readFileSync(htmlPath.fsPath, 'utf-8');

    // Prepare model configuration for injection
    const modelConfigData = {
      models: ModelConfig.getAllActive(),
      providers: {
        openai: ModelConfig.getProviderInfo('openai'),
        anthropic: ModelConfig.getProviderInfo('anthropic'),
        google: ModelConfig.getProviderInfo('google'),
        xai: ModelConfig.getProviderInfo('xai'),
        ollama: ModelConfig.getProviderInfo('ollama')
      }
    };

    // Create inline script to inject model config before main script loads
    const modelConfigScript = `
      <script>
        window.modelConfig = ${JSON.stringify(modelConfigData)};
      </script>
    `;

    // Replace placeholders with actual URIs
    let data = htmlContent
      .replace(/\$\{scriptUri\}/g, mainScriptPath.toString())
      .replace(/\$\{stylesUri\}/g, stylesPath.toString())
      .replace(/\$\{iconUri\}/g, iconPath.toString())
      .replace(/\$\{cspSource\}/g, webview.cspSource);
    
    // Inject model config script before the main script tag
    data = data.replace('</head>', `${modelConfigScript}</head>`);
    
    return data;
  }
}
