(function() {
    const vscode = acquireVsCodeApi();
    const profileSelect = document.getElementById('profileSelect');
    const profileDetails = document.getElementById('profileDetails');
    const profileContent = document.getElementById('profileContent');

    // Request initial profiles and scans
    vscode.postMessage({ type: 'getProfiles' });
    vscode.postMessage({ type: 'getScans' });

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
                    <button class="profile-tab active" id="tab-health">Security Health</button>
                    <button class="profile-tab" id="tab-history">Scan History</button>
                </div>
                <div class="profile-tab-content" id="tabContent-health">
                    <div class="health-summary-v2">
                        <div class="health-score-v2">
                            <div class="health-score-circle">
                                <span class="health-score-value">D</span>
                                <span class="health-score-label">Fair</span>
                            </div>
                        </div>
                        <div class="health-severity-badges">
                            <div class="severity-badge critical"><span class="sev-icon">&#9888;</span> Critical</div>
                            <div class="severity-badge critical-count">1</div>
                            <div class="severity-badge high"><span class="sev-icon">&#9888;</span> High</div>
                            <div class="severity-badge high-count">2</div>
                            <div class="severity-badge medium"><span class="sev-icon">&#9888;</span> Medium</div>
                            <div class="severity-badge medium-count">4</div>
                        </div>
                    </div>
                    <div class="scan-history-title">Reported Issues</div>
                    <div class="scan-history-list-v2">
                        <div class="scan-history-item-v2 critical">
                            <span class="scan-icon critical">&#9888;</span>
                            <div>
                                <div class="scan-title">[Critical] Remote Code Execution in /api/upload</div>
                                <div class="scan-meta">Detected: Today, 5:12 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 high">
                            <span class="scan-icon high">&#9888;</span>
                            <div>
                                <div class="scan-title">[High] SQL Injection in /user/login</div>
                                <div class="scan-meta">Detected: Today, 4:50 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Insecure Cookie Flag</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Cross-Site Scripting (XSS)</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] XML External Entity (XXE) Injection</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Server-side Request Forgery (SSRF)</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Cross-Site Scripting (XSS)</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Server Misconfiguration</div>
                                <div class="scan-meta">Detected: Yesterday, 8:10 PM</div>
                            </div>
                        </div>
                        <div class="scan-history-item-v2 medium">
                            <span class="scan-icon medium">&#9888;</span>
                            <div>
                                <div class="scan-title">[Medium] Cross-Site Scripting (XSS)</div>
                                <div class="scan-meta">Detected: Yesterday, 2:10 PM</div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="profile-tab-content" id="tabContent-history" style="display:none;">
                    <div id="scanList" class="scan-list">
                        <div id="noScans" class="empty-scan-state">
                            <p>No security scans found. Run a SecureFlow review to see scan history here.</p>
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
