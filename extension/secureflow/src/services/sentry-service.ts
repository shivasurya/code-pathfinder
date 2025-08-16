import * as Sentry from '@sentry/node';
import * as vscode from 'vscode';
import { AnalyticsService } from './analytics';

export class SentryService {
  private static instance: SentryService;
  private initialized = false;
  private context?: vscode.ExtensionContext;

  private constructor() {}

  public static getInstance(): SentryService {
    if (!SentryService.instance) {
      SentryService.instance = new SentryService();
    }
    return SentryService.instance;
  }

  /**
   * Initialize Sentry with proper configuration for VS Code extension
   * Following best practices for shared environments
   */
  public async initialize(context: vscode.ExtensionContext): Promise<void> {
    if (this.initialized) {
      return;
    }

    this.context = context;

    // Check if error reporting is enabled (respects user privacy)
    const config = vscode.workspace.getConfiguration('secureflow');
    const errorReportingEnabled = config.get('errorReporting.enabled', true);
    const analyticsEnabled = config.get('analytics.enabled', true);

    // Only initialize if both error reporting and analytics are enabled
    if (!errorReportingEnabled || !analyticsEnabled) {
      console.log('🔒 Sentry: Error reporting disabled in settings');
      return;
    }

    try {
      Sentry.init({
        dsn: 'https://d0115fc459af62674afedf0a2fb2c89e@o4509825969815552.ingest.us.sentry.io/4509825973223424',

        // Release tracking
        release: `secureflow@${context.extension.packageJSON.version}`,

        // Sample rate for performance monitoring (lower for extensions)
        tracesSampleRate: 0.1,

        // Configure for shared environment (VS Code)
        beforeSend: (event, hint) => {
          // Only allow events that originate from our extension
          if (!this.isAppEvent(event, hint)) {
            return null;
          }

          // Filter out sensitive information
          const sanitized = this.sanitizeEvent(event);
          return sanitized as Sentry.ErrorEvent;
        },

        // Disable Sentry's automatic global handlers to avoid collecting
        // unrelated VS Code / other extension errors. We'll manage our own.

        // Set user context using analytics ID
        initialScope: {
          user: {
            id: this.getAnalyticsUserId()
          },
          tags: {
            'vscode.version': vscode.version,
            'extension.version': context.extension.packageJSON.version
          }
        },

        // Don't send default PII for privacy
        sendDefaultPii: false
      });

      this.initialized = true;
      console.log('🔒 Sentry: Initialized successfully');

      // Track initialization
      Sentry.addBreadcrumb({
        message: 'SecureFlow extension activated',
        category: 'extension',
        level: 'info'
      });
    } catch (error) {
      console.error('🔒 Sentry: Failed to initialize:', error);
    }
  }

  /**
   * Capture an exception with additional context
   */
  public captureException(error: Error, context?: Record<string, any>): void {
    if (!this.initialized) {
      console.error(
        '🔒 Sentry: Not initialized, logging error locally:',
        error
      );
      return;
    }

    Sentry.withScope((scope) => {
      if (context) {
        scope.setContext('additional', context);
      }

      // Add extension-specific tags
      scope.setTag('component', 'secureflow-extension');

      Sentry.captureException(error);
    });
  }

  /**
   * Capture a message with level and context
   */
  public captureMessage(
    message: string,
    level: Sentry.SeverityLevel = 'info',
    context?: Record<string, any>
  ): void {
    if (!this.initialized) {
      console.log(
        '🔒 Sentry: Not initialized, logging message locally:',
        message
      );
      return;
    }

    Sentry.withScope((scope) => {
      if (context) {
        scope.setContext('additional', context);
      }

      scope.setTag('component', 'secureflow-extension');
      Sentry.captureMessage(message, level);
    });
  }

  /**
   * Add breadcrumb for debugging
   */
  public addBreadcrumb(
    message: string,
    category: string,
    data?: Record<string, any>
  ): void {
    if (!this.initialized) {
      return;
    }

    Sentry.addBreadcrumb({
      message,
      category,
      data,
      level: 'info',
      timestamp: Date.now() / 1000
    });
  }

  /**
   * Set user context (anonymized)
   */
  public setUserContext(context: Record<string, any>): void {
    if (!this.initialized) {
      return;
    }

    Sentry.setUser({
      id: this.getAnalyticsUserId(),
      ...context
    });
  }

  /**
   * Flush pending events (useful before extension deactivation)
   */
  public async flush(timeout: number = 2000): Promise<boolean> {
    if (!this.initialized) {
      return true;
    }

    try {
      return await Sentry.flush(timeout);
    } catch (error) {
      console.error('🔒 Sentry: Failed to flush events:', error);
      return false;
    }
  }

  /**
   * Close Sentry client
   */
  public async close(): Promise<void> {
    if (!this.initialized) {
      return;
    }

    try {
      await Sentry.close();
      this.initialized = false;
      console.log('🔒 Sentry: Closed successfully');
    } catch (error) {
      console.error('🔒 Sentry: Failed to close:', error);
    }
  }

  /**
   * Sanitize event to remove sensitive information
   */
  private sanitizeEvent(event: Sentry.Event): Sentry.Event | null {
    // Remove or sanitize sensitive data
    if (event.request?.url) {
      // Remove query parameters that might contain sensitive data
      event.request.url = event.request.url.split('?')[0];
    }

    // Sanitize exception stack traces for file paths
    if (event.exception?.values) {
      event.exception.values.forEach((exception) => {
        if (exception.stacktrace?.frames) {
          exception.stacktrace.frames.forEach((frame) => {
            if (frame.filename) {
              // Keep only relative paths, remove absolute paths that might contain usernames
              const parts = frame.filename.split('/');
              const srcIndex = parts.findIndex((part) => part === 'src');
              if (srcIndex !== -1) {
                frame.filename = parts.slice(srcIndex).join('/');
              }
            }
          });
        }
      });
    }

    return event;
  }

