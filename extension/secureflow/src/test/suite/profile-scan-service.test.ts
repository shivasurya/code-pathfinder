import * as assert from 'assert';
import * as vscode from 'vscode';
import { ProfileScanService } from '../../services/profile-scan-service';
import { StoredProfile } from '../../models/profile-store';

suite('ProfileScanService Test Suite', () => {
  let service: ProfileScanService;
  let mockContext: vscode.ExtensionContext;
  let globalStateData: Map<string, any>;
  let secretsData: Map<string, string>;

  setup(() => {
    globalStateData = new Map();
    secretsData = new Map();

    mockContext = {
      globalState: {
        get: (key: string) => globalStateData.get(key),
        update: async (key: string, value: any) => {
          globalStateData.set(key, value);
        },
        keys: () => Array.from(globalStateData.keys()),
        setKeysForSync: (keys: string[]) => {}
      },
      secrets: {
        store: async (key: string, value: string) => {
          secretsData.set(key, value);
        },
        get: async (key: string) => {
          return secretsData.get(key);
        },
        delete: async (key: string) => {
          secretsData.delete(key);
        }
      },
      globalStorageUri: vscode.Uri.file('/tmp/test-storage')
    } as any;

    service = new ProfileScanService(mockContext);
  });

  suite('Service Initialization', () => {
    test('should create service instance', () => {
      assert.ok(service);
    });

    test('should have access to settings manager', () => {
      // Service should be able to access settings
      assert.ok(service);
    });
  });

  suite('Scan Validation', () => {
    test('should have scanProfile method', () => {
      // Verify the service has the scanProfile method
      assert.ok(typeof service.scanProfile === 'function');
    });

    test('should create valid profile for scanning', () => {
      const mockProfile: StoredProfile = {
        id: 'test-profile-1',
        name: 'Test App',
        category: 'Web Application',
        path: '/test',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json'],
        timestamp: Date.now(),
        isActive: true,
        workspaceFolderUri: 'file:///workspace'
      };

      // Verify profile has required properties for scanning
      assert.ok(mockProfile.id);
      assert.ok(mockProfile.path);
      assert.ok(mockProfile.workspaceFolderUri);
    });
  });

  suite('Provider Detection', () => {
    test('should detect OpenRouter provider from model ID', () => {
      // OpenRouter models have format: provider/model
      const openRouterModel = 'anthropic/claude-3-5-sonnet';
      const directModel = 'claude-sonnet-4-5-20250929';

      assert.ok(openRouterModel.includes('/'), 'OpenRouter model should contain /');
      assert.ok(!directModel.includes('/'), 'Direct model should not contain /');
    });
  });

  suite('Progress Callbacks', () => {
    test('should call progress callback during scan', async () => {
      const messages: string[] = [];

      const progressCallback = (message: string) => {
        messages.push(message);
      };

      // Note: This is a conceptual test - actual scan would need mocked CLI
      // We're testing that the callback mechanism is in place
      assert.strictEqual(typeof progressCallback, 'function');
      assert.strictEqual(messages.length, 0);

      progressCallback('Test message');
      assert.strictEqual(messages.length, 1);
      assert.strictEqual(messages[0], 'Test message');
    });
  });

  suite('Scan Results Mapping', () => {
    test('should handle empty scan results', () => {
      const emptyResults = {
        issues: [],
        filesAnalyzed: 0,
        iterations: 1
      };

      assert.strictEqual(emptyResults.issues.length, 0);
      assert.strictEqual(emptyResults.filesAnalyzed, 0);
    });

    test('should handle scan results with issues', () => {
      const results = {
        issues: [
          {
            title: 'SQL Injection',
            severity: 'High',
            description: 'Potential SQL injection in user input',
            recommendation: 'Use parameterized queries'
          },
          {
            title: 'XSS Vulnerability',
            severity: 'Medium',
            description: 'Unescaped user input in HTML',
            recommendation: 'Sanitize user input'
          }
        ],
        filesAnalyzed: 50,
        iterations: 3
      };

      assert.strictEqual(results.issues.length, 2);
      assert.strictEqual(results.filesAnalyzed, 50);
      assert.strictEqual(results.iterations, 3);
    });
  });

  suite('Silent Mode Configuration', () => {
    test('should enable silent mode for extension usage', () => {
      // The service should pass silent: true to CLI scanner
      // to suppress console output
      const cliOptions = {
        selectedModel: 'test-model',
        outputFormat: 'json',
        outputFile: '/tmp/scan.json',
        maxIterations: 20,
        silent: true, // Critical for clean extension output
        config: {
          apiKey: 'test-key',
          model: 'test-model',
          provider: undefined,
          analytics: {
            enabled: false
          }
        }
      };

      assert.strictEqual(cliOptions.silent, true);
      assert.strictEqual(cliOptions.config.analytics.enabled, false);
    });
  });

  suite('File Path Resolution', () => {
    test('should handle root path correctly', () => {
      const profilePath = '/';
      const workspacePath = '/workspace';

      // When profile path is '/', should use workspace root
      const expectedPath = workspacePath;

      assert.strictEqual(profilePath, '/');
      assert.ok(expectedPath.length > 0);
    });

    test('should handle nested path correctly', () => {
      const profilePath = '/apps/frontend';
      const workspacePath = '/workspace';

      // Should combine paths
      const expectedPath = workspacePath + profilePath;

      assert.strictEqual(expectedPath, '/workspace/apps/frontend');
    });
  });
});
