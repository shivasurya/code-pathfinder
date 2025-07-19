/******/ (() => { // webpackBootstrap
/******/ 	"use strict";
/******/ 	var __webpack_modules__ = ([
/* 0 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.activate = activate;
exports.deactivate = deactivate;
// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
const vscode = __importStar(__webpack_require__(1));
const security_analyzer_1 = __webpack_require__(2);
const git_changes_1 = __webpack_require__(12);
const settings_manager_1 = __webpack_require__(15);
// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
function activate(context) {
    console.log('SecureFlow extension is now active!');
    // Create an output channel for security diagnostics
    const outputChannel = vscode.window.createOutputChannel('SecureFlow Security Diagnostics');
    // Initialize the settings manager
    const settingsManager = new settings_manager_1.SettingsManager(context);
    // Register the command that will be triggered with cmd+l
    const analyzeSelectionCommand = vscode.commands.registerCommand('secureflow.analyzeSelection', async () => {
        // Get the active text editor
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            vscode.window.showErrorMessage('No active editor found');
            return;
        }
        // Get the selected text
        const selection = editor.selection;
        const selectedText = editor.document.getText(selection);
        if (!selectedText) {
            vscode.window.showInformationMessage('No text selected for security analysis');
            return;
        }
        // Show progress indicator
        await vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: "Scanning for security issues...",
            cancellable: true
        }, async (progress, token) => {
            // Show output channel
            outputChannel.clear();
            outputChannel.show(true);
            outputChannel.appendLine('🔍 Analyzing code for security vulnerabilities...');
            // Get the selected AI Model
            const selectedModel = settingsManager.getSelectedAIModel();
            outputChannel.appendLine(`🤖 Using AI Model: ${selectedModel}`);
            // Simulate scanning process with some delay
            progress.report({ increment: 0 });
            // First stage - initial scanning
            await new Promise(resolve => setTimeout(resolve, 500));
            progress.report({ increment: 20, message: "Parsing code..." });
            outputChannel.appendLine('⏳ Parsing code structure...');
            // Second stage - deeper analysis
            await new Promise(resolve => setTimeout(resolve, 700));
            progress.report({ increment: 30, message: "Checking for vulnerabilities..." });
            outputChannel.appendLine('⏳ Checking for known vulnerability patterns...');
            // Third stage - final checks
            await new Promise(resolve => setTimeout(resolve, 800));
            progress.report({ increment: 50, message: "Finalizing analysis..." });
            outputChannel.appendLine('⏳ Running final security checks...');
            // Get the API key for the selected AI Model
            const aiModel = settingsManager.getSelectedAIModel();
            let securityIssues = [];
            try {
                // Try to get the API key for the selected model
                const apiKey = await settingsManager.getApiKey();
                if (apiKey) {
                    // If we have an API key, use the AI-powered analysis
                    outputChannel.appendLine(`⏳ Running AI-powered analysis with ${aiModel}...`);
                    securityIssues = await (0, security_analyzer_1.performSecurityAnalysisAsync)(selectedText, aiModel, apiKey);
                }
                else {
                    // Fallback to pattern-based analysis if no API key
                    outputChannel.appendLine('⚠️ No API key found for the selected AI Model. Using pattern-based analysis only.');
                    securityIssues = (0, security_analyzer_1.performSecurityAnalysis)(selectedText, aiModel);
                }
            }
            catch (error) {
                // If there's an error with the API key or AI analysis, fallback to pattern-based
                console.error('Error with AI analysis:', error);
                outputChannel.appendLine(`⚠️ Error connecting to ${aiModel}: ${error}. Using pattern-based analysis only.`);
                securityIssues = (0, security_analyzer_1.performSecurityAnalysis)(selectedText, aiModel);
            }
            // Complete the progress
            await new Promise(resolve => setTimeout(resolve, 500));
            // Display results
            outputChannel.appendLine('\n✅ Security analysis complete!\n');
            if (securityIssues.length === 0) {
                outputChannel.appendLine('🎉 No security issues found in the selected code.');
            }
            else {
                outputChannel.appendLine(`⚠️ Found ${securityIssues.length} potential security issues:\n`);
                securityIssues.forEach((issue, index) => {
                    outputChannel.appendLine(`Issue #${index + 1}: ${issue.title}`);
                    outputChannel.appendLine(`Severity: ${issue.severity}`);
                    outputChannel.appendLine(`Description: ${issue.description}`);
                    outputChannel.appendLine(`Recommendation: ${issue.recommendation}\n`);
                });
            }
        });
    });
    // Register the git changes review command and status bar button
    (0, git_changes_1.registerSecureFlowReviewCommand)(context, outputChannel, settingsManager);
    // Register command to set API key
    const setApiKeyCommand = vscode.commands.registerCommand('secureflow.setApiKey', async () => {
        const aiModel = settingsManager.getSelectedAIModel();
        const apiKey = await vscode.window.showInputBox({
            prompt: `Enter API Key for ${aiModel}`,
            password: true,
            ignoreFocusOut: true,
            placeHolder: 'API Key'
        });
        if (apiKey) {
            await settingsManager.storeApiKey(apiKey);
            vscode.window.showInformationMessage(`API Key for ${aiModel} has been stored securely.`);
        }
    });
    // Add commands to context subscriptions
    context.subscriptions.push(analyzeSelectionCommand, setApiKeyCommand);
}
// This method is called when your extension is deactivated
function deactivate() { }


/***/ }),
/* 1 */
/***/ ((module) => {

module.exports = require("vscode");

/***/ }),
/* 2 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.performSecurityAnalysis = performSecurityAnalysis;
exports.performSecurityAnalysisAsync = performSecurityAnalysisAsync;
const security_analyzer_ai_1 = __webpack_require__(3);
/**
 * Performs security analysis on the given code snippet
 * Can use both pattern-based detection and AI-based analysis
 * @param code The code to analyze
 * @param aiModel Optional parameter to specify which AI Model to use
 * @param apiKey Optional API key for AI Model (if not provided, only pattern-based analysis is done)
 * @returns Array of security issues found
 */
function performSecurityAnalysis(code, aiModel, apiKey) {
    // Log which AI Model would be used (for future implementation)
    console.log(`Using AI Model for analysis: ${aiModel || 'default'}`);
    // This is a pattern-based implementation. It doesn't require an API key.
    const issues = [];
    // Check for SQL injection vulnerability pattern
    if (code.toLowerCase().includes('sql') && code.includes("'") && code.includes("+")) {
        issues.push({
            title: "Potential SQL Injection",
            severity: "High",
            description: "String concatenation used with SQL queries can lead to SQL injection attacks.",
            recommendation: "Use parameterized queries or prepared statements instead of string concatenation."
        });
    }
    // Check for potential XSS vulnerability pattern
    if ((code.includes("innerHTML") || code.includes("document.write")) && code.includes("${")) {
        issues.push({
            title: "Potential Cross-Site Scripting (XSS)",
            severity: "High",
            description: "Directly inserting user input into DOM can lead to XSS attacks.",
            recommendation: "Use textContent instead of innerHTML, or sanitize user input before inserting into DOM."
        });
    }
    // Check for hardcoded secrets
    if (code.toLowerCase().includes("password") || code.toLowerCase().includes("token") || code.toLowerCase().includes("secret")) {
        if (code.includes("=") && (code.includes("'") || code.includes("\""))) {
            issues.push({
                title: "Hardcoded Secret",
                severity: "Medium",
                description: "Sensitive information appears to be hardcoded in the source code.",
                recommendation: "Use environment variables or a secure vault for storing sensitive information."
            });
        }
    }
    // Check for insecure random number generation
    if (code.includes("Math.random(") && (code.toLowerCase().includes("auth") ||
        code.toLowerCase().includes("token") ||
        code.toLowerCase().includes("password") ||
        code.toLowerCase().includes("secure"))) {
        issues.push({
            title: "Insecure Random Number Generation",
            severity: "Medium",
            description: "Using Math.random() for security-sensitive operations is not recommended.",
            recommendation: "Use a cryptographically secure random number generator like crypto.getRandomValues()."
        });
    }
    // Check for potential command injection
    if (code.includes("exec(") || code.includes("spawn(") || code.includes("system(")) {
        if (code.includes("+") || code.includes("`") || code.includes("${")) {
            issues.push({
                title: "Potential Command Injection",
                severity: "Critical",
                description: "Dynamic command execution can lead to command injection vulnerabilities.",
                recommendation: "Avoid using user input in command execution. If necessary, properly validate and sanitize the input."
            });
        }
    }
    // If no issues found but code contains security-sensitive patterns, add a mock issue (for demonstration)
    if (issues.length === 0 && (code.toLowerCase().includes("security") ||
        code.toLowerCase().includes("login") ||
        code.toLowerCase().includes("auth") ||
        code.toLowerCase().includes("password"))) {
        issues.push({
            title: "Potential Weak Authentication",
            severity: "Medium",
            description: "Authentication logic might not follow security best practices.",
            recommendation: "Implement multi-factor authentication and ensure proper password handling."
        });
    }
    return issues;
}
/**
 * Performs security analysis on the given code snippet asynchronously,
 * utilizing both pattern-based detection and AI-based analysis if an API key is provided
 * @param code The code to analyze
 * @param aiModel The AI Model to use
 * @param apiKey API key for the AI Model
 * @returns Promise with array of security issues found
 */
async function performSecurityAnalysisAsync(code, aiModel, apiKey) {
    // If no API key is provided, just return the pattern-based results
    if (!apiKey) {
        return [];
    }
    try {
        // Run the AI-based analysis
        const aiIssues = await (0, security_analyzer_ai_1.analyzeSecurityWithAI)(code, aiModel, apiKey);
        // Merge the results, removing any duplicates
        const allIssues = [];
        // Add AI issues that don't overlap with pattern issues
        for (const aiIssue of aiIssues) {
            allIssues.push(aiIssue);
        }
        return allIssues;
    }
    catch (error) {
        console.error('Error in AI-based analysis:', error);
        // If AI analysis fails, return just the pattern-based results
        return [];
    }
}


/***/ }),
/* 3 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.analyzeSecurityWithAI = analyzeSecurityWithAI;
exports.analyzeSecurityWithAIStreaming = analyzeSecurityWithAIStreaming;
const ai_client_factory_1 = __webpack_require__(4);
/**
 * Analyzes code for security issues using AI models
 * @param code The code to analyze
 * @param aiModel The AI model to use
 * @param apiKey The API key for the AI model
 * @returns Array of security issues found
 */
