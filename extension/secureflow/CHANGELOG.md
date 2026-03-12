# SecureFlow Changelog

## Version 0.0.19 - March 7, 2026

### What's New

- **Next-Gen AI Models**: Added latest flagship models from all major providers
  - **GPT-5.4 & GPT-5.4 Pro**: OpenAI's most capable models (1.05M context, 128K output)
  - **Claude Opus 4.6**: Anthropic's most intelligent model for agents and code (200K context, 128K output, 1M beta)
  - **Claude Sonnet 4.6**: New recommended default — best combination of speed and intelligence (200K context, 64K output, 1M beta)
  - **Gemini 3.1 Pro Preview**: Google's latest with advanced agentic and coding capabilities (1M context)
- **Updated OpenRouter Models**: Upgraded to latest versions of open-source models
  - Qwen3 Coder → **Qwen3 Coder Next** (262K context/output)
  - GLM 4.7 → **GLM 5** (202K context/output)
  - MiniMax M2 → **MiniMax M2.5** (196K context/output)
  - Added **Kimi K2.5** — multimodal model with visual coding capabilities (262K context/output)
- **xAI Grok Models**: Added to model-context-limits configuration for full parity

### Improvements

- **Default Model Updated**: Changed default from deprecated Claude Sonnet 4.5 to Claude Sonnet 4.6 across all components
- **Spec Accuracy Fixes**: Corrected pre-existing specification errors verified against official docs
  - GPT-5 Mini: contextWindow 200K→400K, maxOutput 64K→128K
  - GPT-5 Pro: maxOutput 128K→272K
  - GPT-5 Nano: contextWindow 128K→400K, maxOutput 32K→128K
  - OpenRouter model maxOutput values corrected to match official specs
- **Updated Documentation**: Refreshed README and CLI README with current model listings

### Deprecated

- **Claude Sonnet 4.5** (`claude-sonnet-4-5-20250929`): Superseded by Claude Sonnet 4.6
- **Claude Opus 4.5** (`claude-opus-4-5`): Superseded by Claude Opus 4.6
- **Gemini 3 Pro** (`gemini-3-pro-preview`): Shutting down March 9, 2026 — use Gemini 3.1 Pro instead

## Version 0.0.18 - December 27, 2025

### What's New

- **Latest AI Models**: Support for cutting-edge AI models from OpenAI, Google, xAI, and OpenRouter
  - **GPT-5.2**: OpenAI's best model for coding and agentic tasks across industries (400K context, 128K output)
  - **Gemini 3 Pro & Flash**: Google's newest multimodal AI with thinking capabilities (1M context)
  - **Grok 4.1 Fast**: xAI's frontier model optimized for high-performance agentic tool calling (2M context)
  - **MiniMax M2**: Compact high-efficiency model optimized for coding and agentic workflows via OpenRouter (196K context)
  - **DeepSeek V3.2**: High computational efficiency with strong reasoning and agentic tool-use performance via OpenRouter (163K context)
  - All models available at the top of Settings and Onboarding for easy selection
- **OpenRouter Support**: Access 200+ AI models from multiple providers through a single API key
  - Use models from Anthropic, OpenAI, Google, Meta, Mistral, and many more
  - Switch between models without changing API keys
  - Perfect for comparing different AI models for security analysis
- **Modern User Interface**: Completely redesigned interface with better navigation
  - New Profile management page to organize your scans
  - Dedicated Results page with full scan history
  - Easy-to-use Settings page for configuration
  - Cleaner, more intuitive layout
- **Enhanced Provider Selection**: Choose your preferred AI provider
  - New Provider setting in configuration
  - Auto-detection or manual selection
  - Supported providers: Anthropic Claude, OpenAI, Google Gemini, OpenRouter
- **Improved Workspace Profiling**: Better project detection and scanning
  - Automatically identifies your project's technology stack
  - One-click security scanning for detected profiles
  - Faster, more accurate analysis
- **Scan History**: Keep track of all your security scans
  - View complete history of all scans
  - See which profile each scan belongs to
  - Review severity breakdowns and trends over time

### Improvements

- **Cleaner Output**: Removed unnecessary logging and debug messages
- **Faster Performance**: Optimized for smaller bundle size and faster loading
- **Better Settings Experience**: Improved input validation and helpful guidance

## Version 0.0.17 - November 15, 2025

### 🔧 Improvements

- **TypeScript Build Fixes**: Fixed TypeScript compilation errors in generated model configuration files with proper type annotations

## Version 0.0.16 - November 14, 2025

### 🚀 What's New?

- **GPT-5.1 Model Support**: Added support for OpenAI's flagship GPT-5.1 model with configurable reasoning effort, 400K context window, and 128K output capacity for complex security workflows

## Version 0.0.15 - October 17, 2025

### 🚀 What's New?

- **Claude Haiku 4.5 Support**: Added support for Claude Haiku 4.5 model for enhanced security analysis

## Version 0.0.14 - October 12, 2025

### 🚀 What's New?

- **Updates to AI Model Support**: Added support for GPT-5 model family and updated model context limits with latest specs

## Version 0.0.12 - September 30, 2025

### 🚀 What's New?

- **Secureflow CLI Package**: Introduced standalone CLI tool for security analysis outside VS Code
- **Grok AI Model Support**: Added Grok 4 Fast Reasoning AI model support for enhanced security analysis
- **Say bye to claude-sonnet-3-5 model**: Removed deprecated claude-sonnet-3-5 model

