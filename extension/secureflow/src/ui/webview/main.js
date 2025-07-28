(function() {
    const vscode = acquireVsCodeApi();
    const profileSelect = document.getElementById('profileSelect');
    const profileDetails = document.getElementById('profileDetails');
    const profileContent = document.getElementById('profileContent');

    // Onboarding state
    let currentStep = 1;
    let selectedModel = 'claude-3-5-sonnet-20241022';
    let apiKey = '';

    // Request initial profiles and scans
    vscode.postMessage({ type: 'getProfiles' });
    vscode.postMessage({ type: 'getScans' });
    
    // Check if onboarding is needed
    vscode.postMessage({ type: 'checkOnboardingStatus' });

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
            case 'updateScans':
                updateScanList(message.scans);
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
            case 'onboardingStatus':
                if (message.isConfigured) {
                    // Skip onboarding if already configured
                    showStep(4);
                } else {
                    // Show step 1 for new users
                    showStep(1);
                }
                break;
            case 'configSaved':
                if (message.success) {
                    // Move to final step after successful config save
                    showStep(4);
                    updateConfigSummary();
                } else {
                    alert('Failed to save configuration: ' + message.error);
                }
                break;
        }
    });

    // Update profile dropdown
    function updateProfileList(profiles) {
        const selectContainer = document.querySelector('.select-container');
        const emptyState = document.getElementById('emptyState');

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

    // Add a new container for the card if it doesn't exist
    let profileCardContainer = document.getElementById('profileCardContainer');
    if (!profileCardContainer) {
        profileCardContainer = document.createElement('div');
        profileCardContainer.id = 'profileCardContainer';
        profileDetails.parentNode.insertBefore(profileCardContainer, profileDetails.nextSibling);
    }

    // Format timestamp to human readable format
    function formatTimestamp(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const yesterday = new Date(now);
        yesterday.setDate(yesterday.getDate() - 1);

        const isToday = date.toDateString() === now.toDateString();
        const isYesterday = date.toDateString() === yesterday.toDateString();

        const timeStr = date.toLocaleTimeString('en-US', { 
            hour: 'numeric', 
            minute: '2-digit',
            hour12: true 
        });

        if (isToday) {
            return `Today at ${timeStr}`;
        } else if (isYesterday) {
            return `Yesterday at ${timeStr}`;
        } else {
            return date.toLocaleDateString('en-US', { 
                weekday: 'short',
                month: 'short', 
                day: 'numeric',
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            });
        }
    }

    // Store scans globally so they can be used when profile details are shown
    let globalScans = [];

    // Update scan list
    function updateScanList(scans) {
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

    // View scan function (global scope)
    window.viewScan = function(scanNumber) {
        vscode.postMessage({ type: 'viewScan', scanNumber: scanNumber });
    };

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
        
        // Now that the tab structure is created, update the scan list with stored data
        updateScanList(globalScans);
        // Tab switching logic
        const tabHealth = document.getElementById('tab-health');
        const tabHistory = document.getElementById('tab-history');
        const tabContentHealth = document.getElementById('tabContent-health');
        const tabContentHistory = document.getElementById('tabContent-history');
        tabHealth.onclick = () => {
            tabHealth.classList.add('active');
            tabHistory.classList.remove('active');
            tabContentHealth.style.display = '';
            tabContentHistory.style.display = 'none';
        };
        tabHistory.onclick = () => {
            tabHistory.classList.add('active');
            tabHealth.classList.remove('active');
            tabContentHistory.style.display = '';
            tabContentHealth.style.display = 'none';
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

    // When no profile is selected or no profiles exist, show empty state
    function showNoProfileDetails() {
        profileDetails.style.display = 'none';
        profileCardContainer.style.display = 'none';
        const emptyState = document.getElementById('emptyState');
        if (emptyState) {
            emptyState.style.display = 'flex';
        }
    }

    function getTrashIcon() {
        // SVG trash/dustbin icon, inherits color
        return `<svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M7.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M12.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><rect x="4.5" y="5.5" width="11" height="11" rx="2" stroke="currentColor" stroke-width="1.5"/><path d="M2.5 5.5H17.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M8.5 2.5H11.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    }

    // Handle delete button click
    document.getElementById('deleteProfile').addEventListener('click', () => {
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

    // Handle rescan profile button click
    document.getElementById('rescanProfile').addEventListener('click', () => {
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

    // Handle scan all button click
    document.getElementById('scanAll').addEventListener('click', () => {
        const button = document.getElementById('scanAll');
        const originalText = button.innerHTML;
        button.disabled = true;
        button.innerHTML = '⏳';
        
        vscode.postMessage({ type: 'scanAllProfiles' });
        
        // Re-enable button after a delay
        setTimeout(() => {
            button.disabled = false;
            button.innerHTML = originalText;
        }, 3000);
    });

    // Handle scan workspace button click
    document.getElementById('scanWorkspace').addEventListener('click', () => {
        const button = document.getElementById('scanWorkspace');
        button.disabled = true;
        button.innerHTML = '<span style="font-family: codicon;">⏳</span> Scanning...';
        
        vscode.postMessage({ type: 'scanWorkspace' });
    });

    // Onboarding Functions
    function showStep(step) {
        // Hide all steps
        for (let i = 1; i <= 4; i++) {
            const stepElement = document.getElementById(`onboardingStep${i}`);
            if (stepElement) {
                stepElement.style.display = 'none';
            }
        }
        
        // Show the requested step
        const targetStep = document.getElementById(`onboardingStep${step}`);
        if (targetStep) {
            targetStep.style.display = 'block';
            currentStep = step;
        }
    }

    function updateModelInfo() {
        const modelInfo = document.getElementById('selectedModelInfo');
        const modelName = getModelDisplayName(selectedModel);
        const provider = getModelProvider(selectedModel);
        
        modelInfo.innerHTML = `
            <strong>Selected Model:</strong> ${modelName}<br>
            <strong>Provider:</strong> ${provider}<br>
            <small>Make sure you have a valid API key for ${provider}</small>
        `;
    }

    function updateConfigSummary() {
        const configSummary = document.getElementById('configSummary');
        const modelName = getModelDisplayName(selectedModel);
        const provider = getModelProvider(selectedModel);
        
        configSummary.innerHTML = `
            <div class="config-item">
                <span class="config-label">AI Model:</span>
                <span class="config-value">${modelName}</span>
            </div>
            <div class="config-item">
                <span class="config-label">Provider:</span>
                <span class="config-value">${provider}</span>
            </div>
            <div class="config-item">
                <span class="config-label">API Key:</span>
                <span class="config-value">••••••••••••${apiKey.slice(-4)}</span>
            </div>
        `;
    }

    function getModelDisplayName(model) {
        const modelNames = {
            'claude-3-5-sonnet-20241022': 'Claude 3.5 Sonnet',
            'gpt-4o': 'GPT-4o',
            'gpt-4o-mini': 'GPT-4o Mini',
            'o1-mini': 'O1 Mini',
            'o1': 'O1',
            'gpt-4.1-2025-04-14': 'GPT-4.1',
            'o3-mini-2025-01-31': 'O3 Mini',
            'gemini-2.5-pro': 'Gemini 2.5 Pro',
            'gemini-2.5-flash': 'Gemini 2.5 Flash',
            'claude-opus-4-20250514': 'Claude Opus 4',
            'claude-sonnet-4-20250514': 'Claude Sonnet 4',
            'claude-3-7-sonnet-20250219': 'Claude 3.7 Sonnet',
            'claude-3-5-haiku-20241022': 'Claude 3.5 Haiku'
        };
        return modelNames[model] || model;
    }

    function getModelProvider(model) {
        if (model.startsWith('claude')) return 'Anthropic';
        if (model.startsWith('gpt') || model.startsWith('o1') || model.startsWith('o3')) return 'OpenAI';
        if (model.startsWith('gemini')) return 'Google';
        return 'Unknown';
    }

    // Onboarding Event Listeners
    document.addEventListener('DOMContentLoaded', () => {
        // Step 1: Start onboarding
        const startButton = document.getElementById('startOnboarding');
        if (startButton) {
            startButton.addEventListener('click', () => {
                showStep(2);
            });
        }

        // Step 2: Model selection
        const modelSelect = document.getElementById('modelSelect');
        if (modelSelect) {
            modelSelect.addEventListener('change', (e) => {
                selectedModel = e.target.value;
            });
        }

        const backToStep1 = document.getElementById('backToStep1');
        if (backToStep1) {
            backToStep1.addEventListener('click', () => {
                showStep(1);
            });
        }

        const continueToStep3 = document.getElementById('continueToStep3');
        if (continueToStep3) {
            continueToStep3.addEventListener('click', () => {
                showStep(3);
                updateModelInfo();
            });
        }

        // Step 3: API Key configuration
        const apiKeyInput = document.getElementById('apiKeyInput');
        if (apiKeyInput) {
            apiKeyInput.addEventListener('input', (e) => {
                apiKey = e.target.value;
            });
        }

        const backToStep2 = document.getElementById('backToStep2');
        if (backToStep2) {
            backToStep2.addEventListener('click', () => {
                showStep(2);
            });
        }

        const continueToStep4 = document.getElementById('continueToStep4');
        if (continueToStep4) {
            continueToStep4.addEventListener('click', () => {
                if (!apiKey.trim()) {
                    alert('Please enter an API key to continue.');
                    return;
                }
                
                // Save configuration
                vscode.postMessage({
                    type: 'saveConfig',
                    model: selectedModel,
                    apiKey: apiKey
                });
            });
        }

        // Step 4: Back to API key
        const backToStep3 = document.getElementById('backToStep3');
        if (backToStep3) {
            backToStep3.addEventListener('click', () => {
                showStep(3);
                updateModelInfo();
            });
        }
    });

    // Handle profile selection change
    profileSelect.addEventListener('change', () => {
        const selectedId = profileSelect.value;
        if (selectedId) {
            const profile = globalProfiles.find(p => p.id === selectedId);
            if (profile) {
                displayProfileDetails(profile);
            }
            
            // Show loading state while fetching profile details
            profileCardContainer.innerHTML = '<div>Loading profile details...</div>';
            profileCardContainer.style.display = 'block';
            profileDetails.style.display = 'none';
        } else {
            showNoProfileDetails();
        }
    });
})();
