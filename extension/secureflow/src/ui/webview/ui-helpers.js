/**
 * UI Helper Functions
 * Utility functions for formatting, icons, and common UI operations
 */

// Format timestamp to human readable format
export function formatTimestamp(timestamp) {
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

// Format last updated timestamp
export function formatLastUpdated(ts) {
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

// Get trash/delete icon SVG
export function getTrashIcon() {
    // SVG trash/dustbin icon, inherits color
    return `<svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M7.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M12.5 8.5V14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><rect x="4.5" y="5.5" width="11" height="11" rx="2" stroke="currentColor" stroke-width="1.5"/><path d="M2.5 5.5H17.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M8.5 2.5H11.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`;
}

/**
 * NOTE: Model configuration is auto-generated from config/models.json
 * This file now dynamically loads model data instead of hardcoding it
 */

// Import the model config - this will be available after webpack bundles
// For now, we'll use a dynamic approach that works in the webview context
let modelConfigCache = null;

/**
 * Load model configuration from the extension
 * In the webview context, we can access this via the vscode API
 */
function loadModelConfig() {
    // This will be populated by the extension when it loads the webview
    // For now, return a fallback that will be replaced by proper loading
    if (typeof window !== 'undefined' && window.modelConfig) {
        return window.modelConfig;
    }
    
    // Fallback - will be replaced when webview loads
    return { models: [], providers: {} };
}

// Get display name for AI models
export function getModelDisplayName(model) {
    const config = loadModelConfig();
    const modelData = config.models?.find(m => m.id === model);
    return modelData?.displayName || model;
}

// Get provider name for AI models
export function getModelProvider(model) {
    const config = loadModelConfig();
    const modelData = config.models?.find(m => m.id === model);
    if (modelData && config.providers) {
        return config.providers[modelData.provider]?.name || 'Unknown';
    }
    return 'Unknown';
}
