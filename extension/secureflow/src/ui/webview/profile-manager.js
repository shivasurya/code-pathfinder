/**
 * Profile Manager Module
 * Handles profile-related functionality including display, selection, and management
 */

import { formatLastUpdated, getTrashIcon } from './ui-helpers.js';
import { updateScanList, getGlobalScans } from './scan-manager.js';

// Initialize profile manager with required DOM elements
export function initializeProfileManager() {
    // Add a new container for the card if it doesn't exist
    const profileDetails = document.getElementById('profileDetails');
    let profileCardContainer = document.getElementById('profileCardContainer');
    if (!profileCardContainer) {
        profileCardContainer = document.createElement('div');
        profileCardContainer.id = 'profileCardContainer';
        profileDetails.parentNode.insertBefore(profileCardContainer, profileDetails.nextSibling);
    }
    return { profileDetails, profileCardContainer };
}

// Update profile dropdown list
export function updateProfileList(profiles, vscode) {
    const profileSelect = document.getElementById('profileSelect');
    const selectContainer = document.querySelector('.select-container');
    const emptyState = document.getElementById('emptyState');
    const { profileDetails, profileCardContainer } = initializeProfileManager();

    // Reset UI state
    profileDetails.style.display = 'none';
    profileCardContainer.style.display = 'none';
    profileSelect.innerHTML = '';

    if (!profiles || profiles.length === 0) {
        // No profiles - show empty state
        selectContainer.style.display = 'none';
        emptyState.style.display = 'flex';
        return;
    }

    // We have profiles - show select container and hide empty state
    selectContainer.style.display = 'flex';
    emptyState.style.display = 'none';

    // Add profile options and select first one
    profiles.forEach((profile, index) => {
        const option = document.createElement('option');
        option.value = profile.id;
        option.textContent = profile.name || 'Unnamed Profile';
        profileSelect.appendChild(option);

        // Auto-select first profile
        if (index === 0) {
            option.selected = true;
            // Trigger profile selection
            vscode.postMessage({
                type: 'profileSelected',
                profileId: profile.id
            });
        }
    });
}

// Display profile details in card format
export function displayProfileDetails(profile, vscode) {
    const { profileDetails, profileCardContainer } = initializeProfileManager();
    
    // Hide the old profile details container
    profileDetails.style.display = 'none';
    // Show the card in the new container
    profileCardContainer.style.display = 'block';
    profileCardContainer.innerHTML = `
        <div class="profile-card-naked">
            <div class="profile-card-header">
                <div class="profile-card-title">
                    <span class="profile-card-icon">🛡️</span>
                    <span class="profile-card-name">${profile.name}</span>
                </div>
                <div class="profile-card-actions">
                    <button id="rescanProfileCard" class="profile-action-btn" title="Rescan Profile">↻</button>
                    <button id="deleteProfileCard" class="profile-action-btn" title="Delete Profile">${getTrashIcon()}</button>
                </div>
            </div>
            <span class="profile-card-category">${profile.category}</span>
            <div class="profile-card-divider"></div>
            <div class="profile-card-body">
                <div class="profile-card-row">
                    <span class="profile-card-label">Subcategory</span>
                    <span class="profile-card-value">${profile.subcategory}</span>
                </div>
                <div class="profile-card-row">
                    <span class="profile-card-label">Technology</span>
                    <span class="profile-card-value">${profile.technology}</span>
                </div>
                <div class="profile-card-row">
                    <span class="profile-card-label">Path</span>
                    <span class="profile-card-value profile-card-path" title="${profile.path}">${profile.path}</span>
                </div>
                <div class="profile-card-row">
                    <span class="profile-card-label">Languages</span>
                    <span class="profile-card-value">${(profile.languages || []).map(lang => `<span class='profile-badge'>${lang}</span>`).join('')}</span>
                </div>
                <div class="profile-card-row">
                    <span class="profile-card-label">Confidence</span>
                    <span class="profile-card-value"><span class="profile-badge profile-badge-confidence">${profile.confidence.toFixed(1)}%</span></span>
                </div>
                <div class="profile-card-row">
                    <span class="profile-card-label">Last Updated</span>
                    <span class="profile-card-value">${formatLastUpdated(profile.timestamp)}</span>
                </div>
            </div>
        </div>
        <div class="profile-tabs-section">
            <div class="profile-tabs">
                <button class="profile-tab active" id="tab-history">Scan History</button>
            </div>
            <div class="profile-tab-content active" id="tabContent-history">
                <div id="scanList" class="scan-list">
                    <!-- Scan items will be dynamically added here -->
                </div>
                <div id="noScans" class="no-scans" style="display: none;">
                    <p>No scans yet. Run a scan to see the results here.</p>
                </div>
            </div>
        </div>
    `;
    
    // Attach event listeners to new action buttons
    const profileSelect = document.getElementById('profileSelect');
    document.getElementById('deleteProfileCard').onclick = () => {
        const selectedId = profileSelect.value;
        if (selectedId) {
            vscode.postMessage({
                type: 'confirmDelete',
                profileId: selectedId
            });
        }
    };
    
    document.getElementById('rescanProfileCard').onclick = () => {
        const selectedId = profileSelect.value;
        if (selectedId) {
            vscode.postMessage({
                type: 'rescanProfile',
                profileId: selectedId
            });
        }
    };
    
    // Now that the tab structure is created, update the scan list with stored data
    updateScanList(getGlobalScans());
}