async function analyzeSecurityWithAI(code, aiModel, apiKey) {
    try {
        // Get the appropriate AI client
        const aiClient = ai_client_factory_1.AIClientFactory.getClient(aiModel);
        // Construct the prompt for security analysis
        const prompt = `
            You are a security expert analyzing code for vulnerabilities. 
            Review the following code for security issues and return a JSON array of issues found.
            Each issue should have the following format:
            {
                "title": "Issue title",
                "severity": "Low|Medium|High|Critical",
                "description": "Detailed description of the issue",
                "recommendation": "How to fix the issue"
            }
            
            If no issues are found, return an empty array.
            
            Here's the code to analyze:
            
            \`\`\`
            ${code}
            \`\`\`
            
            Please provide your response as valid JSON only, with no additional text.
        `;
        // Send the request to the AI model
        const response = await aiClient.sendRequest(prompt, {
            apiKey,
            temperature: 0, // Lower temperature for more consistent results
            maxTokens: 2000 // Allow enough tokens for a detailed analysis
        });
        // Parse the response as JSON
        try {
            // Extract the JSON array from the response
            const responseText = response.content.trim();
            const jsonStartIndex = responseText.indexOf('[');
            const jsonEndIndex = responseText.lastIndexOf(']') + 1;
            if (jsonStartIndex !== -1 && jsonEndIndex !== -1) {
                const jsonStr = responseText.substring(jsonStartIndex, jsonEndIndex);
                const issues = JSON.parse(jsonStr);
                return issues;
            }
            // Fallback to parsing the entire response if JSON markers not found
            return JSON.parse(responseText);
        }
        catch (parseError) {
            console.error('Error parsing AI response:', parseError);
            return [{
                    title: 'Error Analyzing Code',
                    severity: 'Medium',
                    description: `The AI response could not be parsed: ${parseError}`,
                    recommendation: 'Try again or use a different AI model.'
                }];
        }
    }
    catch (error) {
        console.error('Error analyzing security with AI:', error);
        return [{
                title: 'AI Analysis Error',
                severity: 'Medium',
                description: `An error occurred while analyzing code with the AI model: ${error}`,
                recommendation: 'Check your API key and internet connection and try again.'
            }];
    }
}
/**
 * Analyzes code for security issues using AI models with streaming response
 * @param code The code to analyze
 * @param aiModel The AI model to use
 * @param apiKey The API key for the AI model
 * @param progressCallback Callback function for progress updates
 * @returns Array of security issues found
 */
