import * as vscode from 'vscode';
import { ScanStorageService } from '../services/scan-storage-service';
import { SentryService } from '../services/sentry-service';

/**
 * Register scan-related commands
 */
export function registerScanCommands(context: vscode.ExtensionContext): void {
  const scanService = new ScanStorageService(context);
  const sentry = SentryService.getInstance();

  // Command to retrieve a scan by number
  const retrieveScanCommand = vscode.commands.registerCommand(
    'secureflow.retrieveScan',
    sentry.withErrorHandling('secureflow.retrieveScan', async () => {
      const scanNumber = await vscode.window.showInputBox({
        prompt: 'Enter scan number to retrieve',
        placeHolder: 'e.g., 1',
        validateInput: (value) => {
          const num = parseInt(value);
          if (isNaN(num) || num < 1) {
            return 'Please enter a valid scan number (1 or greater)';
          }
          return null;
        }
      });

      if (scanNumber) {
        const num = parseInt(scanNumber);
        const scan = scanService.getScanByNumber(num);

        if (scan) {
          // Create a new webview to display the scan results
          const panel = vscode.window.createWebviewPanel(
            'secureflowScanResult',
            `SecureFlow Scan #${scan.scanNumber}`,
            vscode.ViewColumn.One,
            { enableScripts: true }
          );

          panel.webview.html = generateScanResultHtml(scan);
        } else {
          vscode.window.showErrorMessage(`Scan #${num} not found.`);
        }
      }
    })
  );

  // Command to list all scans
  const listScansCommand = vscode.commands.registerCommand(
    'secureflow.listScans',
    sentry.withErrorHandling('secureflow.listScans', async () => {
      const scans = scanService.getAllScans();
      const stats = scanService.getStats();

      if (scans.length === 0) {
        vscode.window.showInformationMessage('No scans found.');
        return;
      }

      const items = scans.map((scan) => ({
        label: `Scan #${scan.scanNumber}`,
        description: `${scan.issues.length} issues - ${scan.timestampFormatted}`,
        detail: `${scan.fileCount} files analyzed with ${scan.model}`,
        scanNumber: scan.scanNumber
      }));

      const selected = await vscode.window.showQuickPick(items, {
        placeHolder: `Select a scan to view (${stats.totalScans} total scans)`
      });

      if (selected) {
        vscode.commands.executeCommand('secureflow.retrieveScan');
        // Pre-fill the input with the selected scan number
        setTimeout(() => {
          vscode.window.showInputBox({
            prompt: 'Enter scan number to retrieve',
            value: selected.scanNumber.toString()
          });
        }, 100);
      }
    })
  );

  context.subscriptions.push(retrieveScanCommand, listScansCommand);
}

/**
 * Generate HTML for displaying scan results
 */
function generateScanResultHtml(scan: any): string {
  const issuesHtml = scan.issues
    .map(
      (item: any) => `
        <div class="issue">
            <h3 class="issue-title severity-${item.issue.severity.toLowerCase()}">${item.issue.title}</h3>
            <div class="issue-meta">
                <span class="severity">${item.issue.severity}</span>
                <span class="file">${item.filePath}:${item.startLine}</span>
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
                body { font-family: var(--vscode-font-family); padding: 20px; }
                .header { border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 15px; margin-bottom: 20px; }
                .scan-info { display: flex; gap: 20px; margin-bottom: 10px; }
                .info-item { display: flex; flex-direction: column; }
                .info-label { font-size: 12px; color: var(--vscode-descriptionForeground); }
                .info-value { font-weight: bold; }
                .issue { border: 1px solid var(--vscode-panel-border); border-radius: 4px; padding: 15px; margin-bottom: 15px; }
                .issue-title { margin: 0 0 10px 0; }
                .issue-meta { display: flex; gap: 15px; margin-bottom: 10px; font-size: 12px; }
                .severity { padding: 2px 6px; border-radius: 3px; font-weight: bold; }
                .severity-critical { background: #ff4444; color: white; }
                .severity-high { background: #ff8800; color: white; }
                .severity-medium { background: #ffaa00; color: black; }
                .severity-low { background: #88cc00; color: black; }
                .file { font-family: monospace; background: var(--vscode-editor-background); padding: 2px 6px; border-radius: 3px; }
                .description { margin: 10px 0; }
                .recommendation { background: var(--vscode-editor-inactiveSelectionBackground); padding: 10px; border-radius: 4px; margin-top: 10px; }
            </style>
        </head>
        <body>
            <div class="header">
                <h1>SecureFlow Scan #${scan.scanNumber}</h1>
                <div class="scan-info">
                    <div class="info-item">
                        <span class="info-label">Timestamp</span>
                        <span class="info-value">${scan.timestampFormatted}</span>
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
                : '<p>No security issues found in this scan.</p>'
            }
        </body>
        </html>
    `;
}
