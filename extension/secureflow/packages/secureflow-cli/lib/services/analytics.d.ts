/**
 * TypeScript declarations for shared Analytics Service
 */

/**
 * Storage adapter interface for distinct ID persistence
 */
export declare class StorageAdapter {
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
}

/**
 * File-based storage for CLI usage
 */
export declare class FileStorageAdapter extends StorageAdapter {
  constructor();
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
}

/**
 * VS Code storage adapter
 */
export declare class VSCodeStorageAdapter extends StorageAdapter {
  constructor(context: any);
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
}

/**
 * Shared Analytics Service
 */
export declare class AnalyticsService {
  static getInstance(): AnalyticsService;

  /**
   * Initialize for CLI usage
   */
  initializeForCLI(metadata?: Record<string, any>): Promise<void>;

  /**
   * Initialize for VS Code extension usage
   */
  initializeForVSCode(context: any, metadata?: Record<string, any>): Promise<void>;

  /**
   * Check if analytics is enabled
   */
  isEnabled(): boolean;

  /**
   * Track an event with properties
   */
  trackEvent(eventName: string, properties?: Record<string, any>): Promise<void>;

  /**
   * Get the distinct ID for this user
   */
  getDistinctId(): string;

  /**
   * Get the current platform
   */
  getPlatform(): string;

  /**
   * Shutdown analytics service
   * @param quick - If true, don't wait for flush (faster exit)
   */
  shutdown(quick?: boolean): Promise<void>;
}