async function analyzeSecurityWithAIStreaming(code, aiModel, apiKey, progressCallback) {
    return new Promise((resolve, reject) => {
        try {
            // Get the appropriate AI client
            const aiClient = ai_client_factory_1.AIClientFactory.getClient(aiModel);
            // Construct the prompt for security analysis
            const prompt = `
                You are a security expert analyzing code for vulnerabilities. 
                Review the following code for security issues and return a JSON array of issues found.
                Each issue should have the following format:
                {
                    "title": "Issue title",
                    "severity": "Low|Medium|High|Critical",
                    "description": "Detailed description of the issue",
                    "recommendation": "How to fix the issue"
                }
                
                If no issues are found, return an empty array.
                
                Here's the code to analyze:
                
                \`\`\`
                ${code}
                \`\`\`
                
                Please provide your response as valid JSON only, with no additional text.
            `;
            let responseText = '';
            // Send the streaming request to the AI model
            aiClient.sendStreamingRequest(prompt, (chunk) => {
                responseText = chunk.content;
                progressCallback(chunk.content);
                if (chunk.isComplete) {
                    try {
                        // Extract the JSON array from the response
                        const jsonStartIndex = responseText.indexOf('[');
                        const jsonEndIndex = responseText.lastIndexOf(']') + 1;
                        if (jsonStartIndex !== -1 && jsonEndIndex !== -1) {
                            const jsonStr = responseText.substring(jsonStartIndex, jsonEndIndex);
                            const issues = JSON.parse(jsonStr);
                            resolve(issues);
                            return;
                        }
                        // Fallback to parsing the entire response if JSON markers not found
                        const issues = JSON.parse(responseText);
                        resolve(issues);
                    }
                    catch (parseError) {
                        console.error('Error parsing AI response:', parseError);
                        resolve([{
                                title: 'Error Analyzing Code',
                                severity: 'Medium',
                                description: `The AI response could not be parsed: ${parseError}`,
                                recommendation: 'Try again or use a different AI model.'
                            }]);
                    }
                }
            }, {
                apiKey,
                temperature: 0.3,
                maxTokens: 2000
            }).catch(error => {
                console.error('Error in streaming request:', error);
                reject(error);
            });
        }
        catch (error) {
            console.error('Error analyzing security with AI streaming:', error);
            resolve([{
                    title: 'AI Analysis Error',
                    severity: 'Medium',
                    description: `An error occurred while analyzing code with the AI model: ${error}`,
                    recommendation: 'Check your API key and internet connection and try again.'
                }]);
        }
    });
}


