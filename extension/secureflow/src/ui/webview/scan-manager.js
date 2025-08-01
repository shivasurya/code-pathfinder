/**
 * Scan Manager Module
 * Handles scan-related functionality including display, management, and interactions
 */

import { formatTimestamp } from './ui-helpers.js';

// Store scans globally so they can be used when profile details are shown
let globalScans = [];

// Update scan list display
export function updateScanList(scans) {
    globalScans = scans || [];
    
    const scanList = document.getElementById('scanList');
    const noScans = document.getElementById('noScans');
    
    // If elements don't exist yet (profile not selected), just store the data
    if (!scanList || !noScans) {
        return;
    }
    
    if (!scans || scans.length === 0) {
        noScans.style.display = 'block';
        // Clear any existing scan items
        const existingItems = scanList.querySelectorAll('.scan-item');
        existingItems.forEach(item => item.remove());
        return;
    }
    
    noScans.style.display = 'none';
    
    // Clear existing scan items
    const existingItems = scanList.querySelectorAll('.scan-item');
    existingItems.forEach(item => item.remove());
    
    // Show only the 5 most recent scans
    const recentScans = scans.slice(0, 5);
    
    recentScans.forEach(scan => {
        const scanItem = document.createElement('div');
        scanItem.className = 'scan-item';
        
        const issueCount = scan.issues.length;
        const issueClass = issueCount > 0 ? 'has-issues' : 'no-issues';
        const issueText = issueCount > 0 ? `${issueCount} issue${issueCount > 1 ? 's' : ''}` : 'No issues';
        
        scanItem.innerHTML = `
            <div class="scan-info">
                <div class="scan-title">Scan #${scan.scanNumber}</div>
                <div class="scan-meta">
                    <span>${formatTimestamp(scan.timestamp)}</span>
                    <span>${scan.fileCount} files</span>
                    <span class="scan-issues ${issueClass}">${issueText}</span>
                </div>
            </div>
            <div class="scan-actions">
                <button class="view-scan-btn" onclick="viewScan(${scan.scanNumber})">
                    <span style="font-family: codicon;">→</span> View
                </button>
            </div>
        `;
        
        scanList.appendChild(scanItem);
    });
}

// View scan function
export function viewScan(scanNumber) {
    // Get vscode from global scope
    const vscode = window.vscode;
    vscode.postMessage({ type: 'viewScan', scanNumber: scanNumber });
}

// Get stored scans
export function getGlobalScans() {
    return globalScans;
}

// Initialize scan-related event listeners
export function initializeScanEventListeners(vscode) {
    // Handle scan all button click
    const scanAllButton = document.getElementById('scanAll');
    if (scanAllButton) {
        scanAllButton.addEventListener('click', () => {
            const button = document.getElementById('scanAll');
            const originalText = button.innerHTML;
            button.disabled = true;
            button.innerHTML = '⏳';
            
            vscode.postMessage({ type: 'scanAll' });
            
            // Re-enable button after a delay
            setTimeout(() => {
                button.disabled = false;
                button.innerHTML = originalText;
            }, 3000);
        });
    }

    // Handle scan workspace button click
    const scanWorkspaceButton = document.getElementById('scanWorkspace');
    if (scanWorkspaceButton) {
        scanWorkspaceButton.addEventListener('click', () => {
            const button = document.getElementById('scanWorkspace');
            button.disabled = true;
            button.innerHTML = '<span style="font-family: codicon;">⏳</span> Scanning...';
            vscode.postMessage({ type: 'scanWorkspace' });
        });
    }
}

// Handle scan complete message
export function handleScanComplete(message) {
    const button = document.getElementById('scanWorkspace');
    if (button) {
        button.disabled = false;
        if (message.success) {
            button.innerHTML = '<span style="font-family: codicon;">✓</span> Scan Complete';
            setTimeout(() => {
                button.innerHTML = '<span style="font-family: codicon;">⚡</span> Secure Your Workspace';
            }, 2000);
        } else {
            button.innerHTML = '<span style="font-family: codicon;">✖</span> Scan Failed';
            setTimeout(() => {
                button.innerHTML = '<span style="font-family: codicon;">⚡</span> Try Again';
            }, 2000);
        }
    }
}

// Make viewScan available globally for onclick handlers
window.viewScan = viewScan;
