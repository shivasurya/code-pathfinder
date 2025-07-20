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
                        border-radius: 6px;
                    }
                    .profile-details h2 {
                        font-size: 1.1em;
                        margin: 0 0 12px 0;
                        padding-bottom: 8px;
                        border-bottom: 1px solid var(--vscode-panel-border);
                        display: flex;
                        justify-content: space-between;
                        align-items: center;
                    }
                    .profile-details .detail-row {
                        display: flex;
                        margin-bottom: 4px;
                        padding: 6px 8px;
                        border-radius: 3px;
                        transition: background-color 0.1s ease;
                    }
                    .profile-details .detail-row:last-child {
                        margin-bottom: 0;
                    }
                    .profile-details .detail-row:hover {
                        background: var(--vscode-list-hoverBackground);
                    }
                    .profile-details .label {
                        flex: 0 0 100px;
                        color: var(--vscode-foreground);
                        opacity: 0.7;
                        font-size: 0.95em;
                        padding-right: 8px;
                    }
                    .profile-details .value {
                        flex: 1;
                        color: var(--vscode-foreground);
                        font-size: 0.95em;
                        line-height: 1.4;
                    }
                    .profile-details .badge {
                        display: inline-block;
                        padding: 2px 6px;
                        border-radius: 3px;
                        font-size: 0.85em;
                        background: var(--vscode-badge-background);
                        color: var(--vscode-badge-foreground);
                        margin: 0 4px 4px 0;
                    }
                    .delete-btn {
                        background: none;
                        border: none;
                        color: var(--vscode-errorForeground);
                        cursor: pointer;
                        padding: 4px 8px;
                        border-radius: 4px;
                        display: flex;
                        align-items: center;
                        gap: 4px;
                        font-size: 12px;
                    }
                    .delete-btn:hover {
                        background: var(--vscode-errorForeground);
                        color: var(--vscode-editor-background);
                    }
                    .delete-btn span {
                        font-family: codicon;
                        font-size: 14px;
                    }
                </style>
            </head>
            <body>
                <h1>SecureFlow Profiles</h1>
                <select id="profileSelect">
                    <option value="">Select a profile...</option>
                </select>
                <div id="profileDetails" class="profile-details" style="display: none;">
                    <h2>
                        Profile Details
                        <button id="deleteProfile" class="delete-btn" style="display: none;">
                            <span>✖</span> Delete Profile
                        </button>
                    </h2>
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
                                case 'deleteSuccess':
                                    profileDetails.style.display = 'none';
                                    profileSelect.value = '';
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

                            // Auto-select first profile if available
                            if (profiles.length > 0) {
                                profileSelect.value = profiles[0].id;
                                vscode.postMessage({
                                    type: 'profileSelected',
                                    profileId: profiles[0].id
                                });
                            }
                        }

                        // Display profile details
                        function displayProfileDetails(profile) {
                            profileDetails.style.display = 'block';
                            const deleteBtn = document.getElementById('deleteProfile');
                            deleteBtn.style.display = 'flex';

                            const createDetailRow = (label, value, type = 'text') => {
                                let valueHtml = value;
                                if (type === 'array' && Array.isArray(value)) {
                                    valueHtml = value.map(v => '<span class="badge">' + v + '</span>').join('');
                                } else if (type === 'percentage') {
                                    valueHtml = '<span class="badge">' + value.toFixed(1) + '%</span>';
                                }
                                return '<div class="detail-row">' +
                                    '<div class="label">' + label + '</div>' +
                                    '<div class="value">' + valueHtml + '</div>' +
                                    '</div>';
                            };

                            profileContent.innerHTML = 
                                createDetailRow('Name', profile.name) +
                                createDetailRow('Category', profile.category) +
                                createDetailRow('Subcategory', profile.subcategory) +
                                createDetailRow('Technology', profile.technology) +
                                createDetailRow('Path', profile.path) +
                                createDetailRow('Languages', profile.languages, 'array') +
                                createDetailRow('Confidence', profile.confidence, 'percentage') +
                                createDetailRow('Last Updated', profile.timestamp);
                        }

                        // Handle delete button click
                        document.getElementById('deleteProfile').addEventListener('click', () => {
                            const selectedId = profileSelect.value;
                            if (selectedId) {
                                vscode.postMessage({
                                    type: 'confirmDelete',
                                    profileId: selectedId
                                });
                            }
                        });

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