/***/ }),
/* 4 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.AIClientFactory = void 0;
const claude_client_1 = __webpack_require__(5);
const gemini_client_1 = __webpack_require__(10);
const openai_client_1 = __webpack_require__(11);
/**
 * Factory class for creating AI clients
 */
class AIClientFactory {
    /**
     * Get the appropriate AI client based on the model
     * @param model The AI model to use
     * @returns The AI client for the specified model
     */
    static getClient(model) {
        switch (model) {
            case 'openai':
                return new openai_client_1.OpenAIClient();
            case 'claude':
                return new claude_client_1.ClaudeClient();
            case 'gemini':
                return new gemini_client_1.GeminiClient();
            case 'claude-3-5-sonnet-20241022':
                return new claude_client_1.ClaudeClient();
            default:
                throw new Error(`Unsupported AI model: ${model}`);
        }
    }
}
exports.AIClientFactory = AIClientFactory;


/***/ }),
/* 5 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.ClaudeClient = void 0;
const http_client_1 = __webpack_require__(6);
class ClaudeClient extends http_client_1.HttpClient {
    static API_URL = 'https://api.anthropic.com/v1/messages';
    defaultModel = 'claude-3-5-sonnet-20241022';
    /**
     * Send a request to the Anthropic Claude API
     * @param prompt The prompt to send
     * @param options Claude-specific options
     * @returns The AI response
     */
    async sendRequest(prompt, options) {
        if (!options?.apiKey) {
            throw new Error('Anthropic Claude API key is required');
        }
        const response = await this.post(ClaudeClient.API_URL, {
            model: options.model || this.defaultModel,
            messages: [{ role: 'user', content: prompt }],
            temperature: options.temperature || 0,
            max_tokens: options.maxTokens || 500,
            stream: false
        }, {
            'x-api-key': options.apiKey,
            'anthropic-version': '2023-06-01', // Use appropriate API version
            'Content-Type': 'application/json'
        });
        // Extract text content from Claude's response
        const content = response.content
            .filter(item => item.type === 'text')
            .map(item => item.text)
            .join('');
        return {
            content,
            model: response.model,
            provider: 'claude'
        };
    }
    /**
     * Send a streaming request to the Anthropic Claude API
     * @param prompt The prompt to send
     * @param callback Callback function for each chunk
     * @param options Claude-specific options
     */
    async sendStreamingRequest(prompt, callback, options) {
        if (!options?.apiKey) {
            throw new Error('Anthropic Claude API key is required');
        }
        let contentSoFar = '';
        await this.streamingPost(ClaudeClient.API_URL, {
            model: options.model || this.defaultModel,
            messages: [{ role: 'user', content: prompt }],
            temperature: options.temperature || 0.7,
            max_tokens: options.maxTokens || 500,
            stream: true
        }, (chunk) => {
            try {
                // Claude also uses SSE format with data: prefix
                const lines = chunk.split('\n').filter(line => line.trim() !== '');
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6); // Remove 'data: ' prefix
                        if (data === '[DONE]') {
                            callback({ content: contentSoFar, isComplete: true });
                            return;
                        }
                        try {
                            const parsed = JSON.parse(data);
                            if (parsed.type === 'content_block_delta' &&
                                parsed.delta &&
                                parsed.delta.text) {
                                const content = parsed.delta.text;
                                contentSoFar += content;
                                callback({ content: contentSoFar, isComplete: false });
                            }
                        }
                        catch (e) {
                            console.error('Error parsing SSE data:', e);
                        }
                    }
                }
            }
            catch (error) {
                console.error('Error processing chunk:', error);
            }
        }, () => {
            // Final completion callback
            callback({ content: contentSoFar, isComplete: true });
        }, {
            'x-api-key': options.apiKey,
            'anthropic-version': '2023-06-01',
            'Content-Type': 'application/json',
            'Accept': 'text/event-stream'
        });
    }
}
exports.ClaudeClient = ClaudeClient;


