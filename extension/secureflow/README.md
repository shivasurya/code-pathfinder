
<div align="center">

[![VS Code Marketplace](https://img.shields.io/visual-studio-marketplace/v/codepathfinder.secureflow?label=VS%20Code&logo=visualstudiocode)](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow)
[![npm version](https://img.shields.io/npm/v/@codepathfinder/secureflow-cli?logo=npm)](https://www.npmjs.com/package/@codepathfinder/secureflow-cli)
[![Open VSX](https://img.shields.io/open-vsx/v/codepathfinder/secureflow?label=Open%20VSX&logo=vscodium)](https://open-vsx.org/extension/codepathfinder/secureflow)

</div>

# SecureFlow AI

[SecureFlow AI](https://codepathfinder.dev/secureflow-ai) is a VS Code extension that runs AI-powered security analysis on your code. It finds potential vulnerabilities without leaving your editor.

## Features

- **Profile-Based Scanning**: Detects your project stack and runs targeted security analysis
- **Multi-Provider Support**: Works with Anthropic Claude, OpenAI, Google Gemini, or OpenRouter (200+ models)
- **Svelte UI**: Interface with intuitive navigation and real-time updates
- **Detailed Reports**: Vulnerability reports include severity ratings, file locations, and recommendations
- **Quick Analysis**: Run security scans on git changes or full workspace
- **Scan History**: Track all security scans with auto-incrementing scan numbers and profile linkage
- **In-Editor**: All analysis runs in VS Code with no context switching

## Getting Started

### 1. Installation

Install from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow) or search for "SecureFlow" in VS Code Extensions.

### 2. Configuration

Configure your AI provider in VS Code settings:

**Required Settings:**
- `secureflow.Provider`: Choose your AI provider (auto/anthropic/openai/google/openrouter)
- `secureflow.AIModel`: Select the AI model for security analysis
- `secureflow.APIKey`: Your API key for the selected provider

**Supported Providers:**
- **Anthropic Claude**: Claude Sonnet 4.5 (recommended)
- **OpenAI**: GPT-4o, o1, and other OpenAI models
- **Google Gemini**: Gemini 2.5 Pro or Flash models
- **OpenRouter**: Access 200+ models from multiple providers through a single API

**Getting API Keys:**
- Anthropic: [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys)
- OpenAI: [platform.openai.com/api-keys](https://platform.openai.com/api-keys)
- Google: [aistudio.google.com/apikey](https://aistudio.google.com/apikey)
- OpenRouter: [openrouter.ai/settings/keys](https://openrouter.ai/settings/keys)

### 3. Usage

**Profile Your Workspace:**
1. Open the SecureFlow view in the Activity Bar
2. Click "Profile Workspace" or run command: `SecureFlow: Profile Workspace for Security Analysis`
3. Review detected application profiles and select one to scan

**Run Security Analysis:**
- **Full Profile Scan**: Click "Scan" button on any detected profile
- **Git Changes**: Run `SecureFlow: Review Git Changes for Security Issues`
- **Quick Scan**: Use the "Scan Profile" action from the profiles list

**View Results:**
- Navigate to the Results tab to see all scan history
- Click on any scan to view detailed vulnerability findings
- Review severity levels: Critical, High, Medium, Low, Info

## License Notice

For full license terms, see the [LICENSE](LICENSE) file
