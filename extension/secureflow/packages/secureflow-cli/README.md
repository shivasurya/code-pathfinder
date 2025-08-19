# SecureFlow CLI (Scaffold)

Barebones CLI scaffold to prepare for reusing the SecureFlow VS Code extension logic.

- Extension remains unaffected.
- Commands are placeholders and will be filled in incrementally.

## Install (local)

From repo root after cloning:

```
node packages/secureflow-cli/bin/secureflow --help
```

Or add an npm script later to run via workspaces.

## Usage (current scaffold)

```
secureflow --version
secureflow scan --file path/to/file --range 10:50 --format json
secureflow profile
```

These commands currently print placeholders and exit with code 0.