/***/ }),
/* 6 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.HttpClient = void 0;
const https = __importStar(__webpack_require__(7));
const http = __importStar(__webpack_require__(8));
const url_1 = __webpack_require__(9);
/**
 * Base HTTP client for making requests to AI APIs
 */
class HttpClient {
    /**
     * Make a GET request
     * @param url The URL to request
     * @param headers Additional headers to include
     * @returns The response data
     */
    async get(url, headers = {}) {
        return this.request(url, 'GET', headers);
    }
    /**
     * Make a POST request
     * @param url The URL to request
     * @param body The request body
     * @param headers Additional headers to include
     * @returns The response data
     */
    async post(url, body, headers = {}) {
        return this.request(url, 'POST', headers, body);
    }
    /**
     * Make a streaming POST request
     * @param url The URL to request
     * @param body The request body
     * @param onChunk Callback for each chunk of data
     * @param headers Additional headers to include
     */
    async streamingPost(url, body, onChunk, onComplete, headers = {}) {
        const parsedUrl = new url_1.URL(url);
        const options = {
            hostname: parsedUrl.hostname,
            port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
            path: parsedUrl.pathname + parsedUrl.search,
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...headers
            }
        };
        return new Promise((resolve, reject) => {
            const req = (parsedUrl.protocol === 'https:' ? https : http).request(options, (res) => {
                res.on('data', (chunk) => {
                    try {
                        onChunk(chunk.toString());
                    }
                    catch (error) {
                        console.error('Error processing chunk:', error);
                    }
                });
                res.on('end', () => {
                    onComplete();
                    resolve();
                });
                res.on('error', (error) => {
                    reject(error);
                });
            });
            req.on('error', (error) => {
                reject(error);
            });
            req.write(JSON.stringify(body));
            req.end();
        });
    }
    /**
     * Make a generic HTTP request
     * @param url The URL to request
     * @param method The HTTP method
     * @param headers Additional headers to include
     * @param body The request body (for POST, PUT, etc.)
     * @returns The response data
     */
    async request(url, method, headers = {}, body) {
        const parsedUrl = new url_1.URL(url);
        const options = {
            hostname: parsedUrl.hostname,
            port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
            path: parsedUrl.pathname + parsedUrl.search,
            method,
            headers: {
                'Content-Type': 'application/json',
                ...headers
            }
        };
        return new Promise((resolve, reject) => {
            const req = (parsedUrl.protocol === 'https:' ? https : http).request(options, (res) => {
                let data = '';
                res.on('data', (chunk) => {
                    data += chunk;
                });
                res.on('end', () => {
                    try {
                        if (res.statusCode && (res.statusCode < 200 || res.statusCode >= 300)) {
                            reject(new Error(`Request failed with status code ${res.statusCode}: ${data}`));
                            return;
                        }
                        try {
                            const jsonData = JSON.parse(data);
                            resolve(jsonData);
                        }
                        catch (e) {
                            // If it's not JSON, return the raw data
                            resolve(data);
                        }
                    }
                    catch (e) {
                        reject(e);
                    }
                });
                res.on('error', (error) => {
                    reject(error);
                });
            });
            req.on('error', (error) => {
                reject(error);
            });
            if (body) {
                req.write(JSON.stringify(body));
            }
            req.end();
        });
    }
}
exports.HttpClient = HttpClient;


