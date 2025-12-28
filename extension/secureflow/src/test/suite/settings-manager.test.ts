import * as assert from 'assert';
import * as vscode from 'vscode';
import { SettingsManager } from '../../settings/settings-manager';

suite('SettingsManager Test Suite', () => {
  let settingsManager: SettingsManager;
  let mockContext: vscode.ExtensionContext;

  setup(() => {
    // Create a mock context with secrets storage
    mockContext = {
      secrets: {
        store: async (key: string, value: string) => {},
        get: async (key: string) => undefined,
        delete: async (key: string) => {}
      },
      workspaceState: {
        get: (key: string) => undefined,
        update: async (key: string, value: any) => {},
        keys: () => []
      },
      globalState: {
        get: (key: string) => undefined,
        update: async (key: string, value: any) => {},
        keys: () => [],
        setKeysForSync: (keys: string[]) => {}
      }
    } as any;

    settingsManager = new SettingsManager(mockContext);
  });

  suite('API Key Management', () => {
    test('should retrieve API key from workspace configuration', async () => {
      const testKey = 'test-api-key-12345';

      // Mock workspace configuration
      const mockConfig = {
        get: (key: string) => {
          if (key === 'APIKey') return testKey;
          return undefined;
        },
        update: async (key: string, value: any) => {},
        has: (key: string) => key === 'APIKey',
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = await settingsManager.getApiKey();
      assert.strictEqual(retrieved, testKey);
    });

    test('should return undefined when API key not configured', async () => {
      const mockConfig = {
        get: (key: string) => undefined,
        update: async (key: string, value: any) => {},
        has: (key: string) => false,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = await settingsManager.getApiKey();
      assert.strictEqual(retrieved, undefined);
    });
  });

  suite('Model Configuration', () => {
    test('should get selected AI model from configuration', () => {
      const testModel = 'claude-sonnet-4-5-20250929';

      // Mock workspace configuration
      const mockConfig = {
        get: (key: string) => {
          if (key === 'AIModel') return testModel;
          return undefined;
        },
        update: async (key: string, value: any) => {},
        has: (key: string) => true,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = settingsManager.getSelectedAIModel();
      assert.strictEqual(retrieved, testModel);
    });

    test('should return default model when not configured', () => {
      const mockConfig = {
        get: (key: string) => undefined,
        update: async (key: string, value: any) => {},
        has: (key: string) => false,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = settingsManager.getSelectedAIModel();
      assert.strictEqual(retrieved, 'claude-sonnet-4-5-20250929');
    });

    test('should retrieve OpenRouter model format', () => {
      const openRouterModel = 'anthropic/claude-3-5-sonnet';

      const mockConfig = {
        get: (key: string) => {
          if (key === 'AIModel') return openRouterModel;
          return undefined;
        },
        update: async (key: string, value: any) => {},
        has: (key: string) => true,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = settingsManager.getSelectedAIModel();
      assert.strictEqual(retrieved, openRouterModel);
      assert.ok(retrieved.includes('/'), 'OpenRouter model should contain /');
    });
  });

  suite('Provider Selection', () => {
    test('should get configured provider', () => {
      const testProvider = 'openrouter';

      const mockConfig = {
        get: (key: string) => {
          if (key === 'Provider') return testProvider;
          return undefined;
        },
        update: async (key: string, value: any) => {},
        has: (key: string) => true,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = settingsManager.getSelectedProvider();
      assert.strictEqual(retrieved, testProvider);
    });

    test('should return auto as default provider', () => {
      const mockConfig = {
        get: (key: string) => undefined,
        update: async (key: string, value: any) => {},
        has: (key: string) => false,
        inspect: (key: string) => undefined
      } as any;

      vscode.workspace.getConfiguration = () => mockConfig;

      const retrieved = settingsManager.getSelectedProvider();
      assert.strictEqual(retrieved, 'auto');
    });
  });

});
