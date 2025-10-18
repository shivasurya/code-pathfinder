import * as vscode from 'vscode';
import { AnalyticsService as SharedAnalyticsService } from '../../packages/secureflow-cli';

/**
 * VS Code Analytics Service
 *
 * This is a thin wrapper around the shared CLI analytics service.
 * It maintains the same public API as before to avoid breaking existing code,
 * but delegates to the shared analytics service from the CLI package.
 *
 * The shared service is initialized with VS Code-specific context and metadata.
 */

export class AnalyticsService {
  private static instance: AnalyticsService;
  private sharedService: SharedAnalyticsService;
  private initialized = false;
  private userEnabled = true; // Application-layer control

  private constructor() {
    this.sharedService = SharedAnalyticsService.getInstance();
  }

  public static getInstance(): AnalyticsService {
    if (!AnalyticsService.instance) {
      AnalyticsService.instance = new AnalyticsService();
    }
    return AnalyticsService.instance;
  }

  public async initialize(context?: vscode.ExtensionContext): Promise<void> {
    if (this.initialized) {
      return;
    }

    // Check user preference from VS Code settings
    this.userEnabled = vscode.workspace
      .getConfiguration('secureflow')
      .get('analytics.enabled', true);

    // Only initialize if user has analytics enabled
    if (!this.userEnabled) {
      this.initialized = true; // Mark as initialized but don't init PostHog
      return;
    }

    const metadata = {
      vscode_extension_version:
        vscode.extensions.getExtension('secureflow')?.packageJSON.version,
      vscode_build_version: vscode.version
    };

    if (context) {
      await this.sharedService.initializeForVSCode(context, metadata);
    }

    this.initialized = true;
  }

  public isEnabled(): boolean {
    // Check application-layer control first
    if (!this.userEnabled) {
      return false;
    }
    return this.sharedService.isEnabled();
  }

  public async trackEvent(
    eventName: string,
    properties: Record<string, any> = {}
  ): Promise<void> {
    // Respect user preference at application layer
    if (!this.userEnabled || !this.isEnabled()) {
      return;
    }

    const vsCodeProperties = {
      ...properties,
      vscode_extension_version:
        vscode.extensions.getExtension('secureflow')?.packageJSON.version,
      vscode_build_version: vscode.version,
      ai_model: vscode.workspace.getConfiguration('secureflow').get('AIModel')
    };

    await this.sharedService.trackEvent(eventName, vsCodeProperties);
  }

  public getDistinctId(): string {
    return this.sharedService.getDistinctId();
  }

  public async shutdown(): Promise<void> {
    await this.sharedService.shutdown();
  }
}