/***/ }),
/* 7 */
/***/ ((module) => {

module.exports = require("https");

/***/ }),
/* 8 */
/***/ ((module) => {

module.exports = require("http");

/***/ }),
/* 9 */
/***/ ((module) => {

module.exports = require("url");

/***/ }),
/* 10 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.GeminiClient = void 0;
const http_client_1 = __webpack_require__(6);
class GeminiClient extends http_client_1.HttpClient {
    static API_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models';
    defaultModel = 'gemini-pro';
    /**
     * Send a request to the Google Gemini API
     * @param prompt The prompt to send
     * @param options Gemini-specific options
     * @returns The AI response
     */
    async sendRequest(prompt, options) {
        if (!options?.apiKey) {
            throw new Error('Google Gemini API key is required');
        }
        const model = options.model || this.defaultModel;
        const url = `${GeminiClient.API_BASE_URL}/${model}:generateContent?key=${options.apiKey}`;
        const response = await this.post(url, {
            contents: [
                {
                    parts: [
                        { text: prompt }
                    ]
                }
            ],
            generationConfig: {
                temperature: options.temperature || 0.7,
                maxOutputTokens: options.maxTokens || 500,
            }
        }, {
            'Content-Type': 'application/json'
        });
        // Extract text from Gemini's response
        const content = response.candidates[0]?.content?.parts
            .map(part => part.text)
            .join('') || '';
        return {
            content,
            model: model,
            provider: 'gemini'
        };
    }
    /**
     * Send a streaming request to the Google Gemini API
     * @param prompt The prompt to send
     * @param callback Callback function for each chunk
     * @param options Gemini-specific options
     */
    async sendStreamingRequest(prompt, callback, options) {
        if (!options?.apiKey) {
            throw new Error('Google Gemini API key is required');
        }
        const model = options.model || this.defaultModel;
        const url = `${GeminiClient.API_BASE_URL}/${model}:streamGenerateContent?key=${options.apiKey}`;
        let contentSoFar = '';
        await this.streamingPost(url, {
            contents: [
                {
                    parts: [
                        { text: prompt }
                    ]
                }
            ],
            generationConfig: {
                temperature: options.temperature || 0.7,
                maxOutputTokens: options.maxTokens || 500,
            }
        }, (chunk) => {
            try {
                // Gemini may return multiple JSON objects in a single chunk
                const jsonObjects = this.parseStreamChunk(chunk);
                for (const obj of jsonObjects) {
                    if (obj.candidates && obj.candidates[0]?.content?.parts) {
                        const parts = obj.candidates[0].content.parts;
                        for (const part of parts) {
                            if (part.text) {
                                contentSoFar += part.text;
                                callback({ content: contentSoFar, isComplete: false });
                            }
                        }
                    }
                }
            }
            catch (error) {
                console.error('Error processing chunk:', error);
            }
        }, () => {
            // Final completion callback
            callback({ content: contentSoFar, isComplete: true });
        }, {
            'Content-Type': 'application/json'
        });
    }
    /**
     * Parse multiple JSON objects from a stream chunk
     * @param chunk Raw stream chunk that may contain multiple JSON objects
     * @returns Array of parsed JSON objects
     */
    parseStreamChunk(chunk) {
        const results = [];
        // Split by newlines and attempt to parse each line as JSON
        const lines = chunk.split('\n');
        for (const line of lines) {
            if (line.trim()) {
                try {
                    const parsed = JSON.parse(line);
                    results.push(parsed);
                }
                catch (e) {
                    console.warn('Failed to parse JSON line:', line);
                }
            }
        }
        return results;
    }
}
exports.GeminiClient = GeminiClient;


/***/ }),
/* 11 */
/***/ ((__unused_webpack_module, exports, __webpack_require__) => {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.OpenAIClient = void 0;
const http_client_1 = __webpack_require__(6);
class OpenAIClient extends http_client_1.HttpClient {
    static API_URL = 'https://api.openai.com/v1/chat/completions';
    defaultModel = 'gpt-3.5-turbo';
    /**
     * Send a request to the OpenAI API
     * @param prompt The prompt to send
     * @param options OpenAI-specific options
     * @returns The AI response
     */
    async sendRequest(prompt, options) {
        if (!options?.apiKey) {
            throw new Error('OpenAI API key is required');
        }
        const response = await this.post(OpenAIClient.API_URL, {
            model: options.model || this.defaultModel,
            messages: [{ role: 'user', content: prompt }],
            temperature: options.temperature || 0.7,
            max_tokens: options.maxTokens || 500,
            stream: false
        }, {
            'Authorization': `Bearer ${options.apiKey}`,
            'Content-Type': 'application/json'
        });
        return {
            content: response.choices[0].message.content,
            model: response.model,
            provider: 'openai'
        };
    }
    /**
     * Send a streaming request to the OpenAI API
     * @param prompt The prompt to send
     * @param callback Callback function for each chunk
     * @param options OpenAI-specific options
     */
    async sendStreamingRequest(prompt, callback, options) {
        if (!options?.apiKey) {
            throw new Error('OpenAI API key is required');
        }
        let contentSoFar = '';
        await this.streamingPost(OpenAIClient.API_URL, {
            model: options.model || this.defaultModel,
            messages: [{ role: 'user', content: prompt }],
            temperature: options.temperature || 0.7,
            max_tokens: options.maxTokens || 500,
            stream: true
        }, (chunk) => {
            try {
                // OpenAI sends "data: " prefixed SSE
                const lines = chunk.split('\n').filter(line => line.trim() !== '');
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6); // Remove 'data: ' prefix
                        if (data === '[DONE]') {
                            callback({ content: contentSoFar, isComplete: true });
                            return;
                        }
                        try {
                            const parsed = JSON.parse(data);
                            if (parsed.choices && parsed.choices[0]?.delta?.content) {
                                const content = parsed.choices[0].delta.content;
                                contentSoFar += content;
                                callback({ content: contentSoFar, isComplete: false });
                            }
                        }
                        catch (e) {
                            console.error('Error parsing SSE data:', e);
                        }
                    }
                }
            }
            catch (error) {
                console.error('Error processing chunk:', error);
            }
        }, () => {
            // Final completion callback
            callback({ content: contentSoFar, isComplete: true });
        }, {
            'Authorization': `Bearer ${options.apiKey}`,
            'Content-Type': 'application/json',
            'Accept': 'text/event-stream'
        });
    }
}
exports.OpenAIClient = OpenAIClient;


