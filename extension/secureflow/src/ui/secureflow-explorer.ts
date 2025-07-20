import * as vscode from 'vscode';
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
            }
        });

        webviewView.webview.html = this._getHtmlContent();
    }

    private _getHtmlContent() {
        return `
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>SecureFlow</title>
                <style>
                    body {
                        padding: 20px;
                        color: var(--vscode-foreground);
                        font-family: var(--vscode-font-family);
                        background-color: var(--vscode-editor-background);
                    }
                    h1 {
                        font-size: 1.2em;
                        margin-bottom: 1em;
                        color: var(--vscode-editor-foreground);
                    }
                    select {
                        width: 100%;
                        padding: 8px;
                        margin-bottom: 16px;
                        background: var(--vscode-input-background);
                        color: var(--vscode-input-foreground);
                        border: 1px solid var(--vscode-input-border);
                        border-radius: 2px;
                    }
                    .profile-details {
                        background: var(--vscode-editor-background);
                        border: 1px solid var(--vscode-panel-border);
                        padding: 12px;
                        margin-top: 12px;
                        border-radius: 4px;
                    }
                    .profile-details h2 {
                        font-size: 1.1em;
                        margin-top: 0;
                        margin-bottom: 8px;
                    }
                    .profile-details div {
                        margin-bottom: 8px;
                    }
                </style>
            </head>
            <body>
                <h1>SecureFlow Profiles</h1>
                <select id="profileSelect">
                    <option value="">Select a profile...</option>
                </select>
                <div id="profileDetails" class="profile-details" style="display: none;">
                    <h2>Profile Details</h2>
                    <div id="profileContent"></div>
                </div>
                <script>
                    (function() {
                        const vscode = acquireVsCodeApi();
                        const profileSelect = document.getElementById('profileSelect');
                        const profileDetails = document.getElementById('profileDetails');
                        const profileContent = document.getElementById('profileContent');

                        // Request initial profiles
                        vscode.postMessage({ type: 'getProfiles' });

                        // Handle messages from extension
                        window.addEventListener('message', event => {
                            const message = event.data;
                            switch (message.type) {
                                case 'updateProfiles':
                                    updateProfileList(message.profiles);
                                    break;
                                case 'profileDetails':
                                    displayProfileDetails(message.profile);
                                    break;
                                case 'error':
                                    profileContent.innerHTML = '<div style="color: var(--vscode-errorForeground);">' + 
                                        message.message + '</div>';
                                    break;
                            }
                        });

                        // Update profile dropdown
                        function updateProfileList(profiles) {
                            // Clear existing profile options
                            while (profileSelect.options.length > 1) {
                                profileSelect.remove(1);
                            }

                            // Add new profile options
                            profiles.forEach(profile => {
                                const option = document.createElement('option');
                                option.value = profile.id;
                                option.textContent = profile.name || 'Unnamed Profile';
                                profileSelect.appendChild(option);
                            });
                        }

                        // Display profile details
                        function displayProfileDetails(profile) {
                            profileDetails.style.display = 'block';
                            profileContent.innerHTML = 
                                '<div><strong>ID:</strong> ' + profile.id + '</div>' +
                                '<div><strong>Name:</strong> ' + profile.name + '</div>' +
                                '<div><strong>Category:</strong> ' + profile.category + '</div>' +
                                '<div><strong>Subcategory:</strong> ' + profile.subcategory + '</div>' +
                                '<div><strong>Technology:</strong> ' + profile.technology + '</div>' +
                                '<div><strong>Path:</strong> ' + profile.path + '</div>' +
                                '<div><strong>Languages:</strong> ' + profile.languages.join(', ') + '</div>' +
                                '<div><strong>Confidence:</strong> ' + (profile.confidence).toFixed(1) + '%</div>' +
                                '<div><strong>Status:</strong> ' + (profile.isActive ? 'Active' : 'Inactive') + '</div>' +
                                '<div><strong>Last Updated:</strong> ' + profile.timestamp + '</div>';
                        }

                        // Handle profile selection
                        profileSelect.addEventListener('change', () => {
                            const selectedId = profileSelect.value;
                            if (selectedId) {
                                vscode.postMessage({
                                    type: 'profileSelected',
                                    profileId: selectedId
                                });
                                
                                // Show loading state
                                profileContent.innerHTML = '<div>Loading profile details...</div>';
                                profileDetails.style.display = 'block';
                            } else {
                                profileDetails.style.display = 'none';
                            }
                        });
                    }())
                </script>
            </body>
            </html>
        `;
    }
}