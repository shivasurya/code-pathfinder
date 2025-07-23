import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import { StoredProfile } from '../models/profile-store';
import { ProfileStorageService } from '../services/profile-storage-service';

export class SecureFlowExplorer {
    private static instance: SecureFlowExplorer;
    private _view?: vscode.WebviewView;

    private constructor(private readonly context: vscode.ExtensionContext) {}

    public static getInstance(context: vscode.ExtensionContext): SecureFlowExplorer {
        if (!SecureFlowExplorer.instance) {
            SecureFlowExplorer.instance = new SecureFlowExplorer(context);
        }
        return SecureFlowExplorer.instance;
    }

    public static register(context: vscode.ExtensionContext): void {
        const provider = new SecureFlowWebViewProvider(context.extensionUri, context);
        context.subscriptions.push(
            vscode.window.registerWebviewViewProvider('secureflow.mainView', provider)
        );
    }
}

class SecureFlowWebViewProvider implements vscode.WebviewViewProvider {
    private _view?: vscode.WebviewView;
    private _profiles: StoredProfile[] = [];
    private _profileService: ProfileStorageService;

    constructor(
        private readonly _extensionUri: vscode.Uri,
        private readonly _context: vscode.ExtensionContext
    ) {
        this._profileService = new ProfileStorageService(this._context);
        this.loadProfiles();
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

    public resolveWebviewView(
        webviewView: vscode.WebviewView,
        context: vscode.WebviewViewResolveContext,
        _token: vscode.CancellationToken,
    ) {
        this._view = webviewView;
        
        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [this._extensionUri]
        };

        webviewView.webview.onDidReceiveMessage(async (message) => {
            switch (message.type) {
                case 'scanWorkspace':
                    try {
                        // Start the workspace scan
                        await this._profileService.scanWorkspace();
                        // Load the new profiles
                        await this.loadProfiles();
                        // Update the UI with success state
                        if (this._view) {
                            this._view.webview.postMessage({
                                type: 'scanComplete',
                                success: true
                            });
                        }
                    } catch (error) {
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
                    await this.loadProfiles();
                    break;
                case 'profileSelected':
                    const profile = this._profiles.find(p => p.id === message.profileId);
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
                    const answer = await vscode.window.showWarningMessage(
                        'Are you sure you want to delete this profile?',
                        { modal: true },
                        'Delete',
                        'Cancel'
                    );
                    if (answer === 'Delete') {
                        try {
                            await this._profileService.deleteProfile(message.profileId);
                            await this.loadProfiles();
                            if (this._view) {
                                this._view.webview.postMessage({
                                    type: 'deleteSuccess',
                                    profileId: message.profileId
                                });
                            }
                        } catch (error) {
                            if (this._view) {
                                this._view.webview.postMessage({
                                    type: 'error',
                                    message: 'Failed to delete profile'
                                });
                            }
                        }
                    }
                    break;
                case 'rescanProfile':
                    try {
                        const profile = this._profiles.find(p => p.id === message.profileId);
                        if (profile) {
                            // Trigger a rescan for the specific profile path
                            // You'll need to implement the actual rescan logic in your service
                            await this._profileService.rescanProfile(profile);
                            await this.loadProfiles();
                        }
                    } catch (error) {
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
            }
        });

        webviewView.webview.html = this._getHtmlContent(webviewView.webview);
    }

    private _getHtmlContent(webview: vscode.Webview) {
        // Get file paths for resources
        const iconPath = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'resources', 'icon.png'));
        const cssPath = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'src', 'ui', 'webview', 'styles.css'));
        const jsPath = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'src', 'ui', 'webview', 'main.js'));
        
        // Read the HTML template
        const htmlTemplate = fs.readFileSync(
            path.join(this._extensionUri.fsPath, 'src', 'ui', 'webview', 'index.html'),
            'utf8'
        );
        
        // Replace placeholders with actual values
        return htmlTemplate
            .replace('{{iconPath}}', iconPath.toString())
            .replace('{{cssPath}}', cssPath.toString())
            .replace('{{jsPath}}', jsPath.toString());
    }
}