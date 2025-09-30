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

// Get display name for AI models
export function getModelDisplayName(model) {
    const modelNames = {
        'claude-sonnet-4-5-20250929': 'Claude 4.5 Sonnet',
        'claude-3-5-sonnet-20241022': 'Claude 3.5 Sonnet (⚠️ DEPRECATED - Use 4.5 instead)',
        'gpt-4o': 'GPT-4o',
        'gpt-4o-mini': 'GPT-4o Mini',
        'o1-mini': 'O1 Mini',
        'o1': 'O1',
        'gpt-4.1-2025-04-14': 'GPT-4.1',
        'o3-mini-2025-01-31': 'O3 Mini',
        'gemini-2.5-pro': 'Gemini 2.5 Pro',
        'gemini-2.5-flash': 'Gemini 2.5 Flash',
        'claude-opus-4-1-20250805': 'Claude Opus 4.1',
        'claude-opus-4-20250514': 'Claude Opus 4',
        'claude-sonnet-4-20250514': 'Claude Sonnet 4',
        'claude-3-7-sonnet-20250219': 'Claude 3.7 Sonnet',
        'claude-3-5-haiku-20241022': 'Claude 3.5 Haiku',
        'grok-4-fast-reasoning': 'Grok 4 Fast Reasoning'
    };
    return modelNames[model] || model;
}

// Get provider name for AI models
export function getModelProvider(model) {
    if (model.startsWith('claude')) return 'Anthropic';
    if (model.startsWith('gpt') || model.startsWith('o1') || model.startsWith('o3')) return 'OpenAI';
    if (model.startsWith('gemini')) return 'Google';
    if (model.startsWith('grok')) return 'xAI';
    return 'Unknown';
}
