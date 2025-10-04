<div align="center">

[![VS Code Marketplace](https://img.shields.io/visual-studio-marketplace/v/codepathfinder.secureflow?label=VS%20Code&logo=visualstudiocode)](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow)
[![npm version](https://img.shields.io/npm/v/@codepathfinder/secureflow-cli?logo=npm)](https://www.npmjs.com/package/@codepathfinder/secureflow-cli)
[![Open VSX](https://img.shields.io/open-vsx/v/codepathfinder/secureflow?label=Open%20VSX&logo=vscodium)](https://open-vsx.org/extension/codepathfinder/secureflow)

</div>

# SecureFlow CLI

🛡️ **AI-Powered Security Analysis for Your Codebase**

SecureFlow CLI is a powerful command-line tool that performs comprehensive security analysis of your projects using advanced AI models. It intelligently analyzes your code structure, identifies vulnerabilities, and provides actionable security insights with a beautiful TUI interface.

## ✨ Features

- 🤖 **AI-Powered Analysis** - Supports 13+ AI models including Claude, GPT, and Gemini
- 🔍 **Intelligent File Discovery** - Smart project analysis with iterative file request system
- 🎯 **Comprehensive Scanning** - Full project security analysis with context-aware insights
- 📊 **Multiple Output Formats** - Text, JSON, and DefectDojo integration
- 🏗️ **Project Profiling** - Technology stack detection and application type identification
- 🎨 **Beautiful TUI** - Claude-style terminal interface with colored output and progress indicators

## 🚀 Quick Start

### 1. Installation

From the repository root:

```bash
# Run directly
node packages/secureflow-cli/bin/secureflow --help

# Or install globally (future)
npm install -g @codepathfinder/secureflow-cli
```

### 2. Configure Your AI Model

SecureFlow CLI requires an AI model to perform analysis. Set up your API key:

```bash
# Check current configuration
secureflow config --show

# The CLI will prompt for API key configuration on first run
# Or manually edit the config file shown in the output
```

**Supported Models:**
- **Anthropic Claude**: `claude-sonnet-4-5-20250929` (recommended), `claude-opus-4-1-20250805`, `claude-sonnet-4-20250514`, `claude-3-7-sonnet-20250219`, `claude-3-5-haiku-20241022`, ~~`claude-3-5-sonnet-20241022`~~ (deprecated)
- **OpenAI**: `gpt-4o`, `gpt-4o-mini`, `o1`, `o1-mini`, `gpt-4.1-2025-04-14`, `o3-mini-2025-01-31`
- **Google Gemini**: `gemini-2.5-pro`, `gemini-2.5-flash`
- **xAI Grok**: `grok-4-fast-reasoning`
- **Ollama**: `qwen3:4b`

### 3. Run Your First Scan

```bash
# Scan current directory with default model
secureflow scan

# Scan specific project with Claude
secureflow scan ./my-project --model claude-sonnet-4-5-20250929

# Get project profile first
secureflow profile ./my-project
```

## 📋 Commands

### `scan` - Security Analysis

Performs comprehensive AI-powered security analysis of your project.

```bash
secureflow scan [path] [options]
```

**Options:**
- `--model <model>` - AI model to use for analysis
- `--format <format>` - Output format: `text`, `json`, or `defectdojo` (default: `text`)
- `--output <file>` - Save results to file
- `--defectdojo` - Export in DefectDojo format (shorthand for `--format defectdojo`)

**DefectDojo Integration:**
```bash
secureflow scan \
  --format defectdojo \
  --defectdojo-url https://defectdojo.example.com \
  --defectdojo-token your-api-token \
  --defectdojo-product-id 123 \
  --output findings.json
```

**DefectDojo Options:**
- `--defectdojo-url <url>` - DefectDojo instance URL
- `--defectdojo-token <token>` - API token for authentication
- `--defectdojo-product-id <id>` - Product ID to submit findings
- `--defectdojo-engagement-id <id>` - Engagement ID (optional, will create if not provided)
- `--defectdojo-test-title <title>` - Test title (default: "SecureFlow Scan")

### `profile` - Project Analysis

Analyzes project structure and identifies technologies, frameworks, and application types.

```bash
secureflow profile [path] [options]
```

**Options:**
- `--model <model>` - AI model to use for analysis
- `--format <format>` - Output format: `text` or `json` (default: `text`)
- `--output <file>` - Save results to file

### `config` - Configuration Management

View and manage CLI configuration.

```bash
secureflow config --show          # Show masked configuration
secureflow config --show --raw    # Show raw configuration (use with caution)
```

## 🔧 Configuration

SecureFlow CLI stores configuration in a local config file. The location is shown when running `secureflow config --show`.

**Example Configuration:**
```json
{
  "model": "grok-4-fast-reasoning",
  "apiKey": "xai-token",
  "provider": "grok",
  "analytics": {
    "enabled": false
  }
}
```

**Getting API Keys:**
- **Anthropic (Claude)**: [console.anthropic.com](https://console.anthropic.com)
- **OpenAI**: [platform.openai.com](https://platform.openai.com)
- **Google**: [ai.google.dev](https://ai.google.dev)
- **Grok (xAI)**: [console.x.ai](https://console.x.ai)

## 🎯 Usage Examples

### Basic Security Scan
```bash
# Scan current directory
secureflow scan

# Scan with specific model
secureflow scan --model grok-4-fast-reasoning

# Save results to file
secureflow scan --output security-report.json --format json
```

### Project Profiling
```bash
# Profile current project
secureflow profile

# Profile specific directory
secureflow profile ./backend --format json
```

### DefectDojo Integration
```bash
# Export to DefectDojo with minimal setup
secureflow scan \
  --defectdojo \
  --defectdojo-url https://defectdojo.company.com \
  --defectdojo-token $DEFECTDOJO_TOKEN \
  --defectdojo-product-id 42

# Full DefectDojo configuration
secureflow scan \
  --format defectdojo \
  --defectdojo-url https://defectdojo.company.com \
  --defectdojo-token $DEFECTDOJO_TOKEN \
  --defectdojo-product-id 42 \
  --defectdojo-engagement-id 123 \
  --defectdojo-test-title "Weekly Security Scan" \
  --output weekly-findings.json
```

## 🏗️ How It Works

SecureFlow CLI uses an innovative **LLM File Request System** that works like tool calling:

1. **Project Discovery** - Analyzes project structure and identifies key files
2. **Iterative Analysis** - AI makes targeted file requests using XML-like syntax:
   ```xml
   <file_request path="./src/auth.js" reason="Analyze authentication logic" />
   <list_file_request path="./src/components" reason="Explore component structure" />
   ```
3. **Security Analysis** - Performs up to 3 iterations of analysis with context building
4. **Report Generation** - Outputs comprehensive security findings with severity levels

**Security Features:**
- ✅ Hidden file filtering (ignores `.git`, `.DS_Store`, etc.)
- ✅ Symlink protection against directory traversal
- ✅ Project scope validation
- ✅ File size limits (large files truncated)
- ✅ Comprehensive request logging

## 🎨 Output Formats

### Text Format (Default)
Beautiful colored terminal output with:
- 🔴 Critical vulnerabilities
- 🟠 High severity issues  
- 🟡 Medium severity warnings
- 🔵 Low severity notes
- ℹ️ Informational findings

### JSON Format
Structured output perfect for CI/CD integration:
```json
{
  "summary": {
    "totalIssues": 5,
    "critical": 1,
    "high": 2,
    "medium": 1,
    "low": 1
  },
  "findings": [...]
}
```

### DefectDojo Format
Direct integration with DefectDojo security platforms:
- Compliant with Generic Findings Import format
- Automatic severity mapping
- File path and line number extraction
- CWE/CVE detection and tagging

---

**Need Help?** 
- Run `secureflow --help` for command overview
- Open an issue or discussion on [GitHub](https://github.com/shivasurya/code-pathfinder/issues)