  /**
   * Set up global error handlers to capture unhandled exceptions and rejections
   */
  public setupGlobalErrorHandlers(): void {
    if (!this.initialized) {
      return;
    }

    // Capture unhandled promise rejections
    process.on('unhandledRejection', (reason, promise) => {
      console.error('Unhandled Promise Rejection:', reason);
      const err = reason instanceof Error ? reason : new Error(String(reason));
      if (this.isErrorFromOurCode(err)) {
        this.captureException(err, {
          context: 'unhandled_promise_rejection',
          promise: promise.toString()
        });
      }
    });

    // Capture uncaught exceptions
    process.on('uncaughtException', (error) => {
      console.error('Uncaught Exception:', error);
      if (this.isErrorFromOurCode(error)) {
        this.captureException(error, { context: 'uncaught_exception' });
      }
    });

    // Capture VS Code extension host errors
    if (typeof process !== 'undefined' && process.on) {
      process.on('warning', (warning) => {
        console.warn('Process Warning:', warning);
        // Only send warnings if they originate from our extension code
        if (this.isStackFromOurExtension(warning.stack)) {
          this.captureMessage(
            `Process Warning: ${warning.message}`,
            'warning',
            {
              context: 'process_warning',
              warning_name: warning.name,
              warning_stack: warning.stack
            }
          );
        }
      });
    }
  }

  /**
   * Wraps a command handler with Sentry error tracking
   */
  public withErrorHandling<T extends any[]>(
    commandName: string,
    handler: (...args: T) => Promise<void> | void
  ): (...args: T) => Promise<void> {
    return async (...args: T) => {
      try {
        this.addBreadcrumb(`Command executed: ${commandName}`, 'user_action');

        await handler(...args);
      } catch (error) {
        console.error(`Error in command ${commandName}:`, error);

        try {
          this.captureException(error as Error, {
            context: 'command_execution',
            command: commandName,
            component: 'command_handler'
          });
        } catch (sentryError) {
          console.error('Failed to capture command error:', sentryError);
        }

        // Show user-friendly error message
        vscode.window.showErrorMessage(
          `SecureFlow: An error occurred while executing ${commandName}. Please check the output panel for details.`
        );
      }
    };
  }

  /**
   * Get analytics user ID for consistent user identification
   */
  private getAnalyticsUserId(): string {
    try {
      const analytics = AnalyticsService.getInstance();
      return analytics.getDistinctId() || vscode.env.machineId;
    } catch (error) {
      // Fallback to VS Code's anonymized machine ID
      return vscode.env.machineId;
    }
  }

  /**
   * Determine if a Sentry event should be considered application-originated.
   * Prefer explicit tagging from our capture calls; otherwise, inspect stack frames.
   */
  private isAppEvent(event: Sentry.Event, hint?: Sentry.EventHint): boolean {
    // If our code set the component tag, accept.
    if (event.tags && event.tags['component'] === 'secureflow-extension') {
      return true;
    }

    // Inspect exception stack frames to see if any frame is from our extension path
    const values = event.exception?.values || [];
    for (const val of values) {
      const frames = (val.stacktrace && val.stacktrace.frames) || [];
      for (const frame of frames) {
        if (frame.filename && this.isFrameFromOurExtension(frame.filename)) {
          return true;
        }
      }
    }

    // Otherwise, drop the event
    return false;
  }

  /**
   * Checks whether a filename from a stack frame points to our extension code.
   */
  private isFrameFromOurExtension(filename: string): boolean {
    const extPath = this.context?.extension.extensionPath || '';
    // Normalize
    const f = filename.toLowerCase();
    const ext = extPath.toLowerCase();

    // Fast-path: any reference containing 'secureflow' is considered our code
    // This handles VS Code webview resource URLs like
    // file+.vscode-resource.../codepathfinder.secureflow-<ver>/dist/webview/main.js
    if (f.includes('secureflow')) {
      return true;
    }

    // Allow if the file is within our extension folder (dist/ or src/)
    if (ext && f.includes(ext)) {
      return true;
    }

    // Heuristic fallbacks useful when source paths are sanitized or minimized
    // Match common identifiers for this project
    if (
      f.includes('/secureflow/') ||
      f.includes('secureflow/dist/') ||
      f.includes('secureflow/src/') ||
      f.includes('code-pathfinder/extension/secureflow') ||
      f.includes('codepathfinder.secureflow')
    ) {
      return true;
    }

    // Exclude obvious non-app frames
    if (
      f.startsWith('node:') ||
      f.includes('/electron/') ||
      f.includes('/vscode/')
    ) {
      return false;
    }

    return false;
  }

  /**
   * Determines whether an Error's stack originates from our extension code.
   */
  private isErrorFromOurCode(error: Error): boolean {
    return this.isStackFromOurExtension(error.stack);
  }

  /**
   * Checks a raw stack string for any frame pointing to our extension.
   */
  private isStackFromOurExtension(stack?: string): boolean {
    if (!stack) {
      return false;
    }
    const lines = stack.split('\n');
    for (const line of lines) {
      if (this.isFrameFromOurExtension(line)) {
        return true;
      }
    }
    return false;
  }
}
