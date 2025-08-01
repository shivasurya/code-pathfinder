/**
 * Message Handler Module
 * Centralized message handling from VSCode extension with routing to appropriate modules
 */

import { updateProfileList, displayProfileDetails, handleDeleteSuccess } from './profile-manager.js';
import { updateScanList, handleScanComplete } from './scan-manager.js';
import { handleOnboardingStatus, handleConfigSaved } from './onboarding.js';

// Initialize message handler
export function initializeMessageHandler(vscode) {
    // Handle messages from extension
    window.addEventListener('message', event => {
        const message = event.data;
        switch (message.type) {
            case 'scanComplete':
                handleScanComplete(message);
                break;
                
            case 'updateProfiles':
                updateProfileList(message.profiles, vscode);
                break;
                
            case 'updateScans':
                updateScanList(message.scans);
                break;
                
            case 'profileDetails':
                displayProfileDetails(message.profile, vscode);
                break;
                
            case 'error':
                handleError(message.message);
                break;
                
            case 'deleteSuccess':
                handleDeleteSuccess();
                break;
                
            case 'onboardingStatus':
                handleOnboardingStatus(message);
                break;
                
            case 'configSaved':
                handleConfigSaved(message);
                break;
                
            default:
                console.warn('Unknown message type:', message.type);
                break;
        }
    });
}

// Handle error messages
function handleError(errorMessage) {
    const profileContent = document.getElementById('profileContent');
    if (profileContent) {
        profileContent.innerHTML = '<div style="color: var(--vscode-errorForeground);">' + 
            errorMessage + '</div>';
    } else {
        console.error('Error from extension:', errorMessage);
    }
}
