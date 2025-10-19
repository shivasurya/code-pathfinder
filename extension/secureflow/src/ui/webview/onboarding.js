/**
 * Onboarding Module
 * Handles the onboarding flow for new users including model selection and API key configuration
 */

import { getModelDisplayName, getModelProvider } from './ui-helpers.js';

// Onboarding state
let currentStep = 1;
let selectedModel = 'claude-sonnet-4-5-20250929';
let apiKey = '';

/**
 * Populate model select dropdown from window.modelConfig
 */
function populateModelDropdown() {
    const modelSelect = document.getElementById('modelSelect');
    if (!modelSelect || !window.modelConfig) {
        return;
    }

    // Clear existing options
    modelSelect.innerHTML = '';

    // Get models sorted by order
    const models = window.modelConfig.models || [];
    
    // Find recommended model
    const recommended = models.find(m => m.recommended);
    
    // Add options
    models.forEach(model => {
        const option = document.createElement('option');
        option.value = model.id;
        option.textContent = model.displayName;
        if (model.recommended) {
            option.textContent += ' (Recommended)';
        }
        modelSelect.appendChild(option);
    });

    // Set default to recommended or first model
    if (recommended) {
        modelSelect.value = recommended.id;
        selectedModel = recommended.id;
    } else if (models.length > 0) {
        modelSelect.value = models[0].id;
        selectedModel = models[0].id;
    }
}

// Show specific onboarding step
export function showStep(step) {
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
        if (step === 4) {
            updateConfigSummary();
        }
    }
}

// Update model information display
export function updateModelInfo() {
    const modelInfo = document.getElementById('selectedModelInfo');
    const modelName = getModelDisplayName(selectedModel);
    const provider = getModelProvider(selectedModel);
    
    modelInfo.innerHTML = `
        <strong>Selected Model:</strong> ${modelName}<br>
        <strong>Provider:</strong> ${provider}<br>
        <small>Make sure you have a valid API key for ${provider}</small>
    `;
}

// Update configuration summary
export function updateConfigSummary() {
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

// Handle onboarding status message
export function handleOnboardingStatus(message) {
    if (message.isConfigured) {
        console.log('Already configured');
        // Skip onboarding if already configured
        showStep(4);
        updateConfigSummary();
    } else {
        // Show step 1 for new users
        showStep(1);
    }
}

// Handle config saved message
export function handleConfigSaved(message) {
    if (message.success) {
        // Move to final step after successful config save
        showStep(4);
        updateConfigSummary();
    } else {
        alert('Failed to save configuration: ' + message.error);
    }
}

// Initialize onboarding event listeners
export function initializeOnboardingEventListeners(vscode) {
    // Populate model dropdown from injected config
    populateModelDropdown();

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
}

// Get current onboarding state
export function getOnboardingState() {
    return {
        currentStep,
        selectedModel,
        apiKey
    };
}
