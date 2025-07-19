# Welcome to your VS Code Extension

## About SecureFlow

SecureFlow is a VS Code extension that helps developers analyze and enhance code security. It provides tools for:

- Security vulnerability scanning
- Code analysis
- Best practice recommendations

## Getting Started

1. Install the extension from VS Code Marketplace
2. Open your project folder
3. Access SecureFlow commands via Command Palette
4. View security analysis results in the dedicated panel

## Features

- Real-time security scanning
- Configurable rule sets
- Detailed vulnerability reports
- Integration with popular security tools

## Security Commands

Access these commands via Command Palette (Cmd/Ctrl + Shift + P):

- `SecureFlow: Scan Project` - Full security scan of workspace
- `SecureFlow: Quick Scan` - Rapid scan of current file
- `SecureFlow: Generate Report` - Create detailed security report
- `SecureFlow: Check Dependencies` - Analyze dependency vulnerabilities
- `SecureFlow: Show Security Panel` - Open security analysis panel

## Workspace Setup

This extension works best with JavaScript/TypeScript projects and requires Node.js installed on your system.

## Configuration

Use `settings.json` to customize:
```json
{
  "secureflow.scanning.enabled": true,
  "secureflow.rules.severity": "high",
  "secureflow.report.format": "html"
}
```

## Support

For issues and feature requests, please visit our GitHub repository.