/***/ }),
/* 12 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.getGitChanges = getGitChanges;
exports.registerSecureFlowReviewCommand = registerSecureFlowReviewCommand;
const vscode = __importStar(__webpack_require__(1));
const cp = __importStar(__webpack_require__(13));
const path = __importStar(__webpack_require__(14));
const security_analyzer_1 = __webpack_require__(2);
/**
 * Gets the git changes (hunks) for a specific file or all files in the workspace
 * @param filePath Optional path to a specific file
 * @returns Array of change information objects
 */
async function getGitChanges(filePath) {
    try {
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders || workspaceFolders.length === 0) {
            throw new Error('No workspace folder found');
        }
        const repoPath = workspaceFolders[0].uri.fsPath;
        const changes = [];
        // Construct the git diff command
        let command = 'git diff --unified=0';
        if (filePath) {
            const relativePath = path.relative(repoPath, filePath);
            command += ` -- "${relativePath}"`;
        }
        // Execute git command
        const output = await executeCommand(command, repoPath);
        // Parse the git diff output
        let currentFile = null;
        const lines = output.split('\n');
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            // Check for file header
            if (line.startsWith('diff --git')) {
                const match = line.match(/diff --git a\/(.*) b\/(.*)/);
                if (match && match[2]) {
                    currentFile = match[2];
                }
                continue;
            }
            // Check for hunk header
            if (line.startsWith('@@') && currentFile) {
                const match = line.match(/@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/);
                if (match) {
                    const startLine = parseInt(match[3], 10);
                    const lineCount = match[4] ? parseInt(match[4], 10) : 1;
                    // Collect the changed lines content
                    let content = '';
                    let j = i + 1;
                    let collectedLines = 0;
                    while (j < lines.length && collectedLines < lineCount) {
                        const nextLine = lines[j];
                        if (!nextLine.startsWith('-') &&
                            !nextLine.startsWith('diff --git') &&
                            !nextLine.startsWith('@@')) {
                            // Include the line if it's an addition or context line
                            if (nextLine.startsWith('+')) {
                                content += nextLine.substring(1) + '\n';
                                collectedLines++;
                            }
                            else {
                                content += nextLine + '\n';
                                collectedLines++;
                            }
                        }
                        j++;
                    }
                    changes.push({
                        filePath: path.join(repoPath, currentFile),
                        startLine,
                        lineCount,
                        content: content.trim()
                    });
                }
            }
        }
        return changes;
    }
    catch (error) {
        console.error('Error getting git changes:', error);
        return [];
    }
}
/**
 * Executes a shell command and returns the output
 * @param command Command to execute
 * @param cwd Current working directory
 * @returns Command output as string
 */
async function executeCommand(command, cwd) {
    return new Promise((resolve, reject) => {
        cp.exec(command, { cwd }, (error, stdout, stderr) => {
            if (error) {
                reject(error);
                return;
            }
            resolve(stdout);
        });
    });
}
/**
 * Registers the SecureFlow review command for git changes
 * @param context VSCode extension context
 * @param outputChannel Output channel for displaying results
 * @param settingsManager Settings manager for the extension
 */
