import { PostHog } from 'posthog-node';
import * as vscode from 'vscode';

export class AnalyticsService {
  private static instance: AnalyticsService;
  private posthog: PostHog | null = null;
  private initialized = false;
  private distinctId: string = '';
  private context: vscode.ExtensionContext | null = null;

  private constructor() {
    // Distinct ID will be loaded from global state in initialize()
  }

  public static getInstance(): AnalyticsService {
    if (!AnalyticsService.instance) {
      AnalyticsService.instance = new AnalyticsService();
    }
    return AnalyticsService.instance;
  }

  public async initialize(context?: vscode.ExtensionContext): Promise<void> {
    if (this.initialized) {
      console.log('📊 Analytics: Already initialized, skipping');
      return;
    }

    // Store context for persistent storage
    if (context) {
      this.context = context;
    }

    // Load or generate persistent distinct ID
    await this.loadOrGenerateDistinctId();

    try {
      console.log('📊 Analytics: Initializing PostHog client...');

      // Initialize PostHog Node.js client
      this.posthog = new PostHog(
        'phc_iOS0SOw2gDax8kq44Z9FBEVAs8m6QG7yANvBF8ItV6g',
        {
          host: 'https://us.i.posthog.com',
          flushAt: 1, // Send events immediately for better debugging
          flushInterval: 0, // Disable automatic flushing,
          disableGeoip: false,
        }
      );

      this.initialized = true;
      console.log(
        '📊 Analytics: PostHog client initialized successfully with distinct ID:',
        this.distinctId
      );
    } catch (error) {
      console.error('📊 Analytics: Failed to initialize PostHog:', error);
    }
  }

  private async loadOrGenerateDistinctId(): Promise<void> {
    const DISTINCT_ID_KEY = 'secureflow.analytics.distinctId';

    if (this.context) {
      // Try to load existing distinct ID from global state
      const existingId = this.context.globalState.get<string>(DISTINCT_ID_KEY);

      if (existingId) {
        this.distinctId = existingId;
        console.log('📊 Analytics: Loaded existing distinct ID from storage');
      } else {
        // Generate new distinct ID and store it
        this.distinctId = this.generateDistinctId();
        await this.context.globalState.update(DISTINCT_ID_KEY, this.distinctId);
        console.log('📊 Analytics: Generated and stored new distinct ID');
      }
    } else {
      // Fallback: generate temporary ID if no context available
      this.distinctId = this.generateDistinctId();
      console.log(
        '📊 Analytics: Generated temporary distinct ID (no context available)'
      );
    }
  }

  private generateDistinctId(): string {
    // Generate a random anonymous ID
    return (
      Math.random().toString(36).substring(2, 15) +
      Math.random().toString(36).substring(2, 15)
    );
  }

  public isEnabled(): boolean {
    return this.initialized && this.posthog !== null;
  }

  public trackEvent(
    eventName: string,
    properties: Record<string, any> = {}
  ): void {
    if (!this.isEnabled()) {
      console.warn(
        '📊 Analytics: Service not enabled, skipping event:',
        eventName
      );
      return;
    }

    // include current vscode extension version and vscode build version
    properties['vscode_extension_version'] =
      vscode.extensions.getExtension('secureflow')?.packageJSON.version;
    properties['vscode_build_version'] = vscode.version;

    // get AI model name from config
    properties['ai_model'] = vscode.workspace
      .getConfiguration('secureflow')
      .get('AIModel');

    // Send event to PostHog using Node.js client
    if (this.posthog) {
      // console.log(`📊 Analytics: Tracking event "${eventName}" with properties:`, properties);

      this.posthog.capture({
        distinctId: this.distinctId,
        event: eventName,
        properties: properties
      });

      // Force flush to ensure events are sent immediately
      this.posthog.flush();
    } else {
      console.warn(
        '📊 Analytics: PostHog client not initialized, event not tracked:',
        eventName
      );
    }
  }

  public getDistinctId(): string {
    return this.distinctId;
  }

  public async shutdown(): Promise<void> {
    if (this.posthog) {
      await this.posthog.shutdown();
    }
  }
}
