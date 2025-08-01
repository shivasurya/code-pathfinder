/**
 * Main Entry Point
 * Coordinates all modules and initializes the webview application
 */

// Import all modules
import { initializeMessageHandler } from './message-handler.js';
import { initializeProfileEventListeners } from './profile-manager.js';
import { initializeScanEventListeners } from './scan-manager.js';
import { initializeOnboardingEventListeners } from './onboarding.js';

// Initialize the application
(function() {
    // Get VSCode API
    const vscode = acquireVsCodeApi();
    
    // Make vscode available globally for modules that need it
    window.vscode = vscode;

    // Request initial data from extension
    vscode.postMessage({ type: 'getProfiles' });
    vscode.postMessage({ type: 'getScans' });
    vscode.postMessage({ type: 'checkOnboardingStatus' });

    // Initialize all modules
    initializeMessageHandler(vscode);
    initializeProfileEventListeners(vscode);
    initializeScanEventListeners(vscode);
    initializeOnboardingEventListeners(vscode);

    console.log('Code PathFinder webview initialized successfully');
})();
