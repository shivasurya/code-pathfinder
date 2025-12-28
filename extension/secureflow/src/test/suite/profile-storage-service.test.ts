import * as assert from 'assert';
import * as vscode from 'vscode';
import { ProfileStorageService } from '../../services/profile-storage-service';
import { ApplicationProfile } from '../../profiler/project-profiler';

suite('ProfileStorageService Test Suite', () => {
  let service: ProfileStorageService;
  let mockContext: vscode.ExtensionContext;
  let globalStateData: Map<string, any>;

  setup(() => {
    globalStateData = new Map();

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
        store: async (key: string, value: string) => {},
        get: async (key: string) => undefined,
        delete: async (key: string) => {}
      }
    } as any;

    service = new ProfileStorageService(mockContext);
  });

  suite('Profile Storage', () => {
    test('should store a profile', async () => {
      const profile: ApplicationProfile = {
        name: 'Test App',
        category: 'Web Application',
        path: '/test/path',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json', 'tsconfig.json']
      };

      const workspaceUri = 'file:///workspace';
      const stored = await service.storeProfile(profile, workspaceUri, false);

      assert.ok(stored.id, 'Should have generated ID');
      assert.strictEqual(stored.name, profile.name);
      assert.strictEqual(stored.category, profile.category);
      assert.strictEqual(stored.workspaceFolderUri, workspaceUri);
      assert.strictEqual(stored.isActive, false);
      assert.ok(stored.timestamp, 'Should have timestamp');
    });

    test('should retrieve profile by ID', async () => {
      const profile: ApplicationProfile = {
        name: 'Test App',
        category: 'Web Application',
        path: '/test/path',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json', 'tsconfig.json']
      };

      const workspaceUri = 'file:///workspace';
      const stored = await service.storeProfile(profile, workspaceUri);

      const retrieved = service.getProfileById(stored.id);

      assert.ok(retrieved);
      assert.strictEqual(retrieved?.id, stored.id);
      assert.strictEqual(retrieved?.name, profile.name);
    });

    test('should return undefined for non-existent profile ID', () => {
      const retrieved = service.getProfileById('non-existent-id');
      assert.strictEqual(retrieved, undefined);
    });
  });

  suite('Workspace Profiles', () => {
    test('should get profiles for workspace', async () => {
      const workspaceUri = 'file:///workspace';

      const profile1: ApplicationProfile = {
        name: 'App 1',
        category: 'Web Application',
        path: '/app1',
        confidence: 0.9,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const profile2: ApplicationProfile = {
        name: 'App 2',
        category: 'API',
        path: '/app2',
        confidence: 0.85,
        languages: ['JavaScript'],
        frameworks: ['Fastify'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      await service.storeProfile(profile1, workspaceUri);
      await service.storeProfile(profile2, workspaceUri);

      const profiles = service.getWorkspaceProfiles(workspaceUri);

      assert.strictEqual(profiles.length, 2);
      assert.ok(profiles.some(p => p.name === 'App 1'));
      assert.ok(profiles.some(p => p.name === 'App 2'));
    });

    test('should return empty array for workspace with no profiles', () => {
      const profiles = service.getWorkspaceProfiles('file:///empty-workspace');
      assert.strictEqual(profiles.length, 0);
    });
  });

  suite('Profile Activation', () => {
    test('should set profile as active', async () => {
      const workspaceUri = 'file:///workspace';

      const profile: ApplicationProfile = {
        name: 'Test App',
        category: 'Web Application',
        path: '/test',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const stored = await service.storeProfile(profile, workspaceUri, false);
      assert.strictEqual(stored.isActive, false);

      const activated = await service.activateProfile(stored.id);

      assert.ok(activated);
      assert.strictEqual(activated?.isActive, true);
    });

    test('should deactivate other profiles when activating one', async () => {
      const workspaceUri = 'file:///workspace';

      const profile1: ApplicationProfile = {
        name: 'App 1',
        category: 'Web Application',
        path: '/app1',
        confidence: 0.9,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const profile2: ApplicationProfile = {
        name: 'App 2',
        category: 'API',
        path: '/app2',
        confidence: 0.85,
        languages: ['JavaScript'],
        frameworks: ['Fastify'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const stored1 = await service.storeProfile(profile1, workspaceUri, true);
      const stored2 = await service.storeProfile(profile2, workspaceUri, false);

      await service.activateProfile(stored2.id);

      const retrieved1 = service.getProfileById(stored1.id);
      const retrieved2 = service.getProfileById(stored2.id);

      assert.strictEqual(retrieved1?.isActive, false);
      assert.strictEqual(retrieved2?.isActive, true);
    });

    test('should get active profile for workspace', async () => {
      const workspaceUri = 'file:///workspace';

      const profile: ApplicationProfile = {
        name: 'Active App',
        category: 'Web Application',
        path: '/active',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      await service.storeProfile(profile, workspaceUri, true);

      const active = service.getActiveProfile(workspaceUri);

      assert.ok(active);
      assert.strictEqual(active?.name, 'Active App');
      assert.strictEqual(active?.isActive, true);
    });
  });

  suite('Profile Deletion', () => {
    test('should delete profile by ID', async () => {
      const workspaceUri = 'file:///workspace';

      const profile: ApplicationProfile = {
        name: 'To Delete',
        category: 'Web Application',
        path: '/delete',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const stored = await service.storeProfile(profile, workspaceUri);
      const deleted = await service.deleteProfile(stored.id);

      assert.strictEqual(deleted, true);

      const retrieved = service.getProfileById(stored.id);
      assert.strictEqual(retrieved, undefined);
    });

    test('should return false when deleting non-existent profile', async () => {
      const deleted = await service.deleteProfile('non-existent-id');
      assert.strictEqual(deleted, false);
    });

    test('should remove profile from workspace list on delete', async () => {
      const workspaceUri = 'file:///workspace';

      const profile: ApplicationProfile = {
        name: 'To Delete',
        category: 'Web Application',
        path: '/delete',
        confidence: 0.95,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const stored = await service.storeProfile(profile, workspaceUri);
      await service.deleteProfile(stored.id);

      const profiles = service.getWorkspaceProfiles(workspaceUri);
      assert.strictEqual(profiles.length, 0);
    });
  });

  suite('Clear All Profiles', () => {
    test('should clear all profiles', async () => {
      const workspaceUri = 'file:///workspace';

      const profile1: ApplicationProfile = {
        name: 'App 1',
        category: 'Web Application',
        path: '/app1',
        confidence: 0.9,
        languages: ['TypeScript'],
        frameworks: ['Express.js'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      const profile2: ApplicationProfile = {
        name: 'App 2',
        category: 'API',
        path: '/app2',
        confidence: 0.85,
        languages: ['JavaScript'],
        frameworks: ['Fastify'],
        buildTools: ['webpack'],
        evidence: ['package.json']
      };

      await service.storeProfile(profile1, workspaceUri);
      await service.storeProfile(profile2, workspaceUri);

      await service.clearAllProfiles();

      const allProfiles = service.getAllProfiles();
      assert.strictEqual(allProfiles.length, 0);
    });
  });
});
