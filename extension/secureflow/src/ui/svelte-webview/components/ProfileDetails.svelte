<script lang="ts">
  import Card from './ui/Card.svelte';
  import Button from './ui/Button.svelte';

  export let vscode: any;
  export let profile: any;
  export let scans: any[] = [];
  export let onBack: () => void;

  // Tab state
  let activeTab: 'overview' | 'history' | 'vulnerabilities' = 'overview';

  // Computed values for dashboard
  $: latestScan = scans[0];
  $: totalIssues = latestScan?.issues?.length || 0;
  $: criticalIssues = latestScan?.issues?.filter(i => i.issue.severity === 'Critical').length || 0;
  $: highIssues = latestScan?.issues?.filter(i => i.issue.severity === 'High').length || 0;
  $: mediumIssues = latestScan?.issues?.filter(i => i.issue.severity === 'Medium').length || 0;
  $: lowIssues = latestScan?.issues?.filter(i => i.issue.severity === 'Low').length || 0;

  // Security score calculation (A-F based on issues)
  $: securityScore = calculateSecurityScore(criticalIssues, highIssues, mediumIssues, lowIssues);
  $: securityGrade = getSecurityGrade(securityScore);

  // Trend calculation (comparing last 2 scans)
  $: trend = calculateTrend(scans);

  // Days since last scan
  $: daysSinceLastScan = latestScan ? Math.floor((Date.now() - new Date(latestScan.timestamp).getTime()) / (1000 * 60 * 60 * 24)) : null;

  function calculateSecurityScore(critical: number, high: number, medium: number, low: number): number {
    // Start with 100, deduct points based on severity
    let score = 100;
    score -= critical * 20; // Critical: -20 each
    score -= high * 10;     // High: -10 each
    score -= medium * 5;    // Medium: -5 each
    score -= low * 2;       // Low: -2 each
    return Math.max(0, Math.min(100, score));
  }

  function getSecurityGrade(score: number): { grade: string; color: string } {
    if (score >= 90) return { grade: 'A', color: '#22c55e' };
    if (score >= 80) return { grade: 'B', color: '#84cc16' };
    if (score >= 70) return { grade: 'C', color: '#eab308' };
    if (score >= 60) return { grade: 'D', color: '#f97316' };
    return { grade: 'F', color: '#ef4444' };
  }

  function calculateTrend(scans: any[]): 'improving' | 'worsening' | 'stable' | null {
    if (scans.length < 2) return null;
    const latest = scans[0]?.issues?.length || 0;
    const previous = scans[1]?.issues?.length || 0;
    if (latest < previous) return 'improving';
    if (latest > previous) return 'worsening';
    return 'stable';
  }

  // Event handlers
  function handleRescan() {
    if (vscode) {
      vscode.postMessage({
        type: 'rescanProfile',
        profileId: profile.id
      });
    }
  }

  function handleScanGitChanges() {
    if (vscode) {
      vscode.postMessage({
        type: 'scanGitChanges'
      });
    }
  }

  function handleDelete() {
    if (confirm(`Are you sure you want to delete the profile "${profile.name}"?`)) {
      if (vscode) {
        vscode.postMessage({
          type: 'deleteProfile',
          profileId: profile.id
        });
      }
      onBack();
    }
  }

  function handleViewScan(scanNumber: number) {
    if (vscode) {
      vscode.postMessage({
        type: 'viewScan',
        scanNumber: scanNumber
      });
    }
  }

  function formatTimestamp(timestamp: number | string): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ago`;
    if (hours > 0) return `${hours}h ago`;
    if (minutes > 0) return `${minutes}m ago`;
    return 'Just now';
  }

  function formatFullDate(timestamp: number | string): string {
    const date = new Date(timestamp);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  // Get deduplicated vulnerabilities from latest scan
  $: uniqueVulnerabilities = getUniqueVulnerabilities(latestScan);

  function getUniqueVulnerabilities(scan: any): Array<{
    title: string;
    severity: string;
    description: string;
    recommendation: string;
    occurrences: Array<{filePath: string; startLine: number}>;
  }> {
    if (!scan?.issues) return [];

    // Group by issue title
    const issueMap: Record<string, any> = {};

    scan.issues.forEach((item: any) => {
      const title = item.issue.title;

      if (!issueMap[title]) {
        issueMap[title] = {
          title: item.issue.title,
          severity: item.issue.severity,
          description: item.issue.description,
          recommendation: item.issue.recommendation,
          occurrences: []
        };
      }

      // Deduplicate occurrences by file:line
      const occurrenceKey = `${item.filePath}:${item.startLine}`;
      const exists = issueMap[title].occurrences.some(
        (occ: any) => `${occ.filePath}:${occ.startLine}` === occurrenceKey
      );

      if (!exists) {
        issueMap[title].occurrences.push({
          filePath: item.filePath,
          startLine: item.startLine
        });
      }
    });

    // Sort by severity (Critical > High > Medium > Low) then by count
    const severityOrder: Record<string, number> = { 'Critical': 0, 'High': 1, 'Medium': 2, 'Low': 3 };
    return Object.values(issueMap).sort((a: any, b: any) => {
      const severityDiff = severityOrder[a.severity] - severityOrder[b.severity];
      if (severityDiff !== 0) return severityDiff;
      return b.occurrences.length - a.occurrences.length;
    });
  }

  function handleVulnerabilityClick(vuln: any) {
    if (vscode) {
      vscode.postMessage({
        type: 'openVulnerabilityDetails',
        vulnerability: vuln
      });
    }
  }
</script>

<div class="profile-details-container">
  <!-- Fixed Header Section -->
  <div class="fixed-header">
    <!-- Back Button -->
    <div class="header">
      <button class="back-btn" on:click={onBack}>
        ← Back to Profiles
      </button>
    </div>

  <!-- Profile Title Card with Details -->
  <Card>
    <div class="profile-header">
      <div class="profile-title-section">
        <div class="profile-icon-wrapper">
          <svg class="profile-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
          </svg>
        </div>
        <div class="profile-info">
          <h1 class="profile-name">{profile.name || 'Unknown Application'}</h1>
          <div class="profile-badges">
            <span class="badge badge-primary">{profile.category || 'Unknown'}</span>
            {#if profile.technology}
              <span class="badge badge-secondary">{profile.technology}</span>
            {/if}
            {#if profile.subcategory}
              <span class="badge badge-tertiary">{profile.subcategory}</span>
            {/if}
          </div>
        </div>
      </div>
      <button class="delete-btn" on:click={handleDelete} title="Delete Profile">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="3 6 5 6 21 6"></polyline>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          <line x1="10" y1="11" x2="10" y2="17"></line>
          <line x1="14" y1="11" x2="14" y2="17"></line>
        </svg>
      </button>
    </div>
  </Card>

  <!-- Tabs -->
  <div class="tabs-container">
    <button
      class="tab"
      class:active={activeTab === 'overview'}
      on:click={() => activeTab = 'overview'}
    >
      Overview
    </button>
    <button
      class="tab"
      class:active={activeTab === 'history'}
      on:click={() => activeTab = 'history'}
    >
      Scan History
    </button>
    <button
      class="tab"
      class:active={activeTab === 'vulnerabilities'}
      on:click={() => activeTab = 'vulnerabilities'}
    >
      Findings
    </button>
  </div>
  </div>

  <!-- Scrollable Content Area -->
  <div class="scrollable-content">
    {#if activeTab === 'overview'}
      <!-- Overview Tab -->
      <div class="tab-content">
        {#if !latestScan}
          <!-- Empty State -->
          <Card>
            <div class="empty-state-large">
              <svg class="empty-icon-large" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="11" cy="11" r="8"></circle>
                <path d="m21 21-4.35-4.35"></path>
              </svg>
              <h2>No Scan Data Available</h2>
              <p class="empty-description">This profile hasn't been scanned yet. Run a security scan to analyze your application for vulnerabilities and security issues.</p>
              <div class="empty-actions">
                <Button variant="primary" size="large" on:click={handleRescan}>
                  <span slot="icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 18px; height: 18px;">
                      <circle cx="11" cy="11" r="8"></circle>
                      <path d="m21 21-4.35-4.35"></path>
                    </svg>
                  </span>
                  Run First Scan
                </Button>
              </div>
              <div class="scan-benefits">
                <div class="benefit-item">
                  <svg class="benefit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
                  </svg>
                  <span>Identify security vulnerabilities</span>
                </div>
                <div class="benefit-item">
                  <svg class="benefit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
                  </svg>
                  <span>Get security score and recommendations</span>
                </div>
                <div class="benefit-item">
                  <svg class="benefit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                    <line x1="16" y1="13" x2="8" y2="13"></line>
                    <line x1="16" y1="17" x2="8" y2="17"></line>
                    <polyline points="10 9 9 9 8 9"></polyline>
                  </svg>
                  <span>Track security improvements over time</span>
                </div>
              </div>
            </div>
          </Card>
        {:else}
          <!-- Security Dashboard -->
          <div class="dashboard-grid">
            <!-- Security Score Card -->
            <Card>
              <div class="stat-card score-card">
                <div class="stat-header">
                  <span class="stat-label">Security Score</span>
                  {#if trend}
                    <span class="trend trend-{trend}">
                      {#if trend === 'improving'}
                        ↑ Improving
                      {:else if trend === 'worsening'}
                        ↓ Needs Attention
                      {:else}
                        → Stable
                      {/if}
                    </span>
                  {/if}
                </div>
                <div class="score-display">
                  <div class="score-circle" style="--score-color: {securityGrade.color}">
                    <svg viewBox="0 0 100 100">
                      <circle cx="50" cy="50" r="45" class="score-bg"></circle>
                      <circle
                        cx="50"
                        cy="50"
                        r="45"
                        class="score-progress"
                        style="stroke-dashoffset: {283 - (283 * securityScore / 100)}"
                      ></circle>
                    </svg>
                    <div class="score-text">
                      <div class="grade">{securityGrade.grade}</div>
                      <div class="score">{securityScore}</div>
                    </div>
                  </div>
                </div>
                <div class="stat-footer">
                  Last scanned {formatTimestamp(latestScan.timestamp)}
                </div>
              </div>
            </Card>

            <!-- Issues Breakdown Card -->
            <Card>
              <div class="stat-card">
                <div class="stat-header">
                  <span class="stat-label">Issues Found</span>
                </div>
                <div class="issues-breakdown">
                  <div class="issue-stat">
                    <div class="issue-count critical">{criticalIssues}</div>
                    <div class="issue-label">Critical</div>
                  </div>
                  <div class="issue-stat">
                    <div class="issue-count high">{highIssues}</div>
                    <div class="issue-label">High</div>
                  </div>
                  <div class="issue-stat">
                    <div class="issue-count medium">{mediumIssues}</div>
                    <div class="issue-label">Medium</div>
                  </div>
                  <div class="issue-stat">
                    <div class="issue-count low">{lowIssues}</div>
                    <div class="issue-label">Low</div>
                  </div>
                </div>
                <div class="total-issues">
                  Total: <strong>{totalIssues}</strong> {totalIssues === 1 ? 'issue' : 'issues'}
                </div>
              </div>
            </Card>

            <!-- Scan Stats Card -->
            <Card>
              <div class="stat-card">
                <div class="stat-header">
                  <span class="stat-label">Scan Statistics</span>
                </div>
                <div class="scan-stats">
                  <div class="scan-stat-item">
                    <div class="scan-stat-top">
                      <svg class="scan-stat-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="18" y1="20" x2="18" y2="10"></line>
                        <line x1="12" y1="20" x2="12" y2="4"></line>
                        <line x1="6" y1="20" x2="6" y2="14"></line>
                      </svg>
                      <div class="scan-stat-value">{scans.length}</div>
                    </div>
                    <div class="scan-stat-label">Total Scans</div>
                  </div>
                  <div class="scan-stat-item">
                    <div class="scan-stat-top">
                      <svg class="scan-stat-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
                        <polyline points="13 2 13 9 20 9"></polyline>
                      </svg>
                      <div class="scan-stat-value">{latestScan.fileCount || 0}</div>
                    </div>
                    <div class="scan-stat-label">Files Scanned</div>
                  </div>
                  {#if daysSinceLastScan !== null}
                    <div class="scan-stat-item">
                      <div class="scan-stat-top">
                        <svg class="scan-stat-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <circle cx="12" cy="12" r="10"></circle>
                          <polyline points="12 6 12 12 16 14"></polyline>
                        </svg>
                        <div class="scan-stat-value">{daysSinceLastScan}d</div>
                      </div>
                      <div class="scan-stat-label">Since Last Scan</div>
                    </div>
                  {/if}
                </div>
              </div>
            </Card>
          </div>

          <!-- Quick Actions -->
          <div class="actions-section">
            <h3 class="section-title">Quick Actions</h3>
            <div class="actions-grid">
              <button class="action-card action-primary" on:click={handleRescan}>
                <svg class="action-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="11" cy="11" r="8"></circle>
                  <path d="m21 21-4.35-4.35"></path>
                </svg>
                <div class="action-content">
                  <div class="action-title">Full Security Scan</div>
                  <div class="action-description">Scan all files in this profile</div>
                </div>
              </button>

              <button class="action-card" on:click={handleScanGitChanges}>
                <svg class="action-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"></path>
                </svg>
                <div class="action-content">
                  <div class="action-title">Scan Git Changes</div>
                  <div class="action-description">Analyze uncommitted changes</div>
                </div>
              </button>
            </div>
          </div>
        {/if}
      </div>
    {:else if activeTab === 'history'}
      <!-- Scan History Tab -->
      <div class="tab-content">
  <div class="history-section">
    {#if scans.length > 5}
      <div class="section-hint">Showing latest 5 of {scans.length} scans</div>
    {/if}

    {#if scans.length === 0}
      <Card>
        <div class="empty-state">
          <svg class="empty-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.35-4.35"></path>
          </svg>
          <h4>No Scans Yet</h4>
          <p>Run your first security scan to analyze this application for vulnerabilities.</p>
          <Button variant="primary" size="medium" on:click={handleRescan}>
            <span slot="icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 16px; height: 16px;">
                <circle cx="11" cy="11" r="8"></circle>
                <path d="m21 21-4.35-4.35"></path>
              </svg>
            </span>
            Start First Scan
          </Button>
        </div>
      </Card>
    {:else}
      <div class="scans-list">
        {#each scans.slice(0, 5) as scan, index}
          <div class="scan-card-wrapper">
            <div class="scan-card-compact">
              <div class="scan-header-compact">
                <div class="scan-title-group">
                  <h4 class="scan-number-compact">Scan #{scan.scanNumber}</h4>
                  {#if index === 0}
                    <span class="latest-badge">Latest</span>
                  {/if}
                </div>
                <div class="scan-date-compact">{formatFullDate(scan.timestamp)}</div>
              </div>

              <div class="scan-info-compact">
                <div class="scan-metrics-row">
                  <div class="metric-compact">
                    <span class="metric-label-compact">Files</span>
                    <span class="metric-value-compact">{scan.fileCount || 0}</span>
                  </div>
                  <div class="metric-compact">
                    <span class="metric-label-compact">Issues</span>
                    <span class="metric-value-compact metric-issues" class:has-issues={scan.issues?.length > 0}>
                      {scan.issues?.length || 0}
                    </span>
                  </div>
                </div>
                <div class="scan-model-row">
                  <span class="model-label-compact">Model</span>
                  <span class="model-value-compact">{scan.model || 'N/A'}</span>
                </div>
              </div>

              {#if scan.issues && scan.issues.length > 0}
                <div class="scan-severity-compact">
                  {#each [
                    { severity: 'Critical', count: scan.issues.filter(i => i.issue.severity === 'Critical').length },
                    { severity: 'High', count: scan.issues.filter(i => i.issue.severity === 'High').length },
                    { severity: 'Medium', count: scan.issues.filter(i => i.issue.severity === 'Medium').length },
                    { severity: 'Low', count: scan.issues.filter(i => i.issue.severity === 'Low').length }
                  ] as sev}
                    {#if sev.count > 0}
                      <span class="severity-badge-compact severity-{sev.severity.toLowerCase()}">
                        {sev.count} {sev.severity}
                      </span>
                    {/if}
                  {/each}
                </div>
              {/if}

              <div class="scan-actions-compact">
                <button class="view-details-btn" on:click={() => handleViewScan(scan.scanNumber)}>
                  View Details
                  <svg class="arrow-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="5" y1="12" x2="19" y2="12"></line>
                    <polyline points="12 5 19 12 12 19"></polyline>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
      </div>
    {:else if activeTab === 'vulnerabilities'}
      <!-- Vulnerabilities Tab -->
      <div class="tab-content">
        <div class="vulnerabilities-list">
          {#if uniqueVulnerabilities.length > 0}
            {#each uniqueVulnerabilities as vuln}
              <div
                class="vuln-card-wrapper"
                on:click={() => handleVulnerabilityClick(vuln)}
              >
                <div class="vuln-card-compact">
                  <div class="vuln-header-compact">
                    <span class="severity-badge-compact severity-{vuln.severity.toLowerCase()}">
                      {vuln.severity}
                    </span>
                    <span class="vuln-count-compact">{vuln.occurrences.length} location{vuln.occurrences.length > 1 ? 's' : ''}</span>
                  </div>
                  <div class="vuln-title-compact">{vuln.title}</div>
                </div>
              </div>
            {/each}
          {:else}
            <div class="empty-state-compact">
              <svg class="empty-icon-compact" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"></path>
                <path d="m9 12 2 2 4-4"></path>
              </svg>
              <h4>No Findings</h4>
              <p>No security issues found in the latest scan.</p>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .profile-details-container {
    width: 100%;
    max-width: 1000px;
    margin: 0 auto;
    height: 100vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* Fixed Header Section */
  .fixed-header {
    flex-shrink: 0;
    padding: 8px;
    padding-bottom: 0;
  }

  /* Header */
  .header {
    margin-bottom: 8px;
  }

  .back-btn {
    background: none;
    border: none;
    color: var(--vscode-textLink-foreground);
    cursor: pointer;
    font-size: 13px;
    padding: 4px 0;
    font-family: var(--vscode-font-family);
    transition: opacity 0.2s;
  }

  .back-btn:hover {
    opacity: 0.8;
    text-decoration: underline;
  }

  /* Profile Header */
  .profile-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }

  .profile-title-section {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 0;
  }

  .profile-icon-wrapper {
    background: var(--vscode-button-background);
    border-radius: 6px;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0.1;
    flex-shrink: 0;
  }

  .profile-icon {
    width: 16px;
    height: 16px;
    stroke: currentColor;
  }

  .profile-name {
    margin: 0 0 4px 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--vscode-foreground);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .profile-badges {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .badge {
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 10px;
    font-weight: 500;
    white-space: nowrap;
  }

  .badge-primary {
    background: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
  }

  .badge-secondary {
    background: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
  }

  .badge-tertiary {
    background: rgba(100, 100, 100, 0.2);
    color: var(--vscode-foreground);
    opacity: 0.8;
  }

  .delete-btn {
    background: none;
    border: 1px solid var(--vscode-widget-border);
    cursor: pointer;
    padding: 4px;
    border-radius: 3px;
    opacity: 0.5;
    transition: all 0.2s;
    color: var(--vscode-errorForeground);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .delete-btn:hover {
    opacity: 1;
    background: rgba(239, 68, 68, 0.1);
    border-color: var(--vscode-errorForeground);
  }

  .delete-btn svg {
    stroke: var(--vscode-errorForeground);
    width: 12px;
    height: 12px;
  }

  /* Dashboard Grid */
  .dashboard-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 8px;
    margin-top: 8px;
  }

  .stat-card {
    padding: 2px 0;
  }

  .stat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .stat-label {
    font-size: 13px;
    font-weight: 500;
    opacity: 0.7;
  }

  .trend {
    font-size: 12px;
    font-weight: 600;
    padding: 3px 8px;
    border-radius: 4px;
  }

  .trend-improving {
    color: #22c55e;
    background: rgba(34, 197, 94, 0.1);
  }

  .trend-worsening {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
  }

  .trend-stable {
    color: #6b7280;
    background: rgba(107, 114, 128, 0.1);
  }

  /* Security Score */
  .score-display {
    display: flex;
    justify-content: center;
    margin: 8px 0;
  }

  .score-circle {
    position: relative;
    width: 100px;
    height: 100px;
  }

  .score-circle svg {
    width: 100%;
    height: 100%;
    transform: rotate(-90deg);
  }

  .score-bg {
    fill: none;
    stroke: var(--vscode-widget-border);
    stroke-width: 6;
  }

  .score-progress {
    fill: none;
    stroke: var(--score-color);
    stroke-width: 6;
    stroke-dasharray: 283;
    stroke-linecap: round;
    transition: stroke-dashoffset 1s ease;
  }

  .score-text {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
  }

  .grade {
    font-size: 32px;
    font-weight: 700;
    line-height: 1;
    color: var(--score-color);
  }

  .score {
    font-size: 11px;
    opacity: 0.7;
    margin-top: 2px;
  }

  .stat-footer {
    text-align: center;
    font-size: 12px;
    opacity: 0.6;
    margin-top: 8px;
  }

  /* Issues Breakdown */
  .issues-breakdown {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    margin: 8px 0;
  }

  .issue-stat {
    text-align: center;
  }

  .issue-count {
    font-size: 20px;
    font-weight: 700;
    line-height: 1;
    margin-bottom: 4px;
  }

  .issue-count.critical { color: #dc2626; }
  .issue-count.high { color: #f97316; }
  .issue-count.medium { color: #eab308; }
  .issue-count.low { color: #22c55e; }

  .issue-label {
    font-size: 10px;
    opacity: 0.7;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .total-issues {
    text-align: center;
    font-size: 13px;
    opacity: 0.7;
    padding-top: 8px;
    border-top: 1px solid var(--vscode-widget-border);
  }

  /* Scan Stats */
  .scan-stats {
    display: flex;
    flex-direction: row;
    justify-content: space-around;
    gap: 8px;
    margin: 6px 0;
  }

  .scan-stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    text-align: center;
  }

  .scan-stat-top {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 6px;
  }

  .scan-stat-icon {
    width: 14px;
    height: 14px;
    opacity: 0.8;
    stroke: currentColor;
  }

  .scan-stat-value {
    font-size: 14px;
    font-weight: 600;
    line-height: 1;
  }

  .scan-stat-label {
    font-size: 9px;
    opacity: 0.6;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  /* Quick Actions */
  .actions-section {
    margin-top: 10px;
  }

  .section-title {
    font-size: 13px;
    font-weight: 600;
    margin: 0 0 6px 0;
    color: var(--vscode-foreground);
  }

  .actions-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 10px;
  }

  .action-card {
    background: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 6px;
    padding: 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s;
    font-family: var(--vscode-font-family);
  }

  .action-card:hover {
    border-color: var(--vscode-button-background);
    transform: translateY(-1px);
  }

  .action-primary {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border-color: var(--vscode-button-background);
  }

  .action-icon {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    stroke: currentColor;
  }

  .action-content {
    flex: 1;
  }

  .action-title {
    font-weight: 600;
    font-size: 13px;
    margin-bottom: 2px;
  }

  .action-description {
    font-size: 11px;
    opacity: 0.7;
  }

  /* Tabs */
  .tabs-container {
    display: flex;
    gap: 4px;
    padding: 8px 8px 0 8px;
    border-bottom: 1px solid var(--vscode-widget-border);
  }

  .tab {
    background: none;
    border: none;
    color: var(--vscode-foreground);
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    padding: 8px 16px;
    border-radius: 4px 4px 0 0;
    opacity: 0.6;
    transition: all 0.2s;
    font-family: var(--vscode-font-family);
  }

  .tab:hover {
    opacity: 0.8;
    background: rgba(255, 255, 255, 0.05);
  }

  .tab.active {
    opacity: 1;
    background: var(--vscode-editor-background);
    border-bottom: 2px solid var(--vscode-button-background);
  }

  /* Scrollable Content */
  .scrollable-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }

  .tab-content {
    height: 100%;
  }

  /* Vulnerabilities List */
  .vulnerabilities-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .vuln-card-wrapper {
    background-color: var(--vscode-editor-background);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 6px;
    padding: 10px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .vuln-card-wrapper:hover {
    border-color: var(--vscode-button-background);
    transform: translateY(-1px);
  }

  .vuln-card-compact {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .vuln-header-compact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .vuln-count-compact {
    font-size: 10px;
    opacity: 0.6;
  }

  .vuln-title-compact {
    font-size: 12px;
    font-weight: 500;
    line-height: 1.3;
  }

  .empty-state-compact {
    text-align: center;
    padding: 40px 20px;
  }

  .empty-icon-compact {
    width: 40px;
    height: 40px;
    margin: 0 auto 12px;
    opacity: 0.3;
    stroke: currentColor;
  }

  .empty-state-compact h4 {
    margin: 0 0 6px 0;
    font-size: 15px;
    font-weight: 600;
  }

  .empty-state-compact p {
    margin: 0;
    opacity: 0.7;
    font-size: 13px;
  }

  /* Scan History */
  .history-section {
    margin-bottom: 10px;
  }

  .section-hint {
    font-size: 11px;
    opacity: 0.6;
    margin-bottom: 8px;
    text-align: right;
  }

  /* Empty State */
  .empty-state {
    text-align: center;
    padding: 40px 20px;
  }

  .empty-icon {
    width: 40px;
    height: 40px;
    display: block;
    margin: 0 auto 12px;
    opacity: 0.3;
    stroke: currentColor;
  }

  .empty-state h4 {
    margin: 0 0 6px 0;
    font-size: 15px;
    font-weight: 600;
  }

  .empty-state p {
    margin: 0 0 20px 0;
    opacity: 0.7;
    font-size: 13px;
  }

  /* Scans List */
  .scans-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  /* Compact Scan Card Wrapper */
  .scan-card-wrapper {
    background-color: var(--vscode-editor-background);
    border: 1px solid var(--vscode-widget-border);
    border-radius: 6px;
    padding: 10px;
  }

  /* Compact Scan Card */
  .scan-card-compact {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .scan-header-compact {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .scan-title-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .scan-number-compact {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
  }

  .scan-date-compact {
    font-size: 10px;
    opacity: 0.6;
  }

  .latest-badge {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    padding: 2px 6px;
    border-radius: 8px;
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .scan-info-compact {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .scan-metrics-row {
    display: flex;
    gap: 10px;
  }

  .metric-compact {
    display: flex;
    align-items: baseline;
    gap: 3px;
  }

  .metric-label-compact {
    font-size: 9px;
    opacity: 0.5;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .metric-value-compact {
    font-size: 12px;
    font-weight: 600;
  }

  .metric-issues.has-issues {
    color: #ef4444;
  }

  .scan-model-row {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }

  .model-label-compact {
    font-size: 9px;
    opacity: 0.5;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    flex-shrink: 0;
  }

  .model-value-compact {
    font-size: 10px;
    font-family: var(--vscode-editor-font-family);
    opacity: 0.7;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .scan-severity-compact {
    display: flex;
    gap: 3px;
    flex-wrap: wrap;
  }

  .severity-badge-compact {
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.2px;
  }

  .scan-actions-compact {
    display: flex;
    justify-content: flex-end;
    margin-top: 0;
  }

  .view-details-btn {
    background: none;
    border: 1px solid var(--vscode-button-background);
    color: var(--vscode-button-background);
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: all 0.2s;
    font-family: var(--vscode-font-family);
  }

  .view-details-btn:hover {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
  }

  .arrow-icon {
    width: 12px;
    height: 12px;
    stroke: currentColor;
  }

  .scan-actions {
    display: flex;
    justify-content: flex-end;
  }

  /* Empty State Large */
  .empty-state-large {
    text-align: center;
    padding: 60px 40px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 24px;
  }

  .empty-icon-large {
    width: 80px;
    height: 80px;
    color: var(--vscode-foreground);
    opacity: 0.3;
    margin-bottom: 8px;
  }

  .empty-state-large h2 {
    font-size: 24px;
    font-weight: 600;
    margin: 0;
    color: var(--vscode-foreground);
  }

  .empty-description {
    font-size: 14px;
    line-height: 1.6;
    opacity: 0.7;
    max-width: 500px;
    margin: 0;
  }

  .empty-actions {
    margin-top: 8px;
  }

  .scan-benefits {
    display: flex;
    flex-direction: column;
    gap: 16px;
    margin-top: 16px;
    padding: 24px;
    background: var(--vscode-textBlockQuote-background);
    border-radius: 8px;
    width: 100%;
    max-width: 500px;
  }

  .benefit-item {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 13px;
    color: var(--vscode-foreground);
  }

  .benefit-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    color: var(--vscode-button-background);
  }

  .empty-icon-svg {
    width: 60px;
    height: 60px;
    color: var(--vscode-foreground);
    opacity: 0.3;
    margin-bottom: 16px;
  }
</style>
