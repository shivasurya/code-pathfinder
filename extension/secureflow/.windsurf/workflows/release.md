---
description: Release Preparation Workflow
---

This workflow is for release preparation step. Follow the step by step instruction below

1. Ensure current git branch is main if not stop proceeding immediately
2. Run git status check and only if result is completely empty or clean proceed to next step or stop proceeding immediately.
3. Under extension/secureflow/package.json, get the current version (example: 0.0.6) from root level
4. Bump the version number and perform `npm install` under `extension/secureflow/` directory to overwrite package-lock.json
5. Create a branch named shiva+windsurf/release-version-{bumped_version_number} from main
6. git add package.json, package-lock.json under `extension/secureflow/` directory
7. Using below template <CHANGELOG_TEMPLATE>, Add placeholder to extension/secureflow/CHANGELOG.md with bumped version. Basically append at the top below Secureflow changelog heading

<CHANGELOG_TEMPLATE>
## Version {BUMPED_VERSION} - {CURRENT_DATE - Format: August 11, 2025}

### 🚀 What's New?

- 

{newline}
<CHANGELOG_TEMPLATE>