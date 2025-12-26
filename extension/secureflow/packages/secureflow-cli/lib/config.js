// CommonJS config loader for SecureFlow CLI
// Sources (precedence high->low):
// 1) Env vars
// 2) ~/.secureflow/config.json
// 3) Defaults

const fs = require('fs');
const os = require('os');
const path = require('path');

const CONFIG_DIR = path.join(os.homedir(), '.secureflow');
const CONFIG_FILE = path.join(CONFIG_DIR, 'config.json');

function readJsonSafe(file) {
  try {
    if (!fs.existsSync(file)) {
      return {};
    }
    const raw = fs.readFileSync(file, 'utf8');
    return JSON.parse(raw || '{}');
  } catch (_) {
    return {};
  }
}

function mask(value) {
  if (!value) {
    return '';
  }
  if (value.length <= 8) {
    return '*'.repeat(value.length);
  }
  return value.slice(0, 4) + '...' + value.slice(-4);
}

function loadConfig() {
  const fileCfg = readJsonSafe(CONFIG_FILE);
  const env = process.env;

  const model = env.SECUREFLOW_MODEL || fileCfg.model || 'claude-sonnet-4-5-20250929';
  const explicitProvider = env.SECUREFLOW_PROVIDER || fileCfg.provider;

  const cfg = {
    model: model,
    apiKey:
      env.SECUREFLOW_API_KEY ||
      env.ANTHROPIC_API_KEY ||
      env.OPENAI_API_KEY ||
      env.OPENROUTER_API_KEY ||
      env.GOOGLE_API_KEY ||
      env.XAI_API_KEY ||
      fileCfg.apiKey || '',
    provider: explicitProvider && explicitProvider !== 'auto'
      ? explicitProvider
      : inferProvider(env, model),
    analytics: {
      enabled: getBool(env.SECUREFLOW_ANALYTICS_ENABLED, fileCfg?.analytics?.enabled, true) // Default: enabled
    }
  };

  return cfg;
}

function inferProvider(env, model) {
  // Only infer provider from model ID format, NOT from API keys
  // This ensures users explicitly choose their provider

  // Check for OpenRouter models (they contain a "/" in the model name)
  if (/^[a-z0-9-]+\/[a-z0-9-]+/i.test(model)) {
    return 'openrouter';
  }

  // Infer from model name patterns
  if (/claude|anthropic/i.test(model)) {
    return 'anthropic';
  }
  if (/gpt|o1|o3|o4|openai/i.test(model)) {
    return 'openai';
  }
  if (/gemini/i.test(model)) {
    return 'google';
  }
  if (/grok/i.test(model)) {
    return 'xai';
  }
  if (/qwen/i.test(model)) {
    return 'ollama';
  }

  // Default to anthropic
  return 'anthropic';
}

function getBool(envVal, fileVal, defVal) {
  if (typeof envVal === 'string') {
    return /^(1|true|yes|on)$/i.test(envVal.trim());
  }
  if (typeof fileVal === 'boolean') {
    return fileVal;
  }
  return defVal;
}

function getMaskedConfig() {
  const cfg = loadConfig();
  return {
    ...cfg,
    apiKey: mask(cfg.apiKey)
  };
}

function setAnalyticsEnabled(enabled) {
  try {
    // Ensure directory exists
    if (!fs.existsSync(CONFIG_DIR)) {
      fs.mkdirSync(CONFIG_DIR, { recursive: true });
    }

    // Read existing config
    const existing = readJsonSafe(CONFIG_FILE);
    
    // Update analytics setting
    existing.analytics = { ...existing.analytics, enabled };
    
    // Write back to file
    fs.writeFileSync(CONFIG_FILE, JSON.stringify(existing, null, 2));
    return true;
  } catch (error) {
    return false;
  }
}

module.exports = {
  loadConfig,
  getMaskedConfig,
  setAnalyticsEnabled,
  CONFIG_DIR,
  CONFIG_FILE
};
