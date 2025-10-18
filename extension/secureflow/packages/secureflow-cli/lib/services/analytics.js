/**
 * Shared Analytics Service for SecureFlow
 * 
 * Platform-agnostic analytics service that works for both:
 * - CLI usage (standalone)
 * - VS Code extension usage (imported)
 * 
 * Privacy-focused: Only tracks command usage and model names, no sensitive data.
 */

const { PostHog } = require('posthog-node');
const fs = require('fs');
const path = require('path');
const os = require('os');

// Constants
const POSTHOG_API_KEY = 'phc_iOS0SOw2gDax8kq44Z9FBEVAs8m6QG7yANvBF8ItV6g';
const POSTHOG_HOST = 'https://us.i.posthog.com';
const DISTINCT_ID_KEY = 'secureflow.analytics.distinctId';
const SHUTDOWN_TIMEOUT_MS = 500; // Reduced from 2000ms to 500ms

/**
 * Storage adapter interface for distinct ID persistence
 * Different implementations for CLI vs VS Code
 */
class StorageAdapter {
  async get(key) {
    throw new Error('StorageAdapter.get() must be implemented');
  }

  async set(key, value) {
    throw new Error('StorageAdapter.set() must be implemented');
  }
}

/**
 * File-based storage for CLI usage
 * Stores analytics data in user's home directory
 */
class FileStorageAdapter extends StorageAdapter {
  constructor() {
    super();
    this.storageDir = path.join(os.homedir(), '.secureflow');
    this.storageFile = path.join(this.storageDir, 'analytics.json');
    this._ensureStorageDir();
  }

  _ensureStorageDir() {
    if (!fs.existsSync(this.storageDir)) {
      fs.mkdirSync(this.storageDir, { recursive: true });
    }
  }

  async get(key) {
    try {
      if (!fs.existsSync(this.storageFile)) {
        return null;
      }
      const data = JSON.parse(fs.readFileSync(this.storageFile, 'utf8'));
      return data[key] || null;
    } catch (error) {
      // Silently fail - analytics should not disrupt CLI
      return null;
    }
  }

  async set(key, value) {
    try {
      let data = {};
      if (fs.existsSync(this.storageFile)) {
        data = JSON.parse(fs.readFileSync(this.storageFile, 'utf8'));
      }
      data[key] = value;
      fs.writeFileSync(this.storageFile, JSON.stringify(data, null, 2));
    } catch (error) {
      // Silently fail - analytics should not disrupt CLI
    }
  }
}

/**
 * VS Code storage adapter
 * Wraps VS Code's ExtensionContext.globalState
 */
class VSCodeStorageAdapter extends StorageAdapter {
  constructor(context) {
    super();
    this.context = context;
  }

  async get(key) {
    return this.context.globalState.get(key) || null;
  }

  async set(key, value) {
    await this.context.globalState.update(key, value);
  }
}

/**
 * Shared Analytics Service
 * Works for both CLI and VS Code extension
 */
class AnalyticsService {
  static instance = null;
  
  constructor() {
    this.posthog = null;
    this.initialized = false;
    this.distinctId = '';
    this.storage = null;
    this.platform = 'unknown'; // 'cli' or 'vscode'
    this.metadata = {}; // Platform-specific metadata
  }

  /**
   * Get singleton instance
   */
  static getInstance() {
    if (!AnalyticsService.instance) {
      AnalyticsService.instance = new AnalyticsService();
    }
    return AnalyticsService.instance;
  }

  /**
   * Initialize for CLI usage
   */
  async initializeForCLI(metadata = {}) {
    if (this.initialized) {
      return;
    }

    this.platform = 'cli';
    this.storage = new FileStorageAdapter();
    this.metadata = {
      platform: 'cli',
      node_version: process.version,
      ...metadata
    };

    await this._initialize();
  }

  /**
   * Initialize for VS Code extension usage
   * @param {object} context - VS Code ExtensionContext
   * @param {object} metadata - Additional metadata (version, etc.)
   */
  async initializeForVSCode(context, metadata = {}) {
    if (this.initialized) {
      return;
    }

    this.platform = 'vscode';
    this.storage = new VSCodeStorageAdapter(context);
    this.metadata = {
      platform: 'vscode',
      ...metadata
    };

    await this._initialize();
  }

  /**
   * Internal initialization logic
   */
  async _initialize() {
    // Load or generate persistent distinct ID
    await this._loadOrGenerateDistinctId();

    try {
      this.posthog = new PostHog(POSTHOG_API_KEY, {
        host: POSTHOG_HOST,
        flushAt: 1,
        flushInterval: 0,
        disableGeoip: false,
      });
      this.initialized = true;
    } catch (error) {
      // Silently fail - analytics should not disrupt application
      this.initialized = false;
    }
  }

  /**
   * Load existing or generate new distinct ID
   */
  async _loadOrGenerateDistinctId() {
    if (this.storage) {
      const existingId = await this.storage.get(DISTINCT_ID_KEY);

      if (existingId) {
        this.distinctId = existingId;
      } else {
        this.distinctId = this._generateDistinctId();
        await this.storage.set(DISTINCT_ID_KEY, this.distinctId);
      }
    } else {
      // Fallback: generate temporary ID
      this.distinctId = this._generateDistinctId();
    }
  }

  /**
   * Generate random anonymous distinct ID
   */
  _generateDistinctId() {
    return (
      Math.random().toString(36).substring(2, 15) +
      Math.random().toString(36).substring(2, 15)
    );
  }

  /**
   * Check if analytics is enabled
   */
  isEnabled() {
    return this.initialized && this.posthog !== null;
  }

  /**
   * Track an event with properties
   * 
   * PRIVACY: Only tracks command usage and model names, no sensitive data
   * 
   * @param {string} eventName - Name of the event
   * @param {object} properties - Event properties (safe data only)
   */
  async trackEvent(eventName, properties = {}) {
    if (!this.isEnabled()) {
      return;
    }

    // Merge platform metadata with event properties
    const eventProperties = {
      ...this.metadata,
      ...properties
    };

    if (!this.posthog) {
      return;
    }

    try {
      this.posthog.capture({
        distinctId: this.distinctId,
        event: eventName,
        properties: eventProperties
      });
      // Fire and forget - don't wait for flush to complete
      this.posthog.flush().catch(() => {});
    } catch (error) {
      // Silently fail - analytics should not disrupt application
    }
  }

  /**
   * Get the distinct ID for this user
   */
  getDistinctId() {
    return this.distinctId;
  }

  /**
   * Get the current platform
   */
  getPlatform() {
    return this.platform;
  }

  /**
   * Shutdown analytics service
   * @param {boolean} quick - If true, don't wait for flush (faster exit)
   */
  async shutdown(quick = false) {
    if (!this.posthog) {
      return;
    }

    try {
      if (quick) {
        // For quick commands, just shutdown without waiting
        this.posthog.shutdown().catch(() => {}); // Fire and forget
        this.posthog = null;
        this.initialized = false;
        return;
      }

      // Normal shutdown with reduced timeout
      const shutdownPromise = this.posthog.shutdown();
      const timeoutPromise = new Promise((_, reject) => 
        setTimeout(() => reject(new Error('Shutdown timeout')), SHUTDOWN_TIMEOUT_MS)
      );
      await Promise.race([shutdownPromise, timeoutPromise]);
    } catch (error) {
      // Silently fail - analytics should not disrupt application
    } finally {
      this.posthog = null;
      this.initialized = false;
    }
  }
}

module.exports = {
  AnalyticsService,
  StorageAdapter,
  FileStorageAdapter,
  VSCodeStorageAdapter
};
