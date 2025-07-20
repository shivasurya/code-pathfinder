import * as vscode from 'vscode';

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
        const provider = new SecureFlowWebViewProvider(context.extensionUri);
        context.subscriptions.push(
            vscode.window.registerWebviewViewProvider('secureflow.mainView', provider)
        );
    }
}

class SecureFlowWebViewProvider implements vscode.WebviewViewProvider {
    constructor(private readonly _extensionUri: vscode.Uri) {}

    public resolveWebviewView(
        webviewView: vscode.WebviewView,
        context: vscode.WebviewViewResolveContext,
        _token: vscode.CancellationToken,
    ) {
        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [this._extensionUri]
        };

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
                </style>
            </head>
            <body>
                <h1>Hello World!</h1>
            </body>
            </html>
        `;
    }
}