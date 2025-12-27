import * as assert from 'assert';
import * as vscode from 'vscode';
import { ScanStorageService } from '../../services/scan-storage-service';

suite('ScanStorageService Test Suite', () => {
  let service: ScanStorageService;
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
      }
    } as any;

    service = new ScanStorageService(mockContext);
  });

  suite('Scan Storage', () => {
    test('should save scan with auto-incrementing scan number', async () => {
      const issues = [
        {
          issue: {
            title: 'SQL Injection',
            severity: 'High' as const,
            description: 'Potential SQL injection vulnerability',
            recommendation: 'Use parameterized queries'
          },
          filePath: 'src/db.ts',
          startLine: 42
        }
      ];

      const scan1 = await service.saveScan(
        issues,
        'Scan complete',
        'Found 1 issue',
        10,
        'claude-sonnet-4-5-20250929'
      );

      assert.strictEqual(scan1.scanNumber, 1);
      assert.strictEqual(scan1.issues.length, 1);
      assert.strictEqual(scan1.fileCount, 10);

      const scan2 = await service.saveScan(
        [],
        'Clean scan',
        'No issues found',
        15,
        'claude-sonnet-4-5-20250929'
      );

      assert.strictEqual(scan2.scanNumber, 2);
      assert.strictEqual(scan2.issues.length, 0);
    });

    test('should link scan to profile', async () => {
      const profileId = 'test-profile-123';
      const issues: any[] = [];

      const scan = await service.saveScan(
        issues,
        'Profile scan',
        'Scan for specific profile',
        5,
        'claude-sonnet-4-5-20250929',
        profileId
      );

      assert.strictEqual(scan.profileId, profileId);
    });

    test('should store issues with different severities', async () => {
      const issues = [
        {
          issue: {
            title: 'Critical Issue 1',
            severity: 'Critical' as const,
            description: 'Critical security issue',
            recommendation: 'Fix immediately'
          },
          filePath: 'file1.ts',
          startLine: 1
        },
        {
          issue: {
            title: 'Critical Issue 2',
            severity: 'Critical' as const,
            description: 'Another critical issue',
            recommendation: 'Fix immediately'
          },
          filePath: 'file2.ts',
          startLine: 2
        },
        {
          issue: {
            title: 'High Issue',
            severity: 'High' as const,
            description: 'High priority issue',
            recommendation: 'Fix soon'
          },
          filePath: 'file3.ts',
          startLine: 3
        },
        {
          issue: {
            title: 'Medium Issue',
            severity: 'Medium' as const,
            description: 'Medium priority issue',
            recommendation: 'Fix when possible'
          },
          filePath: 'file4.ts',
          startLine: 4
        },
        {
          issue: {
            title: 'Low Issue',
            severity: 'Low' as const,
            description: 'Low priority issue',
            recommendation: 'Consider fixing'
          },
          filePath: 'file5.ts',
          startLine: 5
        }
      ];

      const scan = await service.saveScan(
        issues,
        'Test scan',
        'Testing severity calculation',
        10,
        'claude-sonnet-4-5-20250929'
      );

      // Verify all issues were stored
      assert.strictEqual(scan.issues.length, 5);

      // Calculate severity counts from issues
      const criticalCount = scan.issues.filter(i => i.issue.severity === 'Critical').length;
      const highCount = scan.issues.filter(i => i.issue.severity === 'High').length;
      const mediumCount = scan.issues.filter(i => i.issue.severity === 'Medium').length;
      const lowCount = scan.issues.filter(i => i.issue.severity === 'Low').length;

      assert.strictEqual(criticalCount, 2);
      assert.strictEqual(highCount, 1);
      assert.strictEqual(mediumCount, 1);
      assert.strictEqual(lowCount, 1);
    });
  });

  suite('Scan Retrieval', () => {
    setup(async () => {
      // Clear all scans before each test in this suite for isolation
      await service.clearAllScans();
    });

    test('should get all scans', async () => {
      await service.saveScan([], 'Scan 1', 'First scan', 5, 'model-1');
      await service.saveScan([], 'Scan 2', 'Second scan', 10, 'model-2');
      await service.saveScan([], 'Scan 3', 'Third scan', 15, 'model-3');

      const allScans = service.getAllScans();

      assert.strictEqual(allScans.length, 3);
      assert.strictEqual(allScans[0].scanNumber, 3);
      assert.strictEqual(allScans[1].scanNumber, 2);
      assert.strictEqual(allScans[2].scanNumber, 1);
    });

    test('should get scan by number', async () => {
      await service.saveScan([], 'Scan 1', 'First scan', 5, 'model-1');
      const saved = await service.saveScan([], 'Scan 2', 'Second scan', 10, 'model-2');

      const retrieved = service.getScanByNumber(2);

      assert.ok(retrieved);
      assert.strictEqual(retrieved?.scanNumber, 2);
      assert.strictEqual(retrieved?.summary, 'Scan 2');
    });

    test('should return undefined for non-existent scan number', () => {
      const retrieved = service.getScanByNumber(999);
      assert.strictEqual(retrieved, undefined);
    });

    test('should get scans by profile ID', async () => {
      const profileId = 'test-profile-123';

      await service.saveScan([], 'Scan 1', 'Profile scan 1', 5, 'model-1', profileId);
      await service.saveScan([], 'Scan 2', 'Other scan', 10, 'model-2', 'other-profile');
      await service.saveScan([], 'Scan 3', 'Profile scan 2', 15, 'model-3', profileId);

      const profileScans = service.getScansForProfile(profileId);

      assert.strictEqual(profileScans.length, 2);
      assert.ok(profileScans.every(scan => scan.profileId === profileId));
    });

    test('should return empty array when no scans for profile', () => {
      const scans = service.getScansForProfile('non-existent-profile');
      assert.strictEqual(scans.length, 0);
    });
  });

  suite('Latest Scan', () => {
    test('should get latest scan', async () => {
      await service.saveScan([], 'Scan 1', 'First scan', 5, 'model-1');
      await service.saveScan([], 'Scan 2', 'Second scan', 10, 'model-2');
      const latest = await service.saveScan([], 'Scan 3', 'Third scan', 15, 'model-3');

      const retrieved = service.getLatestScan();

      assert.ok(retrieved);
      assert.strictEqual(retrieved?.scanNumber, latest.scanNumber);
      assert.strictEqual(retrieved?.summary, 'Scan 3');
    });

    test('should return undefined when no scans exist', () => {
      const latest = service.getLatestScan();
      assert.strictEqual(latest, undefined);
    });
  });

  suite('Clear Scans', () => {
    test('should clear all scans', async () => {
      await service.saveScan([], 'Scan 1', 'First scan', 5, 'model-1');
      await service.saveScan([], 'Scan 2', 'Second scan', 10, 'model-2');

      await service.clearAllScans();

      const allScans = service.getAllScans();
      assert.strictEqual(allScans.length, 0);
    });

    test('should reset scan number after clearing', async () => {
      await service.saveScan([], 'Scan 1', 'First scan', 5, 'model-1');
      await service.saveScan([], 'Scan 2', 'Second scan', 10, 'model-2');

      await service.clearAllScans();

      const newScan = await service.saveScan([], 'New Scan', 'After clear', 5, 'model-1');
      assert.strictEqual(newScan.scanNumber, 1);
    });
  });

  suite('Scan Metadata', () => {
    test('should include timestamp in saved scan', async () => {
      const beforeSave = Date.now();
      const scan = await service.saveScan([], 'Test', 'Details', 5, 'model-1');
      const afterSave = Date.now();

      const timestamp = new Date(scan.timestamp).getTime();

      assert.ok(timestamp >= beforeSave);
      assert.ok(timestamp <= afterSave);
    });

    test('should store model information', async () => {
      const model = 'anthropic/claude-3-5-sonnet';
      const scan = await service.saveScan([], 'Test', 'Details', 5, model);

      assert.strictEqual(scan.model, model);
    });

    test('should store detailed summary', async () => {
      const summaryMessage = 'Test scan';
      const reviewContent = 'Analyzed 100 files in 3 iterations';
      const scan = await service.saveScan([], summaryMessage, reviewContent, 100, 'model-1');

      assert.strictEqual(scan.summary, summaryMessage);
      assert.strictEqual(scan.reviewContent, reviewContent);
    });
  });
});
