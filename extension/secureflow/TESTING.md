# SecureFlow Extension Testing Strategy

## Overview

This document outlines the testing strategy for the SecureFlow VSCode extension, focusing on the features added in the OpenRouter integration PR.

## Test Infrastructure

### Setup ✅
- **Test Framework**: Mocha + @vscode/test-electron
- **Test Location**: `src/test/suite/`
- **Test Runner**: `.vscode-test.mjs` configured
- **Dependencies**: Installed glob, @types/glob

### Run Tests
```bash
npm run compile-tests  # Compile TypeScript tests
npm test                # Run all tests
```

## Test Coverage Plan

### 1. Settings Manager (`settings-manager.test.ts`) 🚧

**What to Test:**
- ✅ API Key storage/retrieval via VSCode secrets API
- ✅ Model selection get/set via workspace configuration
- ✅ Provider selection (auto/anthropic/openai/openrouter)
- ✅ Analytics settings

**Status**: Created but needs type fixes

**Key Test Cases:**
```typescript
✓ Store and retrieve API key
✓ Handle missing API key (undefined)
✓ Get/set selected AI model
✓ Default to claude-sonnet-4-5-20250929 when not configured
✓ Detect OpenRouter model format (contains '/')
✓ Provider selection (auto/explicit)
```

### 2. Profile Storage Service (`profile-storage-service.test.ts`) 🚧

**What to Test:**
- ✅ Profile CRUD operations
- ✅ Workspace-specific profile lists
- ✅ Profile activation (one active per workspace)
- ✅ Profile deletion and cleanup

**Status**: Created but needs property name fixes (frameworks vs framework)

**Key Test Cases:**
```typescript
✓ Store profile with generated ID
✓ Retrieve profile by ID
✓ Get profiles by workspace
✓ Activate/deactivate profiles
✓ Delete profile removes from workspace list
✓ Clear all profiles
```

### 3. Scan Storage Service (`scan-storage-service.test.ts`) 🚧

**What to Test:**
- ✅ Scan result storage with auto-increment scan numbers
- ✅ Severity summary calculation
- ✅ Profile linkage (profileId)
- ✅ Scan retrieval (by number, by profile, latest)

**Status**: Created but needs property name fixes (summary structure)

**Key Test Cases:**
```typescript
✓ Save scan with auto-incrementing number
✓ Link scan to profile
✓ Calculate severity breakdown (Critical/High/Medium/Low)
✓ Retrieve scan by number
✓ Get scans for profile
✓ Get latest scan
✓ Clear all scans resets counter
```

### 4. Profile Scan Service (`profile-scan-service.test.ts`) 🚧

**What to Test:**
- ✅ Scan orchestration
- ✅ API key validation
- ✅ Provider detection from model ID
- ✅ Silent mode configuration
- ✅ Progress callbacks

**Status**: Created but needs mock refinement

**Key Test Cases:**
```typescript
✓ Require API key for scan
✓ Detect OpenRouter from model ID pattern
✓ Enable silent mode for extension usage
✓ Handle progress callbacks
✓ Map scan results correctly
```

## Fixing the Tests

### Step 1: Type Corrections Needed

1. **ApplicationProfile** properties:
   - Use `frameworks: string[]` (not `framework: string`)
   - Use `languages: string[]` (not `language: string`)
   - Use `buildTools: string[]` (not `buildTool: string`)
   - Include all required fields: `name`, `path`, `category`, `confidence`, `languages`, `frameworks`, `buildTools`, `evidence`

2. **ScanResult** properties:
   - Use `fileCount: number` (not `filesAnalyzed`)
   - `summary` is `string` (not object with `.critical`, `.high`, etc.)
   - Add `reviewContent: string` and `timestampFormatted: string`

3. **SettingsManager** methods:
   - Available: `getSelectedProvider()`, `getSelectedAIModel()`, `getApiKey()`
   - NOT available: `setApiKey()`, `deleteApiKey()`, `setSelectedProvider()`, `isAnalyticsEnabled()`
   - Settings are managed via VSCode workspace config, not direct setters

4. **ScanStorageService** methods:
   - Use `getScansForProfile(profileId)` (not `getScansByProfileId`)
   - Use `getScanByNumber(num)` (not `getScan`)

### Step 2: Mocha Fix

```typescript
// src/test/suite/index.ts
import Mocha from 'mocha';  // Change import

const mocha = new Mocha({   // This will work
  ui: 'tdd',
  color: true,
  timeout: 10000
});
```

### Step 3: Mock Workspace (Don't Assign to vscode.workspace)

VSCode's `workspace` is read-only. Use dependency injection instead:

```typescript
// Instead of: vscode.workspace = mockWorkspace
// Create mock in test setup and pass to constructor if possible
// Or test integration-style with actual VSCode API
```

## Test Execution Strategy

### Phase 1: Unit Tests (Current Focus) 🎯
Focus on isolated testing of business logic:
- SettingsManager configuration management
- ProfileStorageService CRUD operations
- ScanStorageService data persistence
- Utility functions and helpers

### Phase 2: Integration Tests (Future)
Test component interactions:
- Settings → Profile Scan flow
- Profile creation → Scan → Results storage
- Webview message passing
- File system operations

### Phase 3: E2E Tests (Future)
Full workflow testing:
- Onboarding flow
- Profile scanning end-to-end
- Settings changes propagation
- UI interactions

## Coverage Goals

| Component | Target Coverage | Priority |
|-----------|----------------|----------|
| Settings Manager | 80%+ | High |
| Profile Storage | 85%+ | High |
| Scan Storage | 85%+ | High |
| Profile Scan Service | 70%+ | Medium |
| UI Components (Svelte) | 60%+ | Low |
| Explorer (integration) | 50%+ | Low |

## Running Specific Tests

```bash
# Compile and run all tests
npm test

# Run specific test suite
npm test -- --grep "SettingsManager"
npm test -- --grep "ProfileStorage"
npm test -- --grep "ScanStorage"
```

## CI/CD Integration

### GitHub Actions Workflow
```yaml
name: Test Extension
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm run compile-tests
      - run: xvfb-run -a npm test
```

## Current Status

### ✅ Completed
- Test infrastructure setup
- Test directory structure
- Test suite files created
- Test dependencies installed

### 🚧 In Progress
- Fixing type errors
- Aligning with actual API signatures
- Refining mocks

### 📋 Next Steps
1. Fix ApplicationProfile property names in tests
2. Fix ScanResult structure in tests
3. Update SettingsManager tests to use actual API
4. Update ScanStorageService method names
5. Run and verify all tests pass
6. Add coverage reporting
7. Integrate into CI/CD

## Notes

- Tests use Mocha's TDD style (`suite`, `test`, `setup`)
- VSCode extension testing requires `@vscode/test-electron` for integration tests
- Mock data should match production types exactly
- Consider adding test utilities for common mock creation

## Resources

- [VSCode Extension Testing](https://code.visualstudio.com/api/working-with-extensions/testing-extension)
- [Mocha Documentation](https://mochajs.org/)
- [@vscode/test-cli](https://github.com/microsoft/vscode-test-cli)
