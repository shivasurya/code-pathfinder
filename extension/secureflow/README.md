# SecureFlow

SecureFlow is your AI security co-pilot for smarter, safer code, right in your editor. This VS Code extension helps you identify potential security vulnerabilities in your code directly within your development workflow.

## Features

- **Quick Security Analysis**: Select any code snippet and press `Cmd+L` (Mac) to analyze it for security vulnerabilities
- **Real-time Feedback**: Get immediate feedback on potential security issues
- **Detailed Reports**: View comprehensive reports with severity ratings, descriptions, and recommendations
- **In-Editor Experience**: All analysis happens right in your VS Code editor with no need to switch contexts

![SecureFlow Analysis Demo](https://example.com/images/secureflow-demo.png)

## Usage

1. Select a block of code in your editor
2. Press `Cmd+L` (Mac) or `Ctrl+L` (Windows/Linux)
3. View the security analysis results in the output panel

SecureFlow analyzes your code for common security vulnerabilities, including:
- SQL Injection
- Cross-Site Scripting (XSS)
- Hardcoded Secrets
- Insecure Random Number Generation
- And more...

## Requirements

- VS Code version 1.102.0 or higher

## Extension Settings

This extension does not add any settings yet. Settings to customize the analysis will be added in future versions.

## Known Issues

- This is an early version with limited analysis capabilities
- Only a subset of common security vulnerabilities are detected

## Release Notes

### 0.0.1

Initial release of SecureFlow with basic security analysis features:
- Code selection analysis with `Cmd+L`
- Detection of basic security patterns
- Output panel with security reports

---

## Following extension guidelines

Ensure that you've read through the extensions guidelines and follow the best practices for creating your extension.

* [Extension Guidelines](https://code.visualstudio.com/api/references/extension-guidelines)

## Working with Markdown

You can author your README using Visual Studio Code. Here are some useful editor keyboard shortcuts:

* Split the editor (`Cmd+\` on macOS or `Ctrl+\` on Windows and Linux).
* Toggle preview (`Shift+Cmd+V` on macOS or `Shift+Ctrl+V` on Windows and Linux).
* Press `Ctrl+Space` (Windows, Linux, macOS) to see a list of Markdown snippets.

## For more information

* [Visual Studio Code's Markdown Support](http://code.visualstudio.com/docs/languages/markdown)
* [Markdown Syntax Reference](https://help.github.com/articles/markdown-basics/)

**Enjoy!**