function registerSecureFlowReviewCommand(context, outputChannel, settingsManager) {
    // Create status bar item
    const statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.text = "$(shield) SecureFlow";
    statusBarItem.tooltip = "Scan git changes for security issues";
    statusBarItem.command = "secureflow.reviewChanges";
    statusBarItem.show();
    // Register command
    const reviewCommand = vscode.commands.registerCommand("secureflow.reviewChanges", async (uri) => {
        // Get the file path from the URI if provided (from SCM view)
        let currentFilePath;
        if (uri && uri.fsPath) {
            // If command was triggered from SCM view with a specific file
            currentFilePath = uri.fsPath;
        }
        else {
            // Otherwise use the active editor file
            const editor = vscode.window.activeTextEditor;
            currentFilePath = editor?.document.uri.fsPath;
        }
        try {
            // Show progress indicator
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: "SecureFlow: Scanning git changes...",
                cancellable: true
            }, async (progress, token) => {
                // Clear and show output channel
                outputChannel.clear();
                outputChannel.show(true);
                outputChannel.appendLine('🔍 SecureFlow: Scanning git changes for security issues...\n');
                // Get the selected AI Model
                const selectedModel = settingsManager.getSelectedAIModel();
                outputChannel.appendLine(`🤖 Using AI Model: ${selectedModel}`);
                // Report progress
                progress.report({ increment: 0 });
                // Get git changes
                outputChannel.appendLine('⏳ Collecting git changes...');
                const changes = await getGitChanges(currentFilePath);
                if (changes.length === 0) {
                    outputChannel.appendLine('\n⚠️ No git changes found to scan.');
                    vscode.window.showInformationMessage('SecureFlow: No git changes found to scan.');
                    return;
                }
                // Report progress
                progress.report({ increment: 30, message: "Analyzing changes..." });
                // Print file and change information
                outputChannel.appendLine(`\n📄 Found ${changes.length} changed chunks in ${new Set(changes.map(c => c.filePath)).size} files\n`);
                let allIssues = [];
                // Analyze each change
                for (let i = 0; i < changes.length; i++) {
                    const change = changes[i];
                    outputChannel.appendLine(`File: ${path.basename(change.filePath)}`);
                    outputChannel.appendLine(`Lines: ${change.startLine}-${change.startLine + change.lineCount - 1}`);
                    outputChannel.appendLine(`Changes:\n${change.content}\n`);
                    // Analyze the change content with the selected AI Model
                    // For now, we'll use the synchronous pattern-based analysis
                    // In a future update, this would use the async AI-powered analysis
                    const issues = await (0, security_analyzer_1.performSecurityAnalysisAsync)(change.content, selectedModel, await settingsManager.getApiKey());
                    // Map issues to include file path and line number
                    const mappedIssues = issues.map((issue) => ({
                        issue,
                        filePath: change.filePath,
                        startLine: change.startLine
                    }));
                    allIssues = [...allIssues, ...mappedIssues];
                    // Update progress
                    progress.report({
                        increment: 60 / changes.length,
                        message: `Analyzed ${i + 1}/${changes.length} chunks...`
                    });
                }
                // Finalize progress
                progress.report({ increment: 10, message: "Finalizing scan..." });
                await new Promise(resolve => setTimeout(resolve, 500));
                // Display results
                outputChannel.appendLine('\n✅ Security scan complete!\n');
                if (allIssues.length === 0) {
                    outputChannel.appendLine('🎉 No security issues found in the scanned changes.');
                    vscode.window.showInformationMessage('SecureFlow: No security issues found in the scanned changes.');
                }
                else {
                    outputChannel.appendLine(`⚠️ Found ${allIssues.length} potential security issues:\n`);
                    allIssues.forEach((item, index) => {
                        const { issue, filePath, startLine } = item;
                        outputChannel.appendLine(`Issue #${index + 1}: ${issue.title}`);
                        outputChannel.appendLine(`File: ${path.basename(filePath)}`);
                        outputChannel.appendLine(`Location: Line ${startLine}`);
                        outputChannel.appendLine(`Severity: ${issue.severity}`);
                        outputChannel.appendLine(`Description: ${issue.description}`);
                        outputChannel.appendLine(`Recommendation: ${issue.recommendation}\n`);
                    });
                    // Show notification
                    vscode.window.showWarningMessage(`SecureFlow: Found ${allIssues.length} security ${allIssues.length === 1 ? 'issue' : 'issues'} in your code changes.`, 'View Details').then(selection => {
                        if (selection === 'View Details') {
                            outputChannel.show(true);
                        }
                    });
                }
            });
        }
        catch (error) {
            console.error('Error during security review:', error);
            vscode.window.showErrorMessage(`SecureFlow: Error during security review: ${error}`);
        }
    });
    // Add to subscriptions
    context.subscriptions.push(statusBarItem, reviewCommand);
}


/***/ }),
/* 13 */
/***/ ((module) => {

module.exports = require("child_process");

/***/ }),
/* 14 */
/***/ ((module) => {

module.exports = require("path");

/***/ }),
/* 15 */
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.SettingsManager = void 0;
const vscode = __importStar(__webpack_require__(1));
/**
 * Settings manager for SecureFlow extension
 */
class SettingsManager {
    context;
    constructor(context) {
        this.context = context;
    }
    /**
     * Get the selected AI Model from user preferences
     */
    getSelectedAIModel() {
        const config = vscode.workspace.getConfiguration('secureflow');
        return config.get('AIModel') || 'openai';
    }
    /**
     * Get the API Key for the selected model
     */
    async getApiKey() {
        const key = `secureflow.APIKey`;
        // Try to get the key from secure storage first
        let apiKey = await this.context.secrets.get(key);
        // If not found in secure storage, check if it's in settings
        if (!apiKey) {
            const config = vscode.workspace.getConfiguration('secureflow');
            const configKey = config.get('APIKey');
            // If found in settings, store it securely and clear from settings
            if (configKey) {
                await this.context.secrets.store(key, configKey);
                // Clear the key from settings to keep it secure
                await config.update('APIKey', '', vscode.ConfigurationTarget.Global);
                apiKey = configKey;
            }
        }
        return apiKey;
    }
    /**
     * Store API Key securely
     */
    async storeApiKey(apiKey) {
        const key = `secureflow.APIKey`;
        await this.context.secrets.store(key, apiKey);
    }
}
exports.SettingsManager = SettingsManager;


/***/ })
/******/ 	]);
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			// no module.id needed
/******/ 			// no module.loaded needed
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	// This entry module is referenced by other modules so it can't be inlined
/******/ 	var __webpack_exports__ = __webpack_require__(0);
/******/ 	module.exports = __webpack_exports__;
/******/ 	
/******/ })()
;
//# sourceMappingURL=extension.js.map