// Show empty state when no profile is selected
export function showNoProfileDetails() {
    const profileDetails = document.getElementById('profileDetails');
    const profileCardContainer = document.getElementById('profileCardContainer');
    const emptyState = document.getElementById('emptyState');
    
    profileDetails.style.display = 'none';
    if (profileCardContainer) {
        profileCardContainer.style.display = 'none';
    }
    if (emptyState) {
        emptyState.style.display = 'flex';
    }
}

// Initialize profile-related event listeners
export function initializeProfileEventListeners(vscode) {
    const profileSelect = document.getElementById('profileSelect');
    
    // Handle profile selection change
    if (profileSelect) {
        profileSelect.addEventListener('change', (e) => {
            const selectedId = e.target.value;
            if (selectedId) {
                vscode.postMessage({
                    type: 'profileSelected',
                    profileId: selectedId
                });
            } else {
                showNoProfileDetails();
            }
        });
    }

    // Handle delete button click (for old UI)
    const deleteProfileButton = document.getElementById('deleteProfile');
    if (deleteProfileButton) {
        deleteProfileButton.addEventListener('click', () => {
            const selectedId = profileSelect.value;
            if (selectedId) {
                if (confirm('Are you sure you want to delete this profile? This action cannot be undone.')) {
                    vscode.postMessage({ 
                        type: 'deleteProfile', 
                        profileId: selectedId 
                    });
                }
            }
        });
    }

    // Handle rescan profile button click (for old UI)
    const rescanProfileButton = document.getElementById('rescanProfile');
    if (rescanProfileButton) {
        rescanProfileButton.addEventListener('click', () => {
            const selectedId = profileSelect.value;
            if (selectedId) {
                const button = document.getElementById('rescanProfile');
                const originalText = button.innerHTML;
                button.disabled = true;
                button.innerHTML = '⏳';
                
                vscode.postMessage({ 
                    type: 'rescanProfile', 
                    profileId: selectedId 
                });
                
                // Re-enable button after a delay
                setTimeout(() => {
                    button.disabled = false;
                    button.innerHTML = originalText;
                }, 3000);
            }
        });
    }
}

// Handle delete success message
export function handleDeleteSuccess() {
    const profileDetails = document.getElementById('profileDetails');
    const profileSelect = document.getElementById('profileSelect');
    
    profileDetails.style.display = 'none';
    profileSelect.value = '';
}