### 🔧 Improvements

- **Enhanced Security Guidelines**: Improved file request rules and security review guidelines for more comprehensive analysis

## Version 0.0.11 - September 7, 2025

### 🚀 What's New?

- **SecureFlow CLI Package**: Introduced standalone CLI tool for security analysis outside VS Code
- **AI-Powered Security Scanner**: New iterative file analysis system with intelligent file request handling
- **Full Workspace Scanning**: Comprehensive security scanning with up to 10 iterations for thorough analysis
- **CLI Project Profiling**: AI-powered workspace analysis and profiling capabilities via command line
- **Configuration Management**: New CLI config command for managing API keys and settings

### 🔧 Improvements

- **Enhanced Security Guidelines**: Improved file request rules and security review guidelines for more comprehensive analysis
- **Modular Architecture**: Extracted AI client functionality into separate CLI package for better code organization
- **Prompt Template System**: Refactored prompt loading with asynchronous template management
- **Git Integration**: Enhanced git diff parsing logic moved to shared CLI package
- **Analytics Enhancement**: Enabled GeoIP tracking in PostHog analytics for better insights

### 🛠️ Technical Changes

- Removed deprecated workspace-profiler class and related interfaces
- Extracted git diff parsing logic into shared CLI package
- Moved prompts to CLI package with updated import structure
- Added workspace analyzer for AI-based project profiling

## Version 0.0.10 - August 17, 2025

### 🚀 What's New?

### 🔧 Improvements

- Use filename (not full path) in security analysis prompt for cleaner, less noisy context
- Remove debug logging during analysis to reduce noise

### 🐛 Bug Fixes

- Fix markdown rendering in webview output

## Version 0.0.9 - August 16, 2025

### 🐛 Bug Fixes

- **Issue with Sentry Error Reporting**: Fixed issue with Sentry error reporting


## Version 0.0.8 - August 14, 2025

### 🐛 Bug Fixes

- **Improved Parsing**: Enhanced parsing logic to handle XML responses more effectively
- **User Experience**: Update scan results list after each scan to provide immediate feedback


## Version 0.0.7 - August 11, 2025

### 🚀 What's New?

- **Workspace Profile Integration**: Enhanced security analysis with workspace profile context for more accurate and contextual vulnerability detection
- **Improved Selection Analysis UI**: Refined webview interface for better code selection analysis experience
- **Sentry Error Reporting**: Added comprehensive error tracking and reporting system for better debugging and user experience

### 🔧 Improvements

- **Code Formatting**: Applied prettier formatting across the codebase for consistent code style
- **Debug Log Cleanup**: Removed unnecessary debug logs to improve performance and reduce noise
- **Security Analyzer Enhancements**: Updated AI-powered security analysis with improved prompts and analysis logic

### 🐛 Bug Fixes

- **UX Issue Notifications**: Addressed annoying UX notification issues that were impacting user experience

## Version 0.0.6 - August 10, 2025

### 🚀 What's New?

- Added Anthropic Claude 4.1 Opus Model support

## Version 0.0.5 - August 3, 2025

### Chores

- Addressed some performance hiccups
- Fixed analytics identifier being generated multiple times

## Version 0.0.4 - July 28, 2025

### 🚀 What's New?

- Added better onboarding experience
- Fixed AI Client to support openai models


## Version 0.0.3 - July 27, 2025

### 🎯 What's New?
- **Gemini Client**: SecureFlow now supports Gemini 2.5 Pro and Gemini 2.5 Flash models.
- **Analytics**: SecureFlow now collects anonymous usage data to help improve the product. Only aggregated usage metrics are collected with no personal information. Restart Editor to apply this change.
  - **AI Model** - AI model used for security analysis
  

## Version 0.0.2 - July 26, 2025

### 🎯 What's New?
- **Analytics**: SecureFlow now collects anonymous usage data to help improve the product. Only aggregated usage metrics are collected with no personal information. Restart Editor to apply this change.

## Version 0.0.1 - July 25, 2025

🚀 First Release - Your AI Security Guardian

### 🎯 What's New?
- **One-Click Security Analysis**: Select your code and press `Cmd+L` to get instant security insights
- **Git Changes Review**: Automatically scan your code changes for security issues before committing
- **Workspace Security Profile**: Get a complete security overview of your entire project
- **Multi-AI Support**: Choose your preferred AI engine:
  - OpenAI Models
    - GPT-4O
    - GPT-4O Mini
    - O1 & O1 Mini
    - GPT-4.1 (2025)
    - O3 Mini (2025)
  - Google Models
    - Gemini 2.5 Pro
    - Gemini 2.5 Flash
  - Anthropic Models
    - Claude 4 Opus & Sonnet (2025)
    - Claude 3.7 Sonnet (2025)
    - Claude 3.5 Series (2024)

### 🛡️ Key Features
- Real-time security recommendations while you code
- Automatic vulnerability detection
- Built-in threat modeling
- Dark and light theme support
- Simple setup with minimal configuration

### 🔧 Getting Started
1. Install SecureFlow from VS Code Marketplace
2. Add your preferred AI model's API key
3. Start coding - SecureFlow works automatically in the background

### 📝 Note
This is our first release! We're actively working on improvements and would love your feedback.