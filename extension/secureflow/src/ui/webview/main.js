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
            case 'scanComplete':
                const button = document.getElementById('scanWorkspace');
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
                break;
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
        const selectContainer = document.querySelector('.select-container');
        const emptyState = document.getElementById('emptyState');
        const profileDetails = document.getElementById('profileDetails');

        // Clear existing profile options
        while (profileSelect.options.length > 1) {
            profileSelect.remove(1);
        }

        if (!profiles || profiles.length === 0) {
            selectContainer.style.display = 'none';
            emptyState.style.display = 'flex';
            showNoProfileDetails();
            return;
        }

        selectContainer.style.display = 'flex';
        emptyState.style.display = 'none';

        // Add new profile options
        profiles.forEach(profile => {
            const option = document.createElement('option');
            option.value = profile.id;
            option.textContent = profile.name || 'Unnamed Profile';
            profileSelect.appendChild(option);
        });

        // Auto-select first profile if available
        profileSelect.value = profiles[0].id;
        vscode.postMessage({
            type: 'profileSelected',
            profileId: profiles[0].id
        });
    }

    // Add a new container for the card if it doesn't exist
    let profileCardContainer = document.getElementById('profileCardContainer');
    if (!profileCardContainer) {
        profileCardContainer = document.createElement('div');
        profileCardContainer.id = 'profileCardContainer';
        profileDetails.parentNode.insertBefore(profileCardContainer, profileDetails.nextSibling);
    }

    // Display profile details
    function displayProfileDetails(profile) {
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
        `;
        // Attach event listeners to new action buttons
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
    }

    function formatLastUpdated(ts) {
        // Accepts either a string or a Date/number
        let date = ts instanceof Date ? ts : new Date(ts);
        const now = new Date();
        const diffMs = now - date;
        const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
        if (diffDays === 0) {
            // Today
            return `Today, ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
        } else if (diffDays === 1) {
            return `Yesterday, ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
        } else if (now.getFullYear() === date.getFullYear()) {
            // This year, show month/day
            return date.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ', ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        } else {
            // Previous years
            return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' });
        }
    }

    // When no profile is selected, show the old details container and hide the card
    function showNoProfileDetails() {
        profileDetails.style.display = 'block';
        profileCardContainer.style.display = 'none';
    }

    function getTrashIcon() {
        // SVG trash/dustbin icon, inherits color
        return `<svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M7.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M12.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><rect x="4.5" y="5.5" width="11" height="11" rx="2" stroke="currentColor" stroke-width="1.5"/><path d="M2.5 5.5H17.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M8.5 2.5H11.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`;
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

    // Handle rescan profile button click
    document.getElementById('rescanProfile').addEventListener('click', () => {
        const selectedId = profileSelect.value;
        if (selectedId) {
            vscode.postMessage({
                type: 'rescanProfile',
                profileId: selectedId
            });
        }
    });

    // Handle scan all button click
    document.getElementById('scanAll').addEventListener('click', () => {
        vscode.postMessage({ type: 'rescanAll' });
    });

    // Handle scan workspace button click
    document.getElementById('scanWorkspace').addEventListener('click', () => {
        const button = document.getElementById('scanWorkspace');
        button.disabled = true;
        button.innerHTML = '<span style="font-family: codicon; animation: spin 1s linear infinite;">↻</span> Analyzing...';
        
        vscode.postMessage({ type: 'scanWorkspace' });
        
        // Show loading state in empty state
        const emptyState = document.getElementById('emptyState');
        const originalContent = emptyState.innerHTML;
        emptyState.innerHTML = 
            '<div class="empty-state-icon" style="animation: spin 1s linear infinite;">↻</div>' +
            '<div>' +
                '<h3 style="margin: 0 0 8px 0;">Scanning Workspace</h3>' +
                '<p>Analyzing your project for security patterns...</p>' +
            '</div>';
            
        // Add timeout to restore button state if no response
        setTimeout(() => {
            if (button.disabled) {
                button.disabled = false;
                button.innerHTML = '<span style="font-family: codicon;">⚡</span> Secure Your Workspace';
                emptyState.innerHTML = originalContent;
            }
        }, 30000); // 30 second timeout
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
            profileCardContainer.innerHTML = '<div>Loading profile details...</div>';
            profileCardContainer.style.display = 'block';
            profileDetails.style.display = 'none';
        } else {
            showNoProfileDetails();
        }
    });
}());
