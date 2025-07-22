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
            profileDetails.style.display = 'none';
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
            profileContent.innerHTML = '<div>Loading profile details...</div>';
            profileDetails.style.display = 'block';
        } else {
            profileDetails.style.display = 'none';
        }
    });
}